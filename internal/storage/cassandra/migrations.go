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
	`INSERT INTO configs (subject, compatibility_level) VALUES ('', 'BACKWARD') IF NOT EXISTS`,

	// Initialize global mode with default value
	`INSERT INTO modes (subject, mode) VALUES ('', 'READWRITE') IF NOT EXISTS`,
}
