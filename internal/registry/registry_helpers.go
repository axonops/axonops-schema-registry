package registry

import (
	"context"
	"fmt"
	"strconv"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// CheckModeForWrite checks if the current mode allows write operations.
// Returns the blocking mode name ("READONLY" or "READONLY_OVERRIDE") or empty
// string if writes are allowed.
func (r *Registry) CheckModeForWrite(ctx context.Context, registryCtx, subject string) (string, error) {
	mode, err := r.GetMode(ctx, registryCtx, subject)
	if err != nil {
		return "", fmt.Errorf("failed to check mode: %w", err)
	}
	if mode == "READONLY" || mode == "READONLY_OVERRIDE" {
		return mode, nil
	}
	return "", nil
}

// ResolveAlias resolves a subject alias. If the subject has an alias configured,
// the alias target is returned. Otherwise the original subject is returned.
// Alias resolution is single-level (no recursive chaining).
func (r *Registry) ResolveAlias(ctx context.Context, registryCtx, subject string) string {
	if subject == "" {
		return subject
	}
	config, err := r.GetSubjectConfigFull(ctx, registryCtx, subject)
	if err == nil && config.Alias != "" {
		return config.Alias
	}
	return subject
}

// ParseVersion parses a version string. "latest" and "-1" return -1 (sentinel).
// Valid versions are positive integers >= 1. Returns storage.ErrInvalidVersion on failure.
func ParseVersion(s string) (int, error) {
	if s == "latest" || s == "-1" {
		return -1, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, storage.ErrInvalidVersion
	}
	if v < 1 {
		return 0, storage.ErrInvalidVersion
	}
	return v, nil
}
