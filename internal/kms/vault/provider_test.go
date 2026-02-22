package vault

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockVaultTransit creates a test HTTP server that mimics the Vault Transit API.
// It handles any transit mount path (e.g., /v1/transit/encrypt/ or /v1/my-transit/encrypt/).
func mockVaultTransit(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case strings.Contains(path, "/encrypt/"):
			// Vault Transit encrypt endpoint
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
			// Vault Transit decrypt endpoint
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
	p, err := NewProvider(Config{Address: "http://localhost:8200", Token: "test"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if got := p.Type(); got != ProviderType {
		t.Errorf("Type() = %q, want %q", got, ProviderType)
	}
}

func TestWrapUnwrap(t *testing.T) {
	srv := mockVaultTransit(t)
	defer srv.Close()

	p, err := NewProvider(Config{
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

	p, err := NewProvider(Config{
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

	p, err := NewProvider(Config{
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
		{"AES256_SIV", 32},
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
		"vault.address":       srv.URL,
		"vault.token":         "test-token",
		"vault.transit.mount": "my-transit",
	}

	p, err := NewProviderFromProps(props)
	if err != nil {
		t.Fatalf("NewProviderFromProps: %v", err)
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
	p, err := NewProvider(Config{Address: "http://localhost:8200", Token: "test"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestKeySizeForAlgorithm(t *testing.T) {
	tests := []struct {
		algo string
		want int
	}{
		{"AES128_GCM", 16},
		{"AES256_GCM", 32},
		{"AES256_SIV", 32},
		{"UNKNOWN", 32},
	}
	for _, tt := range tests {
		if got := keySizeForAlgorithm(tt.algo); got != tt.want {
			t.Errorf("keySizeForAlgorithm(%q) = %d, want %d", tt.algo, got, tt.want)
		}
	}
}
