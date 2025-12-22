// Package mysql provides a MySQL storage implementation.
package mysql

// migrations contains the database schema migrations.
var migrations = []string{
	// Migration 1: Initial schema with indexes
	"CREATE TABLE IF NOT EXISTS `schemas` (" +
		"id BIGINT AUTO_INCREMENT PRIMARY KEY," +
		"subject VARCHAR(255) NOT NULL," +
		"version INT NOT NULL," +
		"schema_type VARCHAR(50) NOT NULL DEFAULT 'AVRO'," +
		"schema_text MEDIUMTEXT NOT NULL," +
		"fingerprint VARCHAR(64) NOT NULL," +
		"deleted BOOLEAN NOT NULL DEFAULT FALSE," +
		"created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"UNIQUE KEY idx_subject_version (subject, version)," +
		"UNIQUE KEY idx_subject_fingerprint (subject, fingerprint)," +
		"INDEX idx_schemas_subject (subject)," +
		"INDEX idx_schemas_deleted (deleted)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 2: Schema references with indexes
	"CREATE TABLE IF NOT EXISTS schema_references (" +
		"id BIGINT AUTO_INCREMENT PRIMARY KEY," +
		"schema_id BIGINT NOT NULL," +
		"name VARCHAR(255) NOT NULL," +
		"ref_subject VARCHAR(255) NOT NULL," +
		"ref_version INT NOT NULL," +
		"FOREIGN KEY (schema_id) REFERENCES `schemas`(id) ON DELETE CASCADE," +
		"INDEX idx_schema_references_schema_id (schema_id)," +
		"INDEX idx_schema_references_ref (ref_subject, ref_version)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 3: Configuration
	"CREATE TABLE IF NOT EXISTS configs (" +
		"subject VARCHAR(255) PRIMARY KEY," +
		"compatibility_level VARCHAR(50) NOT NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 4: Global configuration
	"INSERT IGNORE INTO configs (subject, compatibility_level) VALUES ('', 'BACKWARD')",

	// Migration 5: Mode configuration
	"CREATE TABLE IF NOT EXISTS modes (" +
		"subject VARCHAR(255) PRIMARY KEY," +
		"mode VARCHAR(50) NOT NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 6: Global mode
	"INSERT IGNORE INTO modes (subject, mode) VALUES ('', 'READWRITE')",
}
