// Package config loads and validates configuration for the Schema Registry UI server.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for the UI server.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Registry RegistryConfig `yaml:"registry"`
	Auth     AuthConfig     `yaml:"auth"`
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RegistryConfig specifies how to connect to the Schema Registry backend.
type RegistryConfig struct {
	URL      string `yaml:"url"`
	APIToken string `yaml:"api_token"`
	APIKey   string `yaml:"api_key"`
}

// AuthConfig controls user authentication.
type AuthConfig struct {
	HtpasswdFile  string `yaml:"htpasswd_file"`
	SessionSecret string `yaml:"session_secret"`
	SessionTTL    int    `yaml:"session_ttl"`
	CookieName    string `yaml:"cookie_name"`
	CookieSecure  bool   `yaml:"cookie_secure"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Registry: RegistryConfig{
			URL: "http://localhost:8081",
		},
		Auth: AuthConfig{
			HtpasswdFile:  "/etc/schema-registry-ui/htpasswd",
			SessionSecret: "",
			SessionTTL:    3600,
			CookieName:    "sr_session",
			CookieSecure:  false,
		},
	}
}

// Load reads a YAML config file and merges it with defaults and environment overrides.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Validate checks that required fields are set and values are in range.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Registry.URL == "" {
		return fmt.Errorf("registry.url is required")
	}
	if !strings.HasPrefix(c.Registry.URL, "http://") && !strings.HasPrefix(c.Registry.URL, "https://") {
		return fmt.Errorf("registry.url must start with http:// or https://")
	}
	if c.Auth.SessionSecret == "" {
		return fmt.Errorf("auth.session_secret is required (set via config or SR_UI_SESSION_SECRET env)")
	}
	if c.Auth.SessionTTL < 60 {
		return fmt.Errorf("auth.session_ttl must be at least 60 seconds, got %d", c.Auth.SessionTTL)
	}
	return nil
}

// applyEnvOverrides reads environment variables and overrides config fields.
// Env vars take precedence over the config file.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SR_UI_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SR_UI_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SR_UI_REGISTRY_URL"); v != "" {
		cfg.Registry.URL = v
	}
	if v := os.Getenv("SR_UI_REGISTRY_API_TOKEN"); v != "" {
		cfg.Registry.APIToken = v
	}
	if v := os.Getenv("SR_UI_REGISTRY_API_KEY"); v != "" {
		cfg.Registry.APIKey = v
	}
	if v := os.Getenv("SR_UI_SESSION_SECRET"); v != "" {
		cfg.Auth.SessionSecret = v
	}
	if v := os.Getenv("SR_UI_HTPASSWD_FILE"); v != "" {
		cfg.Auth.HtpasswdFile = v
	}
	if v := os.Getenv("SR_UI_SESSION_TTL"); v != "" {
		if ttl, err := strconv.Atoi(v); err == nil {
			cfg.Auth.SessionTTL = ttl
		}
	}
	if v := os.Getenv("SR_UI_COOKIE_NAME"); v != "" {
		cfg.Auth.CookieName = v
	}
	if v := os.Getenv("SR_UI_COOKIE_SECURE"); v != "" {
		cfg.Auth.CookieSecure = strings.EqualFold(v, "true") || v == "1"
	}
}
