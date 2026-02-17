package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8081 {
		t.Errorf("Expected port 8081, got %d", cfg.Server.Port)
	}
	if cfg.Storage.Type != "memory" {
		t.Errorf("Expected storage type memory, got %s", cfg.Storage.Type)
	}
	if cfg.Compatibility.DefaultLevel != "BACKWARD" {
		t.Errorf("Expected compatibility BACKWARD, got %s", cfg.Compatibility.DefaultLevel)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid port zero",
			cfg: &Config{
				Server:        ServerConfig{Port: 0},
				Storage:       StorageConfig{Type: "memory"},
				Compatibility: CompatibilityConfig{DefaultLevel: "BACKWARD"},
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			cfg: &Config{
				Server:        ServerConfig{Port: 70000},
				Storage:       StorageConfig{Type: "memory"},
				Compatibility: CompatibilityConfig{DefaultLevel: "BACKWARD"},
			},
			wantErr: true,
		},
		{
			name: "invalid storage type",
			cfg: &Config{
				Server:        ServerConfig{Port: 8081},
				Storage:       StorageConfig{Type: "invalid"},
				Compatibility: CompatibilityConfig{DefaultLevel: "BACKWARD"},
			},
			wantErr: true,
		},
		{
			name: "invalid compatibility level",
			cfg: &Config{
				Server:        ServerConfig{Port: 8081},
				Storage:       StorageConfig{Type: "memory"},
				Compatibility: CompatibilityConfig{DefaultLevel: "INVALID"},
			},
			wantErr: true,
		},
		{
			name: "valid postgresql",
			cfg: &Config{
				Server:        ServerConfig{Port: 8081},
				Storage:       StorageConfig{Type: "postgresql"},
				Compatibility: CompatibilityConfig{DefaultLevel: "FULL"},
			},
			wantErr: false,
		},
		{
			name: "valid all compatibility levels",
			cfg: &Config{
				Server:        ServerConfig{Port: 8081},
				Storage:       StorageConfig{Type: "memory"},
				Compatibility: CompatibilityConfig{DefaultLevel: "FULL_TRANSITIVE"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Address(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 9090,
		},
	}

	addr := cfg.Address()
	if addr != "localhost:9090" {
		t.Errorf("Expected localhost:9090, got %s", addr)
	}
}

func TestConfig_EnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("SCHEMA_REGISTRY_HOST", "127.0.0.1")
	os.Setenv("SCHEMA_REGISTRY_PORT", "9999")
	os.Setenv("SCHEMA_REGISTRY_STORAGE_TYPE", "postgresql")
	os.Setenv("SCHEMA_REGISTRY_COMPATIBILITY_LEVEL", "NONE")
	defer func() {
		os.Unsetenv("SCHEMA_REGISTRY_HOST")
		os.Unsetenv("SCHEMA_REGISTRY_PORT")
		os.Unsetenv("SCHEMA_REGISTRY_STORAGE_TYPE")
		os.Unsetenv("SCHEMA_REGISTRY_COMPATIBILITY_LEVEL")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", cfg.Server.Port)
	}
	if cfg.Storage.Type != "postgresql" {
		t.Errorf("Expected storage type postgresql, got %s", cfg.Storage.Type)
	}
	if cfg.Compatibility.DefaultLevel != "NONE" {
		t.Errorf("Expected compatibility NONE, got %s", cfg.Compatibility.DefaultLevel)
	}
}

func TestConfig_LoadFromFile(t *testing.T) {
	yaml := `
server:
  host: "10.0.0.1"
  port: 9090
  read_timeout: 60
  write_timeout: 60
storage:
  type: mysql
compatibility:
  default_level: FULL
logging:
  level: debug
  format: text
`
	tmpFile := writeTempFile(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "10.0.0.1" {
		t.Errorf("Expected host 10.0.0.1, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 60 {
		t.Errorf("Expected read_timeout 60, got %d", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Type != "mysql" {
		t.Errorf("Expected mysql, got %s", cfg.Storage.Type)
	}
	if cfg.Compatibility.DefaultLevel != "FULL" {
		t.Errorf("Expected FULL, got %s", cfg.Compatibility.DefaultLevel)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected debug, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Expected text, got %s", cfg.Logging.Format)
	}
}

func TestConfig_LoadFromFile_NotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestConfig_LoadFromFile_InvalidYAML(t *testing.T) {
	tmpFile := writeTempFile(t, "{{not: valid: yaml:")
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestConfig_LoadFromFile_InvalidConfig(t *testing.T) {
	yaml := `
server:
  port: 0
storage:
  type: memory
compatibility:
  default_level: BACKWARD
`
	tmpFile := writeTempFile(t, yaml)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("Expected validation error for port 0")
	}
}

func TestConfig_EnvOverrides_PostgreSQL(t *testing.T) {
	envVars := map[string]string{
		"SCHEMA_REGISTRY_PG_HOST":     "pg-host",
		"SCHEMA_REGISTRY_PG_PORT":     "5433",
		"SCHEMA_REGISTRY_PG_DATABASE": "mydb",
		"SCHEMA_REGISTRY_PG_USER":     "admin",
		"SCHEMA_REGISTRY_PG_PASSWORD": "secret",
		"SCHEMA_REGISTRY_PG_SSLMODE":  "require",
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.PostgreSQL.Host != "pg-host" {
		t.Errorf("Expected pg-host, got %s", cfg.Storage.PostgreSQL.Host)
	}
	if cfg.Storage.PostgreSQL.Port != 5433 {
		t.Errorf("Expected 5433, got %d", cfg.Storage.PostgreSQL.Port)
	}
	if cfg.Storage.PostgreSQL.Database != "mydb" {
		t.Errorf("Expected mydb, got %s", cfg.Storage.PostgreSQL.Database)
	}
	if cfg.Storage.PostgreSQL.User != "admin" {
		t.Errorf("Expected admin, got %s", cfg.Storage.PostgreSQL.User)
	}
	if cfg.Storage.PostgreSQL.Password != "secret" {
		t.Errorf("Expected secret, got %s", cfg.Storage.PostgreSQL.Password)
	}
	if cfg.Storage.PostgreSQL.SSLMode != "require" {
		t.Errorf("Expected require, got %s", cfg.Storage.PostgreSQL.SSLMode)
	}
}

func TestConfig_EnvOverrides_MySQL(t *testing.T) {
	envVars := map[string]string{
		"SCHEMA_REGISTRY_MYSQL_HOST":     "mysql-host",
		"SCHEMA_REGISTRY_MYSQL_PORT":     "3307",
		"SCHEMA_REGISTRY_MYSQL_DATABASE": "mydb",
		"SCHEMA_REGISTRY_MYSQL_USER":     "root",
		"SCHEMA_REGISTRY_MYSQL_PASSWORD": "pass",
		"SCHEMA_REGISTRY_MYSQL_TLS":      "skip-verify",
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.MySQL.Host != "mysql-host" {
		t.Errorf("Expected mysql-host, got %s", cfg.Storage.MySQL.Host)
	}
	if cfg.Storage.MySQL.Port != 3307 {
		t.Errorf("Expected 3307, got %d", cfg.Storage.MySQL.Port)
	}
	if cfg.Storage.MySQL.Database != "mydb" {
		t.Errorf("Expected mydb, got %s", cfg.Storage.MySQL.Database)
	}
	if cfg.Storage.MySQL.TLS != "skip-verify" {
		t.Errorf("Expected skip-verify, got %s", cfg.Storage.MySQL.TLS)
	}
}

func TestConfig_EnvOverrides_Cassandra(t *testing.T) {
	envVars := map[string]string{
		"SCHEMA_REGISTRY_CASSANDRA_HOSTS":              "node1, node2, node3",
		"SCHEMA_REGISTRY_CASSANDRA_PORT":               "9043",
		"SCHEMA_REGISTRY_CASSANDRA_KEYSPACE":           "my_keyspace",
		"SCHEMA_REGISTRY_CASSANDRA_LOCAL_DC":           "dc1",
		"SCHEMA_REGISTRY_CASSANDRA_CONSISTENCY":        "QUORUM",
		"SCHEMA_REGISTRY_CASSANDRA_READ_CONSISTENCY":   "LOCAL_ONE",
		"SCHEMA_REGISTRY_CASSANDRA_WRITE_CONSISTENCY":  "LOCAL_QUORUM",
		"SCHEMA_REGISTRY_CASSANDRA_SERIAL_CONSISTENCY": "LOCAL_SERIAL",
		"SCHEMA_REGISTRY_CASSANDRA_USERNAME":           "cassuser",
		"SCHEMA_REGISTRY_CASSANDRA_PASSWORD":           "casspass",
		"SCHEMA_REGISTRY_CASSANDRA_TIMEOUT":            "15s",
		"SCHEMA_REGISTRY_CASSANDRA_CONNECT_TIMEOUT":    "20s",
		"SCHEMA_REGISTRY_CASSANDRA_MAX_RETRIES":        "100",
		"SCHEMA_REGISTRY_CASSANDRA_ID_BLOCK_SIZE":      "200",
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Hosts should be split and trimmed
	if len(cfg.Storage.Cassandra.Hosts) != 3 {
		t.Errorf("Expected 3 hosts, got %d", len(cfg.Storage.Cassandra.Hosts))
	}
	if cfg.Storage.Cassandra.Hosts[0] != "node1" {
		t.Errorf("Expected node1, got %s", cfg.Storage.Cassandra.Hosts[0])
	}
	if cfg.Storage.Cassandra.Hosts[1] != "node2" {
		t.Errorf("Expected node2, got %s", cfg.Storage.Cassandra.Hosts[1])
	}
	if cfg.Storage.Cassandra.Port != 9043 {
		t.Errorf("Expected 9043, got %d", cfg.Storage.Cassandra.Port)
	}
	if cfg.Storage.Cassandra.Keyspace != "my_keyspace" {
		t.Errorf("Expected my_keyspace, got %s", cfg.Storage.Cassandra.Keyspace)
	}
	if cfg.Storage.Cassandra.LocalDC != "dc1" {
		t.Errorf("Expected dc1, got %s", cfg.Storage.Cassandra.LocalDC)
	}
	if cfg.Storage.Cassandra.Consistency != "QUORUM" {
		t.Errorf("Expected QUORUM, got %s", cfg.Storage.Cassandra.Consistency)
	}
	if cfg.Storage.Cassandra.ReadConsistency != "LOCAL_ONE" {
		t.Errorf("Expected LOCAL_ONE, got %s", cfg.Storage.Cassandra.ReadConsistency)
	}
	if cfg.Storage.Cassandra.WriteConsistency != "LOCAL_QUORUM" {
		t.Errorf("Expected LOCAL_QUORUM, got %s", cfg.Storage.Cassandra.WriteConsistency)
	}
	if cfg.Storage.Cassandra.SerialConsistency != "LOCAL_SERIAL" {
		t.Errorf("Expected LOCAL_SERIAL, got %s", cfg.Storage.Cassandra.SerialConsistency)
	}
	if cfg.Storage.Cassandra.Username != "cassuser" {
		t.Errorf("Expected cassuser, got %s", cfg.Storage.Cassandra.Username)
	}
	if cfg.Storage.Cassandra.Password != "casspass" {
		t.Errorf("Expected casspass, got %s", cfg.Storage.Cassandra.Password)
	}
	if cfg.Storage.Cassandra.Timeout != "15s" {
		t.Errorf("Expected 15s, got %s", cfg.Storage.Cassandra.Timeout)
	}
	if cfg.Storage.Cassandra.ConnectTimeout != "20s" {
		t.Errorf("Expected 20s, got %s", cfg.Storage.Cassandra.ConnectTimeout)
	}
	if cfg.Storage.Cassandra.MaxRetries != 100 {
		t.Errorf("Expected 100, got %d", cfg.Storage.Cassandra.MaxRetries)
	}
	if cfg.Storage.Cassandra.IDBlockSize != 200 {
		t.Errorf("Expected 200, got %d", cfg.Storage.Cassandra.IDBlockSize)
	}
}

func TestConfig_EnvOverrides_Bootstrap(t *testing.T) {
	envVars := map[string]string{
		"SCHEMA_REGISTRY_BOOTSTRAP_ENABLED":  "true",
		"SCHEMA_REGISTRY_BOOTSTRAP_USERNAME": "admin",
		"SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD": "adminpass",
		"SCHEMA_REGISTRY_BOOTSTRAP_EMAIL":    "admin@example.com",
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.Security.Auth.Bootstrap.Enabled {
		t.Error("Expected bootstrap enabled")
	}
	if cfg.Security.Auth.Bootstrap.Username != "admin" {
		t.Errorf("Expected admin, got %s", cfg.Security.Auth.Bootstrap.Username)
	}
	if cfg.Security.Auth.Bootstrap.Password != "adminpass" {
		t.Errorf("Expected adminpass, got %s", cfg.Security.Auth.Bootstrap.Password)
	}
	if cfg.Security.Auth.Bootstrap.Email != "admin@example.com" {
		t.Errorf("Expected admin@example.com, got %s", cfg.Security.Auth.Bootstrap.Email)
	}
}

func TestConfig_EnvOverrides_Vault(t *testing.T) {
	envVars := map[string]string{
		"SCHEMA_REGISTRY_VAULT_ADDRESS":    "https://vault.example.com",
		"SCHEMA_REGISTRY_VAULT_TOKEN":      "s.token123",
		"SCHEMA_REGISTRY_VAULT_NAMESPACE":  "ns1",
		"SCHEMA_REGISTRY_VAULT_MOUNT_PATH": "kv",
		"SCHEMA_REGISTRY_VAULT_BASE_PATH":  "sr",
	}
	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.Vault.Address != "https://vault.example.com" {
		t.Errorf("Expected vault address, got %s", cfg.Storage.Vault.Address)
	}
	if cfg.Storage.Vault.Token != "s.token123" {
		t.Errorf("Expected vault token, got %s", cfg.Storage.Vault.Token)
	}
	if cfg.Storage.Vault.Namespace != "ns1" {
		t.Errorf("Expected ns1, got %s", cfg.Storage.Vault.Namespace)
	}
	if cfg.Storage.Vault.MountPath != "kv" {
		t.Errorf("Expected kv, got %s", cfg.Storage.Vault.MountPath)
	}
	if cfg.Storage.Vault.BasePath != "sr" {
		t.Errorf("Expected sr, got %s", cfg.Storage.Vault.BasePath)
	}
}

func TestConfig_EnvOverrides_VaultTokenFallback(t *testing.T) {
	// VAULT_TOKEN should be used as fallback when SCHEMA_REGISTRY_VAULT_TOKEN is not set
	os.Setenv("VAULT_TOKEN", "fallback-token")
	defer os.Unsetenv("VAULT_TOKEN")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.Vault.Token != "fallback-token" {
		t.Errorf("Expected fallback-token, got %s", cfg.Storage.Vault.Token)
	}
}

func TestConfig_EnvOverrides_InvalidPort(t *testing.T) {
	// Non-numeric port should be ignored, keeping default
	os.Setenv("SCHEMA_REGISTRY_PORT", "not-a-number")
	defer os.Unsetenv("SCHEMA_REGISTRY_PORT")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 8081 {
		t.Errorf("Expected default port 8081, got %d", cfg.Server.Port)
	}
}

func TestConfig_Validate_AuthType(t *testing.T) {
	tests := []struct {
		name     string
		authType string
		wantErr  bool
	}{
		{"valid vault", "vault", false},
		{"valid postgresql", "postgresql", false},
		{"valid mysql", "mysql", false},
		{"valid cassandra", "cassandra", false},
		{"valid memory", "memory", false},
		{"invalid", "redis", true},
		{"empty is ok", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage.AuthType = tt.authType
			// vault requires address
			if tt.authType == "vault" {
				cfg.Storage.Vault.Address = "http://localhost:8200"
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_VaultRequiresAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.AuthType = "vault"
	cfg.Storage.Vault.Address = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when vault auth_type has no address")
	}
}

func TestConfig_Validate_AllCompatibilityLevels(t *testing.T) {
	levels := []string{
		"NONE", "BACKWARD", "BACKWARD_TRANSITIVE",
		"FORWARD", "FORWARD_TRANSITIVE",
		"FULL", "FULL_TRANSITIVE",
	}
	for _, level := range levels {
		cfg := DefaultConfig()
		cfg.Compatibility.DefaultLevel = level
		if err := cfg.Validate(); err != nil {
			t.Errorf("level %s should be valid: %v", level, err)
		}
	}
}

func TestConfig_Validate_CaseInsensitiveCompatibility(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compatibility.DefaultLevel = "backward"
	if err := cfg.Validate(); err != nil {
		t.Errorf("lowercase should be valid: %v", err)
	}
}

func TestConfig_Validate_AllStorageTypes(t *testing.T) {
	types := []string{"memory", "postgresql", "mysql", "cassandra"}
	for _, st := range types {
		cfg := DefaultConfig()
		cfg.Storage.Type = st
		if err := cfg.Validate(); err != nil {
			t.Errorf("storage type %s should be valid: %v", st, err)
		}
	}
}

func TestConfig_Validate_PortBoundary(t *testing.T) {
	// Port 1 should be valid
	cfg := DefaultConfig()
	cfg.Server.Port = 1
	if err := cfg.Validate(); err != nil {
		t.Errorf("port 1 should be valid: %v", err)
	}

	// Port 65535 should be valid
	cfg.Server.Port = 65535
	if err := cfg.Validate(); err != nil {
		t.Errorf("port 65535 should be valid: %v", err)
	}

	// Port 65536 should be invalid
	cfg.Server.Port = 65536
	if err := cfg.Validate(); err == nil {
		t.Error("port 65536 should be invalid")
	}

	// Negative port should be invalid
	cfg.Server.Port = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative port should be invalid")
	}
}

func TestConfig_LoadFromFile_EnvExpansion(t *testing.T) {
	os.Setenv("TEST_DB_HOST", "env-expanded-host")
	defer os.Unsetenv("TEST_DB_HOST")

	yaml := `
server:
  port: 8081
storage:
  type: memory
  postgresql:
    host: "${TEST_DB_HOST}"
compatibility:
  default_level: BACKWARD
`
	tmpFile := writeTempFile(t, yaml)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.PostgreSQL.Host != "env-expanded-host" {
		t.Errorf("Expected env-expanded-host, got %s", cfg.Storage.PostgreSQL.Host)
	}
}

func TestConfig_EnvOverrides_LogLevel(t *testing.T) {
	os.Setenv("SCHEMA_REGISTRY_LOG_LEVEL", "error")
	defer os.Unsetenv("SCHEMA_REGISTRY_LOG_LEVEL")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Logging.Level != "error" {
		t.Errorf("Expected error, got %s", cfg.Logging.Level)
	}
}

func TestConfig_EnvOverrides_AuthType(t *testing.T) {
	os.Setenv("SCHEMA_REGISTRY_AUTH_TYPE", "vault")
	os.Setenv("SCHEMA_REGISTRY_VAULT_ADDRESS", "http://vault:8200")
	defer func() {
		os.Unsetenv("SCHEMA_REGISTRY_AUTH_TYPE")
		os.Unsetenv("SCHEMA_REGISTRY_VAULT_ADDRESS")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Storage.AuthType != "vault" {
		t.Errorf("Expected vault, got %s", cfg.Storage.AuthType)
	}
}

func TestConfig_DefaultConfig_CacheRefreshSeconds(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.Auth.APIKey.CacheRefreshSeconds != 60 {
		t.Errorf("Expected default CacheRefreshSeconds 60, got %d", cfg.Security.Auth.APIKey.CacheRefreshSeconds)
	}
}

func TestConfig_LoadEmpty(t *testing.T) {
	// Loading with empty path and no env overrides should return defaults
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	def := DefaultConfig()
	if cfg.Server.Host != def.Server.Host {
		t.Errorf("Expected default host %s, got %s", def.Server.Host, cfg.Server.Host)
	}
	if cfg.Server.Port != def.Server.Port {
		t.Errorf("Expected default port %d, got %d", def.Server.Port, cfg.Server.Port)
	}
}

// writeTempFile creates a temporary file with the given content and returns its path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
