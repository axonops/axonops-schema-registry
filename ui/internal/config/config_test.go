package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "http://localhost:8081", cfg.Registry.URL)
	assert.Equal(t, "/etc/schema-registry-ui/htpasswd", cfg.Auth.HtpasswdFile)
	assert.Equal(t, 3600, cfg.Auth.SessionTTL)
	assert.Equal(t, "sr_session", cfg.Auth.CookieName)
	assert.False(t, cfg.Auth.CookieSecure)
}

func TestLoadFromYAML(t *testing.T) {
	content := `
server:
  host: "127.0.0.1"
  port: 9090
registry:
  url: "https://sr.example.com"
  api_token: "my-token"
auth:
  htpasswd_file: "/tmp/htpasswd"
  session_secret: "test-secret-that-is-long-enough"
  session_ttl: 7200
  cookie_name: "my_cookie"
  cookie_secure: true
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "https://sr.example.com", cfg.Registry.URL)
	assert.Equal(t, "my-token", cfg.Registry.APIToken)
	assert.Equal(t, "/tmp/htpasswd", cfg.Auth.HtpasswdFile)
	assert.Equal(t, "test-secret-that-is-long-enough", cfg.Auth.SessionSecret)
	assert.Equal(t, 7200, cfg.Auth.SessionTTL)
	assert.Equal(t, "my_cookie", cfg.Auth.CookieName)
	assert.True(t, cfg.Auth.CookieSecure)
}

func TestLoadWithoutFile(t *testing.T) {
	t.Setenv("SR_UI_SESSION_SECRET", "env-secret-value")
	cfg, err := Load("")
	require.NoError(t, err)
	// Should use defaults + env overrides
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "env-secret-value", cfg.Auth.SessionSecret)
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid yaml"), 0644))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestEnvOverrides(t *testing.T) {
	content := `
server:
  host: "file-host"
  port: 3000
registry:
  url: "http://file-url:8081"
auth:
  session_secret: "file-secret"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	t.Setenv("SR_UI_HOST", "env-host")
	t.Setenv("SR_UI_PORT", "4000")
	t.Setenv("SR_UI_REGISTRY_URL", "http://env-url:9999")
	t.Setenv("SR_UI_REGISTRY_API_TOKEN", "env-token")
	t.Setenv("SR_UI_REGISTRY_API_KEY", "env-key")
	t.Setenv("SR_UI_SESSION_SECRET", "env-secret")
	t.Setenv("SR_UI_HTPASSWD_FILE", "/env/htpasswd")
	t.Setenv("SR_UI_SESSION_TTL", "1800")
	t.Setenv("SR_UI_COOKIE_NAME", "env_cookie")
	t.Setenv("SR_UI_COOKIE_SECURE", "true")

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "env-host", cfg.Server.Host)
	assert.Equal(t, 4000, cfg.Server.Port)
	assert.Equal(t, "http://env-url:9999", cfg.Registry.URL)
	assert.Equal(t, "env-token", cfg.Registry.APIToken)
	assert.Equal(t, "env-key", cfg.Registry.APIKey)
	assert.Equal(t, "env-secret", cfg.Auth.SessionSecret)
	assert.Equal(t, "/env/htpasswd", cfg.Auth.HtpasswdFile)
	assert.Equal(t, 1800, cfg.Auth.SessionTTL)
	assert.Equal(t, "env_cookie", cfg.Auth.CookieName)
	assert.True(t, cfg.Auth.CookieSecure)
}

func TestValidatePortRange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.SessionSecret = "test-secret"

	cfg.Server.Port = 0
	assert.Error(t, cfg.Validate())

	cfg.Server.Port = 70000
	assert.Error(t, cfg.Validate())

	cfg.Server.Port = 8080
	assert.NoError(t, cfg.Validate())
}

func TestValidateRegistryURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.SessionSecret = "test-secret"

	cfg.Registry.URL = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry.url is required")

	cfg.Registry.URL = "ftp://bad-scheme"
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with http:// or https://")

	cfg.Registry.URL = "https://good.example.com"
	assert.NoError(t, cfg.Validate())
}

func TestValidateSessionSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.SessionSecret = ""
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session_secret is required")
}

func TestValidateSessionTTL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.SessionSecret = "test-secret"

	cfg.Auth.SessionTTL = 10
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session_ttl must be at least 60")

	cfg.Auth.SessionTTL = 60
	assert.NoError(t, cfg.Validate())
}
