package conformance

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func TestMemoryBackend(t *testing.T) {
	RunAll(t, func() storage.Storage {
		return memory.NewStore()
	})
}
