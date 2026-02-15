// Package cassandra provides a Cassandra storage implementation.
package cassandra

import (
	"fmt"
	"strings"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// Migrate creates/updates the Cassandra schema needed by the registry.
// This is intentionally idempotent (IF NOT EXISTS everywhere).
//
// Requires Cassandra 5.0+ for Storage Attached Index (SAI) support.
//
// Design notes:
// - schema_id is INT to match Confluent wire format (4-byte schema id)
// - Schemas are stored with SAI indexes for efficient secondary lookups:
//   - schemas_by_id: primary lookup by global ID, SAI on fingerprint for dedup
//   - subject_versions: versions within a subject, SAI on schema_id + deleted
//   - subject_latest: track latest version per subject (also used for subject listing)
//
// - Block-based ID allocation reduces LWT contention
// - TimeUUID for timestamps (Cassandra-native)
func Migrate(session *gocql.Session, keyspace string) error {
	stmts := []string{
		// Keyspace creation
		fmt.Sprintf(`CREATE KEYSPACE IF NOT EXISTS %s
			WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1}
			AND durable_writes = true`, qident(keyspace)),

		// Table 1: schemas_by_id - lookup by global schema ID
		// Primary lookup table for deserialization.
		// SAI index on fingerprint replaces the old schemas_by_fingerprint table.
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schemas_by_id (
			schema_id      int PRIMARY KEY,
			schema_type    text,
			fingerprint    text,
			schema_text    text,
			canonical_text text,
			created_at     timeuuid,
			metadata       text,
			ruleset        text
		)`, qident(keyspace)),

		// Table 2: subject_versions - versions within a subject
		// Partitioned by subject for efficient queries.
		// SAI indexes on schema_id and deleted enable cross-partition lookups.
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.subject_versions (
			subject     text,
			version     int,
			schema_id   int,
			deleted     boolean,
			created_at  timeuuid,
			metadata    text,
			ruleset     text,
			PRIMARY KEY ((subject), version)
		) WITH CLUSTERING ORDER BY (version ASC)`, qident(keyspace)),

		// Table 3: subject_latest - track latest version per subject
		// Avoids scanning partitions to find latest. Also serves as subject listing
		// (replaces the old bucketed subjects table).
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.subject_latest (
			subject          text PRIMARY KEY,
			latest_version   int,
			latest_schema_id int,
			updated_at       timeuuid
		)`, qident(keyspace)),

		// Table 4: schema_references - schema dependencies
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schema_references (
			schema_id   int,
			name        text,
			ref_subject text,
			ref_version int,
			PRIMARY KEY ((schema_id), name)
		)`, qident(keyspace)),

		// Table 5: references_by_target - reverse lookup for "referenced by"
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.references_by_target (
			ref_subject    text,
			ref_version    int,
			schema_subject text,
			schema_version int,
			PRIMARY KEY ((ref_subject, ref_version), schema_subject, schema_version)
		)`, qident(keyspace)),

		// Table 6: subject_configs - compatibility configuration per subject
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.subject_configs (
			subject              text PRIMARY KEY,
			compatibility        text,
			alias                text,
			normalize            boolean,
			validate_fields      boolean,
			default_metadata     text,
			override_metadata    text,
			default_ruleset      text,
			override_ruleset     text,
			compatibility_group  text,
			updated_at           timeuuid
		)`, qident(keyspace)),

		// Table 7: global_config - global compatibility configuration
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.global_config (
			key                  text PRIMARY KEY,
			compatibility        text,
			alias                text,
			normalize            boolean,
			validate_fields      boolean,
			default_metadata     text,
			override_metadata    text,
			default_ruleset      text,
			override_ruleset     text,
			compatibility_group  text,
			updated_at           timeuuid
		)`, qident(keyspace)),

		// Table 8: modes - registry running mode (READWRITE/READONLY/etc)
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.modes (
			key        text PRIMARY KEY,
			mode       text,
			updated_at timeuuid
		)`, qident(keyspace)),

		// Table 9: id_alloc - block-based ID allocation
		// Uses LWT for atomic block reservation
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.id_alloc (
			name    text PRIMARY KEY,
			next_id int
		)`, qident(keyspace)),

		// Table 10: users_by_id - user records
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.users_by_id (
			user_id       bigint PRIMARY KEY,
			email         text,
			name          text,
			password_hash text,
			roles         set<text>,
			enabled       boolean,
			created_at    timeuuid,
			updated_at    timeuuid
		)`, qident(keyspace)),

		// Table 11: users_by_email - lookup by email (used as users_by_username)
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.users_by_email (
			email         text PRIMARY KEY,
			user_id       bigint,
			name          text,
			password_hash text,
			roles         set<text>,
			enabled       boolean,
			created_at    timeuuid,
			updated_at    timeuuid
		)`, qident(keyspace)),

		// Table 12: api_keys_by_id - API key records
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.api_keys_by_id (
			api_key_id   bigint PRIMARY KEY,
			user_id      bigint,
			name         text,
			api_key_hash text,
			key_prefix   text,
			role         text,
			enabled      boolean,
			created_at   timeuuid,
			expires_at   timestamp,
			last_used    timestamp
		)`, qident(keyspace)),

		// Table 13: api_keys_by_user - lookup by user
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.api_keys_by_user (
			user_id      bigint,
			api_key_id   bigint,
			name         text,
			api_key_hash text,
			key_prefix   text,
			role         text,
			enabled      boolean,
			created_at   timeuuid,
			expires_at   timestamp,
			last_used    timestamp,
			PRIMARY KEY ((user_id), api_key_id)
		)`, qident(keyspace)),

		// Table 14: api_keys_by_hash - lookup by hash for authentication
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.api_keys_by_hash (
			api_key_hash text PRIMARY KEY,
			api_key_id   bigint,
			user_id      bigint,
			name         text,
			key_prefix   text,
			role         text,
			enabled      boolean,
			created_at   timeuuid,
			expires_at   timestamp,
			last_used    timestamp
		)`, qident(keyspace)),
	}

	for _, stmt := range stmts {
		if err := session.Query(stmt).Exec(); err != nil {
			return fmt.Errorf("cassandra migrate failed: %w (stmt=%s)", err, oneLine(stmt))
		}
	}

	// ALTER TABLE migrations — add new columns to tables that may already exist.
	// Each ALTER is executed individually because Cassandra returns an error if the
	// column already exists. We silently ignore "already exist" errors to stay idempotent.
	alterStmts := []string{
		// schemas_by_id: metadata and ruleset stored as JSON text
		fmt.Sprintf(`ALTER TABLE %s.schemas_by_id ADD metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.schemas_by_id ADD ruleset text`, qident(keyspace)),

		// subject_versions: metadata and ruleset stored as JSON text
		fmt.Sprintf(`ALTER TABLE %s.subject_versions ADD metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_versions ADD ruleset text`, qident(keyspace)),

		// subject_configs: alias, normalize, and metadata/ruleset config fields
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD alias text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD normalize boolean`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD default_metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD override_metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD default_ruleset text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD override_ruleset text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD compatibility_group text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.subject_configs ADD validate_fields boolean`, qident(keyspace)),

		// global_config: same config fields
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD normalize boolean`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD alias text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD default_metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD override_metadata text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD default_ruleset text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD override_ruleset text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD compatibility_group text`, qident(keyspace)),
		fmt.Sprintf(`ALTER TABLE %s.global_config ADD validate_fields boolean`, qident(keyspace)),
	}
	for _, stmt := range alterStmts {
		if err := session.Query(stmt).Exec(); err != nil {
			// Cassandra returns an error when the column already exists.
			// Treat that as a no-op so migrations stay idempotent.
			if !strings.Contains(err.Error(), "already exist") {
				return fmt.Errorf("cassandra migrate failed: %w (stmt=%s)", err, oneLine(stmt))
			}
		}
	}

	// SAI indexes (Cassandra 5.0+ required) — replace the old schemas_by_fingerprint
	// and subjects tables with efficient secondary lookups.
	saiStmts := []string{
		// Fingerprint lookup on schemas_by_id — eliminates schemas_by_fingerprint table
		fmt.Sprintf(`CREATE CUSTOM INDEX IF NOT EXISTS idx_schemas_fingerprint ON %s.schemas_by_id (fingerprint) USING 'StorageAttachedIndex'`, qident(keyspace)),
		// Schema ID lookup on subject_versions — enables O(1) GetSubjectsBySchemaID/GetVersionsBySchemaID
		fmt.Sprintf(`CREATE CUSTOM INDEX IF NOT EXISTS idx_sv_schema_id ON %s.subject_versions (schema_id) USING 'StorageAttachedIndex'`, qident(keyspace)),
		// Deleted flag lookup on subject_versions — enables efficient non-deleted filtering
		fmt.Sprintf(`CREATE CUSTOM INDEX IF NOT EXISTS idx_sv_deleted ON %s.subject_versions (deleted) USING 'StorageAttachedIndex'`, qident(keyspace)),
	}
	for _, stmt := range saiStmts {
		if err := session.Query(stmt).Exec(); err != nil {
			// Ignore "already exists" for idempotency
			if !strings.Contains(err.Error(), "already exist") {
				return fmt.Errorf("cassandra migrate failed: %w (stmt=%s)", err, oneLine(stmt))
			}
		}
	}

	// Drop legacy tables replaced by SAI indexes (breaking change).
	// These tables are no longer used by the storage layer.
	dropStmts := []string{
		fmt.Sprintf(`DROP TABLE IF EXISTS %s.schemas_by_fingerprint`, qident(keyspace)),
		fmt.Sprintf(`DROP TABLE IF EXISTS %s.subjects`, qident(keyspace)),
	}
	for _, stmt := range dropStmts {
		if err := session.Query(stmt).Exec(); err != nil {
			return fmt.Errorf("cassandra migrate failed: %w (stmt=%s)", err, oneLine(stmt))
		}
	}

	// Initialize allocator row
	if err := session.Query(
		fmt.Sprintf(`INSERT INTO %s.id_alloc (name, next_id) VALUES (?, ?) IF NOT EXISTS`, qident(keyspace)),
		"schema_id", 1,
	).Exec(); err != nil {
		return fmt.Errorf("cassandra migrate failed inserting allocator row: %w", err)
	}

	// Initialize global config default
	if err := session.Query(
		fmt.Sprintf(`INSERT INTO %s.global_config (key, compatibility, updated_at) VALUES (?, ?, now()) IF NOT EXISTS`, qident(keyspace)),
		"global", "BACKWARD",
	).Exec(); err != nil {
		return fmt.Errorf("cassandra migrate failed inserting global config: %w", err)
	}

	// Initialize global mode default
	if err := session.Query(
		fmt.Sprintf(`INSERT INTO %s.modes (key, mode, updated_at) VALUES (?, ?, now()) IF NOT EXISTS`, qident(keyspace)),
		"global", "READWRITE",
	).Exec(); err != nil {
		return fmt.Errorf("cassandra migrate failed inserting global mode: %w", err)
	}

	return nil
}

// qident quotes a Cassandra identifier if needed.
func qident(keyspace string) string {
	if keyspace == "" {
		return ""
	}
	if isSafeIdent(keyspace) {
		return keyspace
	}
	return `"` + strings.ReplaceAll(keyspace, `"`, `""`) + `"`
}

// isSafeIdent checks if a string is a safe Cassandra identifier (lowercase alphanumeric + underscore).
func isSafeIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			if i == 0 && (r >= '0' && r <= '9') {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// oneLine collapses whitespace in a string for error messages.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
