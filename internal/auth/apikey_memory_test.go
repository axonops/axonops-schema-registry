package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func generateAPIKeyHash(t *testing.T, key string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}
	return string(hash)
}

func TestNewMemoryAPIKeyStore_Valid(t *testing.T) {
	hash := generateAPIKeyHash(t, "my-secret-key")
	store, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "ci-pipeline", KeyHash: hash, Role: "developer"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Count() != 1 {
		t.Errorf("expected 1 key, got %d", store.Count())
	}
}

func TestNewMemoryAPIKeyStore_EmptyKeys(t *testing.T) {
	_, err := NewMemoryAPIKeyStore(nil)
	if err == nil {
		t.Error("expected error for empty keys")
	}
}

func TestNewMemoryAPIKeyStore_EmptyName(t *testing.T) {
	hash := generateAPIKeyHash(t, "key")
	_, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "", KeyHash: hash, Role: "readonly"},
	})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMemoryAPIKeyStore_EmptyHash(t *testing.T) {
	_, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "test", KeyHash: "", Role: "readonly"},
	})
	if err == nil {
		t.Error("expected error for empty hash")
	}
}

func TestNewMemoryAPIKeyStore_EmptyRole(t *testing.T) {
	hash := generateAPIKeyHash(t, "key")
	_, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "test", KeyHash: hash, Role: ""},
	})
	if err == nil {
		t.Error("expected error for empty role")
	}
}

func TestNewMemoryAPIKeyStore_UnsupportedHash(t *testing.T) {
	_, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "test", KeyHash: "{SHA}W6ph5Mm5Pz8GgiULbPgzG37mj9g=", Role: "readonly"},
	})
	if err == nil {
		t.Error("expected error for unsupported hash")
	}
}

func TestMemoryAPIKeyStore_Validate_CorrectKey(t *testing.T) {
	hash := generateAPIKeyHash(t, "my-secret-key")
	store, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "ci-pipeline", KeyHash: hash, Role: "developer"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	name, role, ok := store.Validate("my-secret-key")
	if !ok {
		t.Fatal("expected key to validate")
	}
	if name != "ci-pipeline" {
		t.Errorf("expected name 'ci-pipeline', got %s", name)
	}
	if role != "developer" {
		t.Errorf("expected role 'developer', got %s", role)
	}
}

func TestMemoryAPIKeyStore_Validate_WrongKey(t *testing.T) {
	hash := generateAPIKeyHash(t, "correct-key")
	store, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "test", KeyHash: hash, Role: "readonly"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, ok := store.Validate("wrong-key")
	if ok {
		t.Error("expected wrong key to not validate")
	}
}

func TestMemoryAPIKeyStore_Validate_MultipleKeys(t *testing.T) {
	hash1 := generateAPIKeyHash(t, "key-one")
	hash2 := generateAPIKeyHash(t, "key-two")
	store, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "first", KeyHash: hash1, Role: "admin"},
		{Name: "second", KeyHash: hash2, Role: "readonly"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	name, role, ok := store.Validate("key-two")
	if !ok {
		t.Fatal("expected second key to validate")
	}
	if name != "second" {
		t.Errorf("expected name 'second', got %s", name)
	}
	if role != "readonly" {
		t.Errorf("expected role 'readonly', got %s", role)
	}
}

func TestMemoryAPIKeyStore_Validate_EmptyKey(t *testing.T) {
	hash := generateAPIKeyHash(t, "key")
	store, err := NewMemoryAPIKeyStore([]config.ConfigAPIKey{
		{Name: "test", KeyHash: hash, Role: "readonly"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, ok := store.Validate("")
	if ok {
		t.Error("expected empty key to not validate")
	}
}
