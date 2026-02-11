package conformance

import (
	"context"
	"os"
	"strconv"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// noCloseStore wraps a storage.Storage and makes Close() a no-op.
// Used by DB backend tests so individual sub-tests don't close the shared connection.
type noCloseStore struct {
	storage.Storage
}

func (s *noCloseStore) Close() error                        { return nil }
func (s *noCloseStore) IsHealthy(ctx context.Context) bool  { return s.Storage.IsHealthy(ctx) }

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}
