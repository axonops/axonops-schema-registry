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

	// Migration 7: Users table for authentication
	"CREATE TABLE IF NOT EXISTS users (" +
		"id BIGINT AUTO_INCREMENT PRIMARY KEY," +
		"username VARCHAR(255) NOT NULL," +
		"email VARCHAR(255)," +
		"password_hash VARCHAR(255) NOT NULL," +
		"role VARCHAR(50) NOT NULL DEFAULT 'readonly'," +
		"enabled BOOLEAN NOT NULL DEFAULT TRUE," +
		"created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
		"UNIQUE KEY idx_users_username (username)," +
		"UNIQUE KEY idx_users_email (email)," +
		"INDEX idx_users_role (role)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 8: API Keys table for authentication
	"CREATE TABLE IF NOT EXISTS api_keys (" +
		"id BIGINT AUTO_INCREMENT PRIMARY KEY," +
		"user_id BIGINT," +
		"key_hash VARCHAR(255) NOT NULL," +
		"key_prefix VARCHAR(16) NOT NULL," +
		"name VARCHAR(255) NOT NULL," +
		"role VARCHAR(50) NOT NULL DEFAULT 'readonly'," +
		"enabled BOOLEAN NOT NULL DEFAULT TRUE," +
		"created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"expires_at TIMESTAMP NULL," +
		"last_used TIMESTAMP NULL," +
		"UNIQUE KEY idx_api_keys_key_hash (key_hash)," +
		"INDEX idx_api_keys_user_id (user_id)," +
		"INDEX idx_api_keys_role (role)," +
		"FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 9: ID allocation table for sequential ID generation
	"CREATE TABLE IF NOT EXISTS id_alloc (" +
		"name VARCHAR(50) PRIMARY KEY," +
		"next_id BIGINT NOT NULL DEFAULT 1" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 10: Initialize ID allocation
	"INSERT IGNORE INTO id_alloc (name, next_id) VALUES ('schema_id', 1)",

	// Migration 11: Add metadata column to schemas
	"ALTER TABLE `schemas` ADD COLUMN metadata JSON",

	// Migration 12: Add ruleset column to schemas
	"ALTER TABLE `schemas` ADD COLUMN ruleset JSON",

	// Migration 13: Add alias column to configs
	"ALTER TABLE configs ADD COLUMN alias VARCHAR(255)",

	// Migration 14: Add default_metadata column to configs
	"ALTER TABLE configs ADD COLUMN default_metadata JSON",

	// Migration 15: Add override_metadata column to configs
	"ALTER TABLE configs ADD COLUMN override_metadata JSON",

	// Migration 16: Add default_ruleset column to configs
	"ALTER TABLE configs ADD COLUMN default_ruleset JSON",

	// Migration 17: Add override_ruleset column to configs
	"ALTER TABLE configs ADD COLUMN override_ruleset JSON",

	// Migration 18: Add normalize column to configs
	"ALTER TABLE configs ADD COLUMN normalize BOOLEAN",

	// Migration 19: Add compatibility_group column to configs
	"ALTER TABLE configs ADD COLUMN compatibility_group VARCHAR(255)",

	// Migration 20: Add validate_fields column to configs
	"ALTER TABLE configs ADD COLUMN validate_fields BOOLEAN",
}
