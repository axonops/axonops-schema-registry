package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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
	// Schema statements
	getSchemaByID          *sql.Stmt
	getSchemaBySubjectVer  *sql.Stmt
	getSchemaByFingerprint *sql.Stmt
	getLatestSchema        *sql.Stmt
	softDeleteSchema       *sql.Stmt
	hardDeleteSchema       *sql.Stmt
	countSchemasBySubject  *sql.Stmt
	loadReferences         *sql.Stmt
	getReferencedBy        *sql.Stmt

	// Config statements
	getConfig    *sql.Stmt
	setConfig    *sql.Stmt
	deleteConfig *sql.Stmt

	// Mode statements
	getMode    *sql.Stmt
	setMode    *sql.Stmt
	deleteMode *sql.Stmt

	// User statements
	createUser        *sql.Stmt
	getUserByID       *sql.Stmt
	getUserByUsername *sql.Stmt
	updateUser        *sql.Stmt
	deleteUser        *sql.Stmt
	listUsers         *sql.Stmt

	// API Key statements
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

	// Schema statements
	stmts.getSchemaByID, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE id = ?")
	if err != nil {
		return fmt.Errorf("prepare getSchemaByID: %w", err)
	}

	stmts.getSchemaBySubjectVer, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare getSchemaBySubjectVer: %w", err)
	}

	stmts.getSchemaByFingerprint, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE subject = ? AND fingerprint = ? AND deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getSchemaByFingerprint: %w", err)
	}

	stmts.getLatestSchema, err = s.db.Prepare(
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE subject = ? AND deleted = FALSE ORDER BY version DESC LIMIT 1")
	if err != nil {
		return fmt.Errorf("prepare getLatestSchema: %w", err)
	}

	stmts.softDeleteSchema, err = s.db.Prepare(
		"UPDATE `schemas` SET deleted = TRUE WHERE subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare softDeleteSchema: %w", err)
	}

	stmts.hardDeleteSchema, err = s.db.Prepare(
		"DELETE FROM `schemas` WHERE subject = ? AND version = ?")
	if err != nil {
		return fmt.Errorf("prepare hardDeleteSchema: %w", err)
	}

	stmts.countSchemasBySubject, err = s.db.Prepare(
		"SELECT COUNT(*) FROM `schemas` WHERE subject = ? AND deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare countSchemasBySubject: %w", err)
	}

	stmts.loadReferences, err = s.db.Prepare(
		"SELECT name, ref_subject, ref_version FROM schema_references WHERE schema_id = ?")
	if err != nil {
		return fmt.Errorf("prepare loadReferences: %w", err)
	}

	stmts.getReferencedBy, err = s.db.Prepare(
		"SELECT s.subject, s.version FROM `schemas` s JOIN schema_references r ON r.schema_id = s.id WHERE r.ref_subject = ? AND r.ref_version = ? AND s.deleted = FALSE")
	if err != nil {
		return fmt.Errorf("prepare getReferencedBy: %w", err)
	}

	// Config statements
	stmts.getConfig, err = s.db.Prepare(
		"SELECT subject, compatibility_level, alias, normalize, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group FROM configs WHERE subject = ?")
	if err != nil {
		return fmt.Errorf("prepare getConfig: %w", err)
	}

	stmts.setConfig, err = s.db.Prepare(
		"INSERT INTO configs (subject, compatibility_level, alias, normalize, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE compatibility_level = VALUES(compatibility_level), alias = VALUES(alias), normalize = VALUES(normalize), default_metadata = VALUES(default_metadata), override_metadata = VALUES(override_metadata), default_ruleset = VALUES(default_ruleset), override_ruleset = VALUES(override_ruleset), compatibility_group = VALUES(compatibility_group)")
	if err != nil {
		return fmt.Errorf("prepare setConfig: %w", err)
	}

	stmts.deleteConfig, err = s.db.Prepare(
		"DELETE FROM configs WHERE subject = ?")
	if err != nil {
		return fmt.Errorf("prepare deleteConfig: %w", err)
	}

	// Mode statements
	stmts.getMode, err = s.db.Prepare(
		"SELECT subject, mode FROM modes WHERE subject = ?")
	if err != nil {
		return fmt.Errorf("prepare getMode: %w", err)
	}

	stmts.setMode, err = s.db.Prepare(
		"INSERT INTO modes (subject, mode) VALUES (?, ?) ON DUPLICATE KEY UPDATE mode = VALUES(mode)")
	if err != nil {
		return fmt.Errorf("prepare setMode: %w", err)
	}

	stmts.deleteMode, err = s.db.Prepare(
		"DELETE FROM modes WHERE subject = ?")
	if err != nil {
		return fmt.Errorf("prepare deleteMode: %w", err)
	}

	// User statements
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

	// API Key statements
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
		s.stmts.countSchemasBySubject, s.stmts.loadReferences, s.stmts.getReferencedBy,
		s.stmts.getConfig, s.stmts.setConfig, s.stmts.deleteConfig,
		s.stmts.getMode, s.stmts.setMode, s.stmts.deleteMode,
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
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	// Add fingerprint-only index for global dedup (ignore error if already exists)
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE `schemas` ADD INDEX idx_schemas_fingerprint_global (fingerprint)")
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

// globalSchemaID returns the global schema ID for a fingerprint.
// The global ID is the MIN(id) across all rows with the same fingerprint,
// which ensures the same schema content always gets the same API-visible ID.
func (s *Store) globalSchemaID(ctx context.Context, fingerprint string) (int64, error) {
	var globalID int64
	err := s.db.QueryRowContext(ctx,
		"SELECT MIN(id) FROM `schemas` WHERE fingerprint = ?", fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// globalSchemaIDTx returns the global schema ID within a transaction.
func (s *Store) globalSchemaIDTx(ctx context.Context, tx *sql.Tx, fingerprint string) (int64, error) {
	var globalID int64
	err := tx.QueryRowContext(ctx,
		"SELECT MIN(id) FROM `schemas` WHERE fingerprint = ?", fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// CreateSchema stores a new schema record.
// This implementation handles concurrent insertions by retrying on conflicts.
// MySQL deadlocks are handled as retriable errors.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	const maxRetries = 15
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.createSchemaAttempt(ctx, record)
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

// createSchemaAttempt performs a single attempt to create a schema.
func (s *Store) createSchemaAttempt(ctx context.Context, record *storage.SchemaRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check for existing schema with same fingerprint in this subject (idempotent check)
	var existingVersion int
	var existingDeleted bool
	err = tx.QueryRowContext(ctx,
		"SELECT version, deleted FROM `schemas` WHERE subject = ? AND fingerprint = ?",
		record.Subject, record.Fingerprint,
	).Scan(&existingVersion, &existingDeleted)

	if err == nil && !existingDeleted {
		// Schema already exists in this subject - resolve global ID
		globalID, gErr := s.globalSchemaIDTx(ctx, tx, record.Fingerprint)
		if gErr != nil {
			return gErr
		}
		record.ID = globalID
		record.Version = existingVersion
		return storage.ErrSchemaExists
	}

	// Get next version for this subject (no locking - rely on unique constraint)
	var nextVersion int
	err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) + 1 FROM `schemas` WHERE subject = ?",
		record.Subject,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Serialize metadata and ruleset as JSON
	metadataJSON, err := marshalJSON(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	rulesetJSON, err := marshalJSON(record.RuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal ruleset: %w", err)
	}

	// Insert schema - unique constraint on (subject, version) prevents duplicates
	_, err = tx.ExecContext(ctx,
		"INSERT INTO `schemas` (subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		record.Subject, nextVersion, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	// Resolve global schema ID (MIN(id) across all rows with same fingerprint)
	globalID, err := s.globalSchemaIDTx(ctx, tx, record.Fingerprint)
	if err != nil {
		return fmt.Errorf("failed to get global schema ID: %w", err)
	}

	record.ID = globalID
	record.Version = nextVersion
	record.CreatedAt = time.Now()

	// Insert references using the global ID.
	// Only insert if no references exist yet for this global ID (avoids duplicates
	// when same content is registered under multiple subjects).
	if len(record.References) > 0 {
		var refCount int
		_ = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_references WHERE schema_id = ?", globalID).Scan(&refCount)
		if refCount == 0 {
			for _, ref := range record.References {
				_, err = tx.ExecContext(ctx,
					`INSERT INTO schema_references (schema_id, name, ref_subject, ref_version)
					 VALUES (?, ?, ?, ?)`,
					globalID, ref.Name, ref.Subject, ref.Version,
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
func (s *Store) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var metadataBytes, rulesetBytes []byte

	err := s.stmts.getSchemaByID.QueryRowContext(ctx, id).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	if err := scanSchemaMetadata(record, metadataBytes, rulesetBytes); err != nil {
		return nil, err
	}

	// Load references
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	// Handle "latest" version (-1)
	if version == -1 {
		return s.GetLatestSchema(ctx, subject)
	}

	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	err := s.stmts.getSchemaBySubjectVer.QueryRowContext(ctx, subject, version).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataBytes, &rulesetBytes)

	if err == sql.ErrNoRows {
		// Check if subject exists
		var count int
		_ = s.stmts.countSchemasBySubject.QueryRowContext(ctx, subject).Scan(&count)
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

	// Resolve global ID
	globalID, err := s.globalSchemaID(ctx, record.Fingerprint)
	if err != nil {
		record.ID = rowID // fallback
	} else {
		record.ID = globalID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE subject = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
	}
	query += " ORDER BY version"

	rows, err := s.db.QueryContext(ctx, query, subject)
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
		if globalID, gErr := s.globalSchemaID(ctx, record.Fingerprint); gErr == nil {
			record.ID = globalID
		} else {
			record.ID = rowID
		}

		// Load references using global ID
		refs, err := s.loadReferences(ctx, record.ID)
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
			"SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
		if count == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		// Subject exists but all versions are soft-deleted
		return nil, storage.ErrSubjectNotFound
	}

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string, includeDeleted bool) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte
	var err error

	if includeDeleted {
		query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE subject = ? AND fingerprint = ?"
		err = s.db.QueryRowContext(ctx, query, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes)
	} else {
		err = s.stmts.getSchemaByFingerprint.QueryRowContext(ctx, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataBytes, &rulesetBytes)
	}

	if err == sql.ErrNoRows {
		// Check if subject exists at all
		var count int
		_ = s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
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
	if globalID, gErr := s.globalSchemaID(ctx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint (global lookup).
// Returns the first matching schema regardless of subject.
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, fingerprint string) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	// Query for any schema with this fingerprint (global deduplication)
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE fingerprint = ? AND deleted = false LIMIT 1"
	err := s.db.QueryRowContext(ctx, query, fingerprint).Scan(
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
	if globalID, gErr := s.globalSchemaID(ctx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetLatestSchema retrieves the latest schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, subject string) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var rowID int64
	var metadataBytes, rulesetBytes []byte

	err := s.stmts.getLatestSchema.QueryRowContext(ctx, subject).Scan(
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
	if globalID, gErr := s.globalSchemaID(ctx, record.Fingerprint); gErr == nil {
		record.ID = globalID
	} else {
		record.ID = rowID
	}

	// Load references using global ID
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	if permanent {
		// Check if version exists and is soft-deleted first
		var deleted bool
		err := s.db.QueryRowContext(ctx,
			"SELECT deleted FROM `schemas` WHERE subject = ? AND version = ?",
			subject, version).Scan(&deleted)
		if err == sql.ErrNoRows {
			// Check if subject exists at all
			var count int
			_ = s.db.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
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
		_, err = s.stmts.hardDeleteSchema.ExecContext(ctx, subject, version)
		if err != nil {
			return fmt.Errorf("failed to delete schema: %w", err)
		}
		return nil
	}

	result, err := s.stmts.softDeleteSchema.ExecContext(ctx, subject, version)
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if subject exists
		var count int
		_ = s.stmts.countSchemasBySubject.QueryRowContext(ctx, subject).Scan(&count)
		if count == 0 {
			return storage.ErrSubjectNotFound
		}
		return storage.ErrVersionNotFound
	}

	return nil
}

// ListSubjects returns all subject names.
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	query := "SELECT DISTINCT subject FROM `schemas`"
	if !includeDeleted {
		query += " WHERE deleted = FALSE"
	}
	query += " ORDER BY subject"

	rows, err := s.db.QueryContext(ctx, query)
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
func (s *Store) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	if permanent {
		// For permanent delete, check that all versions are soft-deleted first
		var totalCount, deletedCount int
		err := s.db.QueryRowContext(ctx,
			"SELECT COUNT(*), COALESCE(SUM(CASE WHEN deleted THEN 1 ELSE 0 END), 0) FROM `schemas` WHERE subject = ?",
			subject).Scan(&totalCount, &deletedCount)
		if err != nil {
			return nil, fmt.Errorf("failed to check subject: %w", err)
		}
		if totalCount == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		if deletedCount < totalCount {
			return nil, storage.ErrSubjectNotSoftDeleted
		}

		// Get all versions for the response
		rows, err := s.db.QueryContext(ctx,
			"SELECT version FROM `schemas` WHERE subject = ? ORDER BY version", subject)
		if err != nil {
			return nil, fmt.Errorf("failed to query versions: %w", err)
		}
		var versions []int
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}
			versions = append(versions, v)
		}
		rows.Close()

		_, err = s.db.ExecContext(ctx, "DELETE FROM `schemas` WHERE subject = ?", subject)
		if err != nil {
			return nil, fmt.Errorf("failed to delete schemas: %w", err)
		}
		_, _ = s.db.ExecContext(ctx, "DELETE FROM configs WHERE subject = ?", subject)
		_, _ = s.db.ExecContext(ctx, "DELETE FROM modes WHERE subject = ?", subject)
		return versions, nil
	}

	// Soft-delete: get non-deleted versions
	rows, err := s.db.QueryContext(ctx,
		"SELECT version FROM `schemas` WHERE subject = ? AND deleted = FALSE ORDER BY version", subject)
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
			"SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
		if count > 0 {
			return nil, storage.ErrSubjectDeleted
		}
		return nil, storage.ErrSubjectNotFound
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE `schemas` SET deleted = TRUE WHERE subject = ?",
		subject,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to soft-delete schemas: %w", err)
	}

	return versions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM `schemas` WHERE subject = ? AND deleted = FALSE",
		subject,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check subject: %w", err)
	}
	return count > 0, nil
}

// GetConfig retrieves the compatibility configuration for a subject.
func (s *Store) GetConfig(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	config := &storage.ConfigRecord{}
	var alias sql.NullString
	var normalize sql.NullBool
	var compatibilityGroup sql.NullString
	var defaultMetadataBytes, overrideMetadataBytes []byte
	var defaultRuleSetBytes, overrideRuleSetBytes []byte

	err := s.stmts.getConfig.QueryRowContext(ctx, subject).Scan(
		&config.Subject, &config.CompatibilityLevel,
		&alias, &normalize,
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
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
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

	var compatGroupParam interface{}
	if config.CompatibilityGroup != "" {
		compatGroupParam = config.CompatibilityGroup
	}

	_, err = s.stmts.setConfig.ExecContext(ctx, subject, config.CompatibilityLevel,
		aliasParam, normalizeParam,
		defaultMetadataJSON, overrideMetadataJSON,
		defaultRuleSetJSON, overrideRuleSetJSON,
		compatGroupParam)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	result, err := s.stmts.deleteConfig.ExecContext(ctx, subject)
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
func (s *Store) GetGlobalConfig(ctx context.Context) (*storage.ConfigRecord, error) {
	return s.GetConfig(ctx, "")
}

// SetGlobalConfig sets the global compatibility configuration.
func (s *Store) SetGlobalConfig(ctx context.Context, config *storage.ConfigRecord) error {
	return s.SetConfig(ctx, "", config)
}

// GetMode retrieves the mode for a subject.
func (s *Store) GetMode(ctx context.Context, subject string) (*storage.ModeRecord, error) {
	mode := &storage.ModeRecord{}
	err := s.stmts.getMode.QueryRowContext(ctx, subject).Scan(&mode.Subject, &mode.Mode)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mode: %w", err)
	}

	return mode, nil
}

// SetMode sets the mode for a subject.
func (s *Store) SetMode(ctx context.Context, subject string, mode *storage.ModeRecord) error {
	_, err := s.stmts.setMode.ExecContext(ctx, subject, mode.Mode)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, subject string) error {
	result, err := s.stmts.deleteMode.ExecContext(ctx, subject)
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
func (s *Store) GetGlobalMode(ctx context.Context) (*storage.ModeRecord, error) {
	return s.GetMode(ctx, "")
}

// SetGlobalMode sets the global mode.
func (s *Store) SetGlobalMode(ctx context.Context, mode *storage.ModeRecord) error {
	return s.SetMode(ctx, "", mode)
}

// NextID returns the next available schema ID using the id_alloc table.
// Atomically reads the current value and increments it.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.QueryRowContext(ctx, "SELECT next_id FROM id_alloc WHERE name = 'schema_id' FOR UPDATE").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get next ID: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE id_alloc SET next_id = next_id + 1 WHERE name = 'schema_id'")
	if err != nil {
		return 0, fmt.Errorf("failed to increment next ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit next ID: %w", err)
	}

	return id, nil
}

// GetMaxSchemaID returns the highest schema ID currently assigned.
func (s *Store) GetMaxSchemaID(ctx context.Context) (int64, error) {
	var maxID int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM `schemas`").Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max schema ID: %w", err)
	}
	return maxID, nil
}

// ImportSchema inserts a schema with a specified ID (for migration).
// Returns ErrSchemaIDConflict if the ID already exists.
func (s *Store) ImportSchema(ctx context.Context, record *storage.SchemaRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if schema ID already exists
	var existingFingerprint string
	idExists := false
	err = tx.QueryRowContext(ctx, "SELECT fingerprint FROM `schemas` WHERE id = ?", record.ID).Scan(&existingFingerprint)
	if err == nil {
		// ID exists — allow if same content (fingerprint), reject if different
		if existingFingerprint != record.Fingerprint {
			return storage.ErrSchemaIDConflict
		}
		idExists = true
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	// Check if version already exists for this subject
	var existingVersion int
	err = tx.QueryRowContext(ctx,
		"SELECT version FROM `schemas` WHERE subject = ? AND version = ?",
		record.Subject, record.Version,
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

	if idExists {
		// Same content, different subject — insert a new row (auto-id)
		_, err = tx.ExecContext(ctx,
			"INSERT INTO `schemas` (subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			record.Subject, record.Version, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
		)
	} else {
		// New ID — insert with explicit ID
		_, err = tx.ExecContext(ctx,
			"INSERT INTO `schemas` (id, subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			record.ID, record.Subject, record.Version, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	// Insert references
	for _, ref := range record.References {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)",
			record.ID, ref.Name, ref.Subject, ref.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert reference: %w", err)
		}
	}

	record.CreatedAt = time.Now()

	return tx.Commit()
}

// SetNextID sets the ID sequence to start from the given value.
// Used after import to prevent ID conflicts.
func (s *Store) SetNextID(ctx context.Context, id int64) error {
	// Update the id_alloc table
	_, err := s.db.ExecContext(ctx, "UPDATE id_alloc SET next_id = ? WHERE name = 'schema_id'", id)
	if err != nil {
		return fmt.Errorf("failed to set next ID in id_alloc: %w", err)
	}

	// Also update AUTO_INCREMENT so CreateSchema (which uses AUTO_INCREMENT) stays in sync
	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `schemas` AUTO_INCREMENT = %d", id))
	if err != nil {
		return fmt.Errorf("failed to set AUTO_INCREMENT: %w", err)
	}
	return nil
}

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	rows, err := s.stmts.getReferencedBy.QueryContext(ctx, subject, version)
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

// loadReferences loads references for a schema.
func (s *Store) loadReferences(ctx context.Context, schemaID int64) ([]storage.Reference, error) {
	rows, err := s.stmts.loadReferences.QueryContext(ctx, schemaID)
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

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered.
// Uses fingerprint-based lookup for global deduplication: the same content registered
// under different subjects shares the same API-visible schema ID.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	query := `SELECT DISTINCT s.subject FROM ` + "`schemas`" + ` s
		WHERE s.fingerprint = (SELECT fingerprint FROM ` + "`schemas`" + ` WHERE id = ? LIMIT 1)`
	if !includeDeleted {
		query += " AND s.deleted = FALSE"
	}

	rows, err := s.db.QueryContext(ctx, query, id)
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

// GetVersionsBySchemaID returns all subject-version pairs where the given schema ID is registered.
// Uses fingerprint-based lookup for global deduplication.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	query := `SELECT s.subject, s.version FROM ` + "`schemas`" + ` s
		WHERE s.fingerprint = (SELECT fingerprint FROM ` + "`schemas`" + ` WHERE id = ? LIMIT 1)`
	if !includeDeleted {
		query += " AND s.deleted = FALSE"
	}

	rows, err := s.db.QueryContext(ctx, query, id)
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

// ListSchemas returns schemas matching the given filters.
func (s *Store) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM `schemas` WHERE 1=1"
	args := []interface{}{}

	if !params.Deleted {
		query += " AND deleted = ?"
		args = append(args, false)
	}

	if params.SubjectPrefix != "" {
		query += " AND subject LIKE ?"
		args = append(args, params.SubjectPrefix+"%")
	}

	if params.LatestOnly {
		args = []interface{}{}
		query = "SELECT s.id, s.subject, s.version, s.schema_type, s.schema_text, s.fingerprint, s.deleted, s.created_at, s.metadata, s.ruleset FROM `schemas` s INNER JOIN (SELECT subject, MAX(version) as max_version FROM `schemas` WHERE 1=1"
		if !params.Deleted {
			query += " AND deleted = FALSE"
		}
		if params.SubjectPrefix != "" {
			query += " AND subject LIKE ?"
			args = append(args, params.SubjectPrefix+"%")
		}
		query += " GROUP BY subject) latest ON s.subject = latest.subject AND s.version = latest.max_version"
		if !params.Deleted {
			query += " WHERE s.deleted = FALSE"
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
		if globalID, gErr := s.globalSchemaID(ctx, record.Fingerprint); gErr == nil {
			record.ID = globalID
		} else {
			record.ID = rowID
		}

		schemas = append(schemas, record)
	}

	return schemas, nil
}

// DeleteGlobalConfig resets the global config to default.
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO configs (subject, compatibility_level, alias, normalize, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group)
		 VALUES ('', 'BACKWARD', NULL, NULL, NULL, NULL, NULL, NULL, NULL)
		 ON DUPLICATE KEY UPDATE compatibility_level = 'BACKWARD', alias = NULL, normalize = NULL, default_metadata = NULL, override_metadata = NULL, default_ruleset = NULL, override_ruleset = NULL, compatibility_group = NULL`,
	)
	if err != nil {
		return fmt.Errorf("failed to reset global config: %w", err)
	}
	return nil
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
