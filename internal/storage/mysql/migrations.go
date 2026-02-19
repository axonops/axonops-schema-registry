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

	// Migration 21: schema_fingerprints table for stable global ID resolution.
	// Maps each unique schema fingerprint to an immutable schema_id, matching
	// the Cassandra backend's approach.
	"CREATE TABLE IF NOT EXISTS schema_fingerprints (" +
		"fingerprint VARCHAR(64) PRIMARY KEY," +
		"schema_id BIGINT NOT NULL," +
		"UNIQUE KEY idx_schema_fingerprints_schema_id (schema_id)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// ---------------------------------------------------------------
	// Migrations 22+: Multi-tenant context support (issue #264)
	// Adds registry_ctx column to all schema/config/mode tables.
	// Schema IDs become per-context. Default context is ".".
	// ---------------------------------------------------------------

	// Migration 22: Add registry_ctx column to schemas table.
	"ALTER TABLE `schemas` ADD COLUMN registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'",

	// Migration 23: Drop old unique constraints that don't include registry_ctx.
	// MySQL uses DROP INDEX for unique keys.
	"ALTER TABLE `schemas` DROP INDEX idx_subject_version",

	// Migration 24: Drop old fingerprint unique constraint.
	"ALTER TABLE `schemas` DROP INDEX idx_subject_fingerprint",

	// Migration 25: Create context-scoped unique indexes on schemas.
	"CREATE UNIQUE INDEX idx_schemas_ctx_subj_ver ON `schemas`(registry_ctx, subject, version)",

	// Migration 26: Create context-scoped fingerprint uniqueness.
	"CREATE UNIQUE INDEX idx_schemas_ctx_subj_fp ON `schemas`(registry_ctx, subject, fingerprint)",

	// Migration 27: Add registry_ctx to schema_fingerprints.
	"ALTER TABLE schema_fingerprints ADD COLUMN registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'",

	// Migration 28: Drop old schema_fingerprints primary key (fingerprint only).
	"ALTER TABLE schema_fingerprints DROP PRIMARY KEY, ADD PRIMARY KEY (registry_ctx, fingerprint)",

	// Migration 29: Drop old UNIQUE constraint on schema_id (IDs are now per-context).
	"ALTER TABLE schema_fingerprints DROP INDEX idx_schema_fingerprints_schema_id",

	// Migration 30: Add per-context unique constraint on schema_id.
	"CREATE UNIQUE INDEX idx_schema_fp_ctx_id ON schema_fingerprints(registry_ctx, schema_id)",

	// Migration 31: Add registry_ctx to configs.
	"ALTER TABLE configs ADD COLUMN registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'",

	// Migration 32: Drop old configs primary key (subject only) and add compound key.
	"ALTER TABLE configs DROP PRIMARY KEY, ADD PRIMARY KEY (registry_ctx, subject)",

	// Migration 33: Add registry_ctx to modes.
	"ALTER TABLE modes ADD COLUMN registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'",

	// Migration 34: Drop old modes primary key (subject only) and add compound key.
	"ALTER TABLE modes DROP PRIMARY KEY, ADD PRIMARY KEY (registry_ctx, subject)",

	// Migration 35: Per-context ID allocation table.
	// Each context has its own ID sequence starting at 1.
	"CREATE TABLE IF NOT EXISTS ctx_id_alloc (" +
		"registry_ctx VARCHAR(255) PRIMARY KEY," +
		"next_id BIGINT NOT NULL DEFAULT 1" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 36: Seed ctx_id_alloc for default context with current max ID.
	"INSERT IGNORE INTO ctx_id_alloc (registry_ctx, next_id) SELECT '.', COALESCE(MAX(schema_id), 0) + 1 FROM schema_fingerprints WHERE registry_ctx = '.'",

	// Migration 37: Contexts tracking table.
	"CREATE TABLE IF NOT EXISTS contexts (" +
		"registry_ctx VARCHAR(255) PRIMARY KEY," +
		"created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",

	// Migration 38: Seed default context.
	"INSERT IGNORE INTO contexts (registry_ctx) VALUES ('.')",

	// Migration 39: Add registry_ctx to schema_references.
	"ALTER TABLE schema_references ADD COLUMN registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'",

	// Migration 40: Index for context-scoped queries on schemas.
	"CREATE INDEX idx_schemas_registry_ctx ON `schemas`(registry_ctx)",

	// Migration 41: Relax fingerprint uniqueness per subject.
	// The same schema text (fingerprint) can now appear in multiple versions of
	// the same subject when metadata or ruleSet differ (Confluent compatibility).
	// Drop the unique index and replace with a non-unique index for lookups.
	"DROP INDEX idx_schemas_ctx_subj_fp ON `schemas`",
	"CREATE INDEX idx_schemas_ctx_subj_fp ON `schemas`(registry_ctx, subject, fingerprint)",

	// Migration 42: Make registry_ctx case-sensitive across all tables.
	// MySQL's default utf8mb4_unicode_ci collation is case-insensitive,
	// but context names must be case-sensitive for Confluent compatibility.
	"ALTER TABLE `schemas` MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
	"ALTER TABLE schema_fingerprints MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
	"ALTER TABLE configs MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
	"ALTER TABLE modes MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
	"ALTER TABLE ctx_id_alloc MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
	"ALTER TABLE contexts MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL",
	"ALTER TABLE schema_references MODIFY COLUMN registry_ctx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '.'",
}
