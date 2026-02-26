package registry

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Valid DEK algorithms.
var validAlgorithms = map[string]bool{
	"AES128_GCM": true,
	"AES256_GCM": true,
	"AES256_SIV": true,
}

// CreateKEK creates a new Key Encryption Key.
func (r *Registry) CreateKEK(ctx context.Context, kek *storage.KEKRecord) error {
	if strings.TrimSpace(kek.Name) == "" {
		return fmt.Errorf("KEK name is required")
	}
	if strings.TrimSpace(kek.KmsType) == "" {
		return fmt.Errorf("kmsType is required")
	}
	if strings.TrimSpace(kek.KmsKeyID) == "" {
		return fmt.Errorf("kmsKeyId is required")
	}
	return r.storage.CreateKEK(ctx, kek)
}

// GetKEK retrieves a Key Encryption Key by name.
func (r *Registry) GetKEK(ctx context.Context, name string, includeDeleted bool) (*storage.KEKRecord, error) {
	return r.storage.GetKEK(ctx, name, includeDeleted)
}

// UpdateKEK updates an existing Key Encryption Key.
func (r *Registry) UpdateKEK(ctx context.Context, kek *storage.KEKRecord) error {
	if strings.TrimSpace(kek.Name) == "" {
		return fmt.Errorf("KEK name is required")
	}
	return r.storage.UpdateKEK(ctx, kek)
}

// DeleteKEK deletes a Key Encryption Key.
func (r *Registry) DeleteKEK(ctx context.Context, name string, permanent bool) error {
	return r.storage.DeleteKEK(ctx, name, permanent)
}

// UndeleteKEK restores a soft-deleted Key Encryption Key.
func (r *Registry) UndeleteKEK(ctx context.Context, name string) error {
	return r.storage.UndeleteKEK(ctx, name)
}

// ListKEKs returns all Key Encryption Keys.
func (r *Registry) ListKEKs(ctx context.Context, includeDeleted bool) ([]*storage.KEKRecord, error) {
	return r.storage.ListKEKs(ctx, includeDeleted)
}

// CreateDEK creates a new Data Encryption Key.
// If the parent KEK has shared=true and a KMS provider is configured,
// the registry generates key material and wraps it using the KMS.
// The plaintext key material is returned in dek.KeyMaterial (never stored).
func (r *Registry) CreateDEK(ctx context.Context, dek *storage.DEKRecord) error {
	if strings.TrimSpace(dek.KEKName) == "" {
		return fmt.Errorf("kekName is required")
	}
	if strings.TrimSpace(dek.Subject) == "" {
		return fmt.Errorf("subject is required")
	}
	if dek.Algorithm == "" {
		dek.Algorithm = "AES256_GCM"
	}
	if !validAlgorithms[dek.Algorithm] {
		return fmt.Errorf("invalid algorithm: %s (must be AES128_GCM, AES256_GCM, or AES256_SIV)", dek.Algorithm)
	}

	// If no encrypted key material provided and the KEK is shared with a KMS provider,
	// generate key material server-side.
	if dek.EncryptedKeyMaterial == "" && r.kmsRegistry != nil {
		kek, err := r.storage.GetKEK(ctx, dek.KEKName, false)
		if err != nil {
			return err
		}
		if kek.Shared {
			provider := r.kmsRegistry.Get(kek.KmsType)
			if provider != nil {
				plaintext, wrapped, err := provider.GenerateDataKey(ctx, kek.KmsKeyID, dek.Algorithm, kek.KmsProps)
				if err != nil {
					return fmt.Errorf("KMS generate data key: %w", err)
				}
				dek.EncryptedKeyMaterial = base64.StdEncoding.EncodeToString(wrapped)
				dek.KeyMaterial = base64.StdEncoding.EncodeToString(plaintext)
			}
		}
	}

	return r.storage.CreateDEK(ctx, dek)
}

// GetDEK retrieves a Data Encryption Key.
func (r *Registry) GetDEK(ctx context.Context, kekName, subject string, version int, algorithm string, includeDeleted bool) (*storage.DEKRecord, error) {
	return r.storage.GetDEK(ctx, kekName, subject, version, algorithm, includeDeleted)
}

// ListDEKs returns all subject names for DEKs under a KEK.
func (r *Registry) ListDEKs(ctx context.Context, kekName string, includeDeleted bool) ([]string, error) {
	return r.storage.ListDEKs(ctx, kekName, includeDeleted)
}

// ListDEKVersions returns all version numbers for a DEK subject under a KEK.
func (r *Registry) ListDEKVersions(ctx context.Context, kekName, subject string, algorithm string, includeDeleted bool) ([]int, error) {
	return r.storage.ListDEKVersions(ctx, kekName, subject, algorithm, includeDeleted)
}

// DeleteDEK deletes a Data Encryption Key.
func (r *Registry) DeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string, permanent bool) error {
	return r.storage.DeleteDEK(ctx, kekName, subject, version, algorithm, permanent)
}

// UndeleteDEK restores a soft-deleted Data Encryption Key.
func (r *Registry) UndeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string) error {
	return r.storage.UndeleteDEK(ctx, kekName, subject, version, algorithm)
}

// RewrapDEK re-encrypts a DEK's key material under the current KEK key version.
// This is used after KEK rotation to update existing DEKs.
func (r *Registry) RewrapDEK(ctx context.Context, kekName, subject string, version int, algorithm string) (*storage.DEKRecord, error) {
	if r.kmsRegistry == nil {
		return nil, fmt.Errorf("KMS not configured: rewrap requires a KMS provider")
	}

	kek, err := r.storage.GetKEK(ctx, kekName, false)
	if err != nil {
		return nil, err
	}

	provider := r.kmsRegistry.Get(kek.KmsType)
	if provider == nil {
		return nil, fmt.Errorf("no KMS provider registered for type %q", kek.KmsType)
	}

	dek, err := r.storage.GetDEK(ctx, kekName, subject, version, algorithm, false)
	if err != nil {
		return nil, err
	}

	if dek.EncryptedKeyMaterial == "" {
		return nil, fmt.Errorf("DEK has no encrypted key material to rewrap")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(dek.EncryptedKeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted key material: %w", err)
	}

	plaintext, err := provider.Unwrap(ctx, kek.KmsKeyID, ciphertext, kek.KmsProps)
	if err != nil {
		return nil, fmt.Errorf("KMS unwrap: %w", err)
	}

	newCiphertext, err := provider.Wrap(ctx, kek.KmsKeyID, plaintext, kek.KmsProps)
	if err != nil {
		return nil, fmt.Errorf("KMS rewrap: %w", err)
	}

	dek.EncryptedKeyMaterial = base64.StdEncoding.EncodeToString(newCiphertext)
	dek.Ts = time.Now().UnixMilli()

	if err := r.storage.UpdateDEK(ctx, dek); err != nil {
		return nil, err
	}

	return dek, nil
}

// TestKEK validates KMS connectivity by performing a round-trip encrypt/decrypt test.
func (r *Registry) TestKEK(ctx context.Context, kek *storage.KEKRecord) error {
	if r.kmsRegistry == nil {
		return fmt.Errorf("KMS not configured")
	}
	provider := r.kmsRegistry.Get(kek.KmsType)
	if provider == nil {
		return fmt.Errorf("no KMS provider for type %q", kek.KmsType)
	}

	testData := []byte("schema-registry-kek-test")
	wrapped, err := provider.Wrap(ctx, kek.KmsKeyID, testData, kek.KmsProps)
	if err != nil {
		return fmt.Errorf("wrap failed: %w", err)
	}
	unwrapped, err := provider.Unwrap(ctx, kek.KmsKeyID, wrapped, kek.KmsProps)
	if err != nil {
		return fmt.Errorf("unwrap failed: %w", err)
	}
	if !bytes.Equal(testData, unwrapped) {
		return fmt.Errorf("round-trip test failed: plaintext mismatch")
	}
	return nil
}
