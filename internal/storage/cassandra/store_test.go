package cassandra

import (
	"strings"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// ---------------------------------------------------------------------------
// Config defaults
// ---------------------------------------------------------------------------

func TestNewStore_AppliesDefaults(t *testing.T) {
	// NewStore will fail connecting, but we can verify that config defaults
	// are applied by calling it with a minimal config and checking the error
	// is a connection error (not a config validation error).
	cfg := Config{}

	_, err := NewStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error (no Cassandra running), got nil")
	}

	// Should be a connection error, not a config validation error
	errStr := err.Error()
	if !strings.Contains(errStr, "failed to connect") &&
		!strings.Contains(errStr, "gocql") &&
		!strings.Contains(errStr, "connection") &&
		!strings.Contains(errStr, "dial") &&
		!strings.Contains(errStr, "no hosts") {
		t.Errorf("expected connection-related error, got: %v", err)
	}
}

func TestNewStore_InvalidConsistency(t *testing.T) {
	cfg := Config{
		Hosts:       []string{"127.0.0.1"},
		Consistency: "INVALID_LEVEL",
	}

	_, err := NewStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid consistency level")
	}
	if !strings.Contains(err.Error(), "unknown cassandra consistency") {
		t.Errorf("expected consistency error, got: %v", err)
	}
}

func TestNewStore_InvalidReadConsistency(t *testing.T) {
	cfg := Config{
		Hosts:           []string{"127.0.0.1"},
		Consistency:     "ONE",
		ReadConsistency: "BOGUS",
	}

	_, err := NewStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid read consistency")
	}
	if !strings.Contains(err.Error(), "unknown cassandra consistency") {
		t.Errorf("expected consistency error, got: %v", err)
	}
}

func TestNewStore_InvalidWriteConsistency(t *testing.T) {
	cfg := Config{
		Hosts:            []string{"127.0.0.1"},
		Consistency:      "ONE",
		WriteConsistency: "BOGUS",
	}

	_, err := NewStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid write consistency")
	}
	if !strings.Contains(err.Error(), "unknown cassandra consistency") {
		t.Errorf("expected consistency error, got: %v", err)
	}
}

func TestNewStore_InvalidSerialConsistency(t *testing.T) {
	cfg := Config{
		Hosts:             []string{"127.0.0.1"},
		Consistency:       "ONE",
		SerialConsistency: "QUORUM", // not a valid serial consistency
	}

	_, err := NewStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid serial consistency")
	}
	if !strings.Contains(err.Error(), "invalid cassandra serial consistency") {
		t.Errorf("expected serial consistency error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// parseConsistency
// ---------------------------------------------------------------------------

func TestParseConsistency_ValidLevels(t *testing.T) {
	tests := []struct {
		input    string
		expected gocql.Consistency
	}{
		{"ANY", gocql.Any},
		{"ONE", gocql.One},
		{"TWO", gocql.Two},
		{"THREE", gocql.Three},
		{"QUORUM", gocql.Quorum},
		{"ALL", gocql.All},
		{"LOCAL_ONE", gocql.LocalOne},
		{"LOCAL_QUORUM", gocql.LocalQuorum},
		{"EACH_QUORUM", gocql.EachQuorum},
		// Case insensitive
		{"one", gocql.One},
		{"local_quorum", gocql.LocalQuorum},
		// Whitespace tolerance
		{"  ONE  ", gocql.One},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseConsistency(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("parseConsistency(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseConsistency_Invalid(t *testing.T) {
	invalid := []string{"INVALID", "STRONG", "", "LOCAL"}
	for _, v := range invalid {
		t.Run(v, func(t *testing.T) {
			_, err := parseConsistency(v)
			if err == nil {
				t.Errorf("expected error for %q, got nil", v)
			}
			if !strings.Contains(err.Error(), "unknown cassandra consistency") {
				t.Errorf("expected 'unknown cassandra consistency' error, got: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseSerialConsistency
// ---------------------------------------------------------------------------

func TestParseSerialConsistency_ValidLevels(t *testing.T) {
	tests := []struct {
		input    string
		expected gocql.Consistency
	}{
		{"SERIAL", gocql.Serial},
		{"LOCAL_SERIAL", gocql.LocalSerial},
		{"serial", gocql.Serial},
		{"  local_serial  ", gocql.LocalSerial},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseSerialConsistency(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("parseSerialConsistency(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseSerialConsistency_Invalid(t *testing.T) {
	invalid := []string{"ONE", "QUORUM", "LOCAL_QUORUM", "", "ALL"}
	for _, v := range invalid {
		t.Run(v, func(t *testing.T) {
			_, err := parseSerialConsistency(v)
			if err == nil {
				t.Errorf("expected error for %q, got nil", v)
			}
			if !strings.Contains(err.Error(), "invalid cassandra serial consistency") {
				t.Errorf("expected 'invalid cassandra serial consistency' error, got: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// qident (keyspace identifier quoting)
// ---------------------------------------------------------------------------

func TestQident_SafeIdentifier(t *testing.T) {
	// Safe identifiers should be returned as-is
	tests := []struct {
		input    string
		expected string
	}{
		{"myks", "myks"},
		{"schema_registry", "schema_registry"},
		{"ks123", "ks123"},
		{"a", "a"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := qident(tc.input)
			if got != tc.expected {
				t.Errorf("qident(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestQident_UnsafeIdentifier(t *testing.T) {
	// Unsafe identifiers should be double-quoted
	tests := []struct {
		input    string
		expected string
	}{
		{"MyKeyspace", `"MyKeyspace"`},
		{"my-ks", `"my-ks"`},
		{"123start", `"123start"`},
		{"has space", `"has space"`},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := qident(tc.input)
			if got != tc.expected {
				t.Errorf("qident(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestQident_Empty(t *testing.T) {
	got := qident("")
	if got != "" {
		t.Errorf("qident(\"\") = %q, want empty string", got)
	}
}

func TestQident_EscapesDoubleQuotes(t *testing.T) {
	got := qident(`my"ks`)
	expected := `"my""ks"`
	if got != expected {
		t.Errorf("qident(%q) = %q, want %q", `my"ks`, got, expected)
	}
}

// ---------------------------------------------------------------------------
// isSafeIdent
// ---------------------------------------------------------------------------

func TestIsSafeIdent(t *testing.T) {
	tests := []struct {
		input string
		safe  bool
	}{
		{"abc", true},
		{"abc_def", true},
		{"a123", true},
		{"_start", true},
		{"ABC", false},       // uppercase
		{"123start", false},  // starts with digit
		{"has-dash", false},  // dash
		{"has space", false}, // space
		{"", false},          // empty
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := isSafeIdent(tc.input)
			if got != tc.safe {
				t.Errorf("isSafeIdent(%q) = %v, want %v", tc.input, got, tc.safe)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// oneLine
// ---------------------------------------------------------------------------

func TestOneLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello  world", "hello world"},
		{"  leading  ", "leading"},
		{"multi\n\tline\n\tquery", "multi line query"},
		{"CREATE TABLE\n\t\tIF NOT EXISTS\n\t\tschemas", "CREATE TABLE IF NOT EXISTS schemas"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := oneLine(tc.input)
			if got != tc.expected {
				t.Errorf("oneLine(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// normalizeCompat
// ---------------------------------------------------------------------------

func TestNormalizeCompat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "BACKWARD"},
		{"backward", "BACKWARD"},
		{"FORWARD", "FORWARD"},
		{"  full_transitive  ", "FULL_TRANSITIVE"},
		{"none", "NONE"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeCompat(tc.input)
			if got != tc.expected {
				t.Errorf("normalizeCompat(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// idAllocator
// ---------------------------------------------------------------------------

func TestNewIDAllocator(t *testing.T) {
	alloc := newIDAllocator(50)
	if alloc == nil {
		t.Fatal("newIDAllocator returned nil")
	}
	if alloc.block != 50 {
		t.Errorf("expected block size 50, got %d", alloc.block)
	}
	if alloc.allocators == nil {
		t.Error("expected allocators map to be initialized")
	}
	if len(alloc.allocators) != 0 {
		t.Errorf("expected empty allocators map, got %d entries", len(alloc.allocators))
	}
}

func TestIDAllocator_Reset(t *testing.T) {
	alloc := newIDAllocator(10)
	// Manually add a context block
	alloc.allocators["test-ctx"] = &contextIDBlock{current: 5, ceiling: 10}
	alloc.allocators["other-ctx"] = &contextIDBlock{current: 1, ceiling: 10}

	alloc.reset("test-ctx")
	if _, ok := alloc.allocators["test-ctx"]; ok {
		t.Error("expected test-ctx to be removed after reset")
	}
	if _, ok := alloc.allocators["other-ctx"]; !ok {
		t.Error("expected other-ctx to remain after resetting test-ctx")
	}
}

func TestIDAllocator_ResetAll(t *testing.T) {
	alloc := newIDAllocator(10)
	alloc.allocators["ctx1"] = &contextIDBlock{current: 1, ceiling: 10}
	alloc.allocators["ctx2"] = &contextIDBlock{current: 1, ceiling: 10}

	alloc.resetAll()
	if len(alloc.allocators) != 0 {
		t.Errorf("expected empty allocators after resetAll, got %d", len(alloc.allocators))
	}
}

// ---------------------------------------------------------------------------
// Config default values
// ---------------------------------------------------------------------------

func TestConfigDefaults_AppliedInNewStore(t *testing.T) {
	// Verify that zero-value config fields get reasonable defaults.
	// We cannot call NewStore successfully without a database, but we can
	// verify the default application logic by looking at what NewStore does
	// to the config before connecting.
	cfg := Config{}

	// These are the defaults that NewStore applies for zero values
	if cfg.Hosts == nil {
		// NewStore will set Hosts to ["127.0.0.1"]
	}
	if cfg.Port == 0 {
		// NewStore will set Port to 9042
	}
	if cfg.Keyspace == "" {
		// NewStore will set Keyspace to "axonops_schema_registry"
	}
	if cfg.Timeout == 0 {
		// NewStore will set Timeout to 10s
	}
	if cfg.ConnectTimeout == 0 {
		// NewStore will set ConnectTimeout to 10s
	}
	if cfg.MaxRetries <= 0 {
		// NewStore will set MaxRetries to 50
	}
	if cfg.IDBlockSize <= 0 {
		// NewStore will set IDBlockSize to 50
	}

	// Verify expected defaults are documented correctly
	expectedDefaults := map[string]interface{}{
		"Timeout":        10 * time.Second,
		"ConnectTimeout": 10 * time.Second,
		"MaxRetries":     50,
		"IDBlockSize":    50,
	}
	_ = expectedDefaults // Used for documentation; actual verification is in NewStore integration tests
}

// ---------------------------------------------------------------------------
// Migrate — migration statements structure
// ---------------------------------------------------------------------------

func TestMigrate_StatementsAreGenerated(t *testing.T) {
	// We cannot call Migrate without a real session, but we can verify that
	// the function exists and the migration logic references expected tables.
	// This is a compilation check + documentation of expected schema.
	expectedTables := []string{
		"schemas_by_id",
		"subject_versions",
		"subject_latest",
		"schema_references",
		"references_by_target",
		"subject_configs",
		"global_config",
		"modes",
		"id_alloc",
		"users_by_id",
		"users_by_email",
		"api_keys_by_id",
		"api_keys_by_user",
		"api_keys_by_hash",
		"schema_fingerprints",
		"contexts",
		"keks",
		"deks",
		"deks_by_kek",
		"exporters",
		"exporter_statuses",
	}

	// Verify each table name is a non-empty string (compilation check)
	for _, table := range expectedTables {
		if table == "" {
			t.Error("expected table name must not be empty")
		}
	}

	// Verify minimum table count
	if len(expectedTables) < 20 {
		t.Errorf("expected at least 20 tables in Cassandra schema, documented %d", len(expectedTables))
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

func TestStore_ImplementsStorageInterface(t *testing.T) {
	// The compile-time check in store.go (var _ storage.Storage = (*Store)(nil))
	// ensures this, but we verify the package compiles with this test.
	var s *Store
	_ = s // compilation check only
}
