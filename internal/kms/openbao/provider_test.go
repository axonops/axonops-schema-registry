package openbao

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vaultprovider "github.com/axonops/axonops-schema-registry/internal/kms/vault"
)

// mockVaultTransit creates a test HTTP server that mimics the Vault/OpenBao Transit API.
// OpenBao uses the same Transit API as Vault, so the mock is identical.
func mockVaultTransit(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case strings.Contains(path, "/encrypt/"):
			// Transit encrypt endpoint
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			plaintext := body["plaintext"].(string)
			// Simulate wrapping by prefixing with "vault:v1:"
			ciphertext := "vault:v1:" + plaintext
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"ciphertext": ciphertext,
				},
			})

		case strings.Contains(path, "/decrypt/"):
			// Transit decrypt endpoint
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			ciphertext := body["ciphertext"].(string)
			// Simulate unwrapping by removing "vault:v1:" prefix
			plaintext := strings.TrimPrefix(ciphertext, "vault:v1:")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"plaintext": plaintext,
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []string{"unsupported path: " + path},
			})
		}
	}))
}

func TestProviderType(t *testing.T) {
	p, err := NewProvider(vaultprovider.Config{Address: "http://localhost:8200", Token: "test"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if got := p.Type(); got != ProviderType {
		t.Errorf("Type() = %q, want %q", got, ProviderType)
	}
	if got := p.Type(); got != "openbao" {
		t.Errorf("Type() = %q, want %q", got, "openbao")
	}
}

func TestWrapUnwrap(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	p, err := NewProvider(vaultprovider.Config{
		Address:      srv.URL,
		Token:        "test-token",
		TransitMount: "transit",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	ctx := context.Background()
	plaintext := []byte("secret-key-material-32-bytes!!!!!")

	// Wrap
	wrapped, err := p.Wrap(ctx, "my-kek", plaintext, nil)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if len(wrapped) == 0 {
		t.Fatal("Wrap returned empty ciphertext")
	}

	// Unwrap
	unwrapped, err := p.Unwrap(ctx, "my-kek", wrapped, nil)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if string(unwrapped) != string(plaintext) {
		t.Errorf("Unwrap = %q, want %q", string(unwrapped), string(plaintext))
	}
}

func TestGenerateDataKey(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	p, err := NewProvider(vaultprovider.Config{
		Address:      srv.URL,
		Token:        "test-token",
		TransitMount: "transit",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	ctx := context.Background()
	plaintext, wrapped, err := p.GenerateDataKey(ctx, "my-kek", "AES256_GCM", nil)
	if err != nil {
		t.Fatalf("GenerateDataKey: %v", err)
	}

	if len(plaintext) != 32 {
		t.Errorf("GenerateDataKey plaintext length = %d, want 32", len(plaintext))
	}
	if len(wrapped) == 0 {
		t.Fatal("GenerateDataKey returned empty wrapped key")
	}

	// Verify the wrapped key can be unwrapped back to the original plaintext
	unwrapped, err := p.Unwrap(ctx, "my-kek", wrapped, nil)
	if err != nil {
		t.Fatalf("Unwrap after GenerateDataKey: %v", err)
	}
	if string(unwrapped) != string(plaintext) {
		t.Errorf("round-trip failed: unwrapped != plaintext")
	}
}

func TestGenerateDataKeyAlgorithms(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	p, err := NewProvider(vaultprovider.Config{
		Address:      srv.URL,
		Token:        "test-token",
		TransitMount: "transit",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	tests := []struct {
		algorithm string
		keySize   int
	}{
		{"AES128_GCM", 16},
		{"AES256_GCM", 32},
		{"AES256_SIV", 64},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			plaintext, _, err := p.GenerateDataKey(ctx, "my-kek", tt.algorithm, nil)
			if err != nil {
				t.Fatalf("GenerateDataKey(%s): %v", tt.algorithm, err)
			}
			if len(plaintext) != tt.keySize {
				t.Errorf("GenerateDataKey(%s) key size = %d, want %d", tt.algorithm, len(plaintext), tt.keySize)
			}
		})
	}
}

func TestNewProviderFromProps(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	props := map[string]string{
		"openbao.address":       srv.URL,
		"openbao.token":         "test-token",
		"openbao.transit.mount": "my-transit",
	}

	p, err := NewProviderFromProps(props)
	if err != nil {
		t.Fatalf("NewProviderFromProps: %v", err)
	}

	if got := p.Type(); got != "openbao" {
		t.Errorf("Type() = %q, want %q", got, "openbao")
	}

	// Verify it works by doing a wrap
	ctx := context.Background()
	plaintext := []byte("test")
	wrapped, err := p.Wrap(ctx, "test-key", plaintext, nil)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if len(wrapped) == 0 {
		t.Fatal("Wrap returned empty")
	}
}

func TestClose(t *testing.T) {
	p, err := NewProvider(vaultprovider.Config{Address: "http://localhost:8200", Token: "test"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestEnvVarDefaults(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	// Set OpenBao-specific environment variables
	t.Setenv("BAO_ADDR", srv.URL)
	t.Setenv("BAO_TOKEN", "env-token")
	t.Setenv("BAO_NAMESPACE", "env-namespace")

	// Create provider with empty config — should pick up env vars
	p, err := NewProvider(vaultprovider.Config{})
	if err != nil {
		t.Fatalf("NewProvider with env vars: %v", err)
	}

	if got := p.Type(); got != "openbao" {
		t.Errorf("Type() = %q, want %q", got, "openbao")
	}

	// Verify the provider can communicate with the mock server (proving BAO_ADDR was used)
	ctx := context.Background()
	plaintext := []byte("env-var-test")
	wrapped, err := p.Wrap(ctx, "test-key", plaintext, nil)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if len(wrapped) == 0 {
		t.Fatal("Wrap returned empty ciphertext")
	}

	unwrapped, err := p.Unwrap(ctx, "test-key", wrapped, nil)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if string(unwrapped) != string(plaintext) {
		t.Errorf("round-trip with env vars failed: got %q, want %q", string(unwrapped), string(plaintext))
	}
}

func TestEnvVarDefaultsNotOverrideExplicit(t *testing.T) {
	// Set env vars to values that would fail (bad address)
	t.Setenv("BAO_ADDR", "http://bad-host:9999")
	t.Setenv("BAO_TOKEN", "env-token")

	srv := mockVaultTransit(t)
	defer srv.Close()

	// Explicit config should take precedence over env vars
	p, err := NewProvider(vaultprovider.Config{
		Address: srv.URL,
		Token:   "explicit-token",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	// This should succeed because the explicit address (mock server) was used, not BAO_ADDR
	ctx := context.Background()
	plaintext := []byte("explicit-config-test")
	wrapped, err := p.Wrap(ctx, "test-key", plaintext, nil)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	unwrapped, err := p.Unwrap(ctx, "test-key", wrapped, nil)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if string(unwrapped) != string(plaintext) {
		t.Errorf("round-trip failed: got %q, want %q", string(unwrapped), string(plaintext))
	}
}
