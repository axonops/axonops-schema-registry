// Package postgres provides a PostgreSQL storage implementation.
package postgres

// migrations contains the database schema migrations.
var migrations = []string{
	// Migration 1: Initial schema
	`CREATE TABLE IF NOT EXISTS schemas (
		id BIGSERIAL PRIMARY KEY,
		subject VARCHAR(255) NOT NULL,
		version INTEGER NOT NULL,
		schema_type VARCHAR(50) NOT NULL DEFAULT 'AVRO',
		schema_text TEXT NOT NULL,
		fingerprint VARCHAR(64) NOT NULL,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		UNIQUE (subject, version),
		UNIQUE (subject, fingerprint)
	)`,

	`CREATE INDEX IF NOT EXISTS idx_schemas_subject ON schemas(subject)`,
	`CREATE INDEX IF NOT EXISTS idx_schemas_fingerprint ON schemas(subject, fingerprint)`,
	`CREATE INDEX IF NOT EXISTS idx_schemas_deleted ON schemas(deleted)`,

	// Migration 2: Schema references
	`CREATE TABLE IF NOT EXISTS schema_references (
		id BIGSERIAL PRIMARY KEY,
		schema_id BIGINT NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		ref_subject VARCHAR(255) NOT NULL,
		ref_version INTEGER NOT NULL
	)`,

	`CREATE INDEX IF NOT EXISTS idx_schema_references_schema_id ON schema_references(schema_id)`,
	`CREATE INDEX IF NOT EXISTS idx_schema_references_ref ON schema_references(ref_subject, ref_version)`,

	// Migration 3: Configuration
	`CREATE TABLE IF NOT EXISTS configs (
		subject VARCHAR(255) PRIMARY KEY,
		compatibility_level VARCHAR(50) NOT NULL
	)`,

	// Migration 4: Global configuration (using empty string as subject)
	// Uses WHERE NOT EXISTS instead of ON CONFLICT to stay idempotent after the
	// primary key is later changed from (subject) to (registry_ctx, subject).
	`INSERT INTO configs (subject, compatibility_level) SELECT '', 'BACKWARD' WHERE NOT EXISTS (SELECT 1 FROM configs WHERE subject = '')`,

	// Migration 5: Mode configuration
	`CREATE TABLE IF NOT EXISTS modes (
		subject VARCHAR(255) PRIMARY KEY,
		mode VARCHAR(50) NOT NULL
	)`,

	// Migration 6: Global mode
	`INSERT INTO modes (subject, mode) SELECT '', 'READWRITE' WHERE NOT EXISTS (SELECT 1 FROM modes WHERE subject = '')`,

	// Migration 7: Schema versions view for efficient lookups.
	// Uses DROP+CREATE instead of CREATE OR REPLACE because later migrations
	// add registry_ctx to the view, and PostgreSQL cannot add/drop columns
	// via CREATE OR REPLACE VIEW on re-run.
	`DROP VIEW IF EXISTS schema_versions`,
	`CREATE VIEW schema_versions AS
	SELECT subject, MAX(version) as latest_version, COUNT(*) as version_count
	FROM schemas
	WHERE deleted = FALSE
	GROUP BY subject`,

	// Migration 8: Users table for authentication
	`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		email VARCHAR(255) UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'readonly',
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	)`,

	`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
	`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,

	// Migration 9: API Keys table for authentication
	`CREATE TABLE IF NOT EXISTS api_keys (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
		key_hash VARCHAR(255) NOT NULL UNIQUE,
		key_prefix VARCHAR(16) NOT NULL,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'readonly',
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE,
		last_used TIMESTAMP WITH TIME ZONE
	)`,

	`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash)`,
	`CREATE INDEX IF NOT EXISTS idx_api_keys_role ON api_keys(role)`,

	// Migration 10: Add metadata and ruleset columns to schemas
	`ALTER TABLE schemas ADD COLUMN IF NOT EXISTS metadata JSONB`,
	`ALTER TABLE schemas ADD COLUMN IF NOT EXISTS ruleset JSONB`,

	// Migration 11: Add metadata/ruleset/alias/normalize columns to configs
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS alias VARCHAR(255)`,
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS normalize BOOLEAN`,
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS default_metadata JSONB`,
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS override_metadata JSONB`,
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS default_ruleset JSONB`,
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS override_ruleset JSONB`,

	// Migration 12: Add compatibility_group column to configs
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS compatibility_group VARCHAR(255)`,

	// Migration 13: Add validate_fields column to configs
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS validate_fields BOOLEAN`,

	// Migration 14: schema_fingerprints table for stable global ID resolution.
	// Maps each unique schema fingerprint to an immutable schema_id, matching
	// the Cassandra backend's approach. Replaces the previous MIN(id) query
	// which was mutable when the lowest-ID row was permanently deleted.
	`CREATE TABLE IF NOT EXISTS schema_fingerprints (
		fingerprint VARCHAR(64) PRIMARY KEY,
		schema_id BIGINT NOT NULL UNIQUE
	)`,

	// Migration 15: Remove ON DELETE CASCADE from schema_references FK.
	// References are now keyed by the stable schema_fingerprints.schema_id,
	// not by a specific schemas row id. Cascade deletion would destroy
	// references when any single schemas row is permanently deleted.
	`ALTER TABLE schema_references DROP CONSTRAINT IF EXISTS schema_references_schema_id_fkey`,

	// Migration 16: Backfill schema_fingerprints from existing schemas data.
	// Uses MIN(id) to preserve the same global IDs that were previously returned.
	// Uses WHERE NOT EXISTS instead of ON CONFLICT to stay idempotent after the
	// primary key is later changed from (fingerprint) to (registry_ctx, fingerprint).
	`INSERT INTO schema_fingerprints (fingerprint, schema_id)
	 SELECT s.fingerprint, MIN(s.id) FROM schemas s
	 WHERE NOT EXISTS (SELECT 1 FROM schema_fingerprints sf WHERE sf.fingerprint = s.fingerprint)
	 GROUP BY s.fingerprint`,

	// ---------------------------------------------------------------
	// Migrations 17+: Multi-tenant context support (issue #264)
	// Adds registry_ctx column to all schema/config/mode tables.
	// Schema IDs become per-context. Default context is ".".
	// ---------------------------------------------------------------

	// Migration 17: Add registry_ctx column to schemas table.
	`ALTER TABLE schemas ADD COLUMN IF NOT EXISTS registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'`,

	// Migration 18: Drop old unique constraints that don't include registry_ctx.
	// New context-scoped constraints are added in migration 19.
	`ALTER TABLE schemas DROP CONSTRAINT IF EXISTS schemas_subject_version_key`,

	// Migration 19: Drop old fingerprint unique constraint.
	`ALTER TABLE schemas DROP CONSTRAINT IF EXISTS schemas_subject_fingerprint_key`,

	// Migration 20: Create context-scoped unique indexes on schemas.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_schemas_ctx_subj_ver ON schemas(registry_ctx, subject, version)`,

	// Migration 21: Create context-scoped fingerprint uniqueness.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_schemas_ctx_subj_fp ON schemas(registry_ctx, subject, fingerprint)`,

	// Migration 22: Add registry_ctx to schema_fingerprints.
	`ALTER TABLE schema_fingerprints ADD COLUMN IF NOT EXISTS registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'`,

	// Migration 23: Drop old schema_fingerprints primary key (fingerprint only).
	`ALTER TABLE schema_fingerprints DROP CONSTRAINT IF EXISTS schema_fingerprints_pkey`,

	// Migration 24: Add new compound primary key to schema_fingerprints.
	`ALTER TABLE schema_fingerprints ADD CONSTRAINT schema_fingerprints_pkey PRIMARY KEY (registry_ctx, fingerprint)`,

	// Migration 25: Drop old UNIQUE constraint on schema_id (IDs are now per-context).
	`ALTER TABLE schema_fingerprints DROP CONSTRAINT IF EXISTS schema_fingerprints_schema_id_key`,

	// Migration 26: Add per-context unique constraint on schema_id.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_fp_ctx_id ON schema_fingerprints(registry_ctx, schema_id)`,

	// Migration 27: Add registry_ctx to configs.
	`ALTER TABLE configs ADD COLUMN IF NOT EXISTS registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'`,

	// Migration 28: Drop old configs primary key (subject only).
	`ALTER TABLE configs DROP CONSTRAINT IF EXISTS configs_pkey`,

	// Migration 29: Add new compound primary key to configs.
	`ALTER TABLE configs ADD CONSTRAINT configs_pkey PRIMARY KEY (registry_ctx, subject)`,

	// Migration 30: Add registry_ctx to modes.
	`ALTER TABLE modes ADD COLUMN IF NOT EXISTS registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'`,

	// Migration 31: Drop old modes primary key (subject only).
	`ALTER TABLE modes DROP CONSTRAINT IF EXISTS modes_pkey`,

	// Migration 32: Add new compound primary key to modes.
	`ALTER TABLE modes ADD CONSTRAINT modes_pkey PRIMARY KEY (registry_ctx, subject)`,

	// Migration 33: Per-context ID allocation table.
	// Each context has its own ID sequence starting at 1.
	`CREATE TABLE IF NOT EXISTS ctx_id_alloc (
		registry_ctx VARCHAR(255) PRIMARY KEY,
		next_id BIGINT NOT NULL DEFAULT 1
	)`,

	// Migration 34: Seed ctx_id_alloc for default context with current max ID.
	`INSERT INTO ctx_id_alloc (registry_ctx, next_id)
	 SELECT '.', COALESCE(MAX(schema_id), 0) + 1 FROM schema_fingerprints WHERE registry_ctx = '.'
	 ON CONFLICT (registry_ctx) DO NOTHING`,

	// Migration 35: Contexts tracking table.
	`CREATE TABLE IF NOT EXISTS contexts (
		registry_ctx VARCHAR(255) PRIMARY KEY,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	)`,

	// Migration 36: Seed default context.
	`INSERT INTO contexts (registry_ctx) VALUES ('.') ON CONFLICT DO NOTHING`,

	// Migration 37: Add registry_ctx to schema_references.
	`ALTER TABLE schema_references ADD COLUMN IF NOT EXISTS registry_ctx VARCHAR(255) NOT NULL DEFAULT '.'`,

	// Migration 38: Index for context-scoped queries on schemas.
	`CREATE INDEX IF NOT EXISTS idx_schemas_registry_ctx ON schemas(registry_ctx)`,

	// Migration 39: Drop the old schema_versions view before recreating.
	// PostgreSQL cannot rename columns via CREATE OR REPLACE VIEW.
	`DROP VIEW IF EXISTS schema_versions`,

	// Migration 40: Recreate schema_versions view with registry_ctx.
	`CREATE VIEW schema_versions AS
	SELECT registry_ctx, subject, MAX(version) as latest_version, COUNT(*) as version_count
	FROM schemas
	WHERE deleted = FALSE
	GROUP BY registry_ctx, subject`,

	// Migration 41: Relax fingerprint uniqueness per subject.
	// The same schema text (fingerprint) can now appear in multiple versions of
	// the same subject when metadata or ruleSet differ (Confluent compatibility).
	// Drop the unique index and replace with a non-unique index for lookups.
	`DROP INDEX IF EXISTS idx_schemas_ctx_subj_fp`,
	`CREATE INDEX IF NOT EXISTS idx_schemas_ctx_subj_fp ON schemas(registry_ctx, subject, fingerprint)`,

	// Migration 42: KEKs table (CSFLE)
	`CREATE TABLE IF NOT EXISTS keks (
		name VARCHAR(255) PRIMARY KEY,
		kms_type VARCHAR(50) NOT NULL,
		kms_key_id VARCHAR(500) NOT NULL,
		kms_props JSONB,
		doc TEXT,
		shared BOOLEAN NOT NULL DEFAULT FALSE,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		ts BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	)`,

	// Migration 43: DEKs table (CSFLE)
	`CREATE TABLE IF NOT EXISTS deks (
		kek_name VARCHAR(255) NOT NULL,
		subject VARCHAR(255) NOT NULL,
		version INTEGER NOT NULL,
		algorithm VARCHAR(50) NOT NULL DEFAULT 'AES256_GCM',
		encrypted_key_material TEXT,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		ts BIGINT NOT NULL DEFAULT 0,
		PRIMARY KEY (kek_name, subject, version, algorithm)
	)`,

	`CREATE INDEX IF NOT EXISTS idx_deks_kek_name ON deks(kek_name)`,

	// Migration 44: Exporters table
	`CREATE TABLE IF NOT EXISTS exporters (
		name VARCHAR(255) PRIMARY KEY,
		context_type VARCHAR(50),
		context VARCHAR(255),
		subjects JSONB,
		subject_rename_format VARCHAR(255),
		config JSONB,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	)`,

	// Migration 45: Exporter statuses table
	`CREATE TABLE IF NOT EXISTS exporter_statuses (
		name VARCHAR(255) PRIMARY KEY REFERENCES exporters(name) ON DELETE CASCADE,
		state VARCHAR(50) NOT NULL DEFAULT 'PAUSED',
		"offset" BIGINT NOT NULL DEFAULT 0,
		ts BIGINT NOT NULL DEFAULT 0,
		trace TEXT
	)`,
}
