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
	DocsEnabled  bool   `yaml:"docs_enabled"`
}

// StorageConfig represents storage backend configuration.
type StorageConfig struct {
	Type       string           `yaml:"type"`      // memory, postgresql, mysql, cassandra
	AuthType   string           `yaml:"auth_type"` // Optional: vault, or same as Type if not set
	PostgreSQL PostgreSQLConfig `yaml:"postgresql"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Cassandra  CassandraConfig  `yaml:"cassandra"`
	Vault      VaultConfig      `yaml:"vault"`
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
	Hosts            []string `yaml:"hosts"`
	Keyspace         string   `yaml:"keyspace"`
	Consistency      string   `yaml:"consistency"`       // Default consistency (used if read/write not specified)
	ReadConsistency  string   `yaml:"read_consistency"`  // Consistency for read operations (e.g., LOCAL_ONE)
	WriteConsistency string   `yaml:"write_consistency"` // Consistency for write operations (e.g., LOCAL_QUORUM)
	Username         string   `yaml:"username"`
	Password         string   `yaml:"password"`
}

// VaultConfig represents HashiCorp Vault connection configuration.
type VaultConfig struct {
	Address       string `yaml:"address"`         // Vault server address (e.g., http://localhost:8200)
	Token         string `yaml:"token"`           // Vault token (or use VAULT_TOKEN env var)
	Namespace     string `yaml:"namespace"`       // Vault namespace (enterprise feature)
	MountPath     string `yaml:"mount_path"`      // KV secrets engine mount path (default: "secret")
	BasePath      string `yaml:"base_path"`       // Base path for schema registry data (default: "schema-registry")
	TLSCertFile   string `yaml:"tls_cert_file"`   // Path to client certificate for TLS auth
	TLSKeyFile    string `yaml:"tls_key_file"`    // Path to client key for TLS auth
	TLSCAFile     string `yaml:"tls_ca_file"`     // Path to CA certificate
	TLSSkipVerify bool   `yaml:"tls_skip_verify"` // Skip TLS verification (not recommended)
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
	Enabled   bool            `yaml:"enabled"`
	Methods   []string        `yaml:"methods"` // basic, api_key, jwt, oidc, mtls
	Bootstrap BootstrapConfig `yaml:"bootstrap"`
	Basic     BasicAuthConfig `yaml:"basic"`
	LDAP      LDAPConfig      `yaml:"ldap"`
	OIDC      OIDCConfig      `yaml:"oidc"`
	APIKey    APIKeyConfig    `yaml:"api_key"`
	JWT       JWTConfig       `yaml:"jwt"`
	RBAC      RBACConfig      `yaml:"rbac"`
}

// BootstrapConfig represents initial admin user bootstrap configuration.
// This allows creating an initial admin user when the users table is empty.
// Credentials should be set via environment variables for security.
type BootstrapConfig struct {
	// Enabled controls whether bootstrap is attempted on startup.
	// If true and the users table is empty, an admin user will be created
	// using the provided credentials.
	Enabled bool `yaml:"enabled"`
	// Username for the bootstrap admin user.
	// Recommended to set via SCHEMA_REGISTRY_BOOTSTRAP_USERNAME env var.
	Username string `yaml:"username"`
	// Password for the bootstrap admin user.
	// MUST be set via SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD env var for security.
	// The password in the config file will be ignored if the env var is set.
	Password string `yaml:"password"`
	// Email for the bootstrap admin user (optional).
	Email string `yaml:"email"`
}

// BasicAuthConfig represents basic authentication configuration.
type BasicAuthConfig struct {
	Realm    string            `yaml:"realm"`
	Users    map[string]string `yaml:"users"` // username -> bcrypt hash
	HTPasswd string            `yaml:"htpasswd_file"`
}

// LDAPConfig represents LDAP authentication configuration.
type LDAPConfig struct {
	Enabled            bool              `yaml:"enabled"`
	URL                string            `yaml:"url"`                  // ldap://host:389 or ldaps://host:636
	BindDN             string            `yaml:"bind_dn"`              // Service account DN
	BindPassword       string            `yaml:"bind_password"`        // Service account password
	BaseDN             string            `yaml:"base_dn"`              // Base DN for searches
	UserSearchFilter   string            `yaml:"user_search_filter"`   // e.g., (sAMAccountName=%s)
	UserSearchBase     string            `yaml:"user_search_base"`     // e.g., OU=Users,DC=example,DC=com
	GroupSearchFilter  string            `yaml:"group_search_filter"`  // e.g., (member=%s)
	GroupSearchBase    string            `yaml:"group_search_base"`    // e.g., OU=Groups,DC=example,DC=com
	UsernameAttribute  string            `yaml:"username_attribute"`   // sAMAccountName, uid, userPrincipalName
	EmailAttribute     string            `yaml:"email_attribute"`      // mail
	GroupAttribute     string            `yaml:"group_attribute"`      // memberOf
	RoleMapping        map[string]string `yaml:"role_mapping"`         // LDAP group -> role
	DefaultRole        string            `yaml:"default_role"`         // Role if no group matches
	StartTLS           bool              `yaml:"start_tls"`            // Use STARTTLS
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"` // Skip TLS verification
	CACertFile         string            `yaml:"ca_cert_file"`         // CA cert for TLS
	ConnectionTimeout  int               `yaml:"connection_timeout"`   // Seconds, default 10
	RequestTimeout     int               `yaml:"request_timeout"`      // Seconds, default 30
}

// OIDCConfig represents OpenID Connect authentication configuration.
type OIDCConfig struct {
	Enabled           bool              `yaml:"enabled"`
	IssuerURL         string            `yaml:"issuer_url"`         // https://auth.example.com
	ClientID          string            `yaml:"client_id"`          // For token validation
	ClientSecret      string            `yaml:"client_secret"`      // #nosec G117 -- OIDC config field, not a hardcoded secret
	RedirectURL       string            `yaml:"redirect_url"`       // Callback URL
	Scopes            []string          `yaml:"scopes"`             // openid, profile, email
	UsernameClaim     string            `yaml:"username_claim"`     // sub, preferred_username, email
	RolesClaim        string            `yaml:"roles_claim"`        // roles, groups
	RoleMapping       map[string]string `yaml:"role_mapping"`       // OIDC role -> registry role
	DefaultRole       string            `yaml:"default_role"`       // Role if no mapping
	RequiredAudience  string            `yaml:"required_audience"`  // aud claim validation
	AllowedAlgorithms []string          `yaml:"allowed_algorithms"` // RS256, ES256
	SkipIssuerCheck   bool              `yaml:"skip_issuer_check"`  // For testing only
	SkipExpiryCheck   bool              `yaml:"skip_expiry_check"`  // For testing only
}

// APIKeyConfig represents API key authentication configuration.
type APIKeyConfig struct {
	Header      string `yaml:"header"`       // X-API-Key
	QueryParam  string `yaml:"query_param"`  // api_key
	StorageType string `yaml:"storage_type"` // memory, database
	// Secret is used as a pepper for HMAC-SHA256 hashing of API keys.
	// This provides defense-in-depth: even if the database is compromised,
	// the attacker cannot verify API keys without this secret.
	// CRITICAL: This must be kept secret and should be loaded from environment
	// variable or secrets manager. Use at least 32 bytes of random data.
	// If not set, falls back to plain SHA-256 (less secure but backward compatible).
	Secret string `yaml:"secret"`
	// KeyPrefix is prepended to generated API keys for identification (e.g., "sr_live_")
	KeyPrefix string `yaml:"key_prefix"`
	// CacheRefreshSeconds is how often (in seconds) the API key cache is refreshed
	// from the database. This ensures cluster consistency. Default is 60 seconds.
	CacheRefreshSeconds int `yaml:"cache_refresh_seconds"`
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
		Security: SecurityConfig{
			Auth: AuthConfig{
				APIKey: APIKeyConfig{
					CacheRefreshSeconds: 60, // Default to 60 seconds, 0 means disabled
				},
			},
		},
	}
}

// Load loads configuration from a YAML file and environment variables.
// Environment variables override file configuration.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if provided
	if path != "" {
		// #nosec G304 -- path is from command-line argument, user-controlled input is expected
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

	// Docs enabled override
	if v := os.Getenv("SCHEMA_REGISTRY_DOCS_ENABLED"); v != "" {
		c.Server.DocsEnabled = strings.ToLower(v) == "true" || v == "1"
	}

	// Auth type override
	if v := os.Getenv("SCHEMA_REGISTRY_AUTH_TYPE"); v != "" {
		c.Storage.AuthType = v
	}

	// Bootstrap admin user overrides
	if v := os.Getenv("SCHEMA_REGISTRY_BOOTSTRAP_ENABLED"); v != "" {
		c.Security.Auth.Bootstrap.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_BOOTSTRAP_USERNAME"); v != "" {
		c.Security.Auth.Bootstrap.Username = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_BOOTSTRAP_PASSWORD"); v != "" {
		c.Security.Auth.Bootstrap.Password = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_BOOTSTRAP_EMAIL"); v != "" {
		c.Security.Auth.Bootstrap.Email = v
	}

	// Vault overrides
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_ADDRESS"); v != "" {
		c.Storage.Vault.Address = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_TOKEN"); v != "" {
		c.Storage.Vault.Token = v
	}
	if v := os.Getenv("VAULT_TOKEN"); v != "" && c.Storage.Vault.Token == "" {
		// Also support standard VAULT_TOKEN env var
		c.Storage.Vault.Token = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_NAMESPACE"); v != "" {
		c.Storage.Vault.Namespace = v
	}
	if v := os.Getenv("VAULT_NAMESPACE"); v != "" && c.Storage.Vault.Namespace == "" {
		// Also support standard VAULT_NAMESPACE env var
		c.Storage.Vault.Namespace = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_MOUNT_PATH"); v != "" {
		c.Storage.Vault.MountPath = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_BASE_PATH"); v != "" {
		c.Storage.Vault.BasePath = v
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

	// Validate auth_type if set
	if c.Storage.AuthType != "" {
		validAuthTypes := map[string]bool{
			"vault":      true,
			"postgresql": true,
			"mysql":      true,
			"cassandra":  true,
			"memory":     true,
		}
		if !validAuthTypes[c.Storage.AuthType] {
			return fmt.Errorf("invalid auth type: %s", c.Storage.AuthType)
		}
	}

	// Validate Vault config if auth_type is vault
	if c.Storage.AuthType == "vault" {
		if c.Storage.Vault.Address == "" {
			return fmt.Errorf("vault address is required when auth_type is vault")
		}
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
