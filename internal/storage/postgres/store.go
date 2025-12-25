package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	getSchemaByID            *sql.Stmt
	getSchemaBySubjectVer    *sql.Stmt
	getSchemaByFingerprint   *sql.Stmt
	getLatestSchema          *sql.Stmt
	softDeleteSchema         *sql.Stmt
	hardDeleteSchema         *sql.Stmt
	countSchemasBySubject    *sql.Stmt
	loadReferences           *sql.Stmt
	getSubjectsBySchemaID    *sql.Stmt
	getVersionsBySchemaID    *sql.Stmt
	getReferencedBy          *sql.Stmt

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
	createAPIKey          *sql.Stmt
	getAPIKeyByID         *sql.Stmt
	getAPIKeyByHash       *sql.Stmt
	updateAPIKey          *sql.Stmt
	deleteAPIKey          *sql.Stmt
	listAPIKeys           *sql.Stmt
	listAPIKeysByUserID   *sql.Stmt
	getAPIKeyByUserAndName *sql.Stmt
	updateAPIKeyLastUsed  *sql.Stmt
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

	// Schema statements
	stmts.getSchemaByID, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at
		 FROM schemas WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaByID: %w", err)
	}

	stmts.getSchemaBySubjectVer, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at
		 FROM schemas WHERE subject = $1 AND version = $2`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaBySubjectVer: %w", err)
	}

	stmts.getSchemaByFingerprint, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at
		 FROM schemas WHERE subject = $1 AND fingerprint = $2 AND deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getSchemaByFingerprint: %w", err)
	}

	stmts.getLatestSchema, err = s.db.Prepare(
		`SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at
		 FROM schemas WHERE subject = $1 AND deleted = FALSE
		 ORDER BY version DESC LIMIT 1`)
	if err != nil {
		return fmt.Errorf("prepare getLatestSchema: %w", err)
	}

	stmts.softDeleteSchema, err = s.db.Prepare(
		`UPDATE schemas SET deleted = TRUE WHERE subject = $1 AND version = $2`)
	if err != nil {
		return fmt.Errorf("prepare softDeleteSchema: %w", err)
	}

	stmts.hardDeleteSchema, err = s.db.Prepare(
		`DELETE FROM schemas WHERE subject = $1 AND version = $2`)
	if err != nil {
		return fmt.Errorf("prepare hardDeleteSchema: %w", err)
	}

	stmts.countSchemasBySubject, err = s.db.Prepare(
		`SELECT COUNT(*) FROM schemas WHERE subject = $1`)
	if err != nil {
		return fmt.Errorf("prepare countSchemasBySubject: %w", err)
	}

	stmts.loadReferences, err = s.db.Prepare(
		`SELECT name, ref_subject, ref_version FROM schema_references WHERE schema_id = $1`)
	if err != nil {
		return fmt.Errorf("prepare loadReferences: %w", err)
	}

	stmts.getSubjectsBySchemaID, err = s.db.Prepare(
		`SELECT DISTINCT subject FROM schemas WHERE id = $1 AND deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getSubjectsBySchemaID: %w", err)
	}

	stmts.getVersionsBySchemaID, err = s.db.Prepare(
		`SELECT subject, version FROM schemas WHERE id = $1 AND deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getVersionsBySchemaID: %w", err)
	}

	stmts.getReferencedBy, err = s.db.Prepare(
		`SELECT s.subject, s.version
		 FROM schemas s
		 JOIN schema_references r ON r.schema_id = s.id
		 WHERE r.ref_subject = $1 AND r.ref_version = $2 AND s.deleted = FALSE`)
	if err != nil {
		return fmt.Errorf("prepare getReferencedBy: %w", err)
	}

	// Config statements
	stmts.getConfig, err = s.db.Prepare(
		`SELECT subject, compatibility_level FROM configs WHERE subject = $1`)
	if err != nil {
		return fmt.Errorf("prepare getConfig: %w", err)
	}

	stmts.setConfig, err = s.db.Prepare(
		`INSERT INTO configs (subject, compatibility_level) VALUES ($1, $2)
		 ON CONFLICT (subject) DO UPDATE SET compatibility_level = EXCLUDED.compatibility_level`)
	if err != nil {
		return fmt.Errorf("prepare setConfig: %w", err)
	}

	stmts.deleteConfig, err = s.db.Prepare(
		`DELETE FROM configs WHERE subject = $1`)
	if err != nil {
		return fmt.Errorf("prepare deleteConfig: %w", err)
	}

	// Mode statements
	stmts.getMode, err = s.db.Prepare(
		`SELECT subject, mode FROM modes WHERE subject = $1`)
	if err != nil {
		return fmt.Errorf("prepare getMode: %w", err)
	}

	stmts.setMode, err = s.db.Prepare(
		`INSERT INTO modes (subject, mode) VALUES ($1, $2)
		 ON CONFLICT (subject) DO UPDATE SET mode = EXCLUDED.mode`)
	if err != nil {
		return fmt.Errorf("prepare setMode: %w", err)
	}

	stmts.deleteMode, err = s.db.Prepare(
		`DELETE FROM modes WHERE subject = $1`)
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
		`UPDATE api_keys SET user_id = $1, name = $2, role = $3, enabled = $4, expires_at = $5
		 WHERE id = $6`)
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
	return nil
}

// CreateSchema stores a new schema record.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check for existing schema with same fingerprint
	var existingID int64
	var existingVersion int
	var existingDeleted bool
	err = tx.QueryRowContext(ctx,
		`SELECT id, version, deleted FROM schemas WHERE subject = $1 AND fingerprint = $2`,
		record.Subject, record.Fingerprint,
	).Scan(&existingID, &existingVersion, &existingDeleted)

	if err == nil && !existingDeleted {
		record.ID = existingID
		record.Version = existingVersion
		return storage.ErrSchemaExists
	}

	// Get next version for this subject
	var nextVersion int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) + 1 FROM schemas WHERE subject = $1`,
		record.Subject,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Insert schema
	err = tx.QueryRowContext(ctx,
		`INSERT INTO schemas (subject, version, schema_type, schema_text, fingerprint, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		record.Subject, nextVersion, record.SchemaType, record.Schema, record.Fingerprint, time.Now(),
	).Scan(&record.ID)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	record.Version = nextVersion
	record.CreatedAt = time.Now()

	// Insert references
	for _, ref := range record.References {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO schema_references (schema_id, name, ref_subject, ref_version)
			 VALUES ($1, $2, $3, $4)`,
			record.ID, ref.Name, ref.Subject, ref.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert reference: %w", err)
		}
	}

	return tx.Commit()
}

// GetSchemaByID retrieves a schema by its global ID.
func (s *Store) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string

	err := s.stmts.getSchemaByID.QueryRowContext(ctx, id).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

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

	err := s.stmts.getSchemaBySubjectVer.QueryRowContext(ctx, subject, version).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt)

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

	// Load references
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at
		      FROM schemas WHERE subject = $1`
	if !includeDeleted {
		query += ` AND deleted = FALSE`
	}
	query += ` ORDER BY version`

	rows, err := s.db.QueryContext(ctx, query, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*storage.SchemaRecord
	for rows.Next() {
		record := &storage.SchemaRecord{}
		var schemaType string
		if err := rows.Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)

		// Load references
		refs, err := s.loadReferences(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		record.References = refs

		schemas = append(schemas, record)
	}

	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string) (*storage.SchemaRecord, error) {
	record := &storage.SchemaRecord{}
	var schemaType string

	err := s.stmts.getSchemaByFingerprint.QueryRowContext(ctx, subject, fingerprint).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	// Load references
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

	err := s.stmts.getLatestSchema.QueryRowContext(ctx, subject).Scan(
		&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, storage.ErrSubjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	record.SchemaType = storage.SchemaType(schemaType)

	// Load references
	refs, err := s.loadReferences(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.References = refs

	return record, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	var result sql.Result
	var err error

	if permanent {
		result, err = s.stmts.hardDeleteSchema.ExecContext(ctx, subject, version)
	} else {
		result, err = s.stmts.softDeleteSchema.ExecContext(ctx, subject, version)
	}

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
	query := `SELECT DISTINCT subject FROM schemas`
	if !includeDeleted {
		query += ` WHERE deleted = FALSE`
	}
	query += ` ORDER BY subject`

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
	// First get all versions
	rows, err := s.db.QueryContext(ctx,
		`SELECT version FROM schemas WHERE subject = $1 AND (deleted = FALSE OR $2)`,
		subject, permanent,
	)
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
		return nil, storage.ErrSubjectNotFound
	}

	// Delete or soft-delete
	if permanent {
		_, err = s.db.ExecContext(ctx, `DELETE FROM schemas WHERE subject = $1`, subject)
		if err != nil {
			return nil, fmt.Errorf("failed to delete schemas: %w", err)
		}

		// Also delete configs and modes
		_, _ = s.db.ExecContext(ctx, `DELETE FROM configs WHERE subject = $1`, subject)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM modes WHERE subject = $1`, subject)
	} else {
		_, err = s.db.ExecContext(ctx,
			`UPDATE schemas SET deleted = TRUE WHERE subject = $1`,
			subject,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to soft-delete schemas: %w", err)
		}
	}

	return versions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	var count int
	err := s.stmts.countSchemasBySubject.QueryRowContext(ctx, subject).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check subject: %w", err)
	}
	return count > 0, nil
}

// GetConfig retrieves the compatibility configuration for a subject.
func (s *Store) GetConfig(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	config := &storage.ConfigRecord{}
	err := s.stmts.getConfig.QueryRowContext(ctx, subject).Scan(
		&config.Subject, &config.CompatibilityLevel)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return config, nil
}

// SetConfig sets the compatibility configuration for a subject.
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
	_, err := s.stmts.setConfig.ExecContext(ctx, subject, config.CompatibilityLevel)
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

// NextID returns the next available schema ID.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT nextval('schemas_id_seq')`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get next ID: %w", err)
	}
	return id, nil
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
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	query := `SELECT DISTINCT subject FROM schemas WHERE id = $1`
	if !includeDeleted {
		query += ` AND deleted = FALSE`
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
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	query := `SELECT subject, version FROM schemas WHERE id = $1`
	if !includeDeleted {
		query += ` AND deleted = FALSE`
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
	query := `SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM schemas WHERE 1=1`
	args := []interface{}{}
	argNum := 1

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
		query = `SELECT s.id, s.subject, s.version, s.schema_type, s.schema_text, s.fingerprint, s.deleted, s.created_at
		         FROM schemas s
		         INNER JOIN (
		             SELECT subject, MAX(version) as max_version
		             FROM schemas
		             WHERE 1=1`
		if !params.Deleted {
			query += ` AND deleted = FALSE`
		}
		if params.SubjectPrefix != "" {
			query += fmt.Sprintf(` AND subject LIKE $%d`, argNum-1)
		}
		query += ` GROUP BY subject
		         ) latest ON s.subject = latest.subject AND s.version = latest.max_version`
		if !params.Deleted {
			query += ` WHERE s.deleted = FALSE`
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
		if err := rows.Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		record.SchemaType = storage.SchemaType(schemaType)
		schemas = append(schemas, record)
	}

	return schemas, nil
}

// DeleteGlobalConfig resets the global config to default.
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO configs (subject, compatibility_level) VALUES ('', 'BACKWARD')
		 ON CONFLICT (subject) DO UPDATE SET compatibility_level = 'BACKWARD'`,
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
		key.UserID, key.Name, key.Role, key.Enabled, key.ExpiresAt, key.ID,
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
	_, err := s.stmts.updateAPIKeyLastUsed.ExecContext(ctx, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
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

// isUniqueViolation checks if the error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL error code for unique_violation is 23505
	return err.Error() != "" && (contains(err.Error(), "duplicate key") || contains(err.Error(), "23505"))
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
