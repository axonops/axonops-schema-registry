//go:build conformance

package conformance

import (
	"context"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/cassandra"
)

func TestCassandraBackend(t *testing.T) {
	cfg := cassandra.Config{
		Hosts:          []string{getEnvOrDefault("CASSANDRA_HOSTS", "localhost")},
		Port:           getEnvOrDefaultInt("CASSANDRA_PORT", 9042),
		Keyspace:       getEnvOrDefault("CASSANDRA_KEYSPACE", "schemaregistry"),
		Consistency:    "ONE",
		ConnectTimeout: 30 * time.Second,
		Timeout:        30 * time.Second,
		Migrate:        true,
	}

	store, err := cassandra.NewStore(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to create Cassandra store: %v", err)
	}
	defer store.Close()

	RunAll(t, func() storage.Storage {
		truncateCassandra(t, cfg)
		return &noCloseStore{store}
	})
}

func truncateCassandra(t *testing.T, cfg cassandra.Config) {
	t.Helper()

	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = cfg.Port
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = gocql.One
	cluster.Timeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		t.Fatalf("Failed to connect to Cassandra for cleanup: %v", err)
	}
	defer session.Close()

	tables := []string{
		"api_keys_by_hash", "api_keys_by_user", "api_keys_by_id",
		"users_by_email", "users_by_id",
		"id_alloc", "modes", "global_config", "subject_configs",
		"references_by_target", "schema_references",
		"subject_latest", "subject_versions",
		"schemas_by_id",
	}
	for _, table := range tables {
		if err := session.Query("TRUNCATE " + table).Exec(); err != nil {
			t.Fatalf("Failed to truncate Cassandra table %s: %v", table, err)
		}
	}

	// Re-seed global defaults (matching PostgreSQL/MySQL behavior)
	if err := session.Query("INSERT INTO global_config (key, compatibility, updated_at) VALUES (?, ?, now())",
		"global", "BACKWARD").Exec(); err != nil {
		t.Fatalf("Failed to seed default global config: %v", err)
	}
	if err := session.Query("INSERT INTO modes (key, mode, updated_at) VALUES (?, ?, now())",
		"global", "READWRITE").Exec(); err != nil {
		t.Fatalf("Failed to seed default global mode: %v", err)
	}
	if err := session.Query("INSERT INTO id_alloc (name, next_id) VALUES (?, ?)",
		"schema_id", 1).Exec(); err != nil {
		t.Fatalf("Failed to seed default id_alloc: %v", err)
	}
}
