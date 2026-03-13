//go:build bdd

// Package bdd provides BDD tests using godog (Cucumber for Go).
//
// All tests run against a Docker-deployed binary:
//
//	go test -tags bdd -v -timeout 30m ./tests/bdd/...
//
// Or via Makefile:
//
//	make test-bdd-functional          # Memory backend (Docker)
//	make test-bdd BACKEND=memory      # Memory backend (Docker)
//	make test-bdd BACKEND=postgres    # PostgreSQL backend (Docker)
package bdd

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	"github.com/axonops/axonops-schema-registry/tests/bdd/steps"
)

var (
	registryURL  string
	metricsURL   string // separate metrics endpoint (e.g. JMX exporter sidecar for Confluent)
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

	// Default to Docker with memory backend when no env vars are set.
	if backend == "" && registryURL == "" {
		backend = "memory"
	}

	// If BDD_BACKEND is set but no external URL, start Docker Compose automatically.
	if backend != "" && registryURL == "" {
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

	code := m.Run()

	// Close the long-lived Cassandra session before tearing down containers.
	if cassandraSession != nil {
		cassandraSession.Close()
		cassandraSession = nil
	}

	if composeFiles != nil {
		log.Println("Stopping compose...")
		composeDown(composeFiles)
	}

	os.Exit(code)
}

func TestFeatures(t *testing.T) {
	tags := ""
	if envTags := os.Getenv("BDD_TAGS"); envTags != "" {
		tags = envTags
	} else if backend == "confluent" {
		tags = "~@operational && ~@import && ~@axonops-only && ~@contexts && ~@pending-impl && ~@data-contracts && ~@auth && ~@kms && ~@mcp && ~@analysis && ~@audit && ~@audit-outputs && ~@memory && ~@postgres && ~@mysql && ~@cassandra"
	} else {
		// Docker mode: run operational + functional scenarios for this backend.
		// Auth, MCP, audit, KMS are handled by their own test functions with separate stacks.
		allBackends := []string{"memory", "postgres", "mysql", "cassandra"}
		excludes := []string{"~@pending-impl", "~@auth", "~@ldap", "~@mcp", "~@audit", "~@audit-outputs"}
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

	// Confluent's schema registry has no audit logging — don't wire up
	// audit helpers so that audit assertion steps gracefully no-op.
	var auditFetcher func() (string, error)
	var clearAuditLog func() error
	if backend != "confluent" {
		auditFetcher, clearAuditLog = makeAuditHelpers(composeFiles, "")
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(registryURL)
			tc.MetricsURL = metricsURL
			tc.WebhookURL = webhookURL
			if auditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = auditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if hasTag(sc, "@operational") {
					if err := ensureRegistryRunning(); err != nil {
						return gctx, fmt.Errorf("ensure registry running: %w", err)
					}
				}
				if err := cleanBackend(); err != nil {
					return gctx, fmt.Errorf("clean backend: %w", err)
				}
				if clearAuditLog != nil {
					if err := clearAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, waitForURL(registryURL+"/", 30*time.Second)
			})

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
// Starts a Docker compose stack with auth config, optionally layered with a DB overlay.
func TestAuthFeatures(t *testing.T) {
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	authFiles := []string{"docker-compose.base.yml", "docker-compose.auth.yml"}
	authEnv := []string{
		"REGISTRY_PORT=18082",
		"WEBHOOK_PORT=19001",
	}

	// Add database overlay if backend is a DB type.
	// Use unique DB ports to avoid collisions with the main TestFeatures stack.
	isDBBackend := false
	authDBPort := ""
	if dbOverlay := dbOverlayFile(backend); dbOverlay != "" {
		authFiles = append(authFiles, dbOverlay)
		isDBBackend = true
		switch backend {
		case "postgres":
			authDBPort = "25432"
			authEnv = append(authEnv, "POSTGRES_PORT=25432")
		case "mysql":
			authDBPort = "23306"
			authEnv = append(authEnv, "MYSQL_PORT=23306")
		case "cassandra":
			authDBPort = "29042"
			authEnv = append(authEnv, "CASSANDRA_PORT=29042")
		}
	}

	log.Printf("Starting auth compose stack (backend=%s)...", backend)
	if err := composeUpWithProject(authFiles, "bdd-auth", authEnv); err != nil {
		t.Fatalf("Failed to start auth compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping auth compose stack...")
		composeDownWithProject(authFiles, "bdd-auth")
	})

	authURL := "http://localhost:18082"
	authWebhook := "http://localhost:19001"

	waitTimeout := 120 * time.Second
	if isDBBackend {
		waitTimeout = 180 * time.Second // DB startup takes longer
	}
	log.Printf("Waiting for auth registry at %s ...", authURL)
	if err := waitForURL(authURL+"/", waitTimeout); err != nil {
		composeLogsWithProject(authFiles, "bdd-auth")
		t.Fatalf("Auth registry did not become healthy: %v", err)
	}
	log.Println("Auth registry is healthy.")

	authAuditFetcher, clearAuthAuditLog := makeAuditHelpers(authFiles, "bdd-auth")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@auth && ~@ldap && ~@pending-impl",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(authURL)
			if authAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = authAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				// For DB backends: TRUNCATE tables first to clear persistent data.
				if isDBBackend {
					if err := cleanBackendPort(authDBPort); err != nil {
						return gctx, fmt.Errorf("clean %s backend for auth: %w", backend, err)
					}
				}
				// Restart registry to reset in-memory state (rate limiter, etc.)
				// and re-bootstrap the admin user from config.
				if err := restartAuthRegistry(authURL, authWebhook); err != nil {
					return gctx, fmt.Errorf("restart auth registry: %w", err)
				}
				if clearAuthAuditLog != nil {
					if err := clearAuthAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
			steps.RegisterMCPSteps(ctx, tc)
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("Auth BDD tests failed")
	}
}

// TestMCPFeatures runs MCP BDD tests against a Docker-deployed binary via HTTP Streamable transport.
// Uses a separate compose stack with MCP enabled on port 9081 and auth enabled for admin tools.
// Excludes @mcp-permissions and @mcp-confirmation (they need in-process config per scenario),
// @kms (needs separate KMS compose stack), and @audit (needs audit log access).
func TestMCPFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("MCP Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	mcpFiles := []string{"docker-compose.base.yml", "docker-compose.mcp.yml"}
	mcpEnv := []string{
		"REGISTRY_PORT=18083",
		"WEBHOOK_PORT=19002",
		"MCP_PORT=19081",
	}

	log.Printf("Starting MCP compose stack...")
	if err := composeUpWithProject(mcpFiles, "bdd-mcp", mcpEnv); err != nil {
		t.Fatalf("Failed to start MCP compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping MCP compose stack...")
		composeDownWithProject(mcpFiles, "bdd-mcp")
	})

	mcpRESTURL := "http://localhost:18083"
	mcpURL := "http://localhost:19081/mcp"
	mcpWebhook := "http://localhost:19002"

	log.Printf("Waiting for MCP registry REST at %s ...", mcpRESTURL)
	if err := waitForURL(mcpRESTURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp")
		t.Fatalf("MCP registry did not become healthy: %v", err)
	}

	log.Printf("Waiting for MCP endpoint at %s ...", mcpURL)
	if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp")
		t.Fatalf("MCP endpoint did not become ready: %v", err)
	}
	log.Println("MCP registry is healthy.")

	mcpAuditFetcher, clearMCPAuditLog := makeAuditHelpers(mcpFiles, "bdd-mcp")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@mcp && ~@mcp-permissions && ~@mcp-confirmation && ~@mcp-metrics && ~@kms && ~@audit",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(mcpRESTURL)
			tc.MetricsURL = mcpRESTURL + "/metrics"
			tc.WebhookURL = mcpWebhook
			tc.StoredValues["_mcp_url"] = mcpURL
			tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))
			if mcpAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = mcpAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
					return gctx, fmt.Errorf("clean MCP registry: %w", err)
				}
				if clearMCPAuditLog != nil {
					if err := clearMCPAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("MCP BDD tests failed")
	}
}

// TestMCPKMSFeatures runs MCP + KMS BDD tests against Docker with Vault and OpenBao.
func TestMCPKMSFeatures(t *testing.T) {
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	mcpFiles := []string{
		"docker-compose.base.yml",
		"docker-compose.mcp.yml",
		"docker-compose.kms-overlay.yml",
	}
	mcpEnv := []string{
		"REGISTRY_PORT=18085",
		"WEBHOOK_PORT=19004",
		"MCP_PORT=19083",
		"VAULT_PORT=18202",
		"BAO_PORT=18203",
	}

	// Add database overlay if backend is a DB type.
	// Use unique DB ports to avoid collisions with the main TestFeatures stack.
	if dbOverlay := dbOverlayFile(backend); dbOverlay != "" {
		mcpFiles = append(mcpFiles, dbOverlay)
		switch backend {
		case "postgres":
			mcpEnv = append(mcpEnv, "POSTGRES_PORT=25433")
		case "mysql":
			mcpEnv = append(mcpEnv, "MYSQL_PORT=23307")
		case "cassandra":
			mcpEnv = append(mcpEnv, "CASSANDRA_PORT=29043")
		}
	}

	log.Printf("Starting MCP + KMS compose stack (backend=%s)...", backend)
	if err := composeUpWithProject(mcpFiles, "bdd-mcp-kms", mcpEnv); err != nil {
		t.Fatalf("Failed to start MCP KMS compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping MCP + KMS compose stack...")
		composeDownWithProject(mcpFiles, "bdd-mcp-kms")
	})

	mcpRESTURL := "http://localhost:18085"
	mcpURL := "http://localhost:19083/mcp"
	mcpWebhook := "http://localhost:19004"

	mcpKMSTimeout := 120 * time.Second
	if dbOverlayFile(backend) != "" {
		mcpKMSTimeout = 180 * time.Second
	}
	log.Printf("Waiting for MCP+KMS registry at %s ...", mcpRESTURL)
	if err := waitForURL(mcpRESTURL+"/", mcpKMSTimeout); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-kms")
		t.Fatalf("MCP+KMS registry did not become healthy: %v", err)
	}

	log.Printf("Waiting for MCP+KMS endpoint at %s ...", mcpURL)
	if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-kms")
		t.Fatalf("MCP+KMS endpoint did not become ready: %v", err)
	}
	log.Println("MCP+KMS registry is healthy.")

	// Set KMS env vars for test-side transit decrypt verification.
	// The compose overlay exposes Vault at VAULT_PORT and OpenBao at BAO_PORT.
	t.Setenv("KMS_VAULT_ADDR", "http://localhost:18202")
	t.Setenv("KMS_VAULT_TOKEN", "test-root-token")
	t.Setenv("KMS_BAO_ADDR", "http://localhost:18203")
	t.Setenv("KMS_BAO_TOKEN", "test-bao-token")

	kmsAuditFetcher, clearKMSAuditLog := makeAuditHelpers(mcpFiles, "bdd-mcp-kms")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@mcp && @kms",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(mcpRESTURL)
			tc.MetricsURL = mcpRESTURL + "/metrics"
			tc.WebhookURL = mcpWebhook
			tc.StoredValues["_mcp_url"] = mcpURL
			tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))
			if kmsAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = kmsAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
					return gctx, fmt.Errorf("clean MCP+KMS registry: %w", err)
				}
				if clearKMSAuditLog != nil {
					if err := clearKMSAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("MCP KMS BDD tests failed")
	}
}

// TestMCPMetricsFeatures runs MCP metrics BDD tests against Docker.
// Separate from main MCP tests because metrics tests need to verify Prometheus output
// and some require confirmations or permission variations.
func TestMCPMetricsFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("MCP metrics Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	mcpFiles := []string{"docker-compose.base.yml", "docker-compose.mcp.yml"}
	mcpEnv := []string{
		"REGISTRY_PORT=18086",
		"WEBHOOK_PORT=19005",
		"MCP_PORT=19084",
	}

	log.Printf("Starting MCP metrics compose stack...")
	if err := composeUpWithProject(mcpFiles, "bdd-mcp-metrics", mcpEnv); err != nil {
		t.Fatalf("Failed to start MCP metrics compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping MCP metrics compose stack...")
		composeDownWithProject(mcpFiles, "bdd-mcp-metrics")
	})

	mcpRESTURL := "http://localhost:18086"
	mcpURL := "http://localhost:19084/mcp"
	mcpWebhook := "http://localhost:19005"

	log.Printf("Waiting for MCP metrics registry at %s ...", mcpRESTURL)
	if err := waitForURL(mcpRESTURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-metrics")
		t.Fatalf("MCP metrics registry did not become healthy: %v", err)
	}

	log.Printf("Waiting for MCP metrics endpoint at %s ...", mcpURL)
	if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-metrics")
		t.Fatalf("MCP metrics endpoint did not become ready: %v", err)
	}
	log.Println("MCP metrics registry is healthy.")

	metricsAuditFetcher, clearMetricsAuditLog := makeAuditHelpers(mcpFiles, "bdd-mcp-metrics")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@mcp-metrics && ~@mcp-confirmation && ~@mcp-permissions",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(mcpRESTURL)
			tc.MetricsURL = mcpRESTURL + "/metrics"
			tc.WebhookURL = mcpWebhook
			tc.StoredValues["_mcp_url"] = mcpURL
			tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))
			if metricsAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = metricsAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
					return gctx, fmt.Errorf("clean MCP metrics registry: %w", err)
				}
				if clearMetricsAuditLog != nil {
					if err := clearMetricsAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("MCP metrics BDD tests failed")
	}
}

// TestMCPConfirmationFeatures runs MCP confirmation BDD tests against Docker.
// Uses a compose stack with SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS=true.
func TestMCPConfirmationFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("MCP confirmation Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	mcpFiles := []string{"docker-compose.base.yml", "docker-compose.mcp.yml"}
	mcpEnv := []string{
		"REGISTRY_PORT=18087",
		"WEBHOOK_PORT=19006",
		"MCP_PORT=19085",
		"SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS=true",
		"SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL=300",
	}

	log.Printf("Starting MCP confirmation compose stack...")
	if err := composeUpWithProject(mcpFiles, "bdd-mcp-confirm", mcpEnv); err != nil {
		t.Fatalf("Failed to start MCP confirmation compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping MCP confirmation compose stack...")
		composeDownWithProject(mcpFiles, "bdd-mcp-confirm")
	})

	mcpRESTURL := "http://localhost:18087"
	mcpURL := "http://localhost:19085/mcp"
	mcpWebhook := "http://localhost:19006"

	log.Printf("Waiting for MCP confirmation registry at %s ...", mcpRESTURL)
	if err := waitForURL(mcpRESTURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-confirm")
		t.Fatalf("MCP confirmation registry did not become healthy: %v", err)
	}

	log.Printf("Waiting for MCP confirmation endpoint at %s ...", mcpURL)
	if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
		composeLogsWithProject(mcpFiles, "bdd-mcp-confirm")
		t.Fatalf("MCP confirmation endpoint did not become ready: %v", err)
	}
	log.Println("MCP confirmation registry is healthy.")

	confirmAuditFetcher, clearConfirmAuditLog := makeAuditHelpers(mcpFiles, "bdd-mcp-confirm")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@mcp-confirmation",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(mcpRESTURL)
			tc.MetricsURL = mcpRESTURL + "/metrics"
			tc.WebhookURL = mcpWebhook
			tc.StoredValues["_mcp_url"] = mcpURL
			tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))
			if confirmAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = confirmAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
					return gctx, fmt.Errorf("clean MCP confirmation registry: %w", err)
				}
				if clearConfirmAuditLog != nil {
					if err := clearConfirmAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("MCP confirmation BDD tests failed")
	}
}

// TestMCPPermissionsFeatures runs MCP permission preset BDD tests against Docker.
// Starts separate compose stacks per preset since each preset needs a different
// server configuration (SCHEMA_REGISTRY_MCP_PERMISSION_PRESET env var).
func TestMCPPermissionsFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("MCP permissions Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}

	type presetTest struct {
		preset string
		tag    string
		// For custom scopes, we also need the scopes env var
		extraEnv []string
	}

	presets := []presetTest{
		{preset: "readonly", tag: "@mcp-permissions && @preset-readonly"},
		{preset: "developer", tag: "@mcp-permissions && @preset-developer"},
		{preset: "operator", tag: "@mcp-permissions && @preset-operator"},
		{preset: "admin", tag: "@mcp-permissions && @preset-admin"},
		{
			preset:   "custom",
			tag:      "@mcp-permissions && @preset-custom",
			extraEnv: []string{"SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES=schema_read"},
		},
	}

	mcpFiles := []string{"docker-compose.base.yml", "docker-compose.mcp.yml"}
	basePort := 18090

	for i, pt := range presets {
		pt := pt
		restPort := basePort + i*3
		mcpPort := basePort + i*3 + 1
		webhookPort := basePort + i*3 + 2
		projectName := fmt.Sprintf("bdd-mcp-perm-%s", pt.preset)

		t.Run(pt.preset, func(t *testing.T) {
			mcpEnv := []string{
				fmt.Sprintf("REGISTRY_PORT=%d", restPort),
				fmt.Sprintf("MCP_PORT=%d", mcpPort),
				fmt.Sprintf("WEBHOOK_PORT=%d", webhookPort),
				fmt.Sprintf("SCHEMA_REGISTRY_MCP_PERMISSION_PRESET=%s", pt.preset),
			}
			mcpEnv = append(mcpEnv, pt.extraEnv...)

			log.Printf("Starting MCP permissions compose stack (preset=%s)...", pt.preset)
			if err := composeUpWithProject(mcpFiles, projectName, mcpEnv); err != nil {
				t.Fatalf("Failed to start MCP permissions compose (preset=%s): %v", pt.preset, err)
			}
			t.Cleanup(func() {
				log.Printf("Stopping MCP permissions compose stack (preset=%s)...", pt.preset)
				composeDownWithProject(mcpFiles, projectName)
			})

			mcpRESTURL := fmt.Sprintf("http://localhost:%d", restPort)
			mcpURL := fmt.Sprintf("http://localhost:%d/mcp", mcpPort)
			mcpWebhook := fmt.Sprintf("http://localhost:%d", webhookPort)

			log.Printf("Waiting for MCP permissions registry (preset=%s) at %s ...", pt.preset, mcpRESTURL)
			if err := waitForURL(mcpRESTURL+"/", 120*time.Second); err != nil {
				composeLogsWithProject(mcpFiles, projectName)
				t.Fatalf("MCP permissions registry (preset=%s) did not become healthy: %v", pt.preset, err)
			}

			log.Printf("Waiting for MCP permissions endpoint (preset=%s) at %s ...", pt.preset, mcpURL)
			if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
				composeLogsWithProject(mcpFiles, projectName)
				t.Fatalf("MCP permissions endpoint (preset=%s) did not become ready: %v", pt.preset, err)
			}
			log.Printf("MCP permissions registry (preset=%s) is healthy.", pt.preset)

			permAuditFetcher, clearPermAuditLog := makeAuditHelpers(mcpFiles, projectName)

			opts := godog.Options{
				Format:   "pretty",
				Output:   colors.Colored(os.Stdout),
				Paths:    []string{"features"},
				Tags:     pt.tag,
				Strict:   true,
				TestingT: t,
			}

			suite := godog.TestSuite{
				ScenarioInitializer: func(ctx *godog.ScenarioContext) {
					tc := steps.NewTestContext(mcpRESTURL)
					tc.MetricsURL = mcpRESTURL + "/metrics"
					tc.WebhookURL = mcpWebhook
					tc.StoredValues["_mcp_url"] = mcpURL
					tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))
					if permAuditFetcher != nil {
						tc.StoredValues["_audit_fetcher"] = permAuditFetcher
					}

					ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
						if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
							return gctx, fmt.Errorf("clean MCP permissions registry (preset=%s): %w", pt.preset, err)
						}
						if clearPermAuditLog != nil {
							if err := clearPermAuditLog(); err != nil {
								return gctx, fmt.Errorf("clear audit log: %w", err)
							}
						}
						return gctx, nil
					})

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
				t.Fatalf("MCP permissions BDD tests (preset=%s) failed", pt.preset)
			}
		})
	}
}

// TestMCPAuditFeatures runs MCP audit BDD tests against a Docker-deployed binary.
// The audit log is written to a file inside the container and read via docker exec.
func TestMCPAuditFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("MCP audit Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	auditFiles := []string{"docker-compose.base.yml", "docker-compose.mcp-audit.yml"}
	auditEnv := []string{
		"REGISTRY_PORT=18089",
		"WEBHOOK_PORT=19008",
		"MCP_PORT=19086",
	}
	projectName := "bdd-mcp-audit"

	log.Printf("Starting MCP audit compose stack...")
	if err := composeUpWithProject(auditFiles, projectName, auditEnv); err != nil {
		t.Fatalf("Failed to start MCP audit compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping MCP audit compose stack...")
		composeDownWithProject(auditFiles, projectName)
	})

	mcpRESTURL := "http://localhost:18089"
	mcpURL := "http://localhost:19086/mcp"
	mcpWebhook := "http://localhost:19008"

	log.Printf("Waiting for MCP audit registry at %s ...", mcpRESTURL)
	if err := waitForURL(mcpRESTURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(auditFiles, projectName)
		t.Fatalf("MCP audit registry did not become healthy: %v", err)
	}

	log.Printf("Waiting for MCP audit endpoint at %s ...", mcpURL)
	if err := waitForMCPEndpoint(mcpURL, 30*time.Second); err != nil {
		composeLogsWithProject(auditFiles, projectName)
		t.Fatalf("MCP audit endpoint did not become ready: %v", err)
	}
	log.Println("MCP audit registry is healthy.")

	auditFetcher, clearAuditLog := makeAuditHelpers(auditFiles, projectName)

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@mcp && @audit",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(mcpRESTURL)
			tc.MetricsURL = mcpRESTURL + "/metrics"
			tc.WebhookURL = mcpWebhook
			tc.StoredValues["_mcp_url"] = mcpURL
			tc.StoredValues["_audit_fetcher"] = auditFetcher
			tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin-password"))

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPIWithAuth(mcpRESTURL, "admin", "admin-password"); err != nil {
					return gctx, fmt.Errorf("clean MCP audit registry: %w", err)
				}
				if err := clearAuditLog(); err != nil {
					return gctx, fmt.Errorf("clear audit log: %w", err)
				}
				return gctx, nil
			})

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
		t.Fatal("MCP audit BDD tests failed")
	}
}

// TestKMSFeatures runs REST KMS encryption BDD tests against a Docker-deployed binary
// with Vault Transit and OpenBao Transit engines.
func TestKMSFeatures(t *testing.T) {
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	kmsFiles := []string{
		"docker-compose.base.yml",
		"docker-compose.memory.yml",
		"docker-compose.kms-overlay.yml",
	}
	kmsEnv := []string{
		"REGISTRY_PORT=18088",
		"WEBHOOK_PORT=19007",
		"VAULT_PORT=18204",
		"BAO_PORT=18205",
	}

	// Add database overlay if backend is a DB type.
	// Use unique DB ports to avoid collisions with the main TestFeatures stack.
	if dbOverlay := dbOverlayFile(backend); dbOverlay != "" {
		kmsFiles = append(kmsFiles, dbOverlay)
		switch backend {
		case "postgres":
			kmsEnv = append(kmsEnv, "POSTGRES_PORT=25434")
		case "mysql":
			kmsEnv = append(kmsEnv, "MYSQL_PORT=23308")
		case "cassandra":
			kmsEnv = append(kmsEnv, "CASSANDRA_PORT=29044")
		}
	}

	waitTimeout := 120 * time.Second
	if dbOverlayFile(backend) != "" {
		waitTimeout = 180 * time.Second // DB startup takes longer
	}

	log.Printf("Starting REST KMS compose stack (backend=%s)...", backend)
	if err := composeUpWithProject(kmsFiles, "bdd-rest-kms", kmsEnv); err != nil {
		t.Fatalf("Failed to start REST KMS compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping REST KMS compose stack...")
		composeDownWithProject(kmsFiles, "bdd-rest-kms")
	})

	restURL := "http://localhost:18088"

	log.Printf("Waiting for KMS registry at %s ...", restURL)
	if err := waitForURL(restURL+"/", waitTimeout); err != nil {
		composeLogsWithProject(kmsFiles, "bdd-rest-kms")
		t.Fatalf("KMS registry did not become healthy: %v", err)
	}
	log.Println("KMS registry is healthy.")

	// Set KMS env vars for test-side Transit decrypt calls
	t.Setenv("KMS_VAULT_ADDR", "http://localhost:18204")
	t.Setenv("KMS_VAULT_TOKEN", "test-root-token")
	t.Setenv("KMS_BAO_ADDR", "http://localhost:18205")
	t.Setenv("KMS_BAO_TOKEN", "test-bao-token")

	restKMSAuditFetcher, clearRestKMSAuditLog := makeAuditHelpers(kmsFiles, "bdd-rest-kms")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@kms && ~@mcp",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(restURL)
			tc.MetricsURL = restURL + "/metrics"
			if restKMSAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = restKMSAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := cleanViaAPINoAuth(restURL); err != nil {
					return gctx, fmt.Errorf("clean KMS registry: %w", err)
				}
				if clearRestKMSAuditLog != nil {
					if err := clearRestKMSAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("REST KMS BDD tests failed")
	}
}

// TestRESTAuditFeatures runs REST audit BDD tests against a Docker-deployed binary
// with audit logging enabled (no auth, no MCP).
func TestRESTAuditFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("REST audit Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	auditFiles := []string{"docker-compose.base.yml", "docker-compose.audit.yml"}
	auditEnv := []string{
		"REGISTRY_PORT=18091",
		"WEBHOOK_PORT=19009",
	}
	projectName := "bdd-rest-audit"

	log.Printf("Starting REST audit compose stack...")
	if err := composeUpWithProject(auditFiles, projectName, auditEnv); err != nil {
		t.Fatalf("Failed to start REST audit compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping REST audit compose stack...")
		composeDownWithProject(auditFiles, projectName)
	})

	restURL := "http://localhost:18091"

	log.Printf("Waiting for audit registry at %s ...", restURL)
	if err := waitForURL(restURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(auditFiles, projectName)
		t.Fatalf("Audit registry did not become healthy: %v", err)
	}
	log.Println("Audit registry is healthy.")

	auditFetcher, clearAuditLog := makeAuditHelpers(auditFiles, projectName)

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@audit && ~@mcp && ~@pending-impl",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(restURL)
			tc.StoredValues["_audit_fetcher"] = auditFetcher

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := clearAuditLog(); err != nil {
					return gctx, fmt.Errorf("clear audit log: %w", err)
				}
				if err := cleanViaAPINoAuth(restURL); err != nil {
					return gctx, fmt.Errorf("clean audit registry: %w", err)
				}
				return gctx, nil
			})

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
		t.Fatal("REST audit BDD tests failed")
	}
}

// TestAuditOutputsFeatures runs BDD tests for multi-output audit delivery
// (file + syslog + webhook simultaneously). Uses a docker compose overlay
// with syslog-ng and a custom webhook-receiver container.
func TestAuditOutputsFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("Audit outputs Docker tests only run on memory backend")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	composeFiles := []string{"docker-compose.base.yml", "docker-compose.audit-outputs.yml"}
	composeEnv := []string{
		"REGISTRY_PORT=18110",
		"WEBHOOK_PORT=19012",
		"WEBHOOK_RECEIVER_PORT=19013",
	}
	projectName := "bdd-audit-outputs"

	log.Printf("Starting audit outputs compose stack...")
	if err := composeUpWithProject(composeFiles, projectName, composeEnv); err != nil {
		t.Fatalf("Failed to start audit outputs compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping audit outputs compose stack...")
		composeDownWithProject(composeFiles, projectName)
	})

	restURL := "http://localhost:18110"
	webhookReceiverURL := "http://localhost:19013"

	log.Printf("Waiting for audit outputs registry at %s ...", restURL)
	if err := waitForURL(restURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(composeFiles, projectName)
		t.Fatalf("Audit outputs registry did not become healthy: %v", err)
	}
	log.Println("Audit outputs registry is healthy.")

	// Wait for webhook receiver
	log.Printf("Waiting for webhook receiver at %s ...", webhookReceiverURL)
	if err := waitForURL(webhookReceiverURL+"/health", 30*time.Second); err != nil {
		composeLogsWithProject(composeFiles, projectName)
		t.Fatalf("Webhook receiver did not become healthy: %v", err)
	}
	log.Println("Webhook receiver is healthy.")

	auditFetcher, clearAuditLog := makeAuditHelpers(composeFiles, projectName)
	syslogFetcher := makeSyslogFetcherFile(composeFiles, projectName, "/var/log/syslog-audit/audit.log")
	syslogTLSFetcher := makeSyslogFetcherFile(composeFiles, projectName, "/var/log/syslog-audit/audit-tls.log")

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@audit-outputs && ~@pending-impl",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(restURL)
			tc.StoredValues["_audit_fetcher"] = auditFetcher
			tc.StoredValues["_webhook_receiver_url"] = webhookReceiverURL
			tc.StoredValues["_syslog_fetcher"] = syslogFetcher
			tc.StoredValues["_syslog_tls_fetcher"] = syslogTLSFetcher

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				if err := clearAuditLog(); err != nil {
					return gctx, fmt.Errorf("clear audit log: %w", err)
				}
				// Clear webhook receiver events
				if err := clearWebhookReceiver(webhookReceiverURL); err != nil {
					return gctx, fmt.Errorf("clear webhook receiver: %w", err)
				}
				// Clear syslog file
				if err := clearSyslog(composeFiles, projectName); err != nil {
					return gctx, fmt.Errorf("clear syslog: %w", err)
				}
				if err := cleanViaAPINoAuth(restURL); err != nil {
					return gctx, fmt.Errorf("clean registry: %w", err)
				}
				return gctx, nil
			})

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
			steps.RegisterAuditOutputSteps(ctx, tc)
		},
		Options: &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("Audit outputs BDD tests failed")
	}
}

// TestLDAPFeatures runs LDAP authentication BDD tests against a Docker-deployed binary
// with OpenLDAP for authentication and RBAC.
func TestLDAPFeatures(t *testing.T) {
	if bddBackend := os.Getenv("BDD_BACKEND"); bddBackend != "" && bddBackend != "memory" {
		t.Skip("LDAP Docker tests only run on memory backend (they start their own compose stack)")
	}
	if containerCmd == "" {
		containerCmd = findContainerCmd()
	}
	ldapFiles := []string{"docker-compose.base.yml", "docker-compose.ldap.yml"}
	ldapEnv := []string{
		"REGISTRY_PORT=18092",
		"WEBHOOK_PORT=19010",
		"LDAP_PORT=20636",
	}
	projectName := "bdd-ldap"

	log.Printf("Starting LDAP compose stack...")
	if err := composeUpWithProject(ldapFiles, projectName, ldapEnv); err != nil {
		t.Fatalf("Failed to start LDAP compose: %v", err)
	}
	t.Cleanup(func() {
		log.Println("Stopping LDAP compose stack...")
		composeDownWithProject(ldapFiles, projectName)
	})

	ldapURL := "http://localhost:18092"
	ldapWebhook := "http://localhost:19010"

	log.Printf("Waiting for LDAP registry at %s ...", ldapURL)
	if err := waitForURL(ldapURL+"/", 120*time.Second); err != nil {
		composeLogsWithProject(ldapFiles, projectName)
		t.Fatalf("LDAP registry did not become healthy: %v", err)
	}
	log.Println("LDAP registry is healthy.")

	ldapAuditFetcher, clearLDAPAuditLog := makeAuditHelpers(ldapFiles, projectName)

	opts := godog.Options{
		Format:   "pretty",
		Output:   colors.Colored(os.Stdout),
		Paths:    []string{"features"},
		Tags:     "@ldap && ~@pending-impl",
		Strict:   true,
		TestingT: t,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			tc := steps.NewTestContext(ldapURL)
			tc.WebhookURL = ldapWebhook
			if ldapAuditFetcher != nil {
				tc.StoredValues["_audit_fetcher"] = ldapAuditFetcher
			}

			ctx.Before(func(gctx context.Context, sc *godog.Scenario) (context.Context, error) {
				// Restart registry to reset in-memory state between scenarios.
				// LDAP users are external (in OpenLDAP), so the restart only
				// clears the in-memory storage, rate limiter, etc.
				if err := restartAuthRegistry(ldapURL, ldapWebhook); err != nil {
					return gctx, fmt.Errorf("restart LDAP registry: %w", err)
				}
				if clearLDAPAuditLog != nil {
					if err := clearLDAPAuditLog(); err != nil {
						return gctx, fmt.Errorf("clear audit log: %w", err)
					}
				}
				return gctx, nil
			})

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
		t.Fatal("LDAP BDD tests failed")
	}
}

// makeSyslogFetcherFile creates a function that reads a syslog-ng log file from the container.
func makeSyslogFetcherFile(files []string, project, logPath string) func() (string, error) {
	if containerCmd == "" {
		return nil
	}

	return func() (string, error) {
		args := []string{"compose"}
		if project != "" {
			args = append(args, "--project-name", project)
		}
		for _, f := range files {
			args = append(args, "-f", f)
		}
		args = append(args, "exec", "-T", "syslog-ng", "cat", logPath)
		cmd := exec.Command(containerCmd, args...)
		cmd.Dir = "."
		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "No such file") {
				return "", nil
			}
			return "", fmt.Errorf("read syslog log: %w: %s", err, string(out))
		}
		return string(out), nil
	}
}

// clearWebhookReceiver clears all events from the webhook receiver.
func clearWebhookReceiver(baseURL string) error {
	req, err := http.NewRequest("DELETE", baseURL+"/events", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("clear webhook receiver: %w", err)
	}
	resp.Body.Close()
	return nil
}

// clearSyslog truncates the syslog-ng log files (TCP and TLS) inside the container.
func clearSyslog(files []string, project string) error {
	if containerCmd == "" {
		return nil
	}
	args := []string{"compose"}
	if project != "" {
		args = append(args, "--project-name", project)
	}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	args = append(args, "exec", "-T", "syslog-ng", "sh", "-c",
		"truncate -s 0 /var/log/syslog-audit/audit.log 2>/dev/null; truncate -s 0 /var/log/syslog-audit/audit-tls.log 2>/dev/null; true")
	cmd := exec.Command(containerCmd, args...)
	cmd.Dir = "."
	return cmd.Run()
}

// cleanViaAPINoAuth resets all state via the REST API (no auth required).
func cleanViaAPINoAuth(baseURL string) error {
	return cleanViaAPIWithAuth(baseURL, "", "")
}

// waitForMCPEndpoint waits for the MCP HTTP endpoint to accept connections.
func waitForMCPEndpoint(mcpURL string, timeout time.Duration) error {
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Post(mcpURL, "application/json", strings.NewReader(
			`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"probe","version":"0.1"},"capabilities":{}}}`,
		))
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout waiting for MCP endpoint %s: %v", mcpURL, lastErr)
}

// cleanViaAPIWithAuth resets all state via the REST API using optional Basic auth credentials.
// When username is empty, no Authorization header is sent.
func cleanViaAPIWithAuth(baseURL, username, password string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	var authHeader string
	if username != "" {
		authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	}

	doReq := func(method, path string, body io.Reader) (*http.Response, error) {
		req, err := http.NewRequest(method, baseURL+path, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		return client.Do(req)
	}

	// 1. Reset global mode to READWRITE
	r, err := doReq("PUT", "/mode", strings.NewReader(`{"mode":"READWRITE"}`))
	if err != nil {
		return fmt.Errorf("reset mode: %w", err)
	}
	r.Body.Close()

	// 2. Soft-delete all active subjects
	resp, err := doReq("GET", "/subjects", nil)
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
		r, err := doReq("DELETE", "/subjects/"+url.PathEscape(subj), nil)
		if err != nil {
			return fmt.Errorf("soft-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 3. Permanently delete all subjects (including previously soft-deleted)
	resp, err = doReq("GET", "/subjects?deleted=true", nil)
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
		r, err := doReq("DELETE", "/subjects/"+url.PathEscape(subj)+"?permanent=true", nil)
		if err != nil {
			return fmt.Errorf("permanent-delete subject %s: %w", subj, err)
		}
		r.Body.Close()
	}

	// 4. Delete subject-level configs and modes
	seen := make(map[string]bool)
	for _, subj := range append(activeSubjects, allSubjects...) {
		if seen[subj] {
			continue
		}
		seen[subj] = true
		escaped := url.PathEscape(subj)
		if cr, err := doReq("DELETE", "/config/"+escaped, nil); err == nil {
			cr.Body.Close()
		}
		if mr, err := doReq("DELETE", "/mode/"+escaped, nil); err == nil {
			mr.Body.Close()
		}
	}

	// 6. Reset global config
	r, err = doReq("DELETE", "/config", nil)
	if err != nil {
		return fmt.Errorf("reset config: %w", err)
	}
	r.Body.Close()

	// 6. Delete all KEKs (soft + permanent)
	resp, err = doReq("GET", "/dek-registry/v1/keks", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var keks []map[string]interface{}
		if json.Unmarshal(body, &keks) == nil {
			for _, kek := range keks {
				if name, ok := kek["name"].(string); ok {
					dr, _ := doReq("DELETE", "/dek-registry/v1/keks/"+url.PathEscape(name), nil)
					if dr != nil {
						dr.Body.Close()
					}
					dr, _ = doReq("DELETE", "/dek-registry/v1/keks/"+url.PathEscape(name)+"?permanent=true", nil)
					if dr != nil {
						dr.Body.Close()
					}
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 7. Delete all exporters
	resp, err = doReq("GET", "/exporters", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var names []string
		if json.Unmarshal(body, &names) == nil {
			for _, name := range names {
				dr, _ := doReq("DELETE", "/exporters/"+url.PathEscape(name), nil)
				if dr != nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 8. Delete users created during tests (keep bootstrap admin)
	resp, err = doReq("GET", "/admin/users", nil)
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
					continue
				}
				dr, _ := doReq("DELETE", fmt.Sprintf("/admin/users/%d", int(u.ID)), nil)
				if dr != nil {
					dr.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 9. Delete all contexts (except default)
	resp, err = doReq("GET", "/contexts", nil)
	if err == nil && resp.StatusCode == 200 {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		var contexts []string
		if json.Unmarshal(body, &contexts) == nil {
			for _, ctxName := range contexts {
				if ctxName == "." || ctxName == "" {
					continue
				}
				// Clean subjects within context
				ctxResp, ctxErr := doReq("GET", "/contexts/"+url.PathEscape(ctxName)+"/subjects", nil)
				if ctxErr == nil && ctxResp.StatusCode == 200 {
					ctxBody, _ := io.ReadAll(ctxResp.Body)
					ctxResp.Body.Close()
					var ctxSubjects []string
					if json.Unmarshal(ctxBody, &ctxSubjects) == nil {
						for _, s := range ctxSubjects {
							dr, _ := doReq("DELETE", "/contexts/"+url.PathEscape(ctxName)+"/subjects/"+url.PathEscape(s), nil)
							if dr != nil {
								dr.Body.Close()
							}
							dr, _ = doReq("DELETE", "/contexts/"+url.PathEscape(ctxName)+"/subjects/"+url.PathEscape(s)+"?permanent=true", nil)
							if dr != nil {
								dr.Body.Close()
							}
						}
					}
				} else if ctxResp != nil {
					ctxResp.Body.Close()
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	return nil
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

// makeAuditHelpers creates audit log fetcher and clearer functions for a Docker compose stack.
// Returns nil functions if containerCmd is not set (external registry mode).
func makeAuditHelpers(files []string, project string) (fetcher func() (string, error), clearer func() error) {
	if containerCmd == "" {
		return nil, nil
	}

	buildArgs := func(extraArgs ...string) []string {
		args := []string{"compose"}
		if project != "" {
			args = append(args, "--project-name", project)
		}
		for _, f := range files {
			args = append(args, "-f", f)
		}
		return append(args, extraArgs...)
	}

	fetcher = func() (string, error) {
		args := buildArgs("exec", "-T", "schema-registry", "cat", "/tmp/audit.log")
		cmd := exec.Command(containerCmd, args...)
		cmd.Dir = "."
		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "No such file") {
				return "", nil
			}
			return "", fmt.Errorf("read audit log: %w: %s", err, string(out))
		}
		return string(out), nil
	}

	clearer = func() error {
		args := buildArgs("exec", "-T", "schema-registry", "sh", "-c", "truncate -s 0 /tmp/audit.log 2>/dev/null || true")
		cmd := exec.Command(containerCmd, args...)
		cmd.Dir = "."
		return cmd.Run()
	}

	return fetcher, clearer
}

// cleanBackend resets all state between scenarios using default DB ports.
// For memory: uses API cleanup (delete subjects, reset config/mode).
// For DB backends: truncates all tables and resets sequences.
func cleanBackend() error {
	return cleanBackendPort("")
}

// cleanBackendPort resets all state between scenarios.
// If dbPort is empty, uses the default port for the backend.
func cleanBackendPort(dbPort string) error {
	switch backend {
	case "postgres":
		port := dbPort
		if port == "" {
			port = envOrDefault("POSTGRES_PORT", "15432")
		}
		return cleanPostgresPort(port)
	case "mysql":
		port := dbPort
		if port == "" {
			port = envOrDefault("MYSQL_PORT", "13306")
		}
		return cleanMySQLPort(port)
	case "cassandra":
		port := dbPort
		if port == "" {
			port = envOrDefault("CASSANDRA_PORT", "19042")
		}
		return cleanCassandraPort(port)
	case "confluent":
		return cleanViaAPI()
	default:
		return cleanViaAPI()
	}
}

// cleanPostgresPort truncates all tables and resets the ID sequence.
func cleanPostgresPort(port string) error {
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

// cleanMySQLPort truncates all tables.
func cleanMySQLPort(port string) error {
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

// getCassandraSessionPort returns a Cassandra session for BDD cleanup.
// For the default port, it reuses a long-lived singleton session.
// For non-default ports, it creates a new session each time.
func getCassandraSessionPort(portStr string) (*gocql.Session, error) {
	defaultPort := envOrDefault("CASSANDRA_PORT", "19042")
	if portStr == defaultPort && cassandraSession != nil {
		return cassandraSession, nil
	}
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
	if portStr == defaultPort {
		cassandraSession = session
	}
	return session, nil
}

// cleanCassandraPort truncates all tables in the schemaregistry keyspace.
func cleanCassandraPort(port string) error {
	session, err := getCassandraSessionPort(port)
	if err != nil {
		return err
	}
	// Close non-singleton sessions after use.
	defaultPort := envOrDefault("CASSANDRA_PORT", "19042")
	if port != defaultPort {
		defer session.Close()
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

// dbOverlayFile returns the Docker Compose database overlay file for a given backend.
// Returns "" for memory/confluent/unknown backends (no DB overlay needed).
func dbOverlayFile(backend string) string {
	switch backend {
	case "postgres":
		return "docker-compose.db-postgres.yml"
	case "mysql":
		return "docker-compose.db-mysql.yml"
	case "cassandra":
		return "docker-compose.db-cassandra.yml"
	default:
		return ""
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
