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

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
	"github.com/axonops/axonops-schema-registry/tests/bdd/steps"
)

var (
	dockerMode   bool
	registryURL  string
	webhookURL   string
	backend      string
	composeFiles []string
	containerCmd string // "podman" or "docker"

	// cassandraSession is a long-lived session reused across all BDD scenario cleanups.
	// gocql sessions are expensive to create (topology discovery, connection pool setup),
	// so we create one at first use and close it in TestMain.
	cassandraSession *gocql.Session
)

func TestMain(m *testing.M) {
	backend = os.Getenv("BDD_BACKEND")
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

	code := m.Run()

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
func newTestServer() (*httptest.Server, storage.Storage) {
	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0, DocsEnabled: true},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	server := api.NewServer(cfg, reg, logger)

	return httptest.NewServer(server), store
}

// newAuthTestServer creates an in-process schema registry with authentication enabled.
// It pre-seeds a super_admin user ("admin" / "admin-password") for testing.
func newAuthTestServer() (*httptest.Server, storage.Storage, *auth.Service) {
	store := memory.NewStore()

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

	authenticator := auth.NewAuthenticator(authCfg)
	authService := auth.NewServiceWithConfig(store, auth.ServiceConfig{
		CacheRefreshInterval: 0, // disable background refresh for tests
		UserCacheTTL:         0, // disable credential caching for tests
	})
	authenticator.SetService(authService)

	authorizer := auth.NewAuthorizer(authCfg.RBAC)

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
		Security: config.SecurityConfig{
			Auth: authCfg,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	server := api.NewServer(cfg, reg, logger, api.WithAuth(authenticator, authorizer, authService))

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

func TestFeatures(t *testing.T) {
	// In-process: skip @operational (no Docker infrastructure).
	// Docker mode: run everything for this backend, skip other backends. BDD_TAGS overrides.
	tags := ""
	if envTags := os.Getenv("BDD_TAGS"); envTags != "" {
		tags = envTags
	} else if !dockerMode && registryURL == "" {
		tags = "~@operational && ~@pending-impl && ~@auth"
	} else if backend == "confluent" {
		// Confluent: exclude operational, import (our custom API), axonops-only, contexts (our multi-tenant),
		// pending-impl, data-contracts (ruleSet features require commercial Confluent license),
		// and all backend tags.
		tags = "~@operational && ~@import && ~@axonops-only && ~@contexts && ~@pending-impl && ~@data-contracts && ~@auth && ~@memory && ~@postgres && ~@mysql && ~@cassandra"
	} else if dockerMode {
		// Only run operational scenarios tagged for this backend, exclude other backends.
		// Auth tests require in-process server with auth enabled, skip in Docker mode.
		allBackends := []string{"memory", "postgres", "mysql", "cassandra"}
		excludes := []string{"~@pending-impl", "~@auth"}
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
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			var tc *steps.TestContext

			if registryURL != "" {
				// Docker-based: use external registry, clean state before each scenario
				tc = steps.NewTestContext(registryURL)
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
				// In-process: create fresh server per scenario
				ts, store := newTestServer()
				tc = steps.NewTestContext(ts.URL)
				ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
					ts.Close()
					store.Close()
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
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("BDD tests failed")
	}
}

// TestAuthFeatures runs BDD tests that require authentication enabled.
// These tests use a separate in-process server with auth, RBAC, and a pre-seeded admin user.
func TestAuthFeatures(t *testing.T) {
	// Auth BDD tests only run in-process (no Docker mode for auth tests)
	if dockerMode || registryURL != "" {
		t.Skip("Auth BDD tests only run in-process mode")
	}

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@auth",
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ts, store, authSvc := newAuthTestServer()
			tc := steps.NewTestContext(ts.URL)

			ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
				authSvc.Close()
				ts.Close()
				store.Close()
				return gctx, nil
			})

			// Register all step definitions (auth scenarios may also use schema steps)
			steps.RegisterSchemaSteps(ctx, tc)
			steps.RegisterImportSteps(ctx, tc)
			steps.RegisterModeSteps(ctx, tc)
			steps.RegisterReferenceSteps(ctx, tc)
			steps.RegisterInfraSteps(ctx, tc)
			steps.RegisterAuthSteps(ctx, tc)
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
		db.Exec("TRUNCATE TABLE " + t + " CASCADE") // ignore error
	}

	stmts := []string{
		"TRUNCATE TABLE api_keys, users, schema_references, schema_fingerprints, schemas, modes, configs, ctx_id_alloc, contexts CASCADE",
		"ALTER SEQUENCE schemas_id_seq RESTART WITH 1",
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

// composeFilesForBackend returns the Docker Compose files for a given backend.
func composeFilesForBackend(backend string) []string {
	base := "docker-compose.base.yml"
	switch backend {
	case "memory":
		return []string{base, "docker-compose.memory.yml"}
	case "postgres":
		return []string{base, "docker-compose.postgres.yml"}
	case "mysql":
		return []string{base, "docker-compose.mysql.yml"}
	case "cassandra":
		return []string{base, "docker-compose.cassandra.yml"}
	case "confluent":
		return []string{"docker-compose.confluent.yml"}
	default:
		return []string{base, "docker-compose." + backend + ".yml"}
	}
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
	args := []string{"compose"}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "up", "-d", "--build", "--wait")

	cmd := exec.Command(containerCmd, args...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func composeDown(files []string) {
	args := []string{"compose"}
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

func composeLogs(files []string) {
	args := []string{"compose"}
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
