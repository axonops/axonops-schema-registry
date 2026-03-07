package mysql

import (
	"errors"
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
	if cfg.Port != 3306 {
		t.Errorf("expected Port 3306, got %d", cfg.Port)
	}
	if cfg.Database != "schema_registry" {
		t.Errorf("expected Database schema_registry, got %q", cfg.Database)
	}
	if cfg.Username != "root" {
		t.Errorf("expected Username root, got %q", cfg.Username)
	}
	if cfg.TLS != "false" {
		t.Errorf("expected TLS false, got %q", cfg.TLS)
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
		Port:     3307,
		Database: "mydb",
		Username: "admin",
		Password: "secret",
		TLS:      "true",
	}

	dsn := cfg.DSN()
	expected := "admin:secret@tcp(db.example.com:3307)/mydb?parseTime=true&tls=true&timeout=10s&readTimeout=30s&writeTimeout=30s"
	if dsn != expected {
		t.Errorf("DSN mismatch\n  got:  %q\n  want: %q", dsn, expected)
	}
}

func TestConfig_DSN_DefaultValues(t *testing.T) {
	cfg := DefaultConfig()
	dsn := cfg.DSN()

	if !strings.Contains(dsn, "tcp(localhost:3306)") {
		t.Errorf("expected DSN to contain tcp(localhost:3306), got %q", dsn)
	}
	if !strings.Contains(dsn, "parseTime=true") {
		t.Errorf("expected DSN to contain parseTime=true, got %q", dsn)
	}
	if !strings.Contains(dsn, "tls=false") {
		t.Errorf("expected DSN to contain tls=false, got %q", dsn)
	}
	if !strings.Contains(dsn, "timeout=10s") {
		t.Errorf("expected DSN to contain timeout=10s, got %q", dsn)
	}
	if !strings.Contains(dsn, "readTimeout=30s") {
		t.Errorf("expected DSN to contain readTimeout=30s, got %q", dsn)
	}
	if !strings.Contains(dsn, "writeTimeout=30s") {
		t.Errorf("expected DSN to contain writeTimeout=30s, got %q", dsn)
	}
}

func TestConfig_DSN_TLSOptions(t *testing.T) {
	testCases := []struct {
		tls      string
		expected string
	}{
		{"true", "tls=true"},
		{"false", "tls=false"},
		{"skip-verify", "tls=skip-verify"},
		{"preferred", "tls=preferred"},
	}

	for _, tc := range testCases {
		t.Run(tc.tls, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.TLS = tc.tls
			dsn := cfg.DSN()
			if !strings.Contains(dsn, tc.expected) {
				t.Errorf("expected DSN to contain %q, got %q", tc.expected, dsn)
			}
		})
	}
}

func TestConfig_DSN_SpecialCharsInPassword(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     3306,
		Database: "db",
		Username: "user",
		Password: "p@ss:w0rd",
		TLS:      "false",
	}
	dsn := cfg.DSN()
	expected := fmt.Sprintf("user:p@ss:w0rd@tcp(localhost:3306)/db?parseTime=true&tls=false&timeout=10s&readTimeout=30s&writeTimeout=30s")
	if dsn != expected {
		t.Errorf("DSN with special chars mismatch\n  got:  %q\n  want: %q", dsn, expected)
	}
}

// ---------------------------------------------------------------------------
// isInvalidConnErr
// ---------------------------------------------------------------------------

func TestIsInvalidConnErr_Nil(t *testing.T) {
	if isInvalidConnErr(nil) {
		t.Error("nil error should not be an invalid connection error")
	}
}

func TestIsInvalidConnErr_InvalidConnection(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		invalid bool
	}{
		{"invalid connection", errors.New("invalid connection"), true},
		{"unexpected EOF", errors.New("unexpected EOF"), true},
		{"broken pipe", errors.New("write: broken pipe"), true},
		{"connection reset", errors.New("read: connection reset by peer"), true},
		{"regular error", errors.New("syntax error"), false},
		{"timeout", errors.New("i/o timeout"), false},
		{"wrapped invalid conn", fmt.Errorf("query failed: %w", errors.New("invalid connection")), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isInvalidConnErr(tc.err)
			if got != tc.invalid {
				t.Errorf("isInvalidConnErr(%q) = %v, want %v", tc.err, got, tc.invalid)
			}
		})
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
		"CREATE TABLE IF NOT EXISTS `schemas`",
		"CREATE TABLE IF NOT EXISTS schema_references",
		"CREATE TABLE IF NOT EXISTS configs",
		"CREATE TABLE IF NOT EXISTS modes",
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS api_keys",
		"CREATE TABLE IF NOT EXISTS id_alloc",
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

func TestMigrations_UsesInnoDB(t *testing.T) {
	// All CREATE TABLE statements should use InnoDB engine
	for i, m := range migrations {
		if strings.Contains(m, "CREATE TABLE") && !strings.Contains(m, "ENGINE=InnoDB") {
			t.Errorf("migration %d creates table without ENGINE=InnoDB: %s", i, truncate(m, 80))
		}
	}
}

func TestMigrations_UsesUTF8MB4(t *testing.T) {
	// All CREATE TABLE statements should use utf8mb4 charset
	for i, m := range migrations {
		if strings.Contains(m, "CREATE TABLE") && !strings.Contains(m, "utf8mb4") {
			t.Errorf("migration %d creates table without utf8mb4 charset: %s", i, truncate(m, 80))
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

func TestMigrations_CaseSensitiveContextCollation(t *testing.T) {
	// MySQL needs utf8mb4_bin collation for case-sensitive context names
	found := false
	for _, m := range migrations {
		if strings.Contains(m, "utf8mb4_bin") {
			found = true
			break
		}
	}
	if !found {
		t.Error("migrations must include utf8mb4_bin collation for case-sensitive registry_ctx")
	}
}

// ---------------------------------------------------------------------------
// Migrations count — guard against accidental deletion
// ---------------------------------------------------------------------------

func TestMigrations_MinimumCount(t *testing.T) {
	// As of the current codebase, there are 46+ migration statements.
	minExpected := 40
	if len(migrations) < minExpected {
		t.Errorf("expected at least %d migrations, got %d", minExpected, len(migrations))
	}
}

// ---------------------------------------------------------------------------
// NewStore — config validation (will fail to connect but defaults should apply)
// ---------------------------------------------------------------------------

func TestNewStore_AppliesDefaults(t *testing.T) {
	// NewStore will fail at Ping because there is no real database,
	// but we can verify sql.Open succeeds.
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		TLS:      "false",
	}

	_, err := NewStore(cfg)
	if err == nil {
		t.Fatal("expected error (no database running), got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "failed to ping database") &&
		!strings.Contains(errStr, "failed to open database") {
		t.Errorf("expected connection error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
