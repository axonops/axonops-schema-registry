// Package config provides configuration management for the schema registry.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the schema registry configuration.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Storage       StorageConfig       `yaml:"storage"`
	Compatibility CompatibilityConfig `yaml:"compatibility"`
	Logging       LoggingConfig       `yaml:"logging"`
	Security      SecurityConfig      `yaml:"security"`
}

// ServerConfig represents HTTP server configuration.
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// StorageConfig represents storage backend configuration.
type StorageConfig struct {
	Type       string           `yaml:"type"` // memory, postgresql, mysql, cassandra
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Cassandra  CassandraConfig  `yaml:"cassandra"`
}

// PostgreSQLConfig represents PostgreSQL connection configuration.
type PostgreSQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Database        string `yaml:"database"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // seconds
}

// MySQLConfig represents MySQL connection configuration.
type MySQLConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Database        string `yaml:"database"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	TLS             string `yaml:"tls"` // true, false, skip-verify, preferred
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // seconds
}

// CassandraConfig represents Cassandra connection configuration.
type CassandraConfig struct {
	Hosts       []string `yaml:"hosts"`
	Keyspace    string   `yaml:"keyspace"`
	Consistency string   `yaml:"consistency"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
}

// CompatibilityConfig represents compatibility checking configuration.
type CompatibilityConfig struct {
	DefaultLevel string `yaml:"default_level"`
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"` // json, text
}

// SecurityConfig represents security configuration.
type SecurityConfig struct {
	TLS          TLSConfig       `yaml:"tls"`
	Auth         AuthConfig      `yaml:"auth"`
	RateLimiting RateLimitConfig `yaml:"rate_limiting"`
	Audit        AuditConfig     `yaml:"audit"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	CAFile     string `yaml:"ca_file"`     // For client cert verification
	MinVersion string `yaml:"min_version"` // TLS1.2, TLS1.3
	ClientAuth string `yaml:"client_auth"` // none, request, require, verify
	AutoReload bool   `yaml:"auto_reload"` // Reload certs without restart
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Enabled bool            `yaml:"enabled"`
	Methods []string        `yaml:"methods"` // basic, api_key, jwt, mtls
	Basic   BasicAuthConfig `yaml:"basic"`
	APIKey  APIKeyConfig    `yaml:"api_key"`
	JWT     JWTConfig       `yaml:"jwt"`
	RBAC    RBACConfig      `yaml:"rbac"`
}

// BasicAuthConfig represents basic authentication configuration.
type BasicAuthConfig struct {
	Realm    string            `yaml:"realm"`
	Users    map[string]string `yaml:"users"` // username -> bcrypt hash
	HTPasswd string            `yaml:"htpasswd_file"`
}

// APIKeyConfig represents API key authentication configuration.
type APIKeyConfig struct {
	Header      string `yaml:"header"`       // X-API-Key
	QueryParam  string `yaml:"query_param"`  // api_key
	StorageType string `yaml:"storage_type"` // memory, database
}

// JWTConfig represents JWT authentication configuration.
type JWTConfig struct {
	Issuer        string            `yaml:"issuer"`
	Audience      string            `yaml:"audience"`
	JWKSURL       string            `yaml:"jwks_url"`
	PublicKeyFile string            `yaml:"public_key_file"`
	Algorithm     string            `yaml:"algorithm"` // RS256, ES256
	ClaimsMapping map[string]string `yaml:"claims_mapping"`
}

// RBACConfig represents RBAC configuration.
type RBACConfig struct {
	Enabled     bool     `yaml:"enabled"`
	DefaultRole string   `yaml:"default_role"`
	SuperAdmins []string `yaml:"super_admins"` // Users with full access
}

// RateLimitConfig represents rate limiting configuration.
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	BurstSize         int  `yaml:"burst_size"`
	PerClient         bool `yaml:"per_client"`
	PerEndpoint       bool `yaml:"per_endpoint"`
}

// AuditConfig represents audit logging configuration.
type AuditConfig struct {
	Enabled     bool     `yaml:"enabled"`
	LogFile     string   `yaml:"log_file"`
	Events      []string `yaml:"events"` // schema_register, schema_delete, config_change
	IncludeBody bool     `yaml:"include_body"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8081,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Storage: StorageConfig{
			Type: "memory",
		},
		Compatibility: CompatibilityConfig{
			DefaultLevel: "BACKWARD",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Load loads configuration from a YAML file and environment variables.
// Environment variables override file configuration.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		// Expand environment variables in the config file
		expanded := os.ExpandEnv(string(data))

		if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("SCHEMA_REGISTRY_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_STORAGE_TYPE"); v != "" {
		c.Storage.Type = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_COMPATIBILITY_LEVEL"); v != "" {
		c.Compatibility.DefaultLevel = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}

	// PostgreSQL overrides
	if v := os.Getenv("SCHEMA_REGISTRY_PG_HOST"); v != "" {
		c.Storage.PostgreSQL.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Storage.PostgreSQL.Port = port
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_DATABASE"); v != "" {
		c.Storage.PostgreSQL.Database = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_USER"); v != "" {
		c.Storage.PostgreSQL.User = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_PASSWORD"); v != "" {
		c.Storage.PostgreSQL.Password = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_SSLMODE"); v != "" {
		c.Storage.PostgreSQL.SSLMode = v
	}

	// MySQL overrides
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_HOST"); v != "" {
		c.Storage.MySQL.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Storage.MySQL.Port = port
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_DATABASE"); v != "" {
		c.Storage.MySQL.Database = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_USER"); v != "" {
		c.Storage.MySQL.User = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_PASSWORD"); v != "" {
		c.Storage.MySQL.Password = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_TLS"); v != "" {
		c.Storage.MySQL.TLS = v
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	validStorageTypes := map[string]bool{
		"memory":     true,
		"postgresql": true,
		"mysql":      true,
		"cassandra":  true,
	}
	if !validStorageTypes[c.Storage.Type] {
		return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
	}

	validCompatibility := map[string]bool{
		"NONE":                true,
		"BACKWARD":            true,
		"BACKWARD_TRANSITIVE": true,
		"FORWARD":             true,
		"FORWARD_TRANSITIVE":  true,
		"FULL":                true,
		"FULL_TRANSITIVE":     true,
	}
	level := strings.ToUpper(c.Compatibility.DefaultLevel)
	if !validCompatibility[level] {
		return fmt.Errorf("invalid compatibility level: %s", c.Compatibility.DefaultLevel)
	}

	return nil
}

// Address returns the server address string.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
