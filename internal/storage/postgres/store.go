package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	_ "github.com/lib/pq"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Config holds PostgreSQL connection configuration.
type Config struct {
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	Database        string        `json:"database" yaml:"database"`
	Username        string        `json:"username" yaml:"username"`
	Password        string        `json:"password" yaml:"password"`
	SSLMode         string        `json:"ssl_mode" yaml:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            5432,
		Database:        "schema_registry",
		Username:        "postgres",
		Password:        "",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// DSN returns the connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.Username, c.Password, c.SSLMode,
	)
}

// Store implements the storage.Storage interface using PostgreSQL.
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
	getSubjectsBySchemaID  *sql.Stmt
	getVersionsBySchemaID  *sql.Stmt
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

// NewStore creates a new PostgreSQL store.
func NewStore(config Config) (*Store, error) {
	db, err := sql.Open("postgres", config.DSN())
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
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		 FROM schemas WHERE registry_ctx = $1 AND id = $2`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaByID: %w", err)
	}

	stmts.getSchemaBySubjectVer, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		 FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND version = $3`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaBySubjectVer: %w", err)
	}

	stmts.getSchemaByFingerprint, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		 FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND fingerprint = $3 AND deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaByFingerprint: %w", err)
	}

	stmts.getLatestSchema, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		 FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND deleted = FALSE
		 ORDER BY version DESC LIMIT 1`)
	if err != nil {
		return fmt.Errorf("prepare getLatestSchema: %w", err)
	}

	stmts.softDeleteSchema, err = s.db.Prepare(
		`UPDATE schemas SET deleted = TRUE WHERE registry_ctx = $1 AND subject = $2 AND version = $3`)
	if err != nil {
		return fmt.Errorf("prepare softDeleteSchema: %w", err)
	}

	stmts.hardDeleteSchema, err = s.db.Prepare(
		`DELETE FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND version = $3`)
	if err != nil {
		return fmt.Errorf("prepare hardDeleteSchema: %w", err)
	}

	stmts.countSchemasBySubject, err = s.db.Prepare(
		`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare countSchemasBySubject: %w", err)
	}

	stmts.loadReferences, err = s.db.Prepare(
		`SELECT name, ref_subject, ref_version FROM schema_references WHERE registry_ctx = $1 AND schema_id = $2`)
	if err != nil {
		return fmt.Errorf("prepare loadReferences: %w", err)
	}

	stmts.getSubjectsBySchemaID, err = s.db.Prepare(
		`SELECT DISTINCT s.subject FROM schemas s
		 JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint
		 WHERE s.registry_ctx = $1 AND fp.schema_id = $2 AND s.deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getSubjectsBySchemaID: %w", err)
	}

	stmts.getVersionsBySchemaID, err = s.db.Prepare(
		`SELECT s.subject, s.version FROM schemas s
		 JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint
		 WHERE s.registry_ctx = $1 AND fp.schema_id = $2 AND s.deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getVersionsBySchemaID: %w", err)
	}

	stmts.getReferencedBy, err = s.db.Prepare(
		`SELECT s.subject, s.version
		 FROM schemas s
		 JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint
		 JOIN schema_references r ON r.registry_ctx = fp.registry_ctx AND r.schema_id = fp.schema_id
		 WHERE s.registry_ctx = $1 AND r.ref_subject = $2 AND r.ref_version = $3 AND s.deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getReferencedBy: %w", err)
	}

	// Config statements — scoped by registry_ctx
	stmts.getConfig, err = s.db.Prepare(
		`SELECT subject, compatibility_level, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group
		 FROM configs WHERE registry_ctx = $1 AND subject = $2`)
	if err != nil {
		return fmt.Errorf("prepare getConfig: %w", err)
	}

	stmts.setConfig, err = s.db.Prepare(
		`INSERT INTO configs (registry_ctx, subject, compatibility_level, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (registry_ctx, subject) DO UPDATE SET
		     compatibility_level = EXCLUDED.compatibility_level,
		     alias = EXCLUDED.alias,
		     normalize = EXCLUDED.normalize,
		     validate_fields = EXCLUDED.validate_fields,
		     default_metadata = EXCLUDED.default_metadata,
		     override_metadata = EXCLUDED.override_metadata,
		     default_ruleset = EXCLUDED.default_ruleset,
		     override_ruleset = EXCLUDED.override_ruleset,
		     compatibility_group = EXCLUDED.compatibility_group`)
	if err != nil {
		return fmt.Errorf("prepare setConfig: %w", err)
	}

	stmts.deleteConfig, err = s.db.Prepare(
		`DELETE FROM configs WHERE registry_ctx = $1 AND subject = $2`)
	if err != nil {
		return fmt.Errorf("prepare deleteConfig: %w", err)
	}

	// Mode statements — scoped by registry_ctx
	stmts.getMode, err = s.db.Prepare(
		`SELECT subject, mode FROM modes WHERE registry_ctx = $1 AND subject = $2`)
	if err != nil {
		return fmt.Errorf("prepare getMode: %w", err)
	}

	stmts.setMode, err = s.db.Prepare(
		`INSERT INTO modes (registry_ctx, subject, mode) VALUES ($1, $2, $3)
		 ON CONFLICT (registry_ctx, subject) DO UPDATE SET mode = EXCLUDED.mode`)
	if err != nil {
		return fmt.Errorf("prepare setMode: %w", err)
	}

	stmts.deleteMode, err = s.db.Prepare(
		`DELETE FROM modes WHERE registry_ctx = $1 AND subject = $2`)
	if err != nil {
		return fmt.Errorf("prepare deleteMode: %w", err)
	}

	// User statements
	stmts.createUser, err = s.db.Prepare(
		`INSERT INTO users (username, email, password_hash, role, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare createUser: %w", err)
	}

	stmts.getUserByID, err = s.db.Prepare(
		`SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		 FROM users WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare getUserByID: %w", err)
	}

	stmts.getUserByUsername, err = s.db.Prepare(
		`SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		 FROM users WHERE username = $1`)
	if err != nil {
		return fmt.Errorf("prepare getUserByUsername: %w", err)
	}

	stmts.updateUser, err = s.db.Prepare(
		`UPDATE users SET username = $1, email = $2, password_hash = $3, role = $4,
		 enabled = $5, updated_at = $6 WHERE id = $7`)
	if err != nil {
		return fmt.Errorf("prepare updateUser: %w", err)
	}

	stmts.deleteUser, err = s.db.Prepare(
		`DELETE FROM users WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare deleteUser: %w", err)
	}

	stmts.listUsers, err = s.db.Prepare(
		`SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		 FROM users ORDER BY username`)
	if err != nil {
		return fmt.Errorf("prepare listUsers: %w", err)
	}

	// API Key statements
	stmts.createAPIKey, err = s.db.Prepare(
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare createAPIKey: %w", err)
	}

	stmts.getAPIKeyByID, err = s.db.Prepare(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByID: %w", err)
	}

	stmts.getAPIKeyByHash, err = s.db.Prepare(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys WHERE key_hash = $1`)
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByHash: %w", err)
	}

	stmts.updateAPIKey, err = s.db.Prepare(
		`UPDATE api_keys SET user_id = $1, key_hash = $2, name = $3, role = $4, enabled = $5, expires_at = $6
		 WHERE id = $7`)
	if err != nil {
		return fmt.Errorf("prepare updateAPIKey: %w", err)
	}

	stmts.deleteAPIKey, err = s.db.Prepare(
		`DELETE FROM api_keys WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare deleteAPIKey: %w", err)
	}

	stmts.listAPIKeys, err = s.db.Prepare(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return fmt.Errorf("prepare listAPIKeys: %w", err)
	}

	stmts.listAPIKeysByUserID, err = s.db.Prepare(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`)
	if err != nil {
		return fmt.Errorf("prepare listAPIKeysByUserID: %w", err)
	}

	stmts.getAPIKeyByUserAndName, err = s.db.Prepare(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys WHERE user_id = $1 AND name = $2`)
	if err != nil {
		return fmt.Errorf("prepare getAPIKeyByUserAndName: %w", err)
	}

	stmts.updateAPIKeyLastUsed, err = s.db.Prepare(
		`UPDATE api_keys SET last_used = $1 WHERE id = $2`)
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

	// Close all statements (ignore errors on close)
	stmts := []*sql.Stmt{
		s.stmts.getSchemaByID, s.stmts.getSchemaBySubjectVer, s.stmts.getSchemaByFingerprint,
		s.stmts.getLatestSchema, s.stmts.softDeleteSchema, s.stmts.hardDeleteSchema,
		s.stmts.countSchemasBySubject, s.stmts.loadReferences, s.stmts.getSubjectsBySchemaID,
		s.stmts.getVersionsBySchemaID, s.stmts.getReferencedBy,
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
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	// Add fingerprint-only index for global dedup (ignore error if already exists)
	_, _ = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_schemas_fingerprint_global ON schemas(fingerprint)`)
	return nil
}

// globalSchemaID returns the stable per-context schema ID for a fingerprint.
// Looks up the immutable (registry_ctx, fingerprint) → schema_id mapping in schema_fingerprints.
func (s *Store) globalSchemaID(ctx context.Context, registryCtx, fingerprint string) (int64, error) {
	var globalID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = $1 AND fingerprint = $2`, registryCtx, fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// globalSchemaIDTx returns the stable per-context schema ID within a transaction.
func (s *Store) globalSchemaIDTx(ctx context.Context, tx *sql.Tx, registryCtx, fingerprint string) (int64, error) {
	var globalID int64
	err := tx.QueryRowContext(ctx,
		`SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = $1 AND fingerprint = $2`, registryCtx, fingerprint).Scan(&globalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get global schema ID: %w", err)
	}
	return globalID, nil
}

// ensureContext ensures a context exists in the contexts tracking table.
func (s *Store) ensureContext(ctx context.Context, registryCtx string) {
	_, _ = s.db.ExecContext(ctx,
		`INSERT INTO contexts (registry_ctx) VALUES ($1) ON CONFLICT DO NOTHING`, registryCtx)
}

// CreateSchema stores a new schema record.
// This implementation handles concurrent insertions by retrying on conflicts.
// PostgreSQL serialization errors are handled as retriable errors.
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
		// On unique violation or serialization error, retry with exponential backoff + jitter
		if isUniqueViolation(err) || isSerializationError(err) {
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
func (s *Store) createSchemaAttempt(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check for existing schema with same fingerprint (idempotent check)
	var existingVersion int
	var existingDeleted bool
	err = tx.QueryRowContext(ctx,
		`SELECT version, deleted FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND fingerprint = $3`,
		registryCtx, record.Subject, record.Fingerprint,
	).Scan(&existingVersion, &existingDeleted)

	if err == nil && !existingDeleted {
		// Schema already exists in this subject - resolve per-context ID
		globalID, gErr := s.globalSchemaIDTx(ctx, tx, registryCtx, record.Fingerprint)
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
		`SELECT COALESCE(MAX(version), 0) + 1 FROM schemas WHERE registry_ctx = $1 AND subject = $2`,
		registryCtx, record.Subject,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Serialize metadata and ruleset to JSON
	metadataJSON, err := marshalJSONNullable(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	rulesetJSON, err := marshalJSONNullable(record.RuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal ruleset: %w", err)
	}

	// If a soft-deleted row with the same fingerprint exists, remove it first
	// to avoid violating the unique constraint on (registry_ctx, subject, fingerprint).
	if existingDeleted {
		_, _ = tx.ExecContext(ctx,
			`DELETE FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND fingerprint = $3`,
			registryCtx, record.Subject, record.Fingerprint,
		)
	}

	// Insert schema - unique constraint on (registry_ctx, subject, version) prevents duplicates
	var newRowID int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO schemas (registry_ctx, subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		registryCtx, record.Subject, nextVersion, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
	).Scan(&newRowID)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	// Claim fingerprint in schema_fingerprints (first writer wins, per-context)
	_, _ = tx.ExecContext(ctx,
		`INSERT INTO schema_fingerprints (registry_ctx, fingerprint, schema_id) VALUES ($1, $2, $3) ON CONFLICT (registry_ctx, fingerprint) DO NOTHING`,
		registryCtx, record.Fingerprint, newRowID,
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
			`SELECT COUNT(*) FROM schema_references WHERE registry_ctx = $1 AND schema_id = $2`, registryCtx, globalID).Scan(&refCount)
		if refCount == 0 {
			for _, ref := range record.References {
				_, err = tx.ExecContext(ctx,
					`INSERT INTO schema_references (registry_ctx, schema_id, name, ref_subject, ref_version)
					 VALUES ($1, $2, $3, $4, $5)`,
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
// First tries direct row lookup, then falls back to schema_fingerprints
// for cases where the original row was permanently deleted but the content
// still exists under other subjects.
func (s *Store) GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string
	var metadataJSON, rulesetJSON []byte

	// Look up the per-context schema ID via schema_fingerprints first.
	var fingerprint string
	fpErr := s.db.QueryRowContext(ctx,
		`SELECT fingerprint FROM schema_fingerprints WHERE registry_ctx = $1 AND schema_id = $2`, registryCtx, id).Scan(&fingerprint)
	if fpErr != nil {
		return nil, storage.ErrSchemaNotFound
	}

	// Find the schema row by context + fingerprint
	err := s.db.QueryRowContext(ctx,
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		 FROM schemas WHERE registry_ctx = $1 AND fingerprint = $2 AND deleted = FALSE ORDER BY id LIMIT 1`,
		registryCtx, fingerprint).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataJSON, &rulesetJSON)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Use the per-context schema ID, not the row's auto-generated ID
	record.ID = id

	record.SchemaType = storage.SchemaType(schemaType)

	record.Metadata, err = unmarshalMetadata(metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	record.RuleSet, err = unmarshalRuleSet(rulesetJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ruleset: %w", err)
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
	var metadataJSON, rulesetJSON []byte

	err := s.stmts.getSchemaBySubjectVer.QueryRowContext(ctx, registryCtx, subject, version).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataJSON, &rulesetJSON)

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

	record.Metadata, err = unmarshalMetadata(metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	record.RuleSet, err = unmarshalRuleSet(rulesetJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ruleset: %w", err)
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
	query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		      FROM schemas WHERE registry_ctx = $1 AND subject = $2`
	if !includeDeleted {
		query += ` AND deleted = FALSE`
	}
	query += ` ORDER BY version`

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
		var metadataJSON, rulesetJSON []byte
		if err := rows.Scan(&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataJSON, &rulesetJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)

		record.Metadata, _ = unmarshalMetadata(metadataJSON)
		record.RuleSet, _ = unmarshalRuleSet(rulesetJSON)

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
			`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject).Scan(&count)
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
	var metadataJSON, rulesetJSON []byte
	var err error

	if includeDeleted {
		query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
		          FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND fingerprint = $3`
		err = s.db.QueryRowContext(ctx, query, registryCtx, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataJSON, &rulesetJSON)
	} else {
		err = s.stmts.getSchemaByFingerprint.QueryRowContext(ctx, registryCtx, subject, fingerprint).Scan(
			&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataJSON, &rulesetJSON)
	}

	if err == sql.ErrNoRows {
		// Check if subject exists at all
		var count int
		_ = s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject).Scan(&count)
		if count == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	record.Metadata, _ = unmarshalMetadata(metadataJSON)
	record.RuleSet, _ = unmarshalRuleSet(rulesetJSON)

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
	var metadataJSON, rulesetJSON []byte

	// Query for any schema with this fingerprint within this context
	query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset
	          FROM schemas WHERE registry_ctx = $1 AND fingerprint = $2 AND deleted = false LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, registryCtx, fingerprint).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataJSON, &rulesetJSON)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema by global fingerprint: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	record.Metadata, _ = unmarshalMetadata(metadataJSON)
	record.RuleSet, _ = unmarshalRuleSet(rulesetJSON)

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
	var metadataJSON, rulesetJSON []byte

	err := s.stmts.getLatestSchema.QueryRowContext(ctx, registryCtx, subject).Scan(
		&rowID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
		&metadataJSON, &rulesetJSON)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSubjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	record.Metadata, _ = unmarshalMetadata(metadataJSON)
	record.RuleSet, _ = unmarshalRuleSet(rulesetJSON)

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
			`SELECT deleted, fingerprint FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND version = $3`,
			registryCtx, subject, version).Scan(&deleted, &fingerprint)
		if err == sql.ErrNoRows {
			// Check if subject exists at all
			var count int
			_ = s.db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject).Scan(&count)
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
		// if no other schemas rows share this fingerprint in this context.
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
	query := `SELECT DISTINCT subject FROM schemas WHERE registry_ctx = $1`
	if !includeDeleted {
		query += ` AND deleted = FALSE`
	}
	query += ` ORDER BY subject`

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
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN deleted THEN 1 ELSE 0 END), 0)
			 FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject).Scan(&totalCount, &deletedCount)
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
			`SELECT version, fingerprint FROM schemas WHERE registry_ctx = $1 AND subject = $2 ORDER BY version`, registryCtx, subject)
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

		_, err = s.db.ExecContext(ctx, `DELETE FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject)
		if err != nil {
			return nil, fmt.Errorf("failed to delete schemas: %w", err)
		}
		_, _ = s.db.ExecContext(ctx, `DELETE FROM configs WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM modes WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject)

		// Clean up orphaned schema_fingerprints and schema_references
		for fp := range fingerprintSet {
			s.cleanupOrphanedFingerprint(ctx, registryCtx, fp)
		}

		return versions, nil
	}

	// Soft-delete: get non-deleted versions
	rows, err := s.db.QueryContext(ctx,
		`SELECT version FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND deleted = FALSE ORDER BY version`, registryCtx, subject)
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
			`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND subject = $2`, registryCtx, subject).Scan(&count)
		if count > 0 {
			return nil, storage.ErrSubjectDeleted
		}
		return nil, storage.ErrSubjectNotFound
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE schemas SET deleted = TRUE WHERE registry_ctx = $1 AND subject = $2`,
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
	var defaultMetadataJSON, overrideMetadataJSON, defaultRulesetJSON, overrideRulesetJSON []byte

	err := s.stmts.getConfig.QueryRowContext(ctx, registryCtx, subject).Scan(
		&config.Subject, &config.CompatibilityLevel,
		&alias, &normalize, &validateFields,
		&defaultMetadataJSON, &overrideMetadataJSON,
		&defaultRulesetJSON, &overrideRulesetJSON,
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

	config.DefaultMetadata, _ = unmarshalMetadata(defaultMetadataJSON)
	config.OverrideMetadata, _ = unmarshalMetadata(overrideMetadataJSON)
	config.DefaultRuleSet, _ = unmarshalRuleSet(defaultRulesetJSON)
	config.OverrideRuleSet, _ = unmarshalRuleSet(overrideRulesetJSON)

	return config, nil
}

// SetConfig sets the compatibility configuration for a subject.
func (s *Store) SetConfig(ctx context.Context, registryCtx string, subject string, config *storage.ConfigRecord) error {
	// Serialize nullable fields
	var aliasVal *string
	if config.Alias != "" {
		aliasVal = &config.Alias
	}

	var compatGroupVal *string
	if config.CompatibilityGroup != "" {
		compatGroupVal = &config.CompatibilityGroup
	}

	defaultMetadataJSON, err := marshalJSONNullable(config.DefaultMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal default metadata: %w", err)
	}
	overrideMetadataJSON, err := marshalJSONNullable(config.OverrideMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal override metadata: %w", err)
	}
	defaultRulesetJSON, err := marshalJSONNullable(config.DefaultRuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal default ruleset: %w", err)
	}
	overrideRulesetJSON, err := marshalJSONNullable(config.OverrideRuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal override ruleset: %w", err)
	}

	_, err = s.stmts.setConfig.ExecContext(ctx, registryCtx, subject, config.CompatibilityLevel,
		aliasVal, config.Normalize, config.ValidateFields,
		defaultMetadataJSON, overrideMetadataJSON,
		defaultRulesetJSON, overrideRulesetJSON,
		compatGroupVal)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, registryCtx string, subject string) error {
	result, err := s.stmts.deleteConfig.ExecContext(ctx, registryCtx, subject)
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
	err := s.stmts.getMode.QueryRowContext(ctx, registryCtx, subject).Scan(&mode.Subject, &mode.Mode)

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
	_, err := s.stmts.setMode.ExecContext(ctx, registryCtx, subject, mode.Mode)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, registryCtx string, subject string) error {
	result, err := s.stmts.deleteMode.ExecContext(ctx, registryCtx, subject)
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
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO ctx_id_alloc (registry_ctx, next_id)
		 VALUES ($1, 2)
		 ON CONFLICT (registry_ctx) DO UPDATE SET next_id = ctx_id_alloc.next_id + 1
		 RETURNING next_id - 1`, registryCtx).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get next ID: %w", err)
	}
	return id, nil
}

// GetMaxSchemaID returns the highest per-context schema ID currently assigned.
func (s *Store) GetMaxSchemaID(ctx context.Context, registryCtx string) (int64, error) {
	var maxID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(schema_id), 0) FROM schema_fingerprints WHERE registry_ctx = $1`,
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
		`SELECT fingerprint FROM schema_fingerprints WHERE registry_ctx = $1 AND schema_id = $2`,
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
		`SELECT version FROM schemas WHERE registry_ctx = $1 AND subject = $2 AND version = $3`,
		registryCtx, record.Subject, record.Version,
	).Scan(&existingVersion)
	if err == nil {
		return storage.ErrSchemaExists
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing version: %w", err)
	}

	// Serialize metadata and ruleset to JSON
	metadataJSON, err := marshalJSONNullable(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	rulesetJSON, err := marshalJSONNullable(record.RuleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal ruleset: %w", err)
	}

	// Insert schema row (always auto-id for the row; the per-context ID is in schema_fingerprints)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schemas (registry_ctx, subject, version, schema_type, schema_text, fingerprint, created_at, metadata, ruleset)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		registryCtx, record.Subject, record.Version, record.SchemaType, record.Schema, record.Fingerprint, time.Now(), metadataJSON, rulesetJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	if !idExists {
		// Claim per-context fingerprint → schema_id mapping with the imported ID
		_, err = tx.ExecContext(ctx,
			`INSERT INTO schema_fingerprints (registry_ctx, fingerprint, schema_id) VALUES ($1, $2, $3) ON CONFLICT (registry_ctx, fingerprint) DO NOTHING`,
			registryCtx, record.Fingerprint, record.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert fingerprint mapping: %w", err)
		}

		// Advance ctx_id_alloc past the imported ID if needed
		_, _ = tx.ExecContext(ctx,
			`INSERT INTO ctx_id_alloc (registry_ctx, next_id)
			 VALUES ($1, $2)
			 ON CONFLICT (registry_ctx) DO UPDATE SET next_id = GREATEST(ctx_id_alloc.next_id, $2)`,
			registryCtx, record.ID+1,
		)
	}

	// Insert references using the per-context ID
	if len(record.References) > 0 {
		var refCount int
		_ = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_references WHERE registry_ctx = $1 AND schema_id = $2`, registryCtx, record.ID).Scan(&refCount)
		if refCount == 0 {
			for _, ref := range record.References {
				_, err = tx.ExecContext(ctx,
					`INSERT INTO schema_references (registry_ctx, schema_id, name, ref_subject, ref_version)
					 VALUES ($1, $2, $3, $4, $5)`,
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
		`INSERT INTO ctx_id_alloc (registry_ctx, next_id) VALUES ($1, $2)
		 ON CONFLICT (registry_ctx) DO UPDATE SET next_id = $2`,
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
		`SELECT COUNT(*) FROM schemas WHERE registry_ctx = $1 AND fingerprint = $2`, registryCtx, fingerprint).Scan(&remaining)
	if err != nil || remaining > 0 {
		return
	}
	// No schemas rows left in this context — clean up the stable ID mapping and references
	var schemaID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT schema_id FROM schema_fingerprints WHERE registry_ctx = $1 AND fingerprint = $2`, registryCtx, fingerprint).Scan(&schemaID); err == nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM schema_references WHERE registry_ctx = $1 AND schema_id = $2`, registryCtx, schemaID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM schema_fingerprints WHERE registry_ctx = $1 AND fingerprint = $2`, registryCtx, fingerprint)
	}
}

// loadReferences loads references for a schema within a context.
func (s *Store) loadReferences(ctx context.Context, registryCtx string, schemaID int64) ([]storage.Reference, error) {
	rows, err := s.stmts.loadReferences.QueryContext(ctx, registryCtx, schemaID)
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
	rows, err := s.stmts.getSubjectsBySchemaID.QueryContext(ctx, registryCtx, id)
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

	if includeDeleted && len(subjects) == 0 {
		// Try including deleted schemas
		query := `SELECT DISTINCT s.subject FROM schemas s
			JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint
			WHERE s.registry_ctx = $1 AND fp.schema_id = $2`
		rows2, err := s.db.QueryContext(ctx, query, registryCtx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to query subjects: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			var subject string
			if err := rows2.Scan(&subject); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}
			subjects = append(subjects, subject)
		}
	}

	if len(subjects) == 0 {
		return nil, storage.ErrSchemaNotFound
	}

	return subjects, nil
}

// GetVersionsBySchemaID returns all subject-version pairs where the given per-context schema ID is registered.
// Uses fingerprint-based lookup via schema_fingerprints for global deduplication.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	rows, err := s.stmts.getVersionsBySchemaID.QueryContext(ctx, registryCtx, id)
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

	if includeDeleted && len(versions) == 0 {
		// Try including deleted schemas
		query := `SELECT s.subject, s.version FROM schemas s
			JOIN schema_fingerprints fp ON fp.registry_ctx = s.registry_ctx AND fp.fingerprint = s.fingerprint
			WHERE s.registry_ctx = $1 AND fp.schema_id = $2`
		rows2, err := s.db.QueryContext(ctx, query, registryCtx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to query versions: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			var sv storage.SubjectVersion
			if err := rows2.Scan(&sv.Subject, &sv.Version); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}
			versions = append(versions, sv)
		}
	}

	if len(versions) == 0 {
		return nil, storage.ErrSchemaNotFound
	}

	return versions, nil
}

// ListSchemas returns schemas matching the given filters, scoped to a context.
func (s *Store) ListSchemas(ctx context.Context, registryCtx string, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	// Always start with registry_ctx filter
	query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at, metadata, ruleset FROM schemas WHERE registry_ctx = $1`
	args := []interface{}{registryCtx}
	argNum := 2

	if !params.Deleted {
		query += fmt.Sprintf(` AND deleted = $%d`, argNum)
		args = append(args, false)
		argNum++
	}

	if params.SubjectPrefix != "" {
		query += fmt.Sprintf(` AND subject LIKE $%d`, argNum)
		args = append(args, params.SubjectPrefix+"%")
		argNum++
	}

	if params.LatestOnly {
		args = []interface{}{registryCtx}
		argNum = 2
		query = `SELECT s.id, s.subject, s.version, s.schema_type, s.schema_text, s.fingerprint, s.deleted, s.created_at, s.metadata, s.ruleset
		         FROM schemas s
		         INNER JOIN (
		             SELECT subject, MAX(version) as max_version
		             FROM schemas
		             WHERE registry_ctx = $1`
		if !params.Deleted {
			query += ` AND deleted = FALSE`
		}
		if params.SubjectPrefix != "" {
			query += fmt.Sprintf(` AND subject LIKE $%d`, argNum)
			args = append(args, params.SubjectPrefix+"%")
			argNum++
		}
		query += ` GROUP BY subject
		         ) latest ON s.subject = latest.subject AND s.version = latest.max_version`
		query += ` WHERE s.registry_ctx = $1`
		if !params.Deleted {
			query += ` AND s.deleted = FALSE`
		}
	}

	query += ` ORDER BY id`

	if params.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argNum)
		args = append(args, params.Limit)
		argNum++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argNum)
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
		var metadataJSON, rulesetJSON []byte
		if err := rows.Scan(&rowID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
			&metadataJSON, &rulesetJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)

		record.Metadata, _ = unmarshalMetadata(metadataJSON)
		record.RuleSet, _ = unmarshalRuleSet(rulesetJSON)

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

// DeleteGlobalConfig resets the global config to default within a context.
func (s *Store) DeleteGlobalConfig(ctx context.Context, registryCtx string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO configs (registry_ctx, subject, compatibility_level, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group)
		 VALUES ($1, '', 'BACKWARD', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
		 ON CONFLICT (registry_ctx, subject) DO UPDATE SET
		     compatibility_level = 'BACKWARD',
		     alias = NULL,
		     normalize = NULL,
		     validate_fields = NULL,
		     default_metadata = NULL,
		     override_metadata = NULL,
		     default_ruleset = NULL,
		     override_ruleset = NULL,
		     compatibility_group = NULL`,
		registryCtx,
	)
	if err != nil {
		return fmt.Errorf("failed to reset global config: %w", err)
	}
	return nil
}

// ListContexts returns all registry contexts from the database.
func (s *Store) ListContexts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT registry_ctx FROM contexts ORDER BY registry_ctx`)
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

	err := s.stmts.createUser.QueryRowContext(ctx,
		user.Username, sql.NullString{String: user.Email, Valid: user.Email != ""},
		user.PasswordHash, user.Role, user.Enabled, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return storage.ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

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

	result, err := s.stmts.updateUser.ExecContext(ctx,
		user.Username, sql.NullString{String: user.Email, Valid: user.Email != ""},
		user.PasswordHash, user.Role, user.Enabled, user.UpdatedAt, user.ID,
	)

	if err != nil {
		if isUniqueViolation(err) {
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

	err := s.stmts.createAPIKey.QueryRowContext(ctx,
		key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.Role, key.Enabled, key.CreatedAt, key.ExpiresAt,
	).Scan(&key.ID)

	if err != nil {
		if isUniqueViolation(err) {
			return storage.ErrAPIKeyExists
		}
		return fmt.Errorf("failed to create API key: %w", err)
	}

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
		key.UserID, key.KeyHash, key.Name, key.Role, key.Enabled, key.ExpiresAt, key.ID,
	)

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

// marshalJSONNullable marshals a value to JSON, returning nil if the input is nil.
// It handles both untyped nil and typed nil pointers (e.g., (*Metadata)(nil)).
// Returns *string so the pq driver sends it as text (not bytea) for JSONB columns.
func marshalJSONNullable(v interface{}) (*string, error) {
	if v == nil {
		return nil, nil
	}
	// Handle typed nil pointers (e.g., *storage.Metadata(nil) passed as interface{})
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(data)
	return &s, nil
}

// unmarshalMetadata unmarshals a nullable JSON byte slice into a *storage.Metadata.
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

// unmarshalRuleSet unmarshals a nullable JSON byte slice into a *storage.RuleSet.
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

// isUniqueViolation checks if the error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL error code for unique_violation is 23505
	return err.Error() != "" && (contains(err.Error(), "duplicate key") || contains(err.Error(), "23505"))
}

// isSerializationError checks if the error is a PostgreSQL serialization error.
func isSerializationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL error code 40001 is for serialization_failure
	// PostgreSQL error code 40P01 is for deadlock_detected
	return contains(errStr, "40001") || contains(errStr, "40P01") || contains(errStr, "deadlock")
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close closes all prepared statements and the database connection.
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
