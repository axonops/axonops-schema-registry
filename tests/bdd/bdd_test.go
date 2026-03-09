//go:build bdd

// Package bdd provides BDD tests using godog (Cucumber for Go).
//
// In-process (fast, memory backend, no Docker):
//
//	go test -tags bdd -v ./tests/bdd/...
//
// Docker-based (all tests including operational):
//
//	BDD_BACKEND=memory go test -tags bdd -v -timeout 10m ./tests/bdd/...
//	BDD_BACKEND=postgres go test -tags bdd -v -timeout 15m ./tests/bdd/...
package bdd

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/kms"
	openbaokms "github.com/axonops/axonops-schema-registry/internal/kms/openbao"
	vaultkms "github.com/axonops/axonops-schema-registry/internal/kms/vault"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	cassandrastore "github.com/axonops/axonops-schema-registry/internal/storage/cassandra"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
	mysqlstore "github.com/axonops/axonops-schema-registry/internal/storage/mysql"
	pgstore "github.com/axonops/axonops-schema-registry/internal/storage/postgres"
	"github.com/axonops/axonops-schema-registry/tests/bdd/steps"
)

var (
	dockerMode   bool
	registryURL  string
	metricsURL   string // separate metrics endpoint (e.g. JMX exporter sidecar for Confluent)
	webhookURL   string
	backend      string
	bddStorage   string // BDD_STORAGE: "postgres", "mysql", "cassandra" for in-process DB tests
	composeFiles []string
	containerCmd string // "podman" or "docker"

	// sharedDBStore is a long-lived database storage used for in-process BDD tests
	// against real database backends. Created once in TestMain when BDD_STORAGE is set.
	// All scenario types (functional, MCP, audit, auth, KMS) use this store when available.
	sharedDBStore storage.Storage

	// cassandraSession is a long-lived session reused across all BDD scenario cleanups.
	// gocql sessions are expensive to create (topology discovery, connection pool setup),
	// so we create one at first use and close it in TestMain.
	cassandraSession *gocql.Session
)

func TestMain(m *testing.M) {
	backend = os.Getenv("BDD_BACKEND")
	bddStorage = os.Getenv("BDD_STORAGE")
	registryURL = os.Getenv("BDD_REGISTRY_URL")
	webhookURL = os.Getenv("BDD_WEBHOOK_URL")

	// If BDD_BACKEND is set but no external URL, start Docker Compose automatically.
	if backend != "" && registryURL == "" {
		dockerMode = true
		containerCmd = findContainerCmd()
		composeFiles = composeFilesForBackend(backend)

		log.Printf("Starting %s compose for %s backend...", containerCmd, backend)
		if err := composeUp(composeFiles); err != nil {
			log.Fatalf("Failed to start compose: %v", err)
		}

		registryURL = fmt.Sprintf("http://localhost:%s", envOrDefault("REGISTRY_PORT", "18081"))
		webhookURL = fmt.Sprintf("http://localhost:%s", envOrDefault("WEBHOOK_PORT", "19000"))

		// Confluent: JMX exporter sidecar exposes metrics on a separate port.
		if backend == "confluent" {
			metricsURL = fmt.Sprintf("http://localhost:%s/metrics", envOrDefault("JMX_METRICS_PORT", "19090"))
		}

		waitTimeout := 120 * time.Second
		if backend == "confluent" {
			waitTimeout = 180 * time.Second // Kafka + SR startup takes longer
		}
		log.Printf("Waiting for registry at %s ...", registryURL)
		if err := waitForURL(registryURL+"/", waitTimeout); err != nil {
			composeLogs(composeFiles)
			composeDown(composeFiles)
			log.Fatalf("Registry did not become healthy: %v", err)
		}
		log.Println("Registry is healthy.")
	}

	// Create shared database store for in-process tests with database backends.
	// When BDD_STORAGE is set (e.g., "postgres"), ALL in-process scenarios run
	// using the specified database instead of memory storage. This enables MCP,
	// audit, auth, and functional tests to run against real databases.
	if bddStorage != "" && bddStorage != "memory" && !dockerMode {
		var err error
		sharedDBStore, err = createDBStore(bddStorage)
		if err != nil {
			log.Fatalf("Failed to create %s storage for BDD_STORAGE: %v", bddStorage, err)
		}
		log.Printf("Using %s storage for in-process BDD tests", bddStorage)
	}

	code := m.Run()

	// Close shared database store before tearing down containers.
	if sharedDBStore != nil {
		sharedDBStore.Close()
		sharedDBStore = nil
	}

	// Close the long-lived Cassandra session before tearing down containers.
	if cassandraSession != nil {
		cassandraSession.Close()
		cassandraSession = nil
	}

	if dockerMode {
		log.Println("Stopping compose...")
		composeDown(composeFiles)
	}

	os.Exit(code)
}

// newTestServer creates a fresh in-process schema registry backed by memory storage.
func newTestServer() (*httptest.Server, storage.Storage, *registry.Registry, *metrics.Metrics, chan struct{}) {
	return newTestServerWithAudit(nil)
}

// newTestServerWithAudit creates a fresh in-process schema registry with optional audit logging.
func newTestServerWithAudit(al *auth.AuditLogger) (*httptest.Server, storage.Storage, *registry.Registry, *metrics.Metrics, chan struct{}) {
	return newTestServerWithStoreAndAudit(memory.NewStore(), al)
}

// newTestServerWithStore creates an in-process schema registry using the provided storage backend.
// This enables BDD tests (MCP, audit, analysis, functional) to run against database backends.
func newTestServerWithStore(store storage.Storage) (*httptest.Server, storage.Storage, *registry.Registry, *metrics.Metrics, chan struct{}) {
	return newTestServerWithStoreAndAudit(store, nil)
}

// newTestServerWithStoreAndAudit creates an in-process schema registry with the provided storage
// and optional audit logging. This is the core factory used by all non-auth test server constructors.
// Returns a gaugeStop channel that MUST be closed when the server is torn down.
func newTestServerWithStoreAndAudit(store storage.Storage, al *auth.AuditLogger) (*httptest.Server, storage.Storage, *registry.Registry, *metrics.Metrics, chan struct{}) {
	m := metrics.New()

	// Wrap storage with instrumentation for storage metrics
	instrumentedStore := storage.NewInstrumentedStorage(store, "memory", m)

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(instrumentedStore, schemaRegistry, compatChecker, "BACKWARD")

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0, DocsEnabled: true},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	var opts []api.ServerOption
	opts = append(opts, api.WithMetrics(m))
	if al != nil {
		opts = append(opts, api.WithAuditLogger(al))
	}
	server := api.NewServer(cfg, reg, logger, opts...)

	// Start gauge refresh with 1-second interval for tests
	gaugeStop := make(chan struct{})
	m.StartGaugeRefresh(reg, 1*time.Second, gaugeStop)

	return httptest.NewServer(server), store, reg, m, gaugeStop
}

// newAuthTestServer creates an in-process schema registry with authentication enabled.
// It pre-seeds a super_admin user ("admin" / "admin-password") for testing.
func newAuthTestServer() (*httptest.Server, storage.Storage, *auth.Service) {
	return newAuthTestServerWithStore(memory.NewStore())
}

// newAuthTestServerWithStore creates an in-process schema registry with authentication enabled
// using the provided storage backend. This enables auth BDD tests to run against database backends.
func newAuthTestServerWithStore(store storage.Storage) (*httptest.Server, storage.Storage, *auth.Service) {
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	// Configure auth with basic + api_key methods
	authCfg := config.AuthConfig{
		Enabled: true,
		Methods: []string{"basic", "api_key"},
		APIKey: config.APIKeyConfig{
			Header: "X-API-Key",
		},
		RBAC: config.RBACConfig{
			Enabled:     true,
			DefaultRole: "readonly",
		},
	}

	// Configure rate limiting for BDD rate-limiting scenarios.
	// Low burst (3) and rate (2/s) ensure 429s trigger reliably even when
	// CI runners are slow and serial HTTP round-trips take 50-70ms each.
	rateLimitCfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 2,
		BurstSize:         3,
		PerClient:         false,
		PerEndpoint:       false,
	}

	authenticator := auth.NewAuthenticator(authCfg)
	authService := auth.NewServiceWithConfig(store, auth.ServiceConfig{
		CacheRefreshInterval: 5 * time.Second,  // enable API key cache for BDD metrics
		UserCacheTTL:         10 * time.Second, // enable credential caching for BDD metrics
	})
	authenticator.SetService(authService)

	// Create htpasswd file with test users for auth_htpasswd.feature
	htpasswdFile := createTestHTPasswdFile()
	if htpasswdFile != "" {
		htpasswdStore, err := auth.LoadHTPasswdFile(htpasswdFile)
		if err != nil {
			panic(fmt.Sprintf("failed to load test htpasswd file: %v", err))
		}
		authenticator.SetHTPasswdStore(htpasswdStore)
	}

	// Create memory API key store for auth_apikey_memory.feature
	memAPIKeys := createTestMemoryAPIKeys()
	if memAPIKeys != nil {
		authenticator.SetMemoryAPIKeyStore(memAPIKeys)
	}

	authorizer := auth.NewAuthorizer(authCfg.RBAC)

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
		Security: config.SecurityConfig{
			Auth: authCfg,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	m := metrics.New()
	m.EnablePrincipalMetrics()

	server := api.NewServer(cfg, reg, logger,
		api.WithMetrics(m),
		api.WithAuth(authenticator, authorizer, authService),
		api.WithRateLimiter(auth.NewRateLimiter(rateLimitCfg)),
	)

	// Pre-seed a super_admin user
	_, err := authService.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "admin",
		Password: "admin-password",
		Role:     "super_admin",
		Enabled:  true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create seed admin user: %v", err))
	}

	return httptest.NewServer(server), store, authService
}

// newKMSTestServer creates an in-process schema registry with KMS providers configured
// using memory storage. It reads KMS_VAULT_ADDR/KMS_VAULT_TOKEN and KMS_BAO_ADDR/KMS_BAO_TOKEN
// env vars to configure Vault and OpenBao KMS providers respectively.
func newKMSTestServer() (*httptest.Server, storage.Storage, *registry.Registry) {
	return newKMSTestServerWithStore(memory.NewStore())
}

// newKMSTestServerWithStore creates an in-process schema registry with KMS providers
// using the provided storage backend. This enables KMS tests (including MCP KMS tests)
// to run against database backends (PostgreSQL, MySQL, Cassandra) instead of just memory.
func newKMSTestServerWithStore(store storage.Storage) (*httptest.Server, storage.Storage, *registry.Registry) {
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	// Configure KMS providers from environment variables
	kmsReg := kms.NewRegistry()

	if addr, token := os.Getenv("KMS_VAULT_ADDR"), os.Getenv("KMS_VAULT_TOKEN"); addr != "" && token != "" {
		p, err := vaultkms.NewProvider(vaultkms.Config{Address: addr, Token: token})
		if err != nil {
			panic(fmt.Sprintf("failed to create Vault KMS provider: %v", err))
		}
		if err := kmsReg.Register(p); err != nil {
			panic(fmt.Sprintf("failed to register Vault KMS provider: %v", err))
		}
	}

	if addr, token := os.Getenv("KMS_BAO_ADDR"), os.Getenv("KMS_BAO_TOKEN"); addr != "" && token != "" {
		p, err := openbaokms.NewProvider(vaultkms.Config{Address: addr, Token: token})
		if err != nil {
			panic(fmt.Sprintf("failed to create OpenBao KMS provider: %v", err))
		}
		if err := kmsReg.Register(p); err != nil {
			panic(fmt.Sprintf("failed to register OpenBao KMS provider: %v", err))
		}
	}

	reg.SetKMSRegistry(kmsReg)

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0, DocsEnabled: true},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := api.NewServer(cfg, reg, logger)

	return httptest.NewServer(server), store, reg
}

// createDBStore creates a storage backend for in-process testing with an external database.
// The database must already be running (e.g., via Docker) before calling this function.
// Migrations are run automatically by the storage constructors.
func createDBStore(storageType string) (storage.Storage, error) {
	switch storageType {
	case "postgres":
		port, _ := strconv.Atoi(envOrDefault("POSTGRES_PORT", "15432"))
		return pgstore.NewStore(pgstore.Config{
			Host:     "localhost",
			Port:     port,
			Database: "schemaregistry",
			Username: "schemaregistry",
			Password: "schemaregistry",
			SSLMode:  "disable",
		})
	case "mysql":
		port, _ := strconv.Atoi(envOrDefault("MYSQL_PORT", "13306"))
		return mysqlstore.NewStore(mysqlstore.Config{
			Host:     "localhost",
			Port:     port,
			Database: "schemaregistry",
			Username: "schemaregistry",
			Password: "schemaregistry",
			TLS:      "false",
		})
	case "cassandra":
		port, _ := strconv.Atoi(envOrDefault("CASSANDRA_PORT", "19042"))
		return cassandrastore.NewStore(context.Background(), cassandrastore.Config{
			Hosts:       []string{"localhost"},
			Port:        port,
			Keyspace:    "schemaregistry",
			Consistency: "ONE",
			Migrate:     true,
		})
	default:
		return nil, fmt.Errorf("unknown BDD_STORAGE type: %s", storageType)
	}
}

// cleanDBStore cleans the shared database storage between scenarios.
// Reuses the existing cleanPostgres/cleanMySQL/cleanCassandra functions.
func cleanDBStore() error {
	switch bddStorage {
	case "postgres":
		return cleanPostgres()
	case "mysql":
		return cleanMySQL()
	case "cassandra":
		if err := cleanCassandra(); err != nil {
			return err
		}
		// Reset in-memory ID cache to match the re-seeded id_alloc table.
		// Without this, the block allocator serves stale IDs from a previous
		// scenario while id_alloc.next_id has been reset to 1.
		if cs, ok := sharedDBStore.(interface{ ResetIDCache() }); ok {
			cs.ResetIDCache()
		}
		return nil
	default:
		return nil
	}
}

func TestFeatures(t *testing.T) {
	// In-process: skip @operational (no Docker infrastructure).
	// Docker mode: run everything for this backend, skip other backends. BDD_TAGS overrides.
	tags := ""
	if envTags := os.Getenv("BDD_TAGS"); envTags != "" {
		tags = envTags
	} else if !dockerMode && registryURL == "" {
		tags = "~@operational && ~@pending-impl && ~@auth && ~@kms"
	} else if backend == "confluent" {
		// Confluent: exclude operational, import (our custom API), axonops-only, contexts (our multi-tenant),
		// pending-impl, data-contracts (ruleSet features require commercial Confluent license),
		// and all backend tags.
		tags = "~@operational && ~@import && ~@axonops-only && ~@contexts && ~@pending-impl && ~@data-contracts && ~@auth && ~@kms && ~@mcp && ~@analysis && ~@audit && ~@memory && ~@postgres && ~@mysql && ~@cassandra"
	} else if dockerMode {
		// Only run operational scenarios tagged for this backend, exclude other backends.
		// Auth tests are handled by TestAuthFeatures (separate compose stack).
		// MCP tests require in-process server, skip in Docker mode for now.
		// KMS tests are included when BDD_KMS=true (schema-registry has KMS providers configured).
		allBackends := []string{"memory", "postgres", "mysql", "cassandra"}
		excludes := []string{"~@pending-impl", "~@auth", "~@mcp", "~@audit"}
		if os.Getenv("BDD_KMS") != "true" {
			excludes = append(excludes, "~@kms")
		}
		for _, b := range allBackends {
			if b != backend {
				excludes = append(excludes, "~@"+b)
			}
		}
		tags = strings.Join(excludes, " && ")
	}

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     tags,
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			var tc *steps.TestContext

			if registryURL != "" {
				// Docker-based: use external registry, clean state before each scenario
				tc = steps.NewTestContext(registryURL)
				tc.MetricsURL = metricsURL
				tc.WebhookURL = webhookURL

				// Clean state before each scenario.
				// For operational scenarios, ensure the registry is running first
				// (a previous scenario may have killed or stopped it).
				ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
					if hasTag(sc, "@operational") {
						// Ensure registry is up (previous scenario may have stopped/killed it)
						if err := ensureRegistryRunning(); err != nil {
							return gctx, fmt.Errorf("ensure registry running: %w", err)
						}
					}
					if err := cleanBackend(); err != nil {
						return gctx, fmt.Errorf("clean backend: %w", err)
					}
					return gctx, waitForURL(registryURL+"/", 30*time.Second)
				})
			} else {
				// In-process: create fresh server per scenario.
				// tc must be allocated before step registration (steps capture the pointer).
				tc = steps.NewTestContext("http://placeholder")
				var ts *httptest.Server
				var st storage.Storage
				var m *metrics.Metrics
				var gaugeStop chan struct{}
				var storeOwned bool // false when using sharedDBStore (don't close in After)
				ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
					var reg *registry.Registry
					if hasTag(sc, "@kms") {
						if sharedDBStore != nil {
							// Using database backend: clean between scenarios, reuse store
							if err := cleanDBStore(); err != nil {
								return gctx, fmt.Errorf("clean %s storage: %w", bddStorage, err)
							}
							ts, st, reg = newKMSTestServerWithStore(sharedDBStore)
							storeOwned = false
						} else {
							ts, st, reg = newKMSTestServer()
							storeOwned = true
						}
					} else if hasTag(sc, "@audit") {
						// Wire audit buffer for REST audit assertions
						auditBuf := &bytes.Buffer{}
						al := auth.NewAuditLoggerWithWriter(config.AuditConfig{Enabled: true}, auditBuf)
						if sharedDBStore != nil {
							if err := cleanDBStore(); err != nil {
								return gctx, fmt.Errorf("clean %s storage: %w", bddStorage, err)
							}
							ts, st, reg, m, gaugeStop = newTestServerWithStoreAndAudit(sharedDBStore, al)
							storeOwned = false
						} else {
							ts, st, reg, m, gaugeStop = newTestServerWithAudit(al)
							storeOwned = true
						}
						tc.AuditBuffer = auditBuf
					} else {
						if sharedDBStore != nil {
							if err := cleanDBStore(); err != nil {
								return gctx, fmt.Errorf("clean %s storage: %w", bddStorage, err)
							}
							ts, st, reg, m, gaugeStop = newTestServerWithStore(sharedDBStore)
							storeOwned = false
						} else {
							ts, st, reg, m, gaugeStop = newTestServer()
							storeOwned = true
						}
					}
					tc.BaseURL = ts.URL
					tc.Registry = reg
					tc.StoredValues["_storage"] = st
					tc.StoredValues["_metrics"] = m
					return gctx, nil
				})
				ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
					if gaugeStop != nil {
						close(gaugeStop)
						gaugeStop = nil
					}
					if ts != nil {
						ts.Close()
					}
					if storeOwned && st != nil {
						st.Close()
					}
					return gctx, nil
				})
			}

			// Register step definitions
			steps.RegisterSchemaSteps(ctx, tc)
			steps.RegisterImportSteps(ctx, tc)
			steps.RegisterModeSteps(ctx, tc)
			steps.RegisterReferenceSteps(ctx, tc)
			steps.RegisterInfraSteps(ctx, tc)
			steps.RegisterAuthSteps(ctx, tc)
			steps.RegisterEncryptionSteps(ctx, tc)
			steps.RegisterConcurrencySteps(ctx, tc)
			steps.RegisterRateLimitSteps(ctx, tc)
			steps.RegisterMetricsSteps(ctx, tc)
			steps.RegisterMCPSteps(ctx, tc)
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("BDD tests failed")
	}
}

// TestAuthFeatures runs BDD tests that require authentication enabled.
// In Docker mode: starts a separate compose stack with auth config.
// In-process mode: creates a fresh in-process server with auth, RBAC, and a pre-seeded admin user.
func TestAuthFeatures(t *testing.T) {
	// Docker mode: start auth-specific compose stack
	var authURL string
	var authWebhook string
	var authFiles []string
	authDockerMode := dockerMode || os.Getenv("BDD_BACKEND") != ""

	if authDockerMode {
		if containerCmd == "" {
			containerCmd = findContainerCmd()
		}
		authFiles = []string{"docker-compose.base.yml", "docker-compose.auth.yml"}
		authEnv := []string{
			"REGISTRY_PORT=18082",
			"WEBHOOK_PORT=19001",
		}

		log.Printf("Starting auth compose stack...")
		if err := composeUpWithProject(authFiles, "bdd-auth", authEnv); err != nil {
			t.Fatalf("Failed to start auth compose: %v", err)
		}
		t.Cleanup(func() {
			log.Println("Stopping auth compose stack...")
			composeDownWithProject(authFiles, "bdd-auth")
		})

		authURL = "http://localhost:18082"
		authWebhook = "http://localhost:19001"

		log.Printf("Waiting for auth registry at %s ...", authURL)
		if err := waitForURL(authURL+"/", 120*time.Second); err != nil {
			composeLogsWithProject(authFiles, "bdd-auth")
			t.Fatalf("Auth registry did not become healthy: %v", err)
		}
		log.Println("Auth registry is healthy.")
		_ = authWebhook // available for future operational tests
	}

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@auth && ~@pending-impl",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			var tc *steps.TestContext

			if authDockerMode && authURL != "" {
				// Docker mode: use the external auth registry.
				// Memory backend: restart the service between scenarios via webhook.
				// This resets all state (users, schemas, rate limiter) cleanly.
				// The bootstrap config re-creates the admin user on restart.
				tc = steps.NewTestContext(authURL)

				ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
					if err := restartAuthRegistry(authURL, authWebhook); err != nil {
						return gctx, fmt.Errorf("restart auth registry: %w", err)
					}
					return gctx, nil
				})
			} else {
				// In-process mode: create fresh server per scenario
				var ts *httptest.Server
				var store storage.Storage
				var authSvc *auth.Service
				var storeOwned bool

				tc = steps.NewTestContext("http://placeholder")

				ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
					if sharedDBStore != nil {
						if err := cleanDBStore(); err != nil {
							return gctx, fmt.Errorf("clean %s storage for auth: %w", bddStorage, err)
						}
						ts, store, authSvc = newAuthTestServerWithStore(sharedDBStore)
						storeOwned = false
					} else {
						ts, store, authSvc = newAuthTestServer()
						storeOwned = true
					}
					tc.BaseURL = ts.URL
					return gctx, nil
				})

				ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
					authSvc.Close()
					ts.Close()
					if storeOwned {
						store.Close()
					}
					return gctx, nil
				})
			}

			// Register all step definitions (auth scenarios may also use schema steps)
			steps.RegisterSchemaSteps(ctx, tc)
			steps.RegisterImportSteps(ctx, tc)
			steps.RegisterModeSteps(ctx, tc)
			steps.RegisterReferenceSteps(ctx, tc)
			steps.RegisterInfraSteps(ctx, tc)
			steps.RegisterAuthSteps(ctx, tc)
			steps.RegisterEncryptionSteps(ctx, tc)
			steps.RegisterConcurrencySteps(ctx, tc)
			steps.RegisterRateLimitSteps(ctx, tc)
			steps.RegisterMetricsSteps(ctx, tc)
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("Auth BDD tests failed")
	}
}

// hasTag checks if a scenario has a specific tag.
func hasTag(sc *godog.Scenario, name string) bool {
	for _, t := range sc.Tags {
		if t.Name == name {
			return true
		}
	}
	return false
}

// cleanBackend resets all state between scenarios.
// For memory: uses API cleanup (delete subjects, reset config/mode).
// For DB backends: truncates all tables and resets sequences.
func cleanBackend() error {
	switch backend {
	case "postgres":
		return cleanPostgres()
	case "mysql":
		return cleanMySQL()
	case "cassandra":
		return cleanCassandra()
	case "confluent":
		return cleanViaAPI()
	default:
		return cleanViaAPI()
	}
}

// cleanPostgres truncates all tables and resets the ID sequence.
func cleanPostgres() error {
	port := envOrDefault("POSTGRES_PORT", "15432")
	dsn := fmt.Sprintf("host=localhost port=%s user=schemaregistry password=schemaregistry dbname=schemaregistry sslmode=disable", port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()

	// Truncate new tables first — ignore errors if tables don't exist yet (older migrations)
	optionalTables := []string{"exporter_statuses", "exporters", "deks", "keks"}
	for _, t := range optionalTables {
		db.Exec("TRUNCATE TABLE " + t + " RESTART IDENTITY CASCADE") // ignore error
	}

	stmts := []string{
		// RESTART IDENTITY resets all SERIAL/BIGSERIAL sequences (schemas, users, api_keys, etc.)
		"TRUNCATE TABLE api_keys, users, schema_references, schema_fingerprints, schemas, modes, configs, ctx_id_alloc, contexts RESTART IDENTITY CASCADE",
		// Re-seed per-context ID allocation and context for default context
		"INSERT INTO ctx_id_alloc (registry_ctx, next_id) VALUES ('.', 1) ON CONFLICT (registry_ctx) DO NOTHING",
		"INSERT INTO contexts (registry_ctx) VALUES ('.') ON CONFLICT DO NOTHING",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s, err)
		}
	}
	return nil
}

// cleanMySQL truncates all tables.
func cleanMySQL() error {
	port := envOrDefault("MYSQL_PORT", "13306")
	dsn := fmt.Sprintf("schemaregistry:schemaregistry@tcp(localhost:%s)/schemaregistry", port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disable FK checks: %w", err)
	}
	// Truncate new tables first — ignore errors if tables don't exist yet
	optionalTables := []string{"exporter_statuses", "exporters", "deks", "keks"}
	for _, t := range optionalTables {
		db.Exec("TRUNCATE TABLE `" + t + "`") // ignore error
	}
	tables := []string{"api_keys", "users", "schema_references", "schema_fingerprints", "schemas", "modes", "configs", "ctx_id_alloc", "contexts"}
	for _, t := range tables {
		if _, err := db.Exec("TRUNCATE TABLE `" + t + "`"); err != nil {
			return fmt.Errorf("truncate %s: %w", t, err)
		}
	}
	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return fmt.Errorf("enable FK checks: %w", err)
	}
	// Re-seed per-context ID allocation and context for default context
	if _, err := db.Exec("INSERT IGNORE INTO `ctx_id_alloc` (registry_ctx, next_id) VALUES ('.', 1)"); err != nil {
		return fmt.Errorf("seed ctx_id_alloc: %w", err)
	}
	if _, err := db.Exec("INSERT IGNORE INTO `contexts` (registry_ctx) VALUES ('.')"); err != nil {
		return fmt.Errorf("seed contexts: %w", err)
	}
	return nil
}

// getCassandraSession returns a long-lived session for BDD cleanup.
// The session is created once and reused across all scenarios.
func getCassandraSession() (*gocql.Session, error) {
	if cassandraSession != nil {
		return cassandraSession, nil
	}
	portStr := envOrDefault("CASSANDRA_PORT", "19042")
	port, _ := strconv.Atoi(portStr)
	cluster := gocql.NewCluster("localhost")
	cluster.Port = port
	cluster.Keyspace = "schemaregistry"
	cluster.Consistency = gocql.One
	cluster.Timeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("connect cassandra: %w", err)
	}
	cassandraSession = session
	return session, nil
}

// cleanCassandra truncates all tables in the schemaregistry keyspace.
func cleanCassandra() error {
	session, err := getCassandraSession()
	if err != nil {
		return err
	}

	// Truncate new tables first — ignore errors if tables don't exist yet
	optionalTables := []string{"exporter_statuses", "exporters", "deks", "deks_by_kek", "keks", "schema_fingerprints"}
	for _, t := range optionalTables {
		if err := session.Query("TRUNCATE " + t).Exec(); err != nil {
			if !strings.Contains(err.Error(), "unconfigured table") && !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("truncate %s: %w", t, err)
			}
		}
	}

	tables := []string{
		"api_keys_by_hash", "api_keys_by_user", "api_keys_by_id",
		"users_by_email", "users_by_id",
		"id_alloc", "modes", "global_config", "subject_configs",
		"references_by_target", "schema_references",
		"subject_latest", "subject_versions",
		"schemas_by_id", "contexts",
	}
	for _, t := range tables {
		if err := session.Query("TRUNCATE " + t).Exec(); err != nil {
			return fmt.Errorf("truncate %s: %w", t, err)
		}
	}

	// Re-seed id_alloc and default context for the default "." context.
	if err := session.Query("INSERT INTO id_alloc (registry_ctx, name, next_id) VALUES (?, ?, ?)",
		".", "schema_id", 1).Exec(); err != nil {
		return fmt.Errorf("seed id_alloc: %w", err)
	}
	if err := session.Query("INSERT INTO contexts (registry_ctx, created_at) VALUES (?, now())",
		".").Exec(); err != nil {
		return fmt.Errorf("seed contexts: %w", err)
	}
	return nil
}

// cleanViaAPI resets state via the REST API.
// Order matters: reset mode first (READWRITE allows writes/deletes),
// then delete subjects, then reset config.
func cleanViaAPI() error {
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Reset global mode to READWRITE first — a READONLY mode blocks DELETE operations.
	modeBody := strings.NewReader(`{"mode":"READWRITE"}`)
	req, _ := http.NewRequest("PUT", registryURL+"/mode", modeBody)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reset mode: %w", err)
	}
	r.Body.Close()

	// 2. Soft-delete all active subjects
	resp, err := client.Get(registryURL + "/subjects")
	if err != nil {
		return fmt.Errorf("list subjects: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var activeSubjects []string
	if resp.StatusCode == 200 && len(body) > 0 {
		json.Unmarshal(body, &activeSubjects)
	}
	for _, subj := range activeSubjects {
		req, _ := http.NewRequest("DELETE", registryURL+"/subjects/"+url.PathEscape(subj), nil)
		r, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("soft-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 3. Permanently delete all subjects (including previously soft-deleted)
	resp, err = client.Get(registryURL + "/subjects?deleted=true")
	if err != nil {
		return fmt.Errorf("list deleted subjects: %w", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	var allSubjects []string
	if resp.StatusCode == 200 && len(body) > 0 {
		json.Unmarshal(body, &allSubjects)
	}
	for _, subj := range allSubjects {
		req, _ := http.NewRequest("DELETE", registryURL+"/subjects/"+url.PathEscape(subj)+"?permanent=true", nil)
		r, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("permanent-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 4. Delete subject-level configs and modes for all subjects we deleted.
	// Confluent preserves subject config/mode even after permanent subject delete.
	allSubjectsToClean := append(activeSubjects, allSubjects...)
	seen := make(map[string]bool)
	for _, subj := range allSubjectsToClean {
		if seen[subj] {
			continue
		}
		seen[subj] = true
		escaped := url.PathEscape(subj)
		cfgReq, _ := http.NewRequest("DELETE", registryURL+"/config/"+escaped, nil)
		if cr, err := client.Do(cfgReq); err == nil {
			cr.Body.Close()
		}
		modeReq, _ := http.NewRequest("DELETE", registryURL+"/mode/"+escaped, nil)
		if mr, err := client.Do(modeReq); err == nil {
			mr.Body.Close()
		}
	}

	// 5. Delete global config so there's no stored override.
	// Using DELETE instead of PUT avoids leaving a stored config record
	// that would short-circuit the defaultToGlobal 4-tier fallback chain.
	req, _ = http.NewRequest("DELETE", registryURL+"/config", nil)
	r, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("reset config: %w", err)
	}
	r.Body.Close()

	// 6. Delete all KEKs (and their associated DEKs) via the DEK Registry API.
	resp, err = client.Get(registryURL + "/dek-registry/v1/keks")
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var keks []map[string]interface{}
		if json.Unmarshal(body, &keks) == nil {
			for _, kek := range keks {
				if name, ok := kek["name"].(string); ok {
					// Soft-delete KEK first
					delReq, _ := http.NewRequest("DELETE", registryURL+"/dek-registry/v1/keks/"+url.PathEscape(name), nil)
					if dr, err := client.Do(delReq); err == nil {
						dr.Body.Close()
					}
					// Permanent delete
					delReq, _ = http.NewRequest("DELETE", registryURL+"/dek-registry/v1/keks/"+url.PathEscape(name)+"?permanent=true", nil)
					if dr, err := client.Do(delReq); err == nil {
						dr.Body.Close()
					}
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}
	// Also try listing with deleted=true for soft-deleted KEKs
	resp, err = client.Get(registryURL + "/dek-registry/v1/keks?deleted=true")
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var keks []map[string]interface{}
		if json.Unmarshal(body, &keks) == nil {
			for _, kek := range keks {
				if name, ok := kek["name"].(string); ok {
					delReq, _ := http.NewRequest("DELETE", registryURL+"/dek-registry/v1/keks/"+url.PathEscape(name)+"?permanent=true", nil)
					if dr, err := client.Do(delReq); err == nil {
						dr.Body.Close()
					}
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 7. Delete all exporters via the Exporters API.
	resp, err = client.Get(registryURL + "/exporters")
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var exporterNames []string
		if json.Unmarshal(body, &exporterNames) == nil {
			for _, name := range exporterNames {
				delReq, _ := http.NewRequest("DELETE", registryURL+"/exporters/"+url.PathEscape(name), nil)
				if dr, err := client.Do(delReq); err == nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	return nil
}

// restartAuthRegistry restarts the schema registry via the webhook and waits for it to become healthy.
// For memory-backed Docker containers, this is the cleanest way to reset all state between scenarios:
// the rate limiter, in-memory storage, and credential cache are all reset on restart, and the
// bootstrap config automatically re-creates the admin user.
func restartAuthRegistry(registryURL, webhookURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Restart the service via webhook
	resp, err := client.Post(webhookURL+"/hooks/restart-service", "application/json", nil)
	if err != nil {
		return fmt.Errorf("restart webhook: %w", err)
	}
	resp.Body.Close()

	// Brief pause to let the old process die before polling
	time.Sleep(500 * time.Millisecond)

	// Wait for the registry to become healthy (bootstrap creates admin user)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(registryURL + "/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for auth registry restart at %s", registryURL)
}

// cleanAuthViaAPI resets state via the REST API with admin authentication.
// This is used for Docker-mode auth tests where the registry requires authentication.
// It cleans subjects, config, mode, KEKs, exporters, and non-admin users/API keys.
func cleanAuthViaAPI(baseURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	doReq := func(method, url string, body io.Reader) (*http.Response, error) {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
		}
		req.SetBasicAuth("admin", "admin-password")
		return client.Do(req)
	}

	// 1. Reset global mode to READWRITE
	r, err := doReq("PUT", baseURL+"/mode", strings.NewReader(`{"mode":"READWRITE"}`))
	if err != nil {
		return fmt.Errorf("reset mode: %w", err)
	}
	r.Body.Close()

	// 2. Soft-delete all active subjects
	resp, err := doReq("GET", baseURL+"/subjects", nil)
	if err != nil {
		return fmt.Errorf("list subjects: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var activeSubjects []string
	if resp.StatusCode == 200 && len(body) > 0 {
		json.Unmarshal(body, &activeSubjects)
	}
	for _, subj := range activeSubjects {
		r, err := doReq("DELETE", baseURL+"/subjects/"+url.PathEscape(subj), nil)
		if err != nil {
			return fmt.Errorf("soft-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 3. Permanently delete all subjects (including previously soft-deleted)
	resp, err = doReq("GET", baseURL+"/subjects?deleted=true", nil)
	if err != nil {
		return fmt.Errorf("list deleted subjects: %w", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	var allSubjects []string
	if resp.StatusCode == 200 && len(body) > 0 {
		json.Unmarshal(body, &allSubjects)
	}
	for _, subj := range allSubjects {
		r, err := doReq("DELETE", baseURL+"/subjects/"+url.PathEscape(subj)+"?permanent=true", nil)
		if err != nil {
			return fmt.Errorf("permanent-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 4. Delete subject-level configs and modes
	allSubjectsToClean := append(activeSubjects, allSubjects...)
	seen := make(map[string]bool)
	for _, subj := range allSubjectsToClean {
		if seen[subj] {
			continue
		}
		seen[subj] = true
		escaped := url.PathEscape(subj)
		if cr, err := doReq("DELETE", baseURL+"/config/"+escaped, nil); err == nil {
			cr.Body.Close()
		}
		if mr, err := doReq("DELETE", baseURL+"/mode/"+escaped, nil); err == nil {
			mr.Body.Close()
		}
	}

	// 5. Delete global config
	r, err = doReq("DELETE", baseURL+"/config", nil)
	if err != nil {
		return fmt.Errorf("reset config: %w", err)
	}
	r.Body.Close()

	// 6. Delete all non-admin users.
	// Response format: {"users": [{...}, ...]}
	resp, err = doReq("GET", baseURL+"/admin/users", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var usersResp struct {
			Users []struct {
				ID       float64 `json:"id"`
				Username string  `json:"username"`
			} `json:"users"`
		}
		if json.Unmarshal(body, &usersResp) == nil {
			for _, u := range usersResp.Users {
				if u.Username == "admin" {
					continue // keep the bootstrap admin
				}
				dr, err := doReq("DELETE", fmt.Sprintf("%s/admin/users/%d", baseURL, int(u.ID)), nil)
				if err == nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 7. Delete all API keys.
	// Response format: {"api_keys": [{...}, ...]}
	resp, err = doReq("GET", baseURL+"/admin/apikeys", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var keysResp struct {
			APIKeys []struct {
				ID float64 `json:"id"`
			} `json:"api_keys"`
		}
		if json.Unmarshal(body, &keysResp) == nil {
			for _, k := range keysResp.APIKeys {
				dr, err := doReq("DELETE", fmt.Sprintf("%s/admin/apikeys/%d", baseURL, int(k.ID)), nil)
				if err == nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 8. Delete all KEKs
	resp, err = doReq("GET", baseURL+"/dek-registry/v1/keks", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var keks []map[string]interface{}
		if json.Unmarshal(body, &keks) == nil {
			for _, kek := range keks {
				if name, ok := kek["name"].(string); ok {
					dr, _ := doReq("DELETE", baseURL+"/dek-registry/v1/keks/"+url.PathEscape(name), nil)
					if dr != nil {
						dr.Body.Close()
					}
					dr, _ = doReq("DELETE", baseURL+"/dek-registry/v1/keks/"+url.PathEscape(name)+"?permanent=true", nil)
					if dr != nil {
						dr.Body.Close()
					}
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 9. Delete all exporters
	resp, err = doReq("GET", baseURL+"/exporters", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var exporterNames []string
		if json.Unmarshal(body, &exporterNames) == nil {
			for _, name := range exporterNames {
				dr, _ := doReq("DELETE", baseURL+"/exporters/"+url.PathEscape(name), nil)
				if dr != nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	return nil
}

// composeFilesForBackend returns the Docker Compose files for a given backend.
// When BDD_KMS=true, the KMS overlay is appended to add Vault and OpenBao
// Transit engines alongside the storage backend.
func composeFilesForBackend(backend string) []string {
	base := "docker-compose.base.yml"
	var files []string
	switch backend {
	case "memory":
		files = []string{base, "docker-compose.memory.yml"}
	case "postgres":
		files = []string{base, "docker-compose.postgres.yml"}
	case "mysql":
		files = []string{base, "docker-compose.mysql.yml"}
	case "cassandra":
		files = []string{base, "docker-compose.cassandra.yml"}
	case "confluent":
		files = []string{"docker-compose.confluent.yml"}
	default:
		files = []string{base, "docker-compose." + backend + ".yml"}
	}
	if os.Getenv("BDD_KMS") == "true" && backend != "confluent" {
		files = append(files, "docker-compose.kms-overlay.yml")
	}
	return files
}

// findContainerCmd returns "podman" or "docker", preferring podman.
func findContainerCmd() string {
	if cmd := os.Getenv("CONTAINER_CMD"); cmd != "" {
		return cmd
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker"
	}
	log.Fatal("Neither podman nor docker found in PATH")
	return ""
}

func composeUp(files []string) error {
	return composeUpWithProject(files, "", nil)
}

func composeDown(files []string) {
	composeDownWithProject(files, "")
}

func composeLogs(files []string) {
	composeLogsWithProject(files, "")
}

// composeUpWithProject starts Docker Compose with an optional project name and extra env vars.
func composeUpWithProject(files []string, project string, env []string) error {
	args := []string{"compose"}
	if project != "" {
		args = append(args, "--project-name", project)
	}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "up", "-d", "--build", "--wait")

	cmd := exec.Command(containerCmd, args...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	return cmd.Run()
}

// composeDownWithProject tears down Docker Compose with an optional project name.
func composeDownWithProject(files []string, project string) {
	args := []string{"compose"}
	if project != "" {
		args = append(args, "--project-name", project)
	}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "down", "-v")

	cmd := exec.Command(containerCmd, args...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: %s compose down failed: %v", containerCmd, err)
	}
}

// composeLogsWithProject prints compose logs with an optional project name.
func composeLogsWithProject(files []string, project string) {
	args := []string{"compose"}
	if project != "" {
		args = append(args, "--project-name", project)
	}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "logs", "--tail=50")

	cmd := exec.Command(containerCmd, args...)
	cmd.Dir = "."
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// ensureRegistryRunning checks if the registry is healthy, and if not,
// calls the start-service webhook to bring it up. This is needed before
// cleanup between operational scenarios where a previous scenario may
// have stopped or killed the registry.
func ensureRegistryRunning() error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(registryURL + "/")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			return nil // already running
		}
	}
	// Registry is down — start it via webhook
	r, err := client.Post(webhookURL+"/hooks/start-service", "application/json", nil)
	if err != nil {
		return fmt.Errorf("start-service webhook: %w", err)
	}
	r.Body.Close()
	return waitForURL(registryURL+"/", 30*time.Second)
}

func waitForURL(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for %s: %v", url, lastErr)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// createTestHTPasswdFile creates a temporary htpasswd file with test users.
// Returns the file path, or empty string on failure.
func createTestHTPasswdFile() string {
	dir, err := os.MkdirTemp("", "bdd-htpasswd-*")
	if err != nil {
		return ""
	}
	// bcrypt hashes generated for test passwords
	// htuser1:htpassword1 and htuser2:htpassword2
	hash1, err := bcrypt.GenerateFromPassword([]byte("htpassword1"), bcrypt.MinCost)
	if err != nil {
		return ""
	}
	hash2, err := bcrypt.GenerateFromPassword([]byte("htpassword2"), bcrypt.MinCost)
	if err != nil {
		return ""
	}
	content := fmt.Sprintf("# Test htpasswd file for BDD\nhtuser1:%s\nhtuser2:%s\n", hash1, hash2)
	path := dir + "/htpasswd"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return ""
	}
	return path
}

// createTestMemoryAPIKeys creates a MemoryAPIKeyStore with test keys for BDD.
// Keys: "test-apikey-readonly" (readonly), "test-apikey-admin" (admin)
func createTestMemoryAPIKeys() *auth.MemoryAPIKeyStore {
	hash1, err := bcrypt.GenerateFromPassword([]byte("test-apikey-readonly"), bcrypt.MinCost)
	if err != nil {
		return nil
	}
	hash2, err := bcrypt.GenerateFromPassword([]byte("test-apikey-admin"), bcrypt.MinCost)
	if err != nil {
		return nil
	}
	store, err := auth.NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "readonly-key", KeyHash: string(hash1), Role: "readonly"},
		{Name: "admin-key", KeyHash: string(hash2), Role: "admin"},
	})
	if err != nil {
		return nil
	}
	return store
}

func init() {
	// Ensure the features directory is findable regardless of how go test sets cwd.
	if _, err := os.Stat("features"); err != nil {
		candidates := []string{"tests/bdd/features", "../../tests/bdd/features"}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				os.Chdir(strings.TrimSuffix(c, "/features"))
				break
			}
		}
	}
}
