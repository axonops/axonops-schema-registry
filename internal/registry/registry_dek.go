package registry

import (
	"context"
	"fmt"
	"strings"

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
