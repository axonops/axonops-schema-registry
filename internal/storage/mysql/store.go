package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

	return store, nil
}

// migrate runs database migrations.
func (s *Store) migrate(ctx context.Context) error {
	for i, migration := range migrations {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			// MySQL doesn't support IF NOT EXISTS for indexes in older versions
			// so we ignore "duplicate key" errors for index creation
			if i >= 1 && i <= 3 { // Index creation statements
				continue
			}
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
	defer tx.Rollback()

	// Check for existing schema with same fingerprint
	var existingID int64
	var existingVersion int
	var existingDeleted bool
	err = tx.QueryRowContext(ctx,
		"SELECT id, version, deleted FROM `schemas` WHERE subject = ? AND fingerprint = ?",
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
		"SELECT COALESCE(MAX(version), 0) + 1 FROM `schemas` WHERE subject = ?",
		record.Subject,
	).Scan(&nextVersion)
	if err != nil {
		return fmt.Errorf("failed to get next version: %w", err)
	}

	// Insert schema
	result, err := tx.ExecContext(ctx,
		"INSERT INTO `schemas` (subject, version, schema_type, schema_text, fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		record.Subject, nextVersion, record.SchemaType, record.Schema, record.Fingerprint, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	record.ID = id
	record.Version = nextVersion
	record.CreatedAt = time.Now()

	// Insert references
	for _, ref := range record.References {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO schema_references (schema_id, name, ref_subject, ref_version)
			 VALUES (?, ?, ?, ?)`,
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

	err := s.db.QueryRowContext(ctx,
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE id = ?",
		id,
	).Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
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

	err := s.db.QueryRowContext(ctx,
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE subject = ? AND version = ?",
		subject, version,
	).Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt)

	if err == sql.ErrNoRows {
		// Check if subject exists
		var count int
		s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
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
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE subject = ?"
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

	err := s.db.QueryRowContext(ctx,
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE subject = ? AND fingerprint = ? AND deleted = FALSE",
		subject, fingerprint,
	).Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
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

	err := s.db.QueryRowContext(ctx,
		"SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE subject = ? AND deleted = FALSE ORDER BY version DESC LIMIT 1",
		subject,
	).Scan(&record.ID, &record.Subject, &record.Version, &schemaType,
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
		result, err = s.db.ExecContext(ctx,
			"DELETE FROM `schemas` WHERE subject = ? AND version = ?",
			subject, version,
		)
	} else {
		result, err = s.db.ExecContext(ctx,
			"UPDATE `schemas` SET deleted = TRUE WHERE subject = ? AND version = ?",
			subject, version,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Check if subject exists
		var count int
		s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM `schemas` WHERE subject = ?", subject).Scan(&count)
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
	// First get all versions
	rows, err := s.db.QueryContext(ctx,
		"SELECT version FROM `schemas` WHERE subject = ? AND (deleted = FALSE OR ?)",
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
		_, err = s.db.ExecContext(ctx, "DELETE FROM `schemas` WHERE subject = ?", subject)
		if err != nil {
			return nil, fmt.Errorf("failed to delete schemas: %w", err)
		}

		// Also delete configs and modes
		s.db.ExecContext(ctx, "DELETE FROM configs WHERE subject = ?", subject)
		s.db.ExecContext(ctx, "DELETE FROM modes WHERE subject = ?", subject)
	} else {
		_, err = s.db.ExecContext(ctx,
			"UPDATE `schemas` SET deleted = TRUE WHERE subject = ?",
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
	err := s.db.QueryRowContext(ctx,
		`SELECT subject, compatibility_level FROM configs WHERE subject = ?`,
		subject,
	).Scan(&config.Subject, &config.CompatibilityLevel)

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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO configs (subject, compatibility_level) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE compatibility_level = VALUES(compatibility_level)`,
		subject, config.CompatibilityLevel,
	)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM configs WHERE subject = ?`, subject)
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
	err := s.db.QueryRowContext(ctx,
		`SELECT subject, mode FROM modes WHERE subject = ?`,
		subject,
	).Scan(&mode.Subject, &mode.Mode)

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
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO modes (subject, mode) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE mode = VALUES(mode)`,
		subject, mode.Mode,
	)
	if err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, subject string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM modes WHERE subject = ?`, subject)
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
	// MySQL doesn't have sequences, so we use a dummy insert and get the auto_increment value
	// This is a workaround and may not be ideal for high-concurrency scenarios
	var id int64
	err := s.db.QueryRowContext(ctx, "SELECT AUTO_INCREMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'schemas'").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get next ID: %w", err)
	}
	return id, nil
}

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT s.subject, s.version FROM `schemas` s JOIN schema_references r ON r.schema_id = s.id WHERE r.ref_subject = ? AND r.ref_version = ? AND s.deleted = FALSE",
		subject, version,
	)
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
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, ref_subject, ref_version FROM schema_references WHERE schema_id = ?`,
		schemaID,
	)
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
	query := "SELECT DISTINCT subject FROM `schemas` WHERE id = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
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
	query := "SELECT subject, version FROM `schemas` WHERE id = ?"
	if !includeDeleted {
		query += " AND deleted = FALSE"
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
	query := "SELECT id, subject, version, schema_type, schema_text, fingerprint, deleted, created_at FROM `schemas` WHERE 1=1"
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
		query = "SELECT s.id, s.subject, s.version, s.schema_type, s.schema_text, s.fingerprint, s.deleted, s.created_at FROM `schemas` s INNER JOIN (SELECT subject, MAX(version) as max_version FROM `schemas` WHERE 1=1"
		if !params.Deleted {
			query += " AND deleted = FALSE"
		}
		if params.SubjectPrefix != "" {
			query += " AND subject LIKE ?"
		}
		query += " GROUP BY subject) latest ON s.subject = latest.subject AND s.version = latest.max_version"
		if !params.Deleted {
			query += " WHERE s.deleted = FALSE"
		}
	}

	query += " ORDER BY id"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
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
		 ON DUPLICATE KEY UPDATE compatibility_level = 'BACKWARD'`,
	)
	if err != nil {
		return fmt.Errorf("failed to reset global config: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
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
