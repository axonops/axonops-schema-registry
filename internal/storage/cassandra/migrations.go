// Package cassandra provides a Cassandra storage implementation.
package cassandra

// migrations contains the CQL statements to create tables.
var migrations = []string{
	// Table 1: schemas - stores schema records
	// Partitioned by subject for efficient queries within a subject
	`CREATE TABLE IF NOT EXISTS schemas (
		subject text,
		version int,
		id bigint,
		schema_type text,
		schema_text text,
		fingerprint text,
		deleted boolean,
		created_at timestamp,
		PRIMARY KEY (subject, version)
	) WITH CLUSTERING ORDER BY (version ASC)`,

	// Table 2: schemas_by_id - lookup by global ID
	`CREATE TABLE IF NOT EXISTS schemas_by_id (
		id bigint PRIMARY KEY,
		subject text,
		version int
	)`,

	// Table 3: schemas_by_fingerprint - lookup by fingerprint for deduplication
	`CREATE TABLE IF NOT EXISTS schemas_by_fingerprint (
		subject text,
		fingerprint text,
		id bigint,
		version int,
		PRIMARY KEY (subject, fingerprint)
	)`,

	// Table 4: schema_references - stores schema references
	`CREATE TABLE IF NOT EXISTS schema_references (
		schema_id bigint,
		name text,
		ref_subject text,
		ref_version int,
		PRIMARY KEY (schema_id, name)
	)`,

	// Table 5: references_by_target - reverse lookup for "referenced by"
	`CREATE TABLE IF NOT EXISTS references_by_target (
		ref_subject text,
		ref_version int,
		schema_subject text,
		schema_version int,
		PRIMARY KEY ((ref_subject, ref_version), schema_subject, schema_version)
	)`,

	// Table 6: configs - compatibility configuration per subject
	`CREATE TABLE IF NOT EXISTS configs (
		subject text PRIMARY KEY,
		compatibility_level text
	)`,

	// Table 7: modes - mode configuration per subject
	`CREATE TABLE IF NOT EXISTS modes (
		subject text PRIMARY KEY,
		mode text
	)`,

	// Table 8: id_counter - atomic counter for generating unique IDs
	// Uses a single partition with LWT for atomic increments
	`CREATE TABLE IF NOT EXISTS id_counter (
		name text PRIMARY KEY,
		value counter
	)`,

	// Table 9: subjects - tracks all subjects for efficient listing
	// This avoids expensive DISTINCT queries on the schemas table
	`CREATE TABLE IF NOT EXISTS subjects (
		bucket int,
		subject text,
		PRIMARY KEY (bucket, subject)
	)`,

	// Initialize global config with default value
	// Use __global__ as sentinel since Cassandra doesn't allow empty partition keys
	`INSERT INTO configs (subject, compatibility_level) VALUES ('__global__', 'BACKWARD') IF NOT EXISTS`,

	// Initialize global mode with default value
	`INSERT INTO modes (subject, mode) VALUES ('__global__', 'READWRITE') IF NOT EXISTS`,

	// Table 10: users - stores user records
	`CREATE TABLE IF NOT EXISTS users (
		id bigint PRIMARY KEY,
		username text,
		email text,
		password_hash text,
		role text,
		enabled boolean,
		created_at timestamp,
		updated_at timestamp
	)`,

	// Table 11: users_by_username - lookup by username
	`CREATE TABLE IF NOT EXISTS users_by_username (
		username text PRIMARY KEY,
		id bigint
	)`,

	// Table 12: users_by_email - lookup by email (for uniqueness check)
	`CREATE TABLE IF NOT EXISTS users_by_email (
		email text PRIMARY KEY,
		id bigint
	)`,

	// Table 13: api_keys - stores API key records
	`CREATE TABLE IF NOT EXISTS api_keys (
		id bigint PRIMARY KEY,
		user_id bigint,
		key_hash text,
		key_prefix text,
		name text,
		role text,
		enabled boolean,
		created_at timestamp,
		expires_at timestamp,
		last_used timestamp
	)`,

	// Table 14: api_keys_by_hash - lookup by key hash
	`CREATE TABLE IF NOT EXISTS api_keys_by_hash (
		key_hash text PRIMARY KEY,
		id bigint
	)`,

	// Table 15: api_keys_by_user - lookup by user ID
	`CREATE TABLE IF NOT EXISTS api_keys_by_user (
		user_id bigint,
		id bigint,
		created_at timestamp,
		PRIMARY KEY (user_id, created_at, id)
	) WITH CLUSTERING ORDER BY (created_at DESC, id ASC)`,

	// Table 16: user_id_counter - counter for generating unique user IDs
	`CREATE TABLE IF NOT EXISTS user_id_counter (
		name text PRIMARY KEY,
		value counter
	)`,

	// Table 17: api_key_id_counter - counter for generating unique API key IDs
	`CREATE TABLE IF NOT EXISTS api_key_id_counter (
		name text PRIMARY KEY,
		value counter
	)`,
}
