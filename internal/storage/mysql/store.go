package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Config holds MySQL connection configuration.
type Config struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	Database        string        `json:"database" yaml:"database"`
	Username        string        `json:"username" yaml:"username"`
	Password        string        `json:"password" yaml:"password"`
	TLS             string        `json:"tls" yaml:"tls"` // true, false, skip-verify, preferred, or custom config name
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            3306,
		Database:        "schema_registry",
		Username:        "root",
		Password:        "",
		TLS:             "false",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// DSN returns the connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&tls=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.TLS,
	)
}

// Store implements the storage.Storage interface using MySQL.
type Store struct {
	db     *sql.DB
	config Config

	// Prepared statements for better performance
	stmts *preparedStatements
}

// preparedStatements holds all prepared SQL statements.
type preparedStatements struct {
	// Schema statements — all scoped by registry_ctx
	getSchemaByID          *sql.Stmt
	getSchemaBySubjectVer  *sql.Stmt
	getSchemaByFingerprint *sql.Stmt
	getLatestSchema        *sql.Stmt
	softDeleteSchema       *sql.Stmt
	hardDeleteSchema       *sql.Stmt
	countSchemasBySubject  *sql.Stmt
	getSubjectsBySchemaID  *sql.Stmt
	getVersionsBySchemaID  *sql.Stmt
	getReferencedBy        *sql.Stmt

	// Config and mode statements now use direct queries (s.db.QueryRowContext/ExecContext)
	// for automatic bad-connection retry. See GetConfig, SetConfig, GetMode, SetMode, etc.

	// User statements (global scope — NOT scoped by registry_ctx)
	createUser        *sql.Stmt
	getUserByID       *sql.Stmt
	getUserByUsername *sql.Stmt
	updateUser        *sql.Stmt
	deleteUser        *sql.Stmt
	listUsers         *sql.Stmt

	// API Key statements (global scope — NOT scoped by registry_ctx)
	createAPIKey           *sql.Stmt
	getAPIKeyByID          *sql.Stmt
	getAPIKeyByHash        *sql.Stmt
	updateAPIKey           *sql.Stmt
	deleteAPIKey           *sql.Stmt
	listAPIKeys            *sql.Stmt
	listAPIKeysByUserID    *sql.Stmt
	getAPIKeyByUserAndName *sql.Stmt
	updateAPIKeyLastUsed   *sql.Stmt
}

// NewStore creates a new MySQL store.
func NewStore(config Config) (*Store, error) {
	db, err := sql.Open("mysql", config.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		db:     db,
		config: config,
	}

	// Run migrations
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Prepare statements
	if err := store.prepareStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	return store, nil
}

// prepareStatements prepares all SQL statements for better performance.
func (s *Store) prepareStatements() error {
	var err error
	stmts := &preparedStatements{}

	// Schema statements — all scoped by registry_ctx
	stmts.getSchemaByID, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND id = ?")
	if err != nil {
		return fmt.Errorf("prepare getSchemaByID: %w", err)
	}

	stmts.getSchemaBySubjectVer, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare getSchemaBySubjectVer: %w", err)
	}

	stmts.getSchemaByFingerprint, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND fingerprint = ? AND deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getSchemaByFingerprint: %w", err)
	}

	stmts.getLatestSchema, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND deleted = FALSE ORDER BY version DESC LIMIT 1")
	if err != nil {
		return fmt.Errorf("prepare getLatestSchema: %w", err)
	}

	stmts.softDeleteSchema, err = s.db.Prepare(
		"UPDATE `schemas` SET deleted = TRUE WHERE registry_ctx = ? AND subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare softDeleteSchema: %w", err)
	}

	stmts.hardDeleteSchema, err = s.db.Prepare(
		"DELETE FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare hardDeleteSchema: %w", err)
	}

	stmts.countSchemasBySubject, err = s.db.Prepare(
		"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare countSchemasBySubject: %w", err)
	}

	// loadReferences now uses direct query (s.db.QueryContext) for automatic bad-connection retry.

	stmts.getSubjectsBySchemaID, err = s.db.Prepare(
		"SELECT DISTINCT s.subject FROM `schemas` s" +
			" JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint" +
			" WHERE s.registry_ctx = ? AND fp.schema_id = ? AND s.deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getSubjectsBySchemaID: %w", err)
	}

	stmts.getVersionsBySchemaID, err = s.db.Prepare(
		"SELECT s.subject, s.version FROM `schemas` s" +
			" JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint" +
			" WHERE s.registry_ctx = ? AND fp.schema_id = ? AND s.deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getVersionsBySchemaID: %w", err)
	}

	stmts.getReferencedBy, err = s.db.Prepare(
		"SELECT s.subject, s.version FROM `schemas` s" +
			" JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint" +
			" JOIN schema_references r ON r.registry_ctx = fp.registry_ctx AND r.schema_id = fp.schema_id" +
			" WHERE s.registry_ctx = ? AND r.ref_subject = ? AND r.ref_version = ? AND s.deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getReferencedBy: %w", err)
	}

	// Config and mode statements now use direct queries (s.db.QueryRowContext/ExecContext)
	// for automatic bad-connection retry under concurrent load.

	// User statements (global scope)
	stmts.createUser, err = s.db.Prepare(
		"INSERT INTO users (username, email, password_hash, role, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare createUser: %w", err)
	}

	stmts.getUserByID, err = s.db.Prepare(
		"SELECT id, username, email, password_hash, role, enabled, created_at, updated_at FROM users WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare getUserByID: %w", err)
	}

	stmts.getUserByUsername, err = s.db.Prepare(
		"SELECT id, username, email, password_hash, role, enabled, created_at, updated_at FROM users WHERE username = ?")
	if err != nil {
		return fmt.Errorf("prepare getUserByUsername: %w", err)
	}

	stmts.updateUser, err = s.db.Prepare(
		"UPDATE users SET username = ?, email = ?, password_hash = ?, role = ?, enabled = ?, updated_at = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare updateUser: %w", err)
	}

	stmts.deleteUser, err = s.db.Prepare(
		"DELETE FROM users WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare deleteUser: %w", err)
	}

	stmts.listUsers, err = s.db.Prepare(
		"SELECT id, username, email, password_hash, role, enabled, created_at, updated_at FROM users ORDER BY username")
	if err != nil {
		return fmt.Errorf("prepare listUsers: %w", err)
	}

	// API Key statements (global scope)
	stmts.createAPIKey, err = s.db.Prepare(
		"INSERT INTO api_keys (user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare createAPIKey: %w", err)
	}

	stmts.getAPIKeyByID, err = s.db.Prepare(
		"SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByID: %w", err)
	}

	stmts.getAPIKeyByHash, err = s.db.Prepare(
		"SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys WHERE key_hash = ?")
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByHash: %w", err)
	}

	stmts.updateAPIKey, err = s.db.Prepare(
		"UPDATE api_keys SET user_id = ?, key_hash = ?, name = ?, role = ?, enabled = ?, expires_at = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare updateAPIKey: %w", err)
	}

	stmts.deleteAPIKey, err = s.db.Prepare(
		"DELETE FROM api_keys WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare deleteAPIKey: %w", err)
	}

	stmts.listAPIKeys, err = s.db.Prepare(
		"SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		return fmt.Errorf("prepare listAPIKeys: %w", err)
	}

	stmts.listAPIKeysByUserID, err = s.db.Prepare(
		"SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys WHERE user_id = ? ORDER BY created_at DESC")
	if err != nil {
		return fmt.Errorf("prepare listAPIKeysByUserID: %w", err)
	}

	stmts.getAPIKeyByUserAndName, err = s.db.Prepare(
		"SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys WHERE user_id = ? AND name = ?")
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByUserAndName: %w", err)
	}

	stmts.updateAPIKeyLastUsed, err = s.db.Prepare(
		"UPDATE api_keys SET last_used = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare updateAPIKeyLastUsed: %w", err)
	}

	s.stmts = stmts
	return nil
}

// closeStatements closes all prepared statements.
func (s *Store) closeStatements() {
	if s.stmts == nil {
		return
	}

	stmts := []*sql.Stmt{
		s.stmts.getSchemaByID, s.stmts.getSchemaBySubjectVer, s.stmts.getSchemaByFingerprint,
		s.stmts.getLatestSchema, s.stmts.softDeleteSchema, s.stmts.hardDeleteSchema,
		s.stmts.countSchemasBySubject, s.stmts.getSubjectsBySchemaID,
		s.stmts.getVersionsBySchemaID, s.stmts.getReferencedBy,
		s.stmts.createUser, s.stmts.getUserByID, s.stmts.getUserByUsername,
		s.stmts.updateUser, s.stmts.deleteUser, s.stmts.listUsers,
		s.stmts.createAPIKey, s.stmts.getAPIKeyByID, s.stmts.getAPIKeyByHash,
		s.stmts.updateAPIKey, s.stmts.deleteAPIKey, s.stmts.listAPIKeys,
		s.stmts.listAPIKeysByUserID, s.stmts.getAPIKeyByUserAndName, s.stmts.updateAPIKeyLastUsed,
	}

	for _, stmt := range stmts {
		if stmt != nil {
			stmt.Close()
		}
	}
}

// migrate runs database migrations.
func (s *Store) migrate(ctx context.Context) error {
	for i, migration := range migrations {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			// Ignore "Duplicate column" errors from ALTER TABLE ADD COLUMN
			// when re-running migrations on an already-migrated database.
			if isMySQLDuplicateColumnError(err) {
				continue
			}
			// Ignore "Can't DROP" errors when index/key was already removed
			if isMySQLCantDropError(err) {
				continue
			}
			// Ignore "Duplicate key name" errors from CREATE INDEX
			// when re-running migrations on an already-migrated database.
			if isMySQLDuplicateKeyNameError(err) {
				continue
			}
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	// Add fingerprint-only index for global dedup (ignore error if already exists)
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE `schemas` ADD INDEX idx_schemas_fingerprint_global (fingerprint)")

	// Remove ON DELETE CASCADE from schema_references FK (ignore error if already dropped)
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE schema_references DROP FOREIGN KEY schema_references_ibfk_1")

	// Backfill schema_fingerprints from existing data (first ID per fingerprint wins)
	_, _ = s.db.ExecContext(ctx, "INSERT IGNORE INTO schema_fingerprints (registry_ctx, fingerprint, schema_id) SELECT registry_ctx, fingerprint, MIN(id) FROM `schemas` GROUP BY registry_ctx, fingerprint")

	return nil
}

// isMySQLDuplicateColumnError checks if the error is a MySQL duplicate column error (error code 1060).
func isMySQLDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "Duplicate column") || contains(errStr, "1060")
}

// isMySQLCantDropError checks if the error is a MySQL "Can't DROP" error (error code 1091).
// This happens when trying to drop an index or key that doesn't exist.
func isMySQLCantDropError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "Can't DROP") || contains(errStr, "1091")
}

// isMySQLDuplicateKeyNameError checks if the error is a MySQL "Duplicate key name" error (error code 1061).
// This happens when trying to CREATE INDEX on an index that already exists.
func isMySQLDuplicateKeyNameError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "Duplicate key name") || contains(errStr, "1061")
}

// globalSchemaID returns the stable per-context schema ID for a fingerprint.
// Looks up the immutable (registry_ctx, fingerprint) -> schema_id mapping in schema_fingerprints.
func (s *Store) globalSchemaID(ctx context.Context, registryCtx, fingerprint string) (int64, error) {
	var globalID int64
	err := s.db.QueryRowContext(ctx,
		"SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = ? AND fingerprint = ?", registryCtx, fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// globalSchemaIDTx returns the stable per-context schema ID within a transaction.
func (s *Store) globalSchemaIDTx(ctx context.Context, tx *sql.Tx, registryCtx, fingerprint string) (int64, error) {
	var globalID int64
	err := tx.QueryRowContext(ctx,
		"SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = ? AND fingerprint = ?", registryCtx, fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// ensureContext ensures a context exists in the contexts tracking table.
func (s *Store) ensureContext(ctx context.Context, registryCtx string) {
	_, _ = s.db.ExecContext(ctx,
		"INSERT IGNORE INTO contexts (registry_ctx) VALUES (?)", registryCtx)
}

// CreateSchema stores a new schema record.
// This implementation handles concurrent insertions by retrying on conflicts.
// MySQL deadlocks are handled as retriable errors.
func (s *Store) CreateSchema(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	s.ensureContext(ctx, registryCtx)
	const maxRetries = 15
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.createSchemaAttempt(ctx, registryCtx, record)
		if err == nil {
			return nil
		}
		if err == storage.ErrSchemaExists {
			return err
		}
		// On duplicate key error or deadlock, retry with exponential backoff + jitter
		if isMySQLDuplicateError(err) || isMySQLDeadlock(err) {
			lastErr = err
			// Exponential backoff: 5ms, 10ms, 20ms, 40ms, ... capped at 500ms
			// Plus jitter of 0-50% to prevent thundering herd
			backoff := time.Duration(5<<attempt) * time.Millisecond
			if backoff > 500*time.Millisecond {
				backoff = 500 * time.Millisecond
			}
			// Add jitter: 0-50% of backoff
			jitter := time.Duration(float64(backoff) * (0.5 * float64(time.Now().UnixNano()%100) / 100))
			time.Sleep(backoff + jitter)
			continue
		}
		return err
	}

	return fmt.Errorf("failed to create schema after %d retries: %w", maxRetries, lastErr)
}

// allocateSchemaID atomically allocates the next per-context schema ID using a
// short-lived transaction. This is separated from createSchemaAttempt to minimize
// lock contention on the ctx_id_alloc row under concurrent writes.
func (s *Store) allocateSchemaID(ctx context.Context, registryCtx string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin ID allocation tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, _ = tx.ExecContext(ctx, "INSERT IGNORE INTO ctx_id_alloc (registry_ctx, next_id) VALUES (?, 1)", registryCtx)
	var id int64
	err = tx.QueryRowContext(ctx, "SELECT next_id FROM ctx_id_alloc WHERE registry_ctx = ? FOR UPDATE", registryCtx).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get next ID: %w", err)
	}
	_, err = tx.ExecContext(ctx, "UPDATE ctx_id_alloc SET next_id = next_id + 1 WHERE registry_ctx = ?", registryCtx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment next ID: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit ID allocation: %w", err)
	}
	return id, nil
}

// createSchemaAttempt performs a single attempt to create a schema.
func (s *Store) createSchemaAttempt(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get next version for this subject FIRST.
	// In READ COMMITTED (MySQL REPEATABLE READ default uses consistent snapshot,
	// but within a transaction each statement still sees committed data after locks).
	// By running this before the fingerprint check, we ensure the fingerprint check
	// uses a same-or-newer snapshot, preventing TOCTOU races where the fingerprint
	// check misses a concurrent commit that MAX(version) would see.
	var nextVersion int
	err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) + 1 FROM `schemas` WHERE registry_ctx = ? AND subject = ?",
		registryCtx, record.Subject,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Check for existing schemas with same fingerprint (idempotent check).
	// With metadata/ruleSet support, multiple versions of the same subject can share
	// a fingerprint (same schema text, different metadata/ruleSet), so we must check all rows.
	var existingDeleted bool
	fpRows, err := tx.QueryContext(ctx,
		"SELECT version, deleted, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND fingerprint = ?",
		registryCtx, record.Subject, record.Fingerprint,
	)
	if err != nil {
		return fmt.Errorf("failed to check existing fingerprint: %w", err)
	}
	for fpRows.Next() {
		var rowVersion int
		var rowDeleted bool
		var rowMetadataJSON, rowRulesetJSON []byte
		if scanErr := fpRows.Scan(&rowVersion, &rowDeleted, &rowMetadataJSON, &rowRulesetJSON); scanErr != nil {
			fpRows.Close()
			return fmt.Errorf("failed to scan fingerprint row: %w", scanErr)
		}
		if rowDeleted {
			existingDeleted = true
			continue
		}
		// Fingerprint matches on a non-deleted row — check metadata and ruleSet.
		// Confluent behavior: same schema text + same metadata/ruleSet = duplicate (return existing).
		// Same schema text + different metadata/ruleSet = new version with same global ID.
		existingMetadata, _ := unmarshalMetadata(rowMetadataJSON)
		existingRuleSet, _ := unmarshalRuleSet(rowRulesetJSON)

		if reflect.DeepEqual(normalizeMetadata(existingMetadata), normalizeMetadata(record.Metadata)) &&
			reflect.DeepEqual(normalizeRuleSet(existingRuleSet), normalizeRuleSet(record.RuleSet)) {
			fpRows.Close()
			// Full duplicate — same schema, same metadata, same ruleSet
			globalID, gErr := s.globalSchemaIDTx(ctx, tx, registryCtx, record.Fingerprint)
			if gErr != nil {
				return gErr
			}
			record.ID = globalID
			record.Version = rowVersion
			return storage.ErrSchemaExists
		}
		// Same schema text but different metadata/ruleSet — continue checking other rows
	}
	fpRows.Close()

	// Serialize metadata and ruleset as JSON
	metadataJSON, err := marshalJSON(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	rulesetJSON, err := marshalJSON(record.RuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal ruleset: %w", err)
	}

	// If a soft-deleted row with the same fingerprint exists, remove it first.
	if existingDeleted {
		_, _ = tx.ExecContext(ctx,
			"DELETE FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND fingerprint = ? AND deleted = TRUE",
			registryCtx, record.Subject, record.Fingerprint,
		)
	}

	// Insert schema - unique constraint on (registry_ctx, subject, version) prevents duplicates
	_, err = tx.ExecContext(ctx,
		"INSERT INTO `schemas` (registry_ctx, subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		registryCtx, record.Subject, nextVersion, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	// Allocate per-context schema ID in a separate short transaction to minimize
	// lock contention on ctx_id_alloc (the FOR UPDATE lock is released immediately
	// instead of being held for the duration of the main transaction).
	nextCtxID, err := s.allocateSchemaID(ctx, registryCtx)
	if err != nil {
		return fmt.Errorf("failed to allocate schema ID: %w", err)
	}

	// Claim fingerprint in schema_fingerprints (first writer wins, per-context)
	_, _ = tx.ExecContext(ctx,
		"INSERT IGNORE INTO schema_fingerprints (registry_ctx, fingerprint, schema_id) VALUES (?, ?, ?)",
		registryCtx, record.Fingerprint, nextCtxID,
	)

	// Resolve stable per-context ID from schema_fingerprints
	globalID, err := s.globalSchemaIDTx(ctx, tx, registryCtx, record.Fingerprint)
	if err != nil {
		return fmt.Errorf("failed to get global schema ID: %w", err)
	}

	record.ID = globalID
	record.Version = nextVersion
	record.CreatedAt = time.Now()

	// Insert references using the per-context ID.
	// Only insert if no references exist yet for this ID (avoids duplicates
	// when same content is registered under multiple subjects).
	if len(record.References) > 0 {
		var refCount int
		_ = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_references WHERE registry_ctx = ? AND schema_id = ?", registryCtx, globalID).Scan(&refCount)
		if refCount == 0 {
			for _, ref := range record.References {
				_, err = tx.ExecContext(ctx,
					"INSERT INTO schema_references (registry_ctx, schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?, ?)",
					registryCtx, globalID, ref.Name, ref.Subject, ref.Version,
				)
				if err != nil {
					return fmt.Errorf("failed to insert reference: %w", err)
				}
			}
		}
	}

	return tx.Commit()
}

// GetSchemaByID retrieves a schema by its global ID.
// Uses schema_fingerprints for per-context ID lookup, then finds the schema
// row by fingerprint within the same context.
func (s *Store) GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var metadataBytes, rulesetBytes []byte

	// Look up the per-context schema ID via schema_fingerprints first.
	var fingerprint string
	fpErr := s.db.QueryRowContext(ctx,
		"SELECT fingerprint FROM schema_fingerprints WHERE registry_ctx = ? AND schema_id = ?", registryCtx, id).Scan(&fingerprint)
	if fpErr != nil {
		return nil, storage.ErrSchemaNotFound
	}

	// Find the schema row by context + fingerprint.
	// Prefer non-deleted rows but fall back to deleted ones — schema content
	// must remain accessible by ID even after all subjects are soft-deleted.
	err := s.db.QueryRowContext(ctx,
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset"+
			" FROM `schemas` WHERE registry_ctx = ? AND fingerprint = ? ORDER BY deleted ASC, id ASC LIMIT 1",
		registryCtx, fingerprint).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Use the per-context schema ID, not the row's auto-generated ID
	record.ID = id

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Load references using the per-context schema ID
	refs, err := s.loadReferences(ctx, registryCtx, id)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (*storage.SchemaRecord, error) {
	// Handle "latest" version (-1)
	if version == -1 {
		return s.GetLatestSchema(ctx, registryCtx, subject)
	}

	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	err := s.stmts.getSchemaBySubjectVer.QueryRowContext(ctx, registryCtx, subject, version).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		// Check if subject exists
		var count int
		_ = s.stmts.countSchemasBySubject.QueryRowContext(ctx, registryCtx, subject).Scan(&count)
		if count == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		return nil, storage.ErrVersionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if record.Deleted {
		return nil, storage.ErrVersionNotFound
	}

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Resolve per-context ID
	globalID, err := s.globalSchemaID(ctx, registryCtx, record.Fingerprint)
	if err != nil {
		record.ID = rowID // fallback
	} else {
		record.ID = globalID
	}

	// Load references using per-context ID
	refs, err := s.loadReferences(ctx, registryCtx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, registryCtx string, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
	}
	query += " ORDER BY version"

	rows, err := s.db.QueryContext(ctx, query, registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*storage.SchemaRecord
	for rows.Next() {
		record := &storage.SchemaRecord{}
		var schemaType string
		var rowID int64
		var metadataBytes, rulesetBytes []byte
		if err := rows.Scan(&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)

		if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
			return nil, err
		}

		// Resolve global ID
		if globalID, gErr := s.globalSchemaID(ctx, registryCtx, record.Fingerprint); gErr == nil {
			record.ID = globalID
		} else {
			record.ID = rowID
		}

		// Load references using global ID
		refs, err := s.loadReferences(ctx, registryCtx, record.ID)
		if err != nil {
			return nil, err
		}
		record.References = refs

		schemas = append(schemas, record)
	}

	if len(schemas) == 0 {
		// Check if subject exists at all (including deleted versions)
		var count int
		_ = s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND subject = ?", registryCtx, subject).Scan(&count)
		if count == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		// Subject exists but all versions are soft-deleted
		return nil, storage.ErrSubjectNotFound
	}

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, registryCtx string, subject, fingerprint string, includeDeleted bool) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte
	var err error

	if includeDeleted {
		query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND fingerprint = ?"
		err = s.db.QueryRowContext(ctx, query, registryCtx, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes)
	} else {
		err = s.stmts.getSchemaByFingerprint.QueryRowContext(ctx, registryCtx, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes)
	}

	if err == sql.ErrNoRows {
		// Check if subject exists at all
		var count int
		_ = s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND subject = ?", registryCtx, subject).Scan(&count)
		if count == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Resolve global ID
	if globalID, gErr := s.globalSchemaID(ctx, registryCtx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, registryCtx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint (global lookup).
// Returns the first matching schema regardless of subject.
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, registryCtx string, fingerprint string) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	// Query for any schema with this fingerprint within this context
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ? AND fingerprint = ? AND deleted = false LIMIT 1"
	err := s.db.QueryRowContext(ctx, query, registryCtx, fingerprint).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema by global fingerprint: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Resolve global ID
	if globalID, gErr := s.globalSchemaID(ctx, registryCtx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, registryCtx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetLatestSchema retrieves the latest schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, registryCtx string, subject string) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	err := s.stmts.getLatestSchema.QueryRowContext(ctx, registryCtx, subject).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSubjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Resolve global ID
	if globalID, gErr := s.globalSchemaID(ctx, registryCtx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, registryCtx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, registryCtx string, subject string, version int, permanent bool) error {
	if permanent {
		// Check if version exists and is soft-deleted first; capture fingerprint for cleanup
		var deleted bool
		var fingerprint string
		err := s.db.QueryRowContext(ctx,
			"SELECT deleted, fingerprint FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND version = ?",
			registryCtx, subject, version).Scan(&deleted, &fingerprint)
		if err == sql.ErrNoRows {
			// Check if subject exists at all
			var count int
			_ = s.db.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND subject = ?", registryCtx, subject).Scan(&count)
			if count == 0 {
				return storage.ErrSubjectNotFound
			}
			return storage.ErrVersionNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to check schema: %w", err)
		}
		if !deleted {
			return storage.ErrVersionNotSoftDeleted
		}
		_, err = s.stmts.hardDeleteSchema.ExecContext(ctx, registryCtx, subject, version)
		if err != nil {
			return fmt.Errorf("failed to delete schema: %w", err)
		}

		// Clean up orphaned schema_fingerprints and schema_references
		s.cleanupOrphanedFingerprint(ctx, registryCtx, fingerprint)

		return nil
	}

	result, err := s.stmts.softDeleteSchema.ExecContext(ctx, registryCtx, subject, version)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if subject exists
		var count int
		_ = s.stmts.countSchemasBySubject.QueryRowContext(ctx, registryCtx, subject).Scan(&count)
		if count == 0 {
			return storage.ErrSubjectNotFound
		}
		return storage.ErrVersionNotFound
	}

	return nil
}

// ListSubjects returns all subject names.
func (s *Store) ListSubjects(ctx context.Context, registryCtx string, includeDeleted bool) ([]string, error) {
	query := "SELECT DISTINCT subject FROM `schemas` WHERE registry_ctx = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
	}
	query += " ORDER BY subject"

	rows, err := s.db.QueryContext(ctx, query, registryCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to query subjects: %w", err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// DeleteSubject deletes all versions of a subject.
func (s *Store) DeleteSubject(ctx context.Context, registryCtx string, subject string, permanent bool) ([]int, error) {
	if permanent {
		// For permanent delete, check that all versions are soft-deleted first
		var totalCount, deletedCount int
		err := s.db.QueryRowContext(ctx,
			"SELECT COUNT(*), COALESCE(SUM(CASE WHEN deleted THEN 1 ELSE 0 END), 0) FROM `schemas` WHERE registry_ctx = ? AND subject = ?",
			registryCtx, subject).Scan(&totalCount, &deletedCount)
		if err != nil {
			return nil, fmt.Errorf("failed to check subject: %w", err)
		}
		if totalCount == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		if deletedCount < totalCount {
			return nil, storage.ErrSubjectNotSoftDeleted
		}

		// Get all versions and unique fingerprints for cleanup
		rows, err := s.db.QueryContext(ctx,
			"SELECT version, fingerprint FROM `schemas` WHERE registry_ctx = ? AND subject = ? ORDER BY version", registryCtx, subject)
		if err != nil {
			return nil, fmt.Errorf("failed to query versions: %w", err)
		}
		var versions []int
		fingerprintSet := make(map[string]struct{})
		for rows.Next() {
			var v int
			var fp string
			if err := rows.Scan(&v, &fp); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}
			versions = append(versions, v)
			fingerprintSet[fp] = struct{}{}
		}
		rows.Close()

		_, err = s.db.ExecContext(ctx, "DELETE FROM `schemas` WHERE registry_ctx = ? AND subject = ?", registryCtx, subject)
		if err != nil {
			return nil, fmt.Errorf("failed to delete schemas: %w", err)
		}
		_, _ = s.db.ExecContext(ctx, "DELETE FROM configs WHERE registry_ctx = ? AND subject = ?", registryCtx, subject)
		_, _ = s.db.ExecContext(ctx, "DELETE FROM modes WHERE registry_ctx = ? AND subject = ?", registryCtx, subject)

		// Clean up orphaned schema_fingerprints and schema_references
		for fp := range fingerprintSet {
			s.cleanupOrphanedFingerprint(ctx, registryCtx, fp)
		}

		return versions, nil
	}

	// Soft-delete: get non-deleted versions
	rows, err := s.db.QueryContext(ctx,
		"SELECT version FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND deleted = FALSE ORDER BY version", registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		versions = append(versions, version)
	}
	rows.Close()

	if len(versions) == 0 {
		// Check if subject exists but is already soft-deleted
		var count int
		_ = s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND subject = ?", registryCtx, subject).Scan(&count)
		if count > 0 {
			return nil, storage.ErrSubjectDeleted
		}
		return nil, storage.ErrSubjectNotFound
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE `schemas` SET deleted = TRUE WHERE registry_ctx = ? AND subject = ?",
		registryCtx, subject,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to soft-delete schemas: %w", err)
	}

	return versions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, registryCtx string, subject string) (bool, error) {
	var count int
	err := s.stmts.countSchemasBySubject.QueryRowContext(ctx, registryCtx, subject).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check subject: %w", err)
	}
	return count > 0, nil
}

// GetConfig retrieves the compatibility configuration for a subject.
func (s *Store) GetConfig(ctx context.Context, registryCtx string, subject string) (*storage.ConfigRecord, error) {
	config := &storage.ConfigRecord{}
	var alias sql.NullString
	var normalize sql.NullBool
	var validateFields sql.NullBool
	var compatibilityGroup sql.NullString
	var defaultMetadataBytes, overrideMetadataBytes []byte
	var defaultRuleSetBytes, overrideRuleSetBytes []byte

	// Use direct query (not prepared statement) for automatic bad-connection retry.
	err := s.db.QueryRowContext(ctx,
		"SELECT subject, compatibility_level, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group FROM configs WHERE registry_ctx = ? AND subject = ?",
		registryCtx, subject).Scan(
		&config.Subject, &config.CompatibilityLevel,
		&alias, &normalize, &validateFields,
		&defaultMetadataBytes, &overrideMetadataBytes,
		&defaultRuleSetBytes, &overrideRuleSetBytes,
		&compatibilityGroup)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if alias.Valid {
		config.Alias = alias.String
	}
	if normalize.Valid {
		v := normalize.Bool
		config.Normalize = &v
	}
	if validateFields.Valid {
		v := validateFields.Bool
		config.ValidateFields = &v
	}
	if compatibilityGroup.Valid {
		config.CompatibilityGroup = compatibilityGroup.String
	}

	if len(defaultMetadataBytes) > 0 {
		m, err := unmarshalMetadata(defaultMetadataBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal default_metadata: %w", err)
		}
		config.DefaultMetadata = m
	}
	if len(overrideMetadataBytes) > 0 {
		m, err := unmarshalMetadata(overrideMetadataBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal override_metadata: %w", err)
		}
		config.OverrideMetadata = m
	}
	if len(defaultRuleSetBytes) > 0 {
		r, err := unmarshalRuleSet(defaultRuleSetBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal default_ruleset: %w", err)
		}
		config.DefaultRuleSet = r
	}
	if len(overrideRuleSetBytes) > 0 {
		r, err := unmarshalRuleSet(overrideRuleSetBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal override_ruleset: %w", err)
		}
		config.OverrideRuleSet = r
	}

	return config, nil
}

// SetConfig sets the compatibility configuration for a subject.
func (s *Store) SetConfig(ctx context.Context, registryCtx string, subject string, config *storage.ConfigRecord) error {
	defaultMetadataJSON, err := marshalJSON(config.DefaultMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal default_metadata: %w", err)
	}
	overrideMetadataJSON, err := marshalJSON(config.OverrideMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal override_metadata: %w", err)
	}
	defaultRuleSetJSON, err := marshalJSON(config.DefaultRuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal default_ruleset: %w", err)
	}
	overrideRuleSetJSON, err := marshalJSON(config.OverrideRuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal override_ruleset: %w", err)
	}

	var aliasParam interface{}
	if config.Alias != "" {
		aliasParam = config.Alias
	}

	var normalizeParam interface{}
	if config.Normalize != nil {
		normalizeParam = *config.Normalize
	}

	var validateFieldsParam interface{}
	if config.ValidateFields != nil {
		validateFieldsParam = *config.ValidateFields
	}

	var compatGroupParam interface{}
	if config.CompatibilityGroup != "" {
		compatGroupParam = config.CompatibilityGroup
	}

	// Use direct query (not prepared statement) for automatic bad-connection retry.
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO configs (registry_ctx, subject, compatibility_level, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE compatibility_level = VALUES(compatibility_level), alias = VALUES(alias), normalize = VALUES(normalize), validate_fields = VALUES(validate_fields), default_metadata = VALUES(default_metadata), override_metadata = VALUES(override_metadata), default_ruleset = VALUES(default_ruleset), override_ruleset = VALUES(override_ruleset), compatibility_group = VALUES(compatibility_group)",
		registryCtx, subject, config.CompatibilityLevel,
		aliasParam, normalizeParam, validateFieldsParam,
		defaultMetadataJSON, overrideMetadataJSON,
		defaultRuleSetJSON, overrideRuleSetJSON,
		compatGroupParam)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, registryCtx string, subject string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM configs WHERE registry_ctx = ? AND subject = ?",
		registryCtx, subject)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetGlobalConfig retrieves the global compatibility configuration.
func (s *Store) GetGlobalConfig(ctx context.Context, registryCtx string) (*storage.ConfigRecord, error) {
	return s.GetConfig(ctx, registryCtx, "")
}

// SetGlobalConfig sets the global compatibility configuration.
func (s *Store) SetGlobalConfig(ctx context.Context, registryCtx string, config *storage.ConfigRecord) error {
	return s.SetConfig(ctx, registryCtx, "", config)
}

// GetMode retrieves the mode for a subject.
func (s *Store) GetMode(ctx context.Context, registryCtx string, subject string) (*storage.ModeRecord, error) {
	mode := &storage.ModeRecord{}
	// Use direct query (not prepared statement) for automatic bad-connection retry.
	err := s.db.QueryRowContext(ctx,
		"SELECT subject, mode FROM modes WHERE registry_ctx = ? AND subject = ?",
		registryCtx, subject).Scan(&mode.Subject, &mode.Mode)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mode: %w", err)
	}

	return mode, nil
}

// SetMode sets the mode for a subject.
func (s *Store) SetMode(ctx context.Context, registryCtx string, subject string, mode *storage.ModeRecord) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO modes (registry_ctx, subject, mode) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE mode = VALUES(mode)",
		registryCtx, subject, mode.Mode)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, registryCtx string, subject string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM modes WHERE registry_ctx = ? AND subject = ?",
		registryCtx, subject)
	if err != nil {
		return fmt.Errorf("failed to delete mode: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetGlobalMode retrieves the global mode.
func (s *Store) GetGlobalMode(ctx context.Context, registryCtx string) (*storage.ModeRecord, error) {
	return s.GetMode(ctx, registryCtx, "")
}

// SetGlobalMode sets the global mode.
func (s *Store) SetGlobalMode(ctx context.Context, registryCtx string, mode *storage.ModeRecord) error {
	return s.SetMode(ctx, registryCtx, "", mode)
}

// NextID returns the next available per-context schema ID.
// Uses the ctx_id_alloc table for per-context ID allocation.
func (s *Store) NextID(ctx context.Context, registryCtx string) (int64, error) {
	return s.allocateSchemaID(ctx, registryCtx)
}

// GetMaxSchemaID returns the highest per-context schema ID currently assigned.
func (s *Store) GetMaxSchemaID(ctx context.Context, registryCtx string) (int64, error) {
	var maxID int64
	err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(schema_id), 0) FROM schema_fingerprints WHERE registry_ctx = ?",
		registryCtx).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max schema ID: %w", err)
	}
	return maxID, nil
}

// ImportSchema inserts a schema with a specified ID (for migration).
// Returns ErrSchemaIDConflict if the ID already exists with different content.
func (s *Store) ImportSchema(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	s.ensureContext(ctx, registryCtx)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if per-context schema ID already exists
	var existingFingerprint string
	idExists := false
	err = tx.QueryRowContext(ctx,
		"SELECT fingerprint FROM schema_fingerprints WHERE registry_ctx = ? AND schema_id = ?",
		registryCtx, record.ID).Scan(&existingFingerprint)
	if err == nil {
		// ID exists — allow if same content (fingerprint), reject if different
		if existingFingerprint != record.Fingerprint {
			return storage.ErrSchemaIDConflict
		}
		idExists = true
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	// Check if version already exists for this subject in this context
	var existingVersion int
	err = tx.QueryRowContext(ctx,
		"SELECT version FROM `schemas` WHERE registry_ctx = ? AND subject = ? AND version = ?",
		registryCtx, record.Subject, record.Version,
	).Scan(&existingVersion)
	if err == nil {
		return storage.ErrSchemaExists
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing version: %w", err)
	}

	// Serialize metadata and ruleset as JSON
	metadataJSON, mErr := marshalJSON(record.Metadata)
	if mErr != nil {
		return fmt.Errorf("failed to marshal metadata: %w", mErr)
	}
	rulesetJSON, rErr := marshalJSON(record.RuleSet)
	if rErr != nil {
		return fmt.Errorf("failed to marshal ruleset: %w", rErr)
	}

	// Insert schema row (always auto-id for the row; the per-context ID is in schema_fingerprints)
	_, err = tx.ExecContext(ctx,
		"INSERT INTO `schemas` (registry_ctx, subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		registryCtx, record.Subject, record.Version, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	if !idExists {
		// Claim per-context fingerprint -> schema_id mapping with the imported ID
		_, err = tx.ExecContext(ctx,
			"INSERT IGNORE INTO schema_fingerprints (registry_ctx, fingerprint, schema_id) VALUES (?, ?, ?)",
			registryCtx, record.Fingerprint, record.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert fingerprint mapping: %w", err)
		}

		// Advance ctx_id_alloc past the imported ID if needed
		_, _ = tx.ExecContext(ctx,
			"INSERT INTO ctx_id_alloc (registry_ctx, next_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE next_id = GREATEST(next_id, VALUES(next_id))",
			registryCtx, record.ID+1,
		)
	}

	// Insert references using the per-context ID
	if len(record.References) > 0 {
		var refCount int
		_ = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_references WHERE registry_ctx = ? AND schema_id = ?", registryCtx, record.ID).Scan(&refCount)
		if refCount == 0 {
			for _, ref := range record.References {
				_, err = tx.ExecContext(ctx,
					"INSERT INTO schema_references (registry_ctx, schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?, ?)",
					registryCtx, record.ID, ref.Name, ref.Subject, ref.Version,
				)
				if err != nil {
					return fmt.Errorf("failed to insert reference: %w", err)
				}
			}
		}
	}

	record.CreatedAt = time.Now()

	return tx.Commit()
}

// SetNextID sets the per-context ID allocator to start from the given value.
// Used after import to prevent ID conflicts.
func (s *Store) SetNextID(ctx context.Context, registryCtx string, id int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO ctx_id_alloc (registry_ctx, next_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE next_id = VALUES(next_id)",
		registryCtx, id)
	if err != nil {
		return fmt.Errorf("failed to set next ID: %w", err)
	}
	return nil
}

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, registryCtx string, subject string, version int) ([]storage.SubjectVersion, error) {
	rows, err := s.stmts.getReferencedBy.QueryContext(ctx, registryCtx, subject, version)
	if err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}
	defer rows.Close()

	var refs []storage.SubjectVersion
	for rows.Next() {
		var ref storage.SubjectVersion
		if err := rows.Scan(&ref.Subject, &ref.Version); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// cleanupOrphanedFingerprint removes schema_fingerprints and schema_references entries
// when no more schemas rows exist for a given fingerprint within this context.
// Called after permanent deletes.
func (s *Store) cleanupOrphanedFingerprint(ctx context.Context, registryCtx, fingerprint string) {
	var remaining int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM `schemas` WHERE registry_ctx = ? AND fingerprint = ?", registryCtx, fingerprint).Scan(&remaining)
	if err != nil || remaining > 0 {
		return
	}
	// No schemas rows left in this context — clean up the stable ID mapping and references
	var schemaID int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = ? AND fingerprint = ?", registryCtx, fingerprint).Scan(&schemaID); err == nil {
		_, _ = s.db.ExecContext(ctx, "DELETE FROM schema_references WHERE registry_ctx = ? AND schema_id = ?", registryCtx, schemaID)
		_, _ = s.db.ExecContext(ctx, "DELETE FROM schema_fingerprints WHERE registry_ctx = ? AND fingerprint = ?", registryCtx, fingerprint)
	}
}

// loadReferences loads references for a schema within a context.
func (s *Store) loadReferences(ctx context.Context, registryCtx string, schemaID int64) ([]storage.Reference, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name, ref_subject, ref_version FROM schema_references WHERE registry_ctx = ? AND schema_id = ?",
		registryCtx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}
	defer rows.Close()

	var refs []storage.Reference
	for rows.Next() {
		var ref storage.Reference
		if err := rows.Scan(&ref.Name, &ref.Subject, &ref.Version); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// GetSubjectsBySchemaID returns all subjects where the given per-context schema ID is registered.
// Uses fingerprint-based lookup via schema_fingerprints for global deduplication.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]string, error) {
	var rows *sql.Rows
	var err error
	if includeDeleted {
		// Include all subjects (both deleted and non-deleted)
		rows, err = s.db.QueryContext(ctx,
			"SELECT DISTINCT s.subject FROM `schemas` s"+
				" JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint"+
				" WHERE s.registry_ctx = ? AND fp.schema_id = ?", registryCtx, id)
	} else {
		rows, err = s.stmts.getSubjectsBySchemaID.QueryContext(ctx, registryCtx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query subjects: %w", err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		subjects = append(subjects, subject)
	}

	if len(subjects) == 0 {
		return nil, storage.ErrSchemaNotFound
	}

	return subjects, nil
}

// GetVersionsBySchemaID returns all subject-version pairs where the given per-context schema ID is registered.
// Uses fingerprint-based lookup via schema_fingerprints for global deduplication.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	var rows *sql.Rows
	var err error
	if includeDeleted {
		// Include all versions (both deleted and non-deleted)
		rows, err = s.db.QueryContext(ctx,
			"SELECT s.subject, s.version FROM `schemas` s"+
				" JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint"+
				" WHERE s.registry_ctx = ? AND fp.schema_id = ?", registryCtx, id)
	} else {
		rows, err = s.stmts.getVersionsBySchemaID.QueryContext(ctx, registryCtx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []storage.SubjectVersion
	for rows.Next() {
		var sv storage.SubjectVersion
		if err := rows.Scan(&sv.Subject, &sv.Version); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		versions = append(versions, sv)
	}

	if len(versions) == 0 {
		return nil, storage.ErrSchemaNotFound
	}

	return versions, nil
}

// ListSchemas returns schemas matching the given filters, scoped to a context.
func (s *Store) ListSchemas(ctx context.Context, registryCtx string, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE registry_ctx = ?"
	args := []interface{}{registryCtx}

	if !params.Deleted {
		query += " AND deleted = ?"
		args = append(args, false)
	}

	if params.SubjectPrefix != "" {
		query += " AND subject LIKE ?"
		args = append(args, params.SubjectPrefix+"%")
	}

	if params.LatestOnly {
		args = []interface{}{registryCtx}
		query = "SELECT s.id, s.subject, s.version, s.schema_type, s.schema_text, s.fingerprint, s.deleted, s.created_at, s.metadata, s.ruleset FROM `schemas` s INNER JOIN (SELECT subject, MAX(version) as max_version FROM `schemas` WHERE registry_ctx = ?"
		if !params.Deleted {
			query += " AND deleted = FALSE"
		}
		if params.SubjectPrefix != "" {
			query += " AND subject LIKE ?"
			args = append(args, params.SubjectPrefix+"%")
		}
		query += " GROUP BY subject) latest ON s.subject = latest.subject AND s.version = latest.max_version"
		query += " WHERE s.registry_ctx = ?"
		args = append(args, registryCtx)
		if !params.Deleted {
			query += " AND s.deleted = FALSE"
		}
	}

	query += " ORDER BY id"

	// MySQL requires LIMIT before OFFSET; add a large default LIMIT when only OFFSET is specified
	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	} else if params.Offset > 0 {
		query += " LIMIT ?"
		args = append(args, int64(math.MaxInt64)) // Large LIMIT as MySQL requires LIMIT before OFFSET
	}

	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*storage.SchemaRecord
	for rows.Next() {
		record := &storage.SchemaRecord{}
		var schemaType string
		var rowID int64
		var metadataBytes, rulesetBytes []byte
		if err := rows.Scan(&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)

		if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
			return nil, err
		}

		// Resolve global ID
		if globalID, gErr := s.globalSchemaID(ctx, registryCtx, record.Fingerprint); gErr == nil {
			record.ID = globalID
		} else {
			record.ID = rowID
		}

		schemas = append(schemas, record)
	}

	return schemas, nil
}

// DeleteGlobalConfig deletes the global config row within a context.
// After deletion, GetGlobalConfig will return ErrNotFound.
func (s *Store) DeleteGlobalConfig(ctx context.Context, registryCtx string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM configs WHERE registry_ctx = ? AND subject = ?",
		registryCtx, "")
	if err != nil {
		return fmt.Errorf("failed to delete global config: %w", err)
	}

	return nil
}

// DeleteGlobalMode deletes the global mode row within a context.
// After deletion, GetGlobalMode will return ErrNotFound.
func (s *Store) DeleteGlobalMode(ctx context.Context, registryCtx string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM modes WHERE registry_ctx = ? AND subject = ?",
		registryCtx, "")
	if err != nil {
		return fmt.Errorf("failed to delete global mode: %w", err)
	}

	return nil
}

// ListContexts returns all registry contexts from the database.
func (s *Store) ListContexts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT registry_ctx FROM contexts ORDER BY registry_ctx")
	if err != nil {
		return nil, fmt.Errorf("failed to query contexts: %w", err)
	}
	defer rows.Close()

	var contexts []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

// CreateUser creates a new user record.
func (s *Store) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	var email sql.NullString
	if user.Email != "" {
		email = sql.NullString{String: user.Email, Valid: true}
	}

	result, err := s.stmts.createUser.ExecContext(ctx,
		user.Username, email, user.PasswordHash, user.Role, user.Enabled, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	user.ID = id

	return nil
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	user := &storage.UserRecord{}
	var email sql.NullString

	err := s.stmts.getUserByID.QueryRowContext(ctx, id).Scan(
		&user.ID, &user.Username, &email, &user.PasswordHash,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		user.Email = email.String
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	user := &storage.UserRecord{}
	var email sql.NullString

	err := s.stmts.getUserByUsername.QueryRowContext(ctx, username).Scan(
		&user.ID, &user.Username, &email, &user.PasswordHash,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		user.Email = email.String
	}

	return user, nil
}

// UpdateUser updates an existing user record.
func (s *Store) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	user.UpdatedAt = time.Now()

	var email sql.NullString
	if user.Email != "" {
		email = sql.NullString{String: user.Email, Valid: true}
	}

	result, err := s.stmts.updateUser.ExecContext(ctx,
		user.Username, email, user.PasswordHash, user.Role, user.Enabled, user.UpdatedAt, user.ID)

	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrUserExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrUserNotFound
	}

	return nil
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	result, err := s.stmts.deleteUser.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrUserNotFound
	}

	return nil
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	rows, err := s.stmts.listUsers.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*storage.UserRecord
	for rows.Next() {
		user := &storage.UserRecord{}
		var email sql.NullString
		if err := rows.Scan(&user.ID, &user.Username, &email, &user.PasswordHash,
			&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if email.Valid {
			user.Email = email.String
		}
		users = append(users, user)
	}

	return users, nil
}

// CreateAPIKey creates a new API key record.
func (s *Store) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	key.CreatedAt = time.Now()

	result, err := s.stmts.createAPIKey.ExecContext(ctx,
		key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.Role, key.Enabled, key.CreatedAt, key.ExpiresAt)

	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrAPIKeyExists
		}
		return fmt.Errorf("failed to create API key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	key.ID = id

	return nil
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Store) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	key := &storage.APIKeyRecord{}
	var userID sql.NullInt64
	var expiresAt, lastUsed sql.NullTime

	err := s.stmts.getAPIKeyByID.QueryRowContext(ctx, id).Scan(
		&key.ID, &userID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Role,
		&key.Enabled, &key.CreatedAt, &expiresAt, &lastUsed)

	if err == sql.ErrNoRows {
		return nil, storage.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if userID.Valid {
		key.UserID = userID.Int64
	}
	if expiresAt.Valid {
		key.ExpiresAt = expiresAt.Time
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}

	return key, nil
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	key := &storage.APIKeyRecord{}
	var userID sql.NullInt64
	var expiresAt, lastUsed sql.NullTime

	err := s.stmts.getAPIKeyByHash.QueryRowContext(ctx, keyHash).Scan(
		&key.ID, &userID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Role,
		&key.Enabled, &key.CreatedAt, &expiresAt, &lastUsed)

	if err == sql.ErrNoRows {
		return nil, storage.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if userID.Valid {
		key.UserID = userID.Int64
	}
	if expiresAt.Valid {
		key.ExpiresAt = expiresAt.Time
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}

	return key, nil
}

// UpdateAPIKey updates an existing API key record.
func (s *Store) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	result, err := s.stmts.updateAPIKey.ExecContext(ctx,
		key.UserID, key.KeyHash, key.Name, key.Role, key.Enabled, key.ExpiresAt, key.ID)

	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrAPIKeyNotFound
	}

	return nil
}

// DeleteAPIKey deletes an API key by ID.
func (s *Store) DeleteAPIKey(ctx context.Context, id int64) error {
	result, err := s.stmts.deleteAPIKey.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrAPIKeyNotFound
	}

	return nil
}

// ListAPIKeys returns all API keys.
func (s *Store) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	rows, err := s.stmts.listAPIKeys.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	return s.scanAPIKeys(rows)
}

// ListAPIKeysByUserID returns all API keys for a user.
func (s *Store) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	rows, err := s.stmts.listAPIKeysByUserID.QueryContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	return s.scanAPIKeys(rows)
}

// GetAPIKeyByUserAndName retrieves an API key by user ID and name.
func (s *Store) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
	key := &storage.APIKeyRecord{}
	var keyUserID sql.NullInt64
	var expiresAt, lastUsed sql.NullTime

	err := s.stmts.getAPIKeyByUserAndName.QueryRowContext(ctx, userID, name).Scan(
		&key.ID, &keyUserID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Role,
		&key.Enabled, &key.CreatedAt, &expiresAt, &lastUsed)

	if err == sql.ErrNoRows {
		return nil, storage.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if keyUserID.Valid {
		key.UserID = keyUserID.Int64
	}
	if expiresAt.Valid {
		key.ExpiresAt = expiresAt.Time
	}
	if lastUsed.Valid {
		key.LastUsed = &lastUsed.Time
	}

	return key, nil
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	result, err := s.stmts.updateAPIKeyLastUsed.ExecContext(ctx, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrAPIKeyNotFound
	}

	return nil
}

// scanAPIKeys scans rows into API key records.
func (s *Store) scanAPIKeys(rows *sql.Rows) ([]*storage.APIKeyRecord, error) {
	var keys []*storage.APIKeyRecord
	for rows.Next() {
		key := &storage.APIKeyRecord{}
		var userID sql.NullInt64
		var expiresAt, lastUsed sql.NullTime
		if err := rows.Scan(&key.ID, &userID, &key.KeyHash, &key.KeyPrefix, &key.Name,
			&key.Role, &key.Enabled, &key.CreatedAt, &expiresAt, &lastUsed); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if userID.Valid {
			key.UserID = userID.Int64
		}
		if expiresAt.Valid {
			key.ExpiresAt = expiresAt.Time
		}
		if lastUsed.Valid {
			key.LastUsed = &lastUsed.Time
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// marshalJSON marshals a value to JSON bytes for storage. Returns nil if v is nil.
func marshalJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	// Handle typed nil pointers (e.g., (*storage.Metadata)(nil) passed as interface{})
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, nil
	}
	return json.Marshal(v)
}

// unmarshalMetadata deserializes JSON bytes into a *storage.Metadata.
// Returns nil if data is nil or empty.
func unmarshalMetadata(data []byte) (*storage.Metadata, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var m storage.Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// unmarshalRuleSet deserializes JSON bytes into a *storage.RuleSet.
// Returns nil if data is nil or empty.
func unmarshalRuleSet(data []byte) (*storage.RuleSet, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var r storage.RuleSet
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// scanSchemaMetadata scans metadata and ruleset JSON columns and populates the schema record.
func scanSchemaMetadata(record *storage.SchemaRecord, metadataBytes, rulesetBytes []byte) error {
	if len(metadataBytes) > 0 {
		m, err := unmarshalMetadata(metadataBytes)
		if err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		record.Metadata = m
	}
	if len(rulesetBytes) > 0 {
		r, err := unmarshalRuleSet(rulesetBytes)
		if err != nil {
			return fmt.Errorf("failed to unmarshal ruleset: %w", err)
		}
		record.RuleSet = r
	}
	return nil
}

// normalizeMetadata returns a non-nil Metadata for consistent comparison.
func normalizeMetadata(m *storage.Metadata) *storage.Metadata {
	if m == nil {
		return &storage.Metadata{}
	}
	return m
}

// normalizeRuleSet returns a non-nil RuleSet for consistent comparison.
func normalizeRuleSet(r *storage.RuleSet) *storage.RuleSet {
	if r == nil {
		return &storage.RuleSet{}
	}
	return r
}

// isMySQLDuplicateError checks if the error is a MySQL duplicate entry error.
func isMySQLDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// MySQL error code 1062 is for duplicate entry
	errStr := err.Error()
	return len(errStr) > 0 && (contains(errStr, "Duplicate entry") || contains(errStr, "1062"))
}

// isMySQLDeadlock checks if the error is a MySQL deadlock error.
func isMySQLDeadlock(err error) bool {
	if err == nil {
		return false
	}
	// MySQL error code 1213 is for deadlock
	errStr := err.Error()
	return len(errStr) > 0 && (contains(errStr, "Deadlock found") || contains(errStr, "1213"))
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close closes the database connection.
func (s *Store) Close() error {
	s.closeStatements()
	return s.db.Close()
}

// IsHealthy returns true if the database connection is healthy.
func (s *Store) IsHealthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.db.PingContext(ctx) == nil
}

// Stats returns connection pool statistics.
func (s *Store) Stats() sql.DBStats {
	return s.db.Stats()
}

// Ensure Store implements storage.Storage
var _ storage.Storage = (*Store)(nil)

// MarshalJSON implements json.Marshaler for Config.
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		Password string `json:"password,omitempty"`
		*Alias
	}{
		Password: "***",
		Alias:    (*Alias)(&c),
	})
}

// CreateExporter creates a new exporter record.
func (s *Store) CreateExporter(ctx context.Context, exporter *storage.ExporterRecord) error {
	subjectsJSON, err := marshalJSON(exporter.Subjects)
	if err != nil {
		return fmt.Errorf("failed to marshal subjects: %w", err)
	}
	configJSON, err := marshalJSON(exporter.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	now := time.Now()
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO exporters (name, context_type, context, subjects, subject_rename_format, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		exporter.Name, exporter.ContextType, exporter.Context, subjectsJSON, exporter.SubjectRenameFormat, configJSON, now, now)
	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrExporterExists
		}
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	exporter.CreatedAt = now
	exporter.UpdatedAt = now
	return nil
}

// GetExporter retrieves an exporter by name.
func (s *Store) GetExporter(ctx context.Context, name string) (*storage.ExporterRecord, error) {
	exporter := &storage.ExporterRecord{}
	var subjectsBytes, configBytes []byte
	var contextType, ctxVal, subjectRenameFormat sql.NullString

	err := s.db.QueryRowContext(ctx,
		"SELECT name, context_type, context, subjects, subject_rename_format, config, created_at, updated_at FROM exporters WHERE name = ?",
		name).Scan(
		&exporter.Name, &contextType, &ctxVal, &subjectsBytes, &subjectRenameFormat, &configBytes,
		&exporter.CreatedAt, &exporter.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, storage.ErrExporterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exporter: %w", err)
	}

	if contextType.Valid {
		exporter.ContextType = contextType.String
	}
	if ctxVal.Valid {
		exporter.Context = ctxVal.String
	}
	if subjectRenameFormat.Valid {
		exporter.SubjectRenameFormat = subjectRenameFormat.String
	}

	if len(subjectsBytes) > 0 {
		if err := json.Unmarshal(subjectsBytes, &exporter.Subjects); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subjects: %w", err)
		}
	}
	if len(configBytes) > 0 {
		if err := json.Unmarshal(configBytes, &exporter.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return exporter, nil
}

// UpdateExporter updates an existing exporter.
func (s *Store) UpdateExporter(ctx context.Context, exporter *storage.ExporterRecord) error {
	subjectsJSON, err := marshalJSON(exporter.Subjects)
	if err != nil {
		return fmt.Errorf("failed to marshal subjects: %w", err)
	}
	configJSON, err := marshalJSON(exporter.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		"UPDATE exporters SET context_type = ?, context = ?, subjects = ?, subject_rename_format = ?, config = ?, updated_at = ? WHERE name = ?",
		exporter.ContextType, exporter.Context, subjectsJSON, exporter.SubjectRenameFormat, configJSON, now, exporter.Name)
	if err != nil {
		return fmt.Errorf("failed to update exporter: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrExporterNotFound
	}

	exporter.UpdatedAt = now
	return nil
}

// DeleteExporter deletes an exporter by name and its associated status (via CASCADE).
func (s *Store) DeleteExporter(ctx context.Context, name string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM exporters WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete exporter: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrExporterNotFound
	}

	return nil
}

// ListExporters returns a sorted list of all exporter names.
func (s *Store) ListExporters(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name FROM exporters ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to query exporters: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

// GetExporterStatus retrieves the status of an exporter.
// If no status row exists but the exporter does, returns a default status with State "PAUSED".
func (s *Store) GetExporterStatus(ctx context.Context, name string) (*storage.ExporterStatusRecord, error) {
	// First check if the exporter exists
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM exporters WHERE name = ?", name).Scan(&exists)
	if err == sql.ErrNoRows {
		return nil, storage.ErrExporterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check exporter: %w", err)
	}

	status := &storage.ExporterStatusRecord{}
	var trace sql.NullString
	err = s.db.QueryRowContext(ctx,
		"SELECT name, state, `offset`, ts, trace FROM exporter_statuses WHERE name = ?",
		name).Scan(&status.Name, &status.State, &status.Offset, &status.Ts, &trace)
	if err == sql.ErrNoRows {
		return &storage.ExporterStatusRecord{
			Name:  name,
			State: "PAUSED",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exporter status: %w", err)
	}

	if trace.Valid {
		status.Trace = trace.String
	}
	return status, nil
}

// SetExporterStatus sets the status of an exporter.
func (s *Store) SetExporterStatus(ctx context.Context, name string, status *storage.ExporterStatusRecord) error {
	// Check if the exporter exists
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM exporters WHERE name = ?", name).Scan(&exists)
	if err == sql.ErrNoRows {
		return storage.ErrExporterNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check exporter: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO exporter_statuses (name, state, `offset`, ts, trace) VALUES (?, ?, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE state = VALUES(state), `offset` = VALUES(`offset`), ts = VALUES(ts), trace = VALUES(trace)",
		name, status.State, status.Offset, status.Ts, status.Trace)
	if err != nil {
		return fmt.Errorf("failed to set exporter status: %w", err)
	}

	return nil
}

// GetExporterConfig retrieves the configuration of an exporter.
func (s *Store) GetExporterConfig(ctx context.Context, name string) (map[string]string, error) {
	var configBytes []byte
	err := s.db.QueryRowContext(ctx, "SELECT config FROM exporters WHERE name = ?", name).Scan(&configBytes)
	if err == sql.ErrNoRows {
		return nil, storage.ErrExporterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get exporter config: %w", err)
	}

	config := make(map[string]string)
	if len(configBytes) > 0 {
		if err := json.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}
	return config, nil
}

// UpdateExporterConfig updates the configuration of an exporter.
func (s *Store) UpdateExporterConfig(ctx context.Context, name string, config map[string]string) error {
	configJSON, err := marshalJSON(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		"UPDATE exporters SET config = ?, updated_at = ? WHERE name = ?",
		configJSON, time.Now(), name)
	if err != nil {
		return fmt.Errorf("failed to update exporter config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrExporterNotFound
	}

	return nil
}

// KEK operations (CSFLE - Client-Side Field Level Encryption)

// CreateKEK creates a new Key Encryption Key.
func (s *Store) CreateKEK(ctx context.Context, kek *storage.KEKRecord) error {
	kmsPropsJSON, err := marshalJSON(kek.KmsProps)
	if err != nil {
		return fmt.Errorf("failed to marshal kms_props: %w", err)
	}

	now := time.Now()
	kek.Ts = now.UnixMilli()

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO keks (name, kms_type, kms_key_id, kms_props, doc, shared, deleted, ts, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		kek.Name, kek.KmsType, kek.KmsKeyID, kmsPropsJSON, kek.Doc, kek.Shared, kek.Deleted, kek.Ts, now, now)
	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrKEKExists
		}
		return fmt.Errorf("failed to create KEK: %w", err)
	}

	kek.CreatedAt = now
	kek.UpdatedAt = now
	return nil
}

// GetKEK retrieves a Key Encryption Key by name.
func (s *Store) GetKEK(ctx context.Context, name string, includeDeleted bool) (*storage.KEKRecord, error) {
	kek := &storage.KEKRecord{}
	var kmsPropsBytes []byte
	var doc sql.NullString

	err := s.db.QueryRowContext(ctx,
		"SELECT name, kms_type, kms_key_id, kms_props, doc, shared, deleted, ts, created_at, updated_at FROM keks WHERE name = ?",
		name).Scan(
		&kek.Name, &kek.KmsType, &kek.KmsKeyID, &kmsPropsBytes, &doc,
		&kek.Shared, &kek.Deleted, &kek.Ts, &kek.CreatedAt, &kek.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, storage.ErrKEKNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get KEK: %w", err)
	}

	if !includeDeleted && kek.Deleted {
		return nil, storage.ErrKEKNotFound
	}

	if doc.Valid {
		kek.Doc = doc.String
	}
	if len(kmsPropsBytes) > 0 {
		if err := json.Unmarshal(kmsPropsBytes, &kek.KmsProps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kms_props: %w", err)
		}
	}

	return kek, nil
}

// UpdateKEK updates an existing Key Encryption Key.
func (s *Store) UpdateKEK(ctx context.Context, kek *storage.KEKRecord) error {
	kmsPropsJSON, err := marshalJSON(kek.KmsProps)
	if err != nil {
		return fmt.Errorf("failed to marshal kms_props: %w", err)
	}

	now := time.Now()
	kek.Ts = now.UnixMilli()

	result, err := s.db.ExecContext(ctx,
		"UPDATE keks SET kms_type = ?, kms_key_id = ?, kms_props = ?, doc = ?, shared = ?, deleted = ?, ts = ?, updated_at = ? WHERE name = ?",
		kek.KmsType, kek.KmsKeyID, kmsPropsJSON, kek.Doc, kek.Shared, kek.Deleted, kek.Ts, now, kek.Name)
	if err != nil {
		return fmt.Errorf("failed to update KEK: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrKEKNotFound
	}

	kek.UpdatedAt = now
	return nil
}

// DeleteKEK soft-deletes or permanently deletes a Key Encryption Key.
// Permanent delete also removes all associated DEKs.
func (s *Store) DeleteKEK(ctx context.Context, name string, permanent bool) error {
	if permanent {
		// Permanently delete the KEK and all associated DEKs
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback() }()

		// Check KEK exists
		var exists int
		err = tx.QueryRowContext(ctx, "SELECT 1 FROM keks WHERE name = ?", name).Scan(&exists)
		if err == sql.ErrNoRows {
			return storage.ErrKEKNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to check KEK: %w", err)
		}

		// Delete all DEKs under this KEK
		_, err = tx.ExecContext(ctx, "DELETE FROM deks WHERE kek_name = ?", name)
		if err != nil {
			return fmt.Errorf("failed to delete DEKs: %w", err)
		}

		// Delete the KEK
		_, err = tx.ExecContext(ctx, "DELETE FROM keks WHERE name = ?", name)
		if err != nil {
			return fmt.Errorf("failed to delete KEK: %w", err)
		}

		return tx.Commit()
	}

	// Soft delete
	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		"UPDATE keks SET deleted = TRUE, ts = ? WHERE name = ?",
		now.UnixMilli(), name)
	if err != nil {
		return fmt.Errorf("failed to soft-delete KEK: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrKEKNotFound
	}

	return nil
}

// UndeleteKEK restores a soft-deleted Key Encryption Key.
func (s *Store) UndeleteKEK(ctx context.Context, name string) error {
	// Check if the KEK exists and is deleted
	var deleted bool
	err := s.db.QueryRowContext(ctx, "SELECT deleted FROM keks WHERE name = ?", name).Scan(&deleted)
	if err == sql.ErrNoRows {
		return storage.ErrKEKNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check KEK: %w", err)
	}

	if !deleted {
		return storage.ErrKEKNotFound
	}

	now := time.Now()
	_, err = s.db.ExecContext(ctx,
		"UPDATE keks SET deleted = FALSE, ts = ? WHERE name = ?",
		now.UnixMilli(), name)
	if err != nil {
		return fmt.Errorf("failed to undelete KEK: %w", err)
	}

	return nil
}

// ListKEKs returns all Key Encryption Keys, sorted by name.
func (s *Store) ListKEKs(ctx context.Context, includeDeleted bool) ([]*storage.KEKRecord, error) {
	query := "SELECT name, kms_type, kms_key_id, kms_props, doc, shared, deleted, ts, created_at, updated_at FROM keks"
	if !includeDeleted {
		query += " WHERE deleted = FALSE"
	}
	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query KEKs: %w", err)
	}
	defer rows.Close()

	var keks []*storage.KEKRecord
	for rows.Next() {
		kek := &storage.KEKRecord{}
		var kmsPropsBytes []byte
		var doc sql.NullString
		if err := rows.Scan(&kek.Name, &kek.KmsType, &kek.KmsKeyID, &kmsPropsBytes, &doc,
			&kek.Shared, &kek.Deleted, &kek.Ts, &kek.CreatedAt, &kek.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if doc.Valid {
			kek.Doc = doc.String
		}
		if len(kmsPropsBytes) > 0 {
			if err := json.Unmarshal(kmsPropsBytes, &kek.KmsProps); err != nil {
				return nil, fmt.Errorf("failed to unmarshal kms_props: %w", err)
			}
		}
		keks = append(keks, kek)
	}

	return keks, nil
}

// DEK operations (CSFLE - Client-Side Field Level Encryption)

// CreateDEK creates a new Data Encryption Key under an existing KEK.
// Auto-assigns version if version <= 0 (SELECT MAX(version) + 1).
func (s *Store) CreateDEK(ctx context.Context, dek *storage.DEKRecord) error {
	// Check that the KEK exists
	var kekExists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM keks WHERE name = ?", dek.KEKName).Scan(&kekExists)
	if err == sql.ErrNoRows {
		return storage.ErrKEKNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check KEK: %w", err)
	}

	// Default algorithm
	if dek.Algorithm == "" {
		dek.Algorithm = "AES256_GCM"
	}

	// Auto-assign version if not specified
	if dek.Version <= 0 {
		var maxVersion sql.NullInt64
		err = s.db.QueryRowContext(ctx,
			"SELECT MAX(version) FROM deks WHERE kek_name = ? AND subject = ?",
			dek.KEKName, dek.Subject).Scan(&maxVersion)
		if err != nil {
			return fmt.Errorf("failed to get max version: %w", err)
		}
		if maxVersion.Valid {
			dek.Version = int(maxVersion.Int64) + 1
		} else {
			dek.Version = 1
		}
	}

	dek.Ts = time.Now().UnixMilli()

	_, err = s.db.ExecContext(ctx,
		"INSERT INTO deks (kek_name, subject, version, algorithm, encrypted_key_material, deleted, ts) VALUES (?, ?, ?, ?, ?, ?, ?)",
		dek.KEKName, dek.Subject, dek.Version, dek.Algorithm, dek.EncryptedKeyMaterial, dek.Deleted, dek.Ts)
	if err != nil {
		if isMySQLDuplicateError(err) {
			return storage.ErrDEKExists
		}
		return fmt.Errorf("failed to create DEK: %w", err)
	}

	return nil
}

// GetDEK retrieves a Data Encryption Key.
// If version <= 0, returns the latest version. If algorithm is non-empty, filters by it.
func (s *Store) GetDEK(ctx context.Context, kekName, subject string, version int, algorithm string, includeDeleted bool) (*storage.DEKRecord, error) {
	dek := &storage.DEKRecord{}
	var encryptedKeyMaterial sql.NullString

	if version <= 0 {
		// Find the latest version
		query := "SELECT kek_name, subject, version, algorithm, encrypted_key_material, deleted, ts FROM deks WHERE kek_name = ? AND subject = ?"
		args := []interface{}{kekName, subject}
		if !includeDeleted {
			query += " AND deleted = FALSE"
		}
		if algorithm != "" {
			query += " AND algorithm = ?"
			args = append(args, algorithm)
		}
		query += " ORDER BY version DESC LIMIT 1"

		err := s.db.QueryRowContext(ctx, query, args...).Scan(
			&dek.KEKName, &dek.Subject, &dek.Version, &dek.Algorithm,
			&encryptedKeyMaterial, &dek.Deleted, &dek.Ts)
		if err == sql.ErrNoRows {
			return nil, storage.ErrDEKNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get DEK: %w", err)
		}
	} else {
		query := "SELECT kek_name, subject, version, algorithm, encrypted_key_material, deleted, ts FROM deks WHERE kek_name = ? AND subject = ? AND version = ?"
		args := []interface{}{kekName, subject, version}
		if algorithm != "" {
			query += " AND algorithm = ?"
			args = append(args, algorithm)
		}

		err := s.db.QueryRowContext(ctx, query, args...).Scan(
			&dek.KEKName, &dek.Subject, &dek.Version, &dek.Algorithm,
			&encryptedKeyMaterial, &dek.Deleted, &dek.Ts)
		if err == sql.ErrNoRows {
			return nil, storage.ErrDEKNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get DEK: %w", err)
		}

		if !includeDeleted && dek.Deleted {
			return nil, storage.ErrDEKNotFound
		}
	}

	if encryptedKeyMaterial.Valid {
		dek.EncryptedKeyMaterial = encryptedKeyMaterial.String
	}

	return dek, nil
}

// ListDEKs returns the sorted list of unique subject names under a KEK.
func (s *Store) ListDEKs(ctx context.Context, kekName string, includeDeleted bool) ([]string, error) {
	// Check that the KEK exists
	var kekExists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM keks WHERE name = ?", kekName).Scan(&kekExists)
	if err == sql.ErrNoRows {
		return nil, storage.ErrKEKNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check KEK: %w", err)
	}

	query := "SELECT DISTINCT subject FROM deks WHERE kek_name = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
	}
	query += " ORDER BY subject"

	rows, err := s.db.QueryContext(ctx, query, kekName)
	if err != nil {
		return nil, fmt.Errorf("failed to query DEK subjects: %w", err)
	}
	defer rows.Close()

	subjects := []string{}
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// ListDEKVersions returns the sorted list of version numbers for a KEK+subject combination.
func (s *Store) ListDEKVersions(ctx context.Context, kekName, subject string, algorithm string, includeDeleted bool) ([]int, error) {
	// Check that the KEK exists
	var kekExists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM keks WHERE name = ?", kekName).Scan(&kekExists)
	if err == sql.ErrNoRows {
		return nil, storage.ErrKEKNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check KEK: %w", err)
	}

	query := "SELECT version FROM deks WHERE kek_name = ? AND subject = ?"
	args := []interface{}{kekName, subject}
	if !includeDeleted {
		query += " AND deleted = FALSE"
	}
	if algorithm != "" {
		query += " AND algorithm = ?"
		args = append(args, algorithm)
	}
	query += " ORDER BY version"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query DEK versions: %w", err)
	}
	defer rows.Close()

	versions := []int{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// DeleteDEK soft-deletes or permanently deletes a Data Encryption Key.
// Version -1 means delete all versions for the kekName+subject combination.
func (s *Store) DeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string, permanent bool) error {
	if version == -1 {
		// Delete all versions matching the criteria
		query := "SELECT COUNT(*) FROM deks WHERE kek_name = ? AND subject = ?"
		args := []interface{}{kekName, subject}
		if algorithm != "" {
			query += " AND algorithm = ?"
			args = append(args, algorithm)
		}

		var count int
		if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return fmt.Errorf("failed to count DEKs: %w", err)
		}
		if count == 0 {
			return storage.ErrDEKNotFound
		}

		if permanent {
			delQuery := "DELETE FROM deks WHERE kek_name = ? AND subject = ?"
			delArgs := []interface{}{kekName, subject}
			if algorithm != "" {
				delQuery += " AND algorithm = ?"
				delArgs = append(delArgs, algorithm)
			}
			_, err := s.db.ExecContext(ctx, delQuery, delArgs...)
			if err != nil {
				return fmt.Errorf("failed to delete DEKs: %w", err)
			}
		} else {
			updQuery := "UPDATE deks SET deleted = TRUE, ts = ? WHERE kek_name = ? AND subject = ?"
			updArgs := []interface{}{time.Now().UnixMilli(), kekName, subject}
			if algorithm != "" {
				updQuery += " AND algorithm = ?"
				updArgs = append(updArgs, algorithm)
			}
			_, err := s.db.ExecContext(ctx, updQuery, updArgs...)
			if err != nil {
				return fmt.Errorf("failed to soft-delete DEKs: %w", err)
			}
		}
		return nil
	}

	// Single version delete
	if algorithm == "" {
		algorithm = "AES256_GCM"
	}

	// Check if the DEK exists
	var exists int
	err := s.db.QueryRowContext(ctx,
		"SELECT 1 FROM deks WHERE kek_name = ? AND subject = ? AND version = ? AND algorithm = ?",
		kekName, subject, version, algorithm).Scan(&exists)
	if err == sql.ErrNoRows {
		return storage.ErrDEKNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check DEK: %w", err)
	}

	if permanent {
		_, err = s.db.ExecContext(ctx,
			"DELETE FROM deks WHERE kek_name = ? AND subject = ? AND version = ? AND algorithm = ?",
			kekName, subject, version, algorithm)
	} else {
		_, err = s.db.ExecContext(ctx,
			"UPDATE deks SET deleted = TRUE, ts = ? WHERE kek_name = ? AND subject = ? AND version = ? AND algorithm = ?",
			time.Now().UnixMilli(), kekName, subject, version, algorithm)
	}
	if err != nil {
		return fmt.Errorf("failed to delete DEK: %w", err)
	}

	return nil
}

// UndeleteDEK restores a soft-deleted Data Encryption Key.
// Version -1 means undelete all deleted versions for the kekName+subject combination.
func (s *Store) UndeleteDEK(ctx context.Context, kekName, subject string, version int, algorithm string) error {
	if version == -1 {
		// Undelete all deleted versions matching the criteria
		query := "SELECT COUNT(*) FROM deks WHERE kek_name = ? AND subject = ? AND deleted = TRUE"
		args := []interface{}{kekName, subject}
		if algorithm != "" {
			query += " AND algorithm = ?"
			args = append(args, algorithm)
		}

		var count int
		if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return fmt.Errorf("failed to count deleted DEKs: %w", err)
		}
		if count == 0 {
			return storage.ErrDEKNotFound
		}

		updQuery := "UPDATE deks SET deleted = FALSE, ts = ? WHERE kek_name = ? AND subject = ? AND deleted = TRUE"
		updArgs := []interface{}{time.Now().UnixMilli(), kekName, subject}
		if algorithm != "" {
			updQuery += " AND algorithm = ?"
			updArgs = append(updArgs, algorithm)
		}
		_, err := s.db.ExecContext(ctx, updQuery, updArgs...)
		if err != nil {
			return fmt.Errorf("failed to undelete DEKs: %w", err)
		}
		return nil
	}

	// Single version undelete
	if algorithm == "" {
		algorithm = "AES256_GCM"
	}

	var deleted bool
	err := s.db.QueryRowContext(ctx,
		"SELECT deleted FROM deks WHERE kek_name = ? AND subject = ? AND version = ? AND algorithm = ?",
		kekName, subject, version, algorithm).Scan(&deleted)
	if err == sql.ErrNoRows {
		return storage.ErrDEKNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check DEK: %w", err)
	}

	if !deleted {
		return storage.ErrDEKNotFound
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE deks SET deleted = FALSE, ts = ? WHERE kek_name = ? AND subject = ? AND version = ? AND algorithm = ?",
		time.Now().UnixMilli(), kekName, subject, version, algorithm)
	if err != nil {
		return fmt.Errorf("failed to undelete DEK: %w", err)
	}

	return nil
}
