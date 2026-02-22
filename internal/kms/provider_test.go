package kms

import (
	"context"
	"fmt"
	"testing"
)

// mockProvider is a test KMS provider.
type mockProvider struct {
	kmsType string
}

func (m *mockProvider) Type() string { return m.kmsType }
func (m *mockProvider) Close() error { return nil }
func (m *mockProvider) Wrap(_ context.Context, _ string, plaintext []byte, _ map[string]string) ([]byte, error) {
	return append([]byte("wrapped:"), plaintext...), nil
}
func (m *mockProvider) Unwrap(_ context.Context, _ string, ciphertext []byte, _ map[string]string) ([]byte, error) {
	return ciphertext[8:], nil // strip "wrapped:" prefix
}
func (m *mockProvider) GenerateDataKey(_ context.Context, _ string, _ string, _ map[string]string) ([]byte, []byte, error) {
	return []byte("plaintext"), []byte("wrapped"), nil
}

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()

	p := &mockProvider{kmsType: "test-kms"}
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Duplicate registration should fail
	if err := reg.Register(p); err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()

	p := &mockProvider{kmsType: "test-kms"}
	reg.Register(p)

	got := reg.Get("test-kms")
	if got == nil {
		t.Fatal("Get returned nil for registered provider")
	}
	if got.Type() != "test-kms" {
		t.Errorf("Get().Type() = %q, want %q", got.Type(), "test-kms")
	}

	if reg.Get("unknown") != nil {
		t.Fatal("Get returned non-nil for unregistered provider")
	}
}

func TestRegistryHas(t *testing.T) {
	reg := NewRegistry()

	p := &mockProvider{kmsType: "test-kms"}
	reg.Register(p)

	if !reg.Has("test-kms") {
		t.Fatal("Has returned false for registered provider")
	}
	if reg.Has("unknown") {
		t.Fatal("Has returned true for unregistered provider")
	}
}

func TestRegistryClose(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&mockProvider{kmsType: "p1"})
	reg.Register(&mockProvider{kmsType: "p2"})

	if err := reg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, registry should be empty
	if reg.Has("p1") || reg.Has("p2") {
		t.Fatal("registry not empty after Close")
	}
}

func TestRegistryCloseError(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&errorProvider{kmsType: "err"})
	if err := reg.Close(); err == nil {
		t.Fatal("expected error from Close")
	}
}

type errorProvider struct {
	kmsType string
}

func (m *errorProvider) Type() string { return m.kmsType }
func (m *errorProvider) Close() error { return fmt.Errorf("close error") }
func (m *errorProvider) Wrap(_ context.Context, _ string, _ []byte, _ map[string]string) ([]byte, error) {
	return nil, nil
}
func (m *errorProvider) Unwrap(_ context.Context, _ string, _ []byte, _ map[string]string) ([]byte, error) {
	return nil, nil
}
func (m *errorProvider) GenerateDataKey(_ context.Context, _ string, _ string, _ map[string]string) ([]byte, []byte, error) {
	return nil, nil, nil
}
