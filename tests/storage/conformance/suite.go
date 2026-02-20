// Package conformance provides a shared test suite that every storage backend must pass.
// Usage: call RunAll(t, factory) where factory creates a fresh store for each sub-test.
package conformance

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// StoreFactory creates a fresh, empty storage.Storage for each sub-test.
type StoreFactory func() storage.Storage

// RunAll runs every conformance test category against the given store factory.
func RunAll(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("Schema", func(t *testing.T) { RunSchemaTests(t, newStore) })
	t.Run("Subject", func(t *testing.T) { RunSubjectTests(t, newStore) })
	t.Run("Config", func(t *testing.T) { RunConfigTests(t, newStore) })
	t.Run("Auth", func(t *testing.T) { RunAuthTests(t, newStore) })
	t.Run("Import", func(t *testing.T) { RunImportTests(t, newStore) })
	t.Run("Error", func(t *testing.T) { RunErrorTests(t, newStore) })
	t.Run("KEK", func(t *testing.T) { RunKEKTests(t, newStore) })
	t.Run("DEK", func(t *testing.T) { RunDEKTests(t, newStore) })
	t.Run("Exporter", func(t *testing.T) { RunExporterTests(t, newStore) })
}
