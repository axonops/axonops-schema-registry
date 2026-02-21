//go:build conformance

package conformance

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/mysql"
)

func TestMySQLBackend(t *testing.T) {
	cfg := mysql.Config{
		Host:     getEnvOrDefault("MYSQL_HOST", "localhost"),
		Port:     getEnvOrDefaultInt("MYSQL_PORT", 3306),
		Username: getEnvOrDefault("MYSQL_USER", "schemaregistry"),
		Password: getEnvOrDefault("MYSQL_PASSWORD", "schemaregistry"),
		Database: getEnvOrDefault("MYSQL_DATABASE", "schemaregistry"),
	}

	store, err := mysql.NewStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create MySQL store: %v", err)
	}
	defer store.Close()

	RunAll(t, func() storage.Storage {
		truncateMySQL(t, cfg)
		return &noCloseStore{store}
	})
}

func truncateMySQL(t *testing.T, cfg mysql.Config) {
	t.Helper()

	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to MySQL for cleanup: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		t.Fatalf("Failed to disable FK checks: %v", err)
	}

	tables := []string{"exporter_statuses", "exporters", "deks", "keks", "api_keys", "users", "schema_references", "schema_fingerprints", "schemas", "modes", "configs", "id_alloc", "ctx_id_alloc", "contexts"}
	for _, table := range tables {
		if _, err := db.Exec("TRUNCATE TABLE `" + table + "`"); err != nil {
			t.Fatalf("Failed to truncate MySQL table %s: %v", table, err)
		}
	}

	if _, err := db.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		t.Fatalf("Failed to enable FK checks: %v", err)
	}

	// Re-seed ID allocation and context but NOT global config/mode — the
	// conformance tests start from a clean state and set their own.
	if _, err := db.Exec("INSERT IGNORE INTO `id_alloc` (name, next_id) VALUES ('schema_id', 1)"); err != nil {
		t.Fatalf("Failed to insert default id_alloc: %v", err)
	}
	if _, err := db.Exec("INSERT IGNORE INTO `ctx_id_alloc` (registry_ctx, next_id) VALUES ('.', 1)"); err != nil {
		t.Fatalf("Failed to insert default ctx_id_alloc: %v", err)
	}
	if _, err := db.Exec("INSERT IGNORE INTO `contexts` (registry_ctx) VALUES ('.')"); err != nil {
		t.Fatalf("Failed to insert default context: %v", err)
	}
}
