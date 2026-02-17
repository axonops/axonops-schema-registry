//go:build conformance

package conformance

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/postgres"
)

func TestPostgresBackend(t *testing.T) {
	cfg := postgres.Config{
		Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
		Port:     getEnvOrDefaultInt("POSTGRES_PORT", 5432),
		Username: getEnvOrDefault("POSTGRES_USER", "schemaregistry"),
		Password: getEnvOrDefault("POSTGRES_PASSWORD", "schemaregistry"),
		Database: getEnvOrDefault("POSTGRES_DATABASE", "schemaregistry"),
		SSLMode:  "disable",
	}

	store, err := postgres.NewStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL store: %v", err)
	}
	defer store.Close()

	RunAll(t, func() storage.Storage {
		truncatePostgres(t, cfg)
		return &noCloseStore{store}
	})
}

func truncatePostgres(t *testing.T, cfg postgres.Config) {
	t.Helper()

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL for cleanup: %v", err)
	}
	defer db.Close()

	stmts := []string{
		"TRUNCATE TABLE api_keys, users, schema_references, schema_fingerprints, schemas, modes, configs CASCADE",
		"ALTER SEQUENCE schemas_id_seq RESTART WITH 1",
		"INSERT INTO configs (subject, compatibility_level) VALUES ('', 'BACKWARD') ON CONFLICT (subject) DO NOTHING",
		"INSERT INTO modes (subject, mode) VALUES ('', 'READWRITE') ON CONFLICT (subject) DO NOTHING",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("Failed to clean PostgreSQL (%s): %v", s, err)
		}
	}
}
