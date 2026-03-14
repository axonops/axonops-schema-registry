// Package config provides configuration management for the schema registry.
package config

import (
	"fmt"
	"log/slog"
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
	MCP           MCPConfig           `yaml:"mcp"`
}

// MCPConfig represents MCP (Model Context Protocol) server configuration.
type MCPConfig struct {
	Enabled              bool     `yaml:"enabled"`
	Host                 string   `yaml:"host"`
	Port                 int      `yaml:"port"`
	AuthToken            string   `yaml:"auth_token"`            // Bearer token for v1 auth
	ReadOnly             bool     `yaml:"read_only"`             // Restrict to read-only tools
	ToolPolicy           string   `yaml:"tool_policy"`           // "allow_all" (default), "deny_list", "allow_list"
	AllowedTools         []string `yaml:"allowed_tools"`         // Tools to allow (for allow_list mode)
	DeniedTools          []string `yaml:"denied_tools"`          // Tools to deny (for deny_list mode)
	AllowedOrigins       []string `yaml:"allowed_origins"`       // Origin header allowlist (empty = allow all)
	RequireConfirmations bool     `yaml:"require_confirmations"` // Enable two-phase confirmations for destructive ops
	ConfirmationTTLSecs  int      `yaml:"confirmation_ttl"`      // Confirmation token TTL in seconds (default: 300)
	LogSchemas           bool     `yaml:"log_schemas"`           // Log full schema bodies in debug output (default: false)
	PermissionPreset     string   `yaml:"permission_preset"`     // "readonly", "developer", "operator", "admin", "full"
	PermissionScopes     []string `yaml:"permission_scopes"`     // Individual scopes when preset is empty
	ReadHeaderTimeout    int      `yaml:"read_header_timeout"`   // HTTP ReadHeaderTimeout in seconds (default: 10)
}

// ServerConfig represents HTTP server configuration.
type ServerConfig struct {
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	ReadTimeout            int    `yaml:"read_timeout"`
	WriteTimeout           int    `yaml:"write_timeout"`
	ShutdownTimeout        int    `yaml:"shutdown_timeout"` // Graceful shutdown timeout in seconds (default: 30)
	DocsEnabled            bool   `yaml:"docs_enabled"`
	ClusterID              string `yaml:"cluster_id"`
	MaxRequestBodySize     int64  `yaml:"max_request_body_size"`
	MetricsRefreshInterval int    `yaml:"metrics_refresh_interval"` // Gauge metrics refresh interval in seconds (default: 300)
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
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Database           string `yaml:"database"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	SSLMode            string `yaml:"ssl_mode"`
	MaxOpenConns       int    `yaml:"max_open_conns"`
	MaxIdleConns       int    `yaml:"max_idle_conns"`
	ConnMaxLifetime    int    `yaml:"conn_max_lifetime"`    // seconds
	ConnectTimeout     int    `yaml:"connect_timeout"`      // Initial connection ping timeout in seconds (default: 5)
	HealthCheckTimeout int    `yaml:"health_check_timeout"` // Health check timeout in seconds (default: 2)
	SchemaMaxRetries   int    `yaml:"schema_max_retries"`   // Max retries for schema creation (default: 15)
}

// MySQLConfig represents MySQL connection configuration.
type MySQLConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Database           string `yaml:"database"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	TLS                string `yaml:"tls"` // true, false, skip-verify, preferred
	MaxOpenConns       int    `yaml:"max_open_conns"`
	MaxIdleConns       int    `yaml:"max_idle_conns"`
	ConnMaxLifetime    int    `yaml:"conn_max_lifetime"`    // seconds
	ConnectTimeout     int    `yaml:"connect_timeout"`      // Initial connection ping timeout in seconds (default: 5)
	HealthCheckTimeout int    `yaml:"health_check_timeout"` // Health check timeout in seconds (default: 2)
	SchemaMaxRetries   int    `yaml:"schema_max_retries"`   // Max retries for schema creation (default: 15)
}

// CassandraConfig represents Cassandra connection configuration.
type CassandraConfig struct {
	Hosts             []string `yaml:"hosts"`
	Port              int      `yaml:"port"`
	Keyspace          string   `yaml:"keyspace"`
	LocalDC           string   `yaml:"local_dc"`
	Consistency       string   `yaml:"consistency"`        // Default consistency (used if read/write not specified)
	ReadConsistency   string   `yaml:"read_consistency"`   // Consistency for read operations (e.g., LOCAL_ONE)
	WriteConsistency  string   `yaml:"write_consistency"`  // Consistency for write operations (e.g., LOCAL_QUORUM)
	SerialConsistency string   `yaml:"serial_consistency"` // Serial consistency for LWT operations (SERIAL or LOCAL_SERIAL)
	Username          string   `yaml:"username"`
	Password          string   `yaml:"password"`
	Timeout           string   `yaml:"timeout"`         // Query timeout (e.g., "10s")
	ConnectTimeout    string   `yaml:"connect_timeout"` // Connection timeout (e.g., "10s")
	MaxRetries        int      `yaml:"max_retries"`     // Max retries for CAS operations
	IDBlockSize       int      `yaml:"id_block_size"`   // IDs reserved per LWT call
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
	Metrics      SecurityMetrics `yaml:"metrics"`
}

// SecurityMetrics represents security-related metrics configuration.
type SecurityMetrics struct {
	// PerPrincipalMetrics enables per-principal (user identity) Prometheus metrics
	// tracking request counts, errors, and endpoint usage by authenticated principal.
	// This adds a `principal` label to metrics, which MAY increase cardinality.
	// Default: true.
	PerPrincipalMetrics *bool `yaml:"per_principal_metrics"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	Enabled              bool     `yaml:"enabled"`
	CertFile             string   `yaml:"cert_file"`
	KeyFile              string   `yaml:"key_file"`
	CAFile               string   `yaml:"ca_file"`                // For client cert verification
	MinVersion           string   `yaml:"min_version"`            // TLS1.2, TLS1.3 (default: TLS1.3, minimum: TLS1.2)
	ClientAuth           string   `yaml:"client_auth"`            // none, request, require, verify
	AutoReload           bool     `yaml:"auto_reload"`            // Reload certs via SIGHUP without restart
	CipherSuites         []string `yaml:"cipher_suites"`          // Explicit cipher suite names; omit for Go's safe defaults
	AllowInsecureCiphers bool     `yaml:"allow_insecure_ciphers"` // Allow ciphers from tls.InsecureCipherSuites() (default: false)
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Enabled   bool            `yaml:"enabled"`
	Methods   []string        `yaml:"methods"` // basic, api_key, jwt, oidc
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
	Users    map[string]string `yaml:"users"`         // username -> bcrypt hash
	HTPasswd string            `yaml:"htpasswd_file"` // Path to Apache-style htpasswd file (bcrypt only)
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
	GroupSearchFilter  string            `yaml:"group_search_filter"`  // e.g., (member=%s) — searches for groups containing user DN
	GroupSearchBase    string            `yaml:"group_search_base"`    // e.g., OU=Groups,DC=example,DC=com
	UsernameAttribute  string            `yaml:"username_attribute"`   // sAMAccountName, uid, userPrincipalName
	EmailAttribute     string            `yaml:"email_attribute"`      // mail
	GroupAttribute     string            `yaml:"group_attribute"`      // memberOf
	RoleMapping        map[string]string `yaml:"role_mapping"`         // LDAP group -> role
	DefaultRole        string            `yaml:"default_role"`         // Role if no group matches
	StartTLS           bool              `yaml:"start_tls"`            // Use STARTTLS
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify"` // Skip TLS verification
	CACertFile         string            `yaml:"ca_cert_file"`         // CA cert for TLS
	ClientCertFile     string            `yaml:"client_cert_file"`     // Client cert for mTLS
	ClientKeyFile      string            `yaml:"client_key_file"`      // Client key for mTLS
	ConnectionTimeout  int               `yaml:"connection_timeout"`   // Seconds, default 10
	RequestTimeout     int               `yaml:"request_timeout"`      // Seconds, default 30
	AllowFallback      *bool             `yaml:"allow_fallback"`       // Allow fallback to DB/htpasswd when LDAP fails (default: true)
}

// OIDCConfig represents OpenID Connect authentication configuration.
type OIDCConfig struct {
	Enabled           bool              `yaml:"enabled"`
	IssuerURL         string            `yaml:"issuer_url"`         // https://auth.example.com
	ClientID          string            `yaml:"client_id"`          // For token validation
	ClientSecret      string            `yaml:"client_secret"`      // #nosec G117 -- OIDC config field, not a hardcoded secret
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
	StorageType string `yaml:"storage_type"` // "database" (default) or "memory" (config-defined keys)
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
	// Keys defines API keys in config (used when storage_type is "memory").
	Keys []ConfigAPIKey `yaml:"keys"`
}

// ConfigAPIKey represents an API key defined in the config file.
type ConfigAPIKey struct {
	Name    string `yaml:"name"`     // Identifier for the key
	KeyHash string `yaml:"key_hash"` // bcrypt hash of the API key
	Role    string `yaml:"role"`     // Role assigned to this key
}

// JWTConfig represents JWT authentication configuration.
type JWTConfig struct {
	Issuer        string            `yaml:"issuer"`
	Audience      string            `yaml:"audience"`
	JWKSURL       string            `yaml:"jwks_url"`
	PublicKeyFile string            `yaml:"public_key_file"`
	Algorithm     string            `yaml:"algorithm"` // RS256, ES256
	ClaimsMapping map[string]string `yaml:"claims_mapping"`
	DefaultRole   string            `yaml:"default_role"`   // Fallback role when no claim matches (default: "readonly")
	JWKSCacheTTL  int               `yaml:"jwks_cache_ttl"` // JWKS cache TTL in seconds (default: 300)
	HTTPTimeout   int               `yaml:"http_timeout"`   // JWKS HTTP client timeout in seconds (default: 10)
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
	Enabled     bool               `yaml:"enabled"`
	LogFile     string             `yaml:"log_file,omitempty"` // Deprecated: use Outputs.File instead
	Events      []string           `yaml:"events"`             // schema_register, schema_delete, config_update
	IncludeBody bool               `yaml:"include_body"`
	BufferSize  int                `yaml:"buffer_size"` // Async channel buffer size (default: 10000, 0 = sync)
	Outputs     AuditOutputsConfig `yaml:"outputs"`
}

// AuditOutputsConfig represents the multi-output audit configuration.
type AuditOutputsConfig struct {
	Stdout  AuditStdoutConfig  `yaml:"stdout"`
	File    AuditFileConfig    `yaml:"file"`
	Syslog  AuditSyslogConfig  `yaml:"syslog"`
	Webhook AuditWebhookConfig `yaml:"webhook"`
}

// AuditStdoutConfig configures stdout audit output.
type AuditStdoutConfig struct {
	Enabled    bool   `yaml:"enabled"`
	FormatType string `yaml:"format"` // "json" (default) or "cef"
}

// AuditFileConfig configures file audit output with rotation.
type AuditFileConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Path        string `yaml:"path"`
	FormatType  string `yaml:"format"`       // "json" (default) or "cef"
	MaxSizeMB   int    `yaml:"max_size_mb"`  // Max size in MB before rotation (default: 100)
	MaxBackups  int    `yaml:"max_backups"`  // Max number of old files to retain (default: 5)
	MaxAgeDays  int    `yaml:"max_age_days"` // Max age in days before deletion (default: 30)
	Compress    *bool  `yaml:"compress"`     // Compress rotated files (default: true)
	Permissions string `yaml:"permissions"`  // File permissions (default: "0600")
}

// AuditSyslogConfig configures syslog audit output (RFC 5424).
type AuditSyslogConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Network    string `yaml:"network"`  // "tcp", "udp", "tcp+tls" (default: "tcp")
	Address    string `yaml:"address"`  // host:port
	AppName    string `yaml:"app_name"` // syslog APP-NAME (default: "schema-registry")
	Facility   string `yaml:"facility"` // syslog facility (default: "local0")
	FormatType string `yaml:"format"`   // "json" (default) or "cef"
	TLSCert    string `yaml:"tls_cert"` // Client certificate for TLS
	TLSKey     string `yaml:"tls_key"`  // Client key for TLS
	TLSCA      string `yaml:"tls_ca"`   // CA certificate for TLS
}

// AuditWebhookConfig configures webhook audit output.
type AuditWebhookConfig struct {
	Enabled       bool              `yaml:"enabled"`
	URL           string            `yaml:"url"`            // Webhook endpoint URL
	FormatType    string            `yaml:"format"`         // "json" (default) or "cef"
	Headers       map[string]string `yaml:"headers"`        // Custom HTTP headers
	BatchSize     int               `yaml:"batch_size"`     // Events per batch (default: 100)
	FlushInterval string            `yaml:"flush_interval"` // Flush interval (default: "5s")
	Timeout       string            `yaml:"timeout"`        // HTTP timeout (default: "10s")
	MaxRetries    int               `yaml:"max_retries"`    // Retry count for 5xx (default: 3)
	BufferSize    int               `yaml:"buffer_size"`    // Channel buffer size (default: 10000)
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8081,
			ReadTimeout:     30,
			WriteTimeout:    30,
			ShutdownTimeout: 30,
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
		MCP: MCPConfig{
			Host: "127.0.0.1",
			Port: 9081,
			AllowedOrigins: []string{
				"http://localhost:*",
				"https://localhost:*",
				"vscode-webview://*",
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

// envInt parses an integer from an env var value, logging a warning on failure.
func envInt(envVar, value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil {
		slog.Warn("ignoring invalid env var value",
			slog.String("var", envVar),
			slog.String("value", value),
			slog.String("error", err.Error()),
		)
		return 0, false
	}
	return n, true
}

// applyEnvOverrides applies environment variable overrides.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("SCHEMA_REGISTRY_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PORT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_PORT", v); ok {
			c.Server.Port = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_SHUTDOWN_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_SHUTDOWN_TIMEOUT", v); ok {
			c.Server.ShutdownTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_METRICS_REFRESH_INTERVAL"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_METRICS_REFRESH_INTERVAL", v); ok {
			c.Server.MetricsRefreshInterval = n
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
		if n, ok := envInt("SCHEMA_REGISTRY_PG_PORT", v); ok {
			c.Storage.PostgreSQL.Port = n
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
	if v := os.Getenv("SCHEMA_REGISTRY_PG_CONNECT_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_PG_CONNECT_TIMEOUT", v); ok {
			c.Storage.PostgreSQL.ConnectTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_HEALTH_CHECK_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_PG_HEALTH_CHECK_TIMEOUT", v); ok {
			c.Storage.PostgreSQL.HealthCheckTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_PG_SCHEMA_MAX_RETRIES"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_PG_SCHEMA_MAX_RETRIES", v); ok {
			c.Storage.PostgreSQL.SchemaMaxRetries = n
		}
	}

	// MySQL overrides
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_HOST"); v != "" {
		c.Storage.MySQL.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_PORT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MYSQL_PORT", v); ok {
			c.Storage.MySQL.Port = n
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
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_CONNECT_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MYSQL_CONNECT_TIMEOUT", v); ok {
			c.Storage.MySQL.ConnectTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_HEALTH_CHECK_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MYSQL_HEALTH_CHECK_TIMEOUT", v); ok {
			c.Storage.MySQL.HealthCheckTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MYSQL_SCHEMA_MAX_RETRIES"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MYSQL_SCHEMA_MAX_RETRIES", v); ok {
			c.Storage.MySQL.SchemaMaxRetries = n
		}
	}

	// Cassandra overrides
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_HOSTS"); v != "" {
		hosts := strings.Split(v, ",")
		for i := range hosts {
			hosts[i] = strings.TrimSpace(hosts[i])
		}
		c.Storage.Cassandra.Hosts = hosts
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_PORT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_CASSANDRA_PORT", v); ok {
			c.Storage.Cassandra.Port = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_KEYSPACE"); v != "" {
		c.Storage.Cassandra.Keyspace = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_LOCAL_DC"); v != "" {
		c.Storage.Cassandra.LocalDC = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_CONSISTENCY"); v != "" {
		c.Storage.Cassandra.Consistency = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_READ_CONSISTENCY"); v != "" {
		c.Storage.Cassandra.ReadConsistency = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_WRITE_CONSISTENCY"); v != "" {
		c.Storage.Cassandra.WriteConsistency = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_SERIAL_CONSISTENCY"); v != "" {
		c.Storage.Cassandra.SerialConsistency = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_USERNAME"); v != "" {
		c.Storage.Cassandra.Username = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_PASSWORD"); v != "" {
		c.Storage.Cassandra.Password = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_TIMEOUT"); v != "" {
		c.Storage.Cassandra.Timeout = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_CONNECT_TIMEOUT"); v != "" {
		c.Storage.Cassandra.ConnectTimeout = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_MAX_RETRIES"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_CASSANDRA_MAX_RETRIES", v); ok {
			c.Storage.Cassandra.MaxRetries = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_CASSANDRA_ID_BLOCK_SIZE"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_CASSANDRA_ID_BLOCK_SIZE", v); ok {
			c.Storage.Cassandra.IDBlockSize = n
		}
	}

	// MCP overrides
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_ENABLED"); v != "" {
		c.MCP.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_HOST"); v != "" {
		c.MCP.Host = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_PORT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MCP_PORT", v); ok {
			c.MCP.Port = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_AUTH_TOKEN"); v != "" {
		c.MCP.AuthToken = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_READ_ONLY"); v != "" {
		c.MCP.ReadOnly = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_ALLOWED_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		c.MCP.AllowedOrigins = origins
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_REQUIRE_CONFIRMATIONS"); v != "" {
		c.MCP.RequireConfirmations = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MCP_CONFIRMATION_TTL", v); ok {
			c.MCP.ConfirmationTTLSecs = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_LOG_SCHEMAS"); v != "" {
		c.MCP.LogSchemas = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_PERMISSION_PRESET"); v != "" {
		c.MCP.PermissionPreset = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_PERMISSION_SCOPES"); v != "" {
		scopes := strings.Split(v, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
		c.MCP.PermissionScopes = scopes
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_TOOL_POLICY"); v != "" {
		c.MCP.ToolPolicy = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_ALLOWED_TOOLS"); v != "" {
		tools := strings.Split(v, ",")
		for i := range tools {
			tools[i] = strings.TrimSpace(tools[i])
		}
		c.MCP.AllowedTools = tools
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_DENIED_TOOLS"); v != "" {
		tools := strings.Split(v, ",")
		for i := range tools {
			tools[i] = strings.TrimSpace(tools[i])
		}
		c.MCP.DeniedTools = tools
	}
	if v := os.Getenv("SCHEMA_REGISTRY_MCP_READ_HEADER_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_MCP_READ_HEADER_TIMEOUT", v); ok {
			c.MCP.ReadHeaderTimeout = n
		}
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
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_TLS_CERT_FILE"); v != "" {
		c.Storage.Vault.TLSCertFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_TLS_KEY_FILE"); v != "" {
		c.Storage.Vault.TLSKeyFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_TLS_CA_FILE"); v != "" {
		c.Storage.Vault.TLSCAFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_VAULT_TLS_SKIP_VERIFY"); v != "" {
		c.Storage.Vault.TLSSkipVerify = strings.ToLower(v) == "true" || v == "1"
	}

	// JWT overrides
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_ISSUER"); v != "" {
		c.Security.Auth.JWT.Issuer = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_AUDIENCE"); v != "" {
		c.Security.Auth.JWT.Audience = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_JWKS_URL"); v != "" {
		c.Security.Auth.JWT.JWKSURL = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_PUBLIC_KEY_FILE"); v != "" {
		c.Security.Auth.JWT.PublicKeyFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_ALGORITHM"); v != "" {
		c.Security.Auth.JWT.Algorithm = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_DEFAULT_ROLE"); v != "" {
		c.Security.Auth.JWT.DefaultRole = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_JWKS_CACHE_TTL"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_JWT_JWKS_CACHE_TTL", v); ok {
			c.Security.Auth.JWT.JWKSCacheTTL = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_JWT_HTTP_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_JWT_HTTP_TIMEOUT", v); ok {
			c.Security.Auth.JWT.HTTPTimeout = n
		}
	}

	// Auth overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUTH_ENABLED"); v != "" {
		c.Security.Auth.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUTH_METHODS"); v != "" {
		methods := strings.Split(v, ",")
		for i := range methods {
			methods[i] = strings.TrimSpace(methods[i])
		}
		c.Security.Auth.Methods = methods
	}

	// LDAP overrides
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_ENABLED"); v != "" {
		c.Security.Auth.LDAP.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_URL"); v != "" {
		c.Security.Auth.LDAP.URL = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_BIND_DN"); v != "" {
		c.Security.Auth.LDAP.BindDN = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_BIND_PASSWORD"); v != "" {
		c.Security.Auth.LDAP.BindPassword = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_BASE_DN"); v != "" {
		c.Security.Auth.LDAP.BaseDN = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_USER_SEARCH_FILTER"); v != "" {
		c.Security.Auth.LDAP.UserSearchFilter = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_USER_SEARCH_BASE"); v != "" {
		c.Security.Auth.LDAP.UserSearchBase = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_GROUP_SEARCH_FILTER"); v != "" {
		c.Security.Auth.LDAP.GroupSearchFilter = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_GROUP_SEARCH_BASE"); v != "" {
		c.Security.Auth.LDAP.GroupSearchBase = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_USERNAME_ATTRIBUTE"); v != "" {
		c.Security.Auth.LDAP.UsernameAttribute = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_EMAIL_ATTRIBUTE"); v != "" {
		c.Security.Auth.LDAP.EmailAttribute = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_GROUP_ATTRIBUTE"); v != "" {
		c.Security.Auth.LDAP.GroupAttribute = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_DEFAULT_ROLE"); v != "" {
		c.Security.Auth.LDAP.DefaultRole = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_START_TLS"); v != "" {
		c.Security.Auth.LDAP.StartTLS = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_INSECURE_SKIP_VERIFY"); v != "" {
		c.Security.Auth.LDAP.InsecureSkipVerify = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_CA_CERT_FILE"); v != "" {
		c.Security.Auth.LDAP.CACertFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_CLIENT_CERT_FILE"); v != "" {
		c.Security.Auth.LDAP.ClientCertFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_CLIENT_KEY_FILE"); v != "" {
		c.Security.Auth.LDAP.ClientKeyFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_CONNECTION_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_LDAP_CONNECTION_TIMEOUT", v); ok {
			c.Security.Auth.LDAP.ConnectionTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_REQUEST_TIMEOUT"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_LDAP_REQUEST_TIMEOUT", v); ok {
			c.Security.Auth.LDAP.RequestTimeout = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_LDAP_ALLOW_FALLBACK"); v != "" {
		b := strings.ToLower(v) == "true" || v == "1"
		c.Security.Auth.LDAP.AllowFallback = &b
	}

	// OIDC overrides
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_ENABLED"); v != "" {
		c.Security.Auth.OIDC.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_ISSUER_URL"); v != "" {
		c.Security.Auth.OIDC.IssuerURL = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_CLIENT_ID"); v != "" {
		c.Security.Auth.OIDC.ClientID = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_CLIENT_SECRET"); v != "" {
		c.Security.Auth.OIDC.ClientSecret = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_USERNAME_CLAIM"); v != "" {
		c.Security.Auth.OIDC.UsernameClaim = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_ROLES_CLAIM"); v != "" {
		c.Security.Auth.OIDC.RolesClaim = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_DEFAULT_ROLE"); v != "" {
		c.Security.Auth.OIDC.DefaultRole = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_REQUIRED_AUDIENCE"); v != "" {
		c.Security.Auth.OIDC.RequiredAudience = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_SKIP_ISSUER_CHECK"); v != "" {
		c.Security.Auth.OIDC.SkipIssuerCheck = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_OIDC_SKIP_EXPIRY_CHECK"); v != "" {
		c.Security.Auth.OIDC.SkipExpiryCheck = strings.ToLower(v) == "true" || v == "1"
	}

	// TLS overrides
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_ENABLED"); v != "" {
		c.Security.TLS.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_CERT_FILE"); v != "" {
		c.Security.TLS.CertFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_KEY_FILE"); v != "" {
		c.Security.TLS.KeyFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_CA_FILE"); v != "" {
		c.Security.TLS.CAFile = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_MIN_VERSION"); v != "" {
		c.Security.TLS.MinVersion = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_CLIENT_AUTH"); v != "" {
		c.Security.TLS.ClientAuth = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_CIPHER_SUITES"); v != "" {
		suites := strings.Split(v, ",")
		for i := range suites {
			suites[i] = strings.TrimSpace(suites[i])
		}
		c.Security.TLS.CipherSuites = suites
	}
	if v := os.Getenv("SCHEMA_REGISTRY_TLS_ALLOW_INSECURE_CIPHERS"); v != "" {
		c.Security.TLS.AllowInsecureCiphers = strings.ToLower(v) == "true" || v == "1"
	}

	// Audit overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_ENABLED"); v != "" {
		c.Security.Audit.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_INCLUDE_BODY"); v != "" {
		c.Security.Audit.IncludeBody = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_BUFFER_SIZE"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_BUFFER_SIZE", v); ok {
			c.Security.Audit.BufferSize = n
		}
	}

	// Audit stdout output overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_STDOUT_ENABLED"); v != "" {
		c.Security.Audit.Outputs.Stdout.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_STDOUT_FORMAT"); v != "" {
		c.Security.Audit.Outputs.Stdout.FormatType = v
	}

	// Audit file output overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_ENABLED"); v != "" {
		c.Security.Audit.Outputs.File.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_PATH"); v != "" {
		c.Security.Audit.Outputs.File.Path = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_FORMAT"); v != "" {
		c.Security.Audit.Outputs.File.FormatType = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_MAX_SIZE_MB"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_FILE_MAX_SIZE_MB", v); ok {
			c.Security.Audit.Outputs.File.MaxSizeMB = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_MAX_BACKUPS"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_FILE_MAX_BACKUPS", v); ok {
			c.Security.Audit.Outputs.File.MaxBackups = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_MAX_AGE_DAYS"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_FILE_MAX_AGE_DAYS", v); ok {
			c.Security.Audit.Outputs.File.MaxAgeDays = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_COMPRESS"); v != "" {
		b := strings.ToLower(v) == "true" || v == "1"
		c.Security.Audit.Outputs.File.Compress = &b
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_FILE_PERMISSIONS"); v != "" {
		c.Security.Audit.Outputs.File.Permissions = v
	}

	// Audit syslog output overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_ENABLED"); v != "" {
		c.Security.Audit.Outputs.Syslog.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_NETWORK"); v != "" {
		c.Security.Audit.Outputs.Syslog.Network = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_ADDRESS"); v != "" {
		c.Security.Audit.Outputs.Syslog.Address = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_APP_NAME"); v != "" {
		c.Security.Audit.Outputs.Syslog.AppName = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_FACILITY"); v != "" {
		c.Security.Audit.Outputs.Syslog.Facility = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_FORMAT"); v != "" {
		c.Security.Audit.Outputs.Syslog.FormatType = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_CERT"); v != "" {
		c.Security.Audit.Outputs.Syslog.TLSCert = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_KEY"); v != "" {
		c.Security.Audit.Outputs.Syslog.TLSKey = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_SYSLOG_TLS_CA"); v != "" {
		c.Security.Audit.Outputs.Syslog.TLSCA = v
	}

	// Audit webhook output overrides
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_ENABLED"); v != "" {
		c.Security.Audit.Outputs.Webhook.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_URL"); v != "" {
		c.Security.Audit.Outputs.Webhook.URL = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_FORMAT"); v != "" {
		c.Security.Audit.Outputs.Webhook.FormatType = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_BATCH_SIZE"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_WEBHOOK_BATCH_SIZE", v); ok {
			c.Security.Audit.Outputs.Webhook.BatchSize = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_FLUSH_INTERVAL"); v != "" {
		c.Security.Audit.Outputs.Webhook.FlushInterval = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_TIMEOUT"); v != "" {
		c.Security.Audit.Outputs.Webhook.Timeout = v
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_MAX_RETRIES"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_WEBHOOK_MAX_RETRIES", v); ok {
			c.Security.Audit.Outputs.Webhook.MaxRetries = n
		}
	}
	if v := os.Getenv("SCHEMA_REGISTRY_AUDIT_WEBHOOK_BUFFER_SIZE"); v != "" {
		if n, ok := envInt("SCHEMA_REGISTRY_AUDIT_WEBHOOK_BUFFER_SIZE", v); ok {
			c.Security.Audit.Outputs.Webhook.BufferSize = n
		}
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

	// Validate audit config
	if err := c.validateAuditConfig(); err != nil {
		return err
	}

	return nil
}

// validateAuditConfig validates the audit configuration.
func (c *Config) validateAuditConfig() error {
	audit := &c.Security.Audit

	// Validate format types
	validFormats := map[string]bool{"": true, "json": true, "cef": true}
	if !validFormats[audit.Outputs.Stdout.FormatType] {
		return fmt.Errorf("invalid audit stdout format: %q (must be \"json\" or \"cef\")", audit.Outputs.Stdout.FormatType)
	}
	if !validFormats[audit.Outputs.File.FormatType] {
		return fmt.Errorf("invalid audit file format: %q (must be \"json\" or \"cef\")", audit.Outputs.File.FormatType)
	}
	if !validFormats[audit.Outputs.Syslog.FormatType] {
		return fmt.Errorf("invalid audit syslog format: %q (must be \"json\" or \"cef\")", audit.Outputs.Syslog.FormatType)
	}
	if !validFormats[audit.Outputs.Webhook.FormatType] {
		return fmt.Errorf("invalid audit webhook format: %q (must be \"json\" or \"cef\")", audit.Outputs.Webhook.FormatType)
	}

	// File output requires a path when enabled
	if audit.Outputs.File.Enabled && audit.Outputs.File.Path == "" {
		return fmt.Errorf("audit file output enabled but no path specified")
	}

	// Syslog output requires an address when enabled
	if audit.Outputs.Syslog.Enabled && audit.Outputs.Syslog.Address == "" {
		return fmt.Errorf("audit syslog output enabled but no address specified")
	}

	// Webhook output requires a URL when enabled
	if audit.Outputs.Webhook.Enabled && audit.Outputs.Webhook.URL == "" {
		return fmt.Errorf("audit webhook output enabled but no URL specified")
	}

	return nil
}

// Address returns the server address string.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// MCPAddress returns the MCP server address string.
func (c *Config) MCPAddress() string {
	return fmt.Sprintf("%s:%d", c.MCP.Host, c.MCP.Port)
}
