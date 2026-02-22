// Package kms provides a pluggable KMS (Key Management Service) provider
// interface for server-side DEK wrapping and unwrapping in the DEK Registry.
//
// When a KEK has shared=true, the registry uses the configured KMS provider
// to generate and wrap Data Encryption Keys on behalf of clients.
// When shared=false (default), clients are responsible for calling the KMS
// directly — the registry only stores pre-wrapped key material.
package kms

import (
	"context"
	"fmt"
	"sync"
)

// Provider defines the interface for KMS-backed key wrapping operations.
// Implementations exist for HashiCorp Vault Transit, OpenBao, AWS KMS,
// Azure Key Vault, and GCP Cloud KMS.
type Provider interface {
	// Wrap encrypts plaintext key material using the KMS key identified by kmsKeyID.
	// Returns the ciphertext (wrapped key material).
	Wrap(ctx context.Context, kmsKeyID string, plaintext []byte, props map[string]string) ([]byte, error)

	// Unwrap decrypts wrapped key material using the KMS key identified by kmsKeyID.
	// Returns the plaintext key material.
	Unwrap(ctx context.Context, kmsKeyID string, ciphertext []byte, props map[string]string) ([]byte, error)

	// GenerateDataKey generates a new data encryption key, returning both
	// the plaintext and KMS-wrapped (encrypted) forms. The algorithm parameter
	// specifies the key type (e.g., "AES256_GCM", "AES128_GCM", "AES256_SIV").
	GenerateDataKey(ctx context.Context, kmsKeyID string, algorithm string, props map[string]string) (plaintext []byte, wrapped []byte, err error)

	// Type returns the KMS provider type identifier (e.g., "hcvault", "openbao",
	// "aws-kms", "azure-kms", "gcp-kms").
	Type() string

	// Close releases any resources held by the provider.
	Close() error
}

// Registry manages available KMS providers, keyed by their type identifier.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty KMS provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a KMS provider to the registry.
// Returns an error if a provider with the same type is already registered.
func (r *Registry) Register(p Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := p.Type()
	if _, exists := r.providers[t]; exists {
		return fmt.Errorf("kms provider %q already registered", t)
	}
	r.providers[t] = p
	return nil
}

// Get returns the KMS provider for the given type.
// Returns nil if no provider is registered for that type.
func (r *Registry) Get(kmsType string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[kmsType]
}

// Has returns true if a provider is registered for the given type.
func (r *Registry) Has(kmsType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[kmsType]
	return exists
}

// Close closes all registered providers.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for _, p := range r.providers {
		if err := p.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	r.providers = make(map[string]Provider)
	return firstErr
}
