package aws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockAWSKMS creates a test HTTP server that mimics the AWS KMS API.
func mockAWSKMS(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		switch target {
		case "TrentService.Encrypt":
			// Simulate encryption by base64 "wrapping" the plaintext
			resp := map[string]interface{}{
				"CiphertextBlob": "ZW5jcnlwdGVk", // base64 of "encrypted"
				"KeyId":          body["KeyId"],
			}
			json.NewEncoder(w).Encode(resp)

		case "TrentService.Decrypt":
			resp := map[string]interface{}{
				"Plaintext": "cGxhaW50ZXh0", // base64 of "plaintext"
				"KeyId":     body["KeyId"],
			}
			json.NewEncoder(w).Encode(resp)

		case "TrentService.GenerateDataKey":
			resp := map[string]interface{}{
				"Plaintext":      "cGxhaW50ZXh0a2V5bWF0ZXJpYWwxMjM0NTY3OA==", // 32 bytes
				"CiphertextBlob": "ZW5jcnlwdGVka2V5",                         // base64 of "encryptedkey"
				"KeyId":          body["KeyId"],
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"__type":  "UnknownOperationException",
				"Message": "unsupported operation: " + target,
			})
		}
	}))
}

func TestProviderType(t *testing.T) {
	srv := mockAWSKMS(t)
	defer srv.Close()

	p, err := NewProvider(context.Background(), Config{
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        srv.URL,
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if got := p.Type(); got != ProviderType {
		t.Errorf("Type() = %q, want %q", got, ProviderType)
	}
}

func TestNewProviderFromProps(t *testing.T) {
	srv := mockAWSKMS(t)
	defer srv.Close()

	props := map[string]string{
		"aws.region":            "us-west-2",
		"aws.access.key.id":     "test-key",
		"aws.secret.access.key": "test-secret",
		"aws.endpoint":          srv.URL,
	}

	p, err := NewProviderFromProps(context.Background(), props)
	if err != nil {
		t.Fatalf("NewProviderFromProps: %v", err)
	}
	if p.Type() != ProviderType {
		t.Errorf("Type() = %q, want %q", p.Type(), ProviderType)
	}
}

func TestClose(t *testing.T) {
	p, err := NewProvider(context.Background(), Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDataKeySpecForAlgorithm(t *testing.T) {
	tests := []struct {
		algo    string
		wantLen int
	}{
		{"AES128_GCM", 16},
		{"AES256_GCM", 32},
		{"AES256_SIV", 32},
		{"UNKNOWN", 32},
	}
	for _, tt := range tests {
		if got := keySizeForAlgorithm(tt.algo); got != tt.wantLen {
			t.Errorf("keySizeForAlgorithm(%q) = %d, want %d", tt.algo, got, tt.wantLen)
		}
	}
}
