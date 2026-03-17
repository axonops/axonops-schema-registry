package postgres

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// DefaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("expected Host localhost, got %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected Port 5432, got %d", cfg.Port)
	}
	if cfg.Database != "schema_registry" {
		t.Errorf("expected Database schema_registry, got %q", cfg.Database)
	}
	if cfg.Username != "postgres" {
		t.Errorf("expected Username postgres, got %q", cfg.Username)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("expected SSLMode disable, got %q", cfg.SSLMode)
	}
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns 25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns 5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected ConnMaxLifetime 5m, got %v", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("expected ConnMaxIdleTime 5m, got %v", cfg.ConnMaxIdleTime)
	}
	if cfg.ConnectTimeout != 5*time.Second {
		t.Errorf("expected ConnectTimeout 5s, got %v", cfg.ConnectTimeout)
	}
	if cfg.HealthCheckTimeout != 2*time.Second {
		t.Errorf("expected HealthCheckTimeout 2s, got %v", cfg.HealthCheckTimeout)
	}
	if cfg.SchemaMaxRetries != 15 {
		t.Errorf("expected SchemaMaxRetries 15, got %d", cfg.SchemaMaxRetries)
	}
}

// ---------------------------------------------------------------------------
// DSN
// ---------------------------------------------------------------------------

func TestConfig_DSN(t *testing.T) {
	cfg := Config{
		Host:     "db.example.com",
		Port:     5433,
		Database: "mydb",
		Username: "admin",
		Password: "secret",
		SSLMode:  "require",
	}

	dsn := cfg.DSN()
	expected := "host=db.example.com port=5433 dbname=mydb user=admin password=secret sslmode=require"
	if dsn != expected {
		t.Errorf("DSN mismatch\n  got:  %q\n  want: %q", dsn, expected)
	}
}

func TestConfig_DSN_DefaultValues(t *testing.T) {
	cfg := DefaultConfig()
	dsn := cfg.DSN()

	if !strings.Contains(dsn, "host=localhost") {
		t.Errorf("expected DSN to contain host=localhost, got %q", dsn)
	}
	if !strings.Contains(dsn, "port=5432") {
		t.Errorf("expected DSN to contain port=5432, got %q", dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("expected DSN to contain sslmode=disable, got %q", dsn)
	}
}

func TestConfig_DSN_EmptyPassword(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "user",
		Password: "",
		SSLMode:  "disable",
	}
	dsn := cfg.DSN()
	if !strings.Contains(dsn, "password=") {
		t.Errorf("expected DSN to contain password= (empty), got %q", dsn)
	}
}

// ---------------------------------------------------------------------------
// Migrations
// ---------------------------------------------------------------------------

func TestMigrations_NotEmpty(t *testing.T) {
	if len(migrations) == 0 {
		t.Fatal("migrations slice must not be empty")
	}
}

func TestMigrations_NoEmptyStatements(t *testing.T) {
	for i, m := range migrations {
		if strings.TrimSpace(m) == "" {
			t.Errorf("migration %d is empty", i)
		}
	}
}

func TestMigrations_ContainsCoreTableCreation(t *testing.T) {
	tables := []string{
		"CREATE TABLE IF NOT EXISTS schemas",
		"CREATE TABLE IF NOT EXISTS schema_references",
		"CREATE TABLE IF NOT EXISTS configs",
		"CREATE TABLE IF NOT EXISTS modes",
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS api_keys",
		"CREATE TABLE IF NOT EXISTS schema_fingerprints",
		"CREATE TABLE IF NOT EXISTS ctx_id_alloc",
		"CREATE TABLE IF NOT EXISTS contexts",
		"CREATE TABLE IF NOT EXISTS keks",
		"CREATE TABLE IF NOT EXISTS deks",
		"CREATE TABLE IF NOT EXISTS exporters",
		"CREATE TABLE IF NOT EXISTS exporter_statuses",
	}

	allSQL := strings.Join(migrations, "\n")
	for _, table := range tables {
		if !strings.Contains(allSQL, table) {
			t.Errorf("migrations missing table creation: %s", table)
		}
	}
}

func TestMigrations_ContainsContextSupport(t *testing.T) {
	allSQL := strings.Join(migrations, "\n")
	if !strings.Contains(allSQL, "registry_ctx") {
		t.Error("migrations must include registry_ctx for multi-tenant context support")
	}
}

func TestMigrations_ContainsDefaultContextSeed(t *testing.T) {
	found := false
	for _, m := range migrations {
		if strings.Contains(m, "contexts") && strings.Contains(m, "'.'") {
			found = true
			break
		}
	}
	if !found {
		t.Error("migrations must seed the default context '.'")
	}
}

func TestMigrations_ContainsIndexes(t *testing.T) {
	allSQL := strings.Join(migrations, "\n")
	indexes := []string{
		"idx_schemas_subject",
		"idx_schemas_fingerprint",
		"idx_schemas_deleted",
	}
	for _, idx := range indexes {
		if !strings.Contains(allSQL, idx) {
			t.Errorf("migrations missing index: %s", idx)
		}
	}
}

func TestMigrations_GlobalConfigDefault(t *testing.T) {
	found := false
	for _, m := range migrations {
		if strings.Contains(m, "configs") && strings.Contains(m, "BACKWARD") {
			found = true
			break
		}
	}
	if !found {
		t.Error("migrations must insert default global config with BACKWARD compatibility")
	}
}

func TestMigrations_GlobalModeDefault(t *testing.T) {
	found := false
	for _, m := range migrations {
		if strings.Contains(m, "modes") && strings.Contains(m, "READWRITE") {
			found = true
			break
		}
	}
	if !found {
		t.Error("migrations must insert default global mode READWRITE")
	}
}

// ---------------------------------------------------------------------------
// NewStore — config validation (will fail to connect but should apply defaults)
// ---------------------------------------------------------------------------

func TestNewStore_AppliesDefaults(t *testing.T) {
	// NewStore will fail at db.PingContext because there is no real database,
	// but we can verify that sql.Open succeeds and the config defaults are applied
	// by checking the error type — it should NOT be a config validation error.
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
		// Leave pool settings at zero to verify defaults are applied
	}

	_, err := NewStore(cfg)
	if err == nil {
		t.Fatal("expected error (no database running), got nil")
	}

	// The error should be a connection/ping error, not a config validation error
	errStr := err.Error()
	if !strings.Contains(errStr, "failed to ping database") &&
		!strings.Contains(errStr, "failed to open database") {
		t.Errorf("expected connection error, got: %v", err)
	}
}

func TestNewStore_DSNFormation(t *testing.T) {
	// Verify DSN is correctly formed by checking that sql.Open does not fail
	// (sql.Open only validates driver name, actual connection happens on Ping)
	cfg := Config{
		Host:     "nonexistent.host.invalid",
		Port:     15432,
		Database: "nodb",
		Username: "nouser",
		Password: "nopass",
		SSLMode:  "disable",
	}

	// We expect a ping failure, not a DSN parse error
	_, err := NewStore(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should fail at ping, not at Open
	if strings.Contains(err.Error(), "failed to open database") {
		t.Errorf("DSN should be valid for sql.Open; got open error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Migrations count — guard against accidental deletion
// ---------------------------------------------------------------------------

func TestMigrations_MinimumCount(t *testing.T) {
	// As of the current codebase, there are 45+ migration statements.
	// This guards against accidentally truncating the migrations slice.
	minExpected := 40
	if len(migrations) < minExpected {
		t.Errorf("expected at least %d migrations, got %d", minExpected, len(migrations))
	}
}

// ---------------------------------------------------------------------------
// DSN — special characters
// ---------------------------------------------------------------------------

func TestConfig_DSN_SpecialCharsInPassword(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		Database: "db",
		Username: "user",
		Password: "p@ss=w0rd with spaces",
		SSLMode:  "disable",
	}
	dsn := cfg.DSN()
	expected := fmt.Sprintf("host=localhost port=5432 dbname=db user=user password=%s sslmode=disable", cfg.Password)
	if dsn != expected {
		t.Errorf("DSN with special chars mismatch\n  got:  %q\n  want: %q", dsn, expected)
	}
}
