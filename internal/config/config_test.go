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
