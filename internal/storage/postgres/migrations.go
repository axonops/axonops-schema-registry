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
	`INSERT INTO configs (subject, compatibility_level) VALUES ('', 'BACKWARD') ON CONFLICT (subject) DO NOTHING`,

	// Migration 5: Mode configuration
	`CREATE TABLE IF NOT EXISTS modes (
		subject VARCHAR(255) PRIMARY KEY,
		mode VARCHAR(50) NOT NULL
	)`,

	// Migration 6: Global mode
	`INSERT INTO modes (subject, mode) VALUES ('', 'READWRITE') ON CONFLICT (subject) DO NOTHING`,

	// Migration 7: Schema versions view for efficient lookups
	`CREATE OR REPLACE VIEW schema_versions AS
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
}
