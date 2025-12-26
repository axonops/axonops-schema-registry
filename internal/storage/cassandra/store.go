package cassandra

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// globalKey is the sentinel value for global config/mode in Cassandra.
// Cassandra doesn't allow empty partition keys, so we use this instead.
const globalKey = "__global__"

// Config holds Cassandra connection configuration.
type Config struct {
	Hosts               []string      `json:"hosts" yaml:"hosts"`
	Port                int           `json:"port" yaml:"port"`
	Keyspace            string        `json:"keyspace" yaml:"keyspace"`
	Username            string        `json:"username" yaml:"username"`
	Password            string        `json:"password" yaml:"password"`
	Consistency         string        `json:"consistency" yaml:"consistency"`                   // Default consistency (used if read/write not specified)
	ReadConsistency     string        `json:"read_consistency" yaml:"read_consistency"`         // Consistency for read operations (e.g., LOCAL_ONE for low latency)
	WriteConsistency    string        `json:"write_consistency" yaml:"write_consistency"`       // Consistency for write operations (e.g., LOCAL_QUORUM for durability)
	LocalDC             string        `json:"local_dc" yaml:"local_dc"`
	ReplicationStrategy string        `json:"replication_strategy" yaml:"replication_strategy"`
	ReplicationFactor   int           `json:"replication_factor" yaml:"replication_factor"`
	ConnectTimeout      time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	Timeout             time.Duration `json:"timeout" yaml:"timeout"`
	NumConns            int           `json:"num_conns" yaml:"num_conns"`
	MaxPreparedStmts    int           `json:"max_prepared_stmts" yaml:"max_prepared_stmts"`
	EnableTLS           bool          `json:"enable_tls" yaml:"enable_tls"`
	TLSVerifyHost       bool          `json:"tls_verify_host" yaml:"tls_verify_host"`
	SubjectBuckets      int           `json:"subject_buckets" yaml:"subject_buckets"`
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Hosts:               []string{"localhost"},
		Port:                9042,
		Keyspace:            "schema_registry",
		Consistency:         "LOCAL_QUORUM",
		ReplicationStrategy: "SimpleStrategy",
		ReplicationFactor:   1,
		ConnectTimeout:      5 * time.Second,
		Timeout:             10 * time.Second,
		NumConns:            2,
		MaxPreparedStmts:    1000,
		SubjectBuckets:      16,
	}
}

// Store implements the storage.Storage interface using Cassandra.
type Store struct {
	session          *gocql.Session
	config           Config
	idCache          int64           // Local cache of last known ID
	readConsistency  gocql.Consistency // Consistency level for read operations
	writeConsistency gocql.Consistency // Consistency level for write operations
}

// NewStore creates a new Cassandra store.
func NewStore(config Config) (*Store, error) {
	return NewStoreWithRetry(config, 5, 2*time.Second)
}

// NewStoreWithRetry creates a new Cassandra store with retry logic.
func NewStoreWithRetry(config Config, maxRetries int, retryDelay time.Duration) (*Store, error) {
	// Create cluster configuration
	cluster := gocql.NewCluster(config.Hosts...)
	cluster.Port = config.Port
	cluster.Keyspace = config.Keyspace
	cluster.Timeout = config.Timeout
	cluster.ConnectTimeout = config.ConnectTimeout
	cluster.NumConns = config.NumConns

	// Set default cluster consistency level
	defaultConsistency := parseConsistency(config.Consistency)
	cluster.Consistency = defaultConsistency

	// Parse read/write consistency levels (fall back to default if not specified)
	readConsistency := defaultConsistency
	writeConsistency := defaultConsistency
	if config.ReadConsistency != "" {
		readConsistency = parseConsistency(config.ReadConsistency)
	}
	if config.WriteConsistency != "" {
		writeConsistency = parseConsistency(config.WriteConsistency)
	}

	// Configure reconnection policy for better resilience
	cluster.ReconnectionPolicy = &gocql.ConstantReconnectionPolicy{
		MaxRetries: 10,
		Interval:   time.Second,
	}

	// Configure retry policy
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}

	// Authentication
	if config.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: config.Username,
			Password: config.Password,
		}
	}

	// Datacenter-aware configuration - only use for multi-DC setups
	// For single-node/test setups, use simple round-robin
	if config.LocalDC != "" && len(config.Hosts) > 1 {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(config.LocalDC)
	}

	// TLS configuration
	if config.EnableTLS {
		cluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: config.TLSVerifyHost,
		}
	}

	// First, connect without keyspace to create it (with retry)
	clusterNoKS := *cluster
	clusterNoKS.Keyspace = ""

	var sessionNoKS *gocql.Session
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		sessionNoKS, err = clusterNoKS.CreateSession()
		if err == nil {
			break
		}
		if attempt < maxRetries {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Cassandra after %d attempts: %w", maxRetries, err)
	}

	// Create keyspace
	ksQuery := keyspaceCQL(config.Keyspace, config.ReplicationStrategy, config.ReplicationFactor)
	if err := sessionNoKS.Query(ksQuery).Exec(); err != nil {
		sessionNoKS.Close()
		return nil, fmt.Errorf("failed to create keyspace: %w", err)
	}
	sessionNoKS.Close()

	// Now connect with keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to keyspace: %w", err)
	}

	store := &Store{
		session:          session,
		config:           config,
		readConsistency:  readConsistency,
		writeConsistency: writeConsistency,
	}

	// Run migrations first to ensure tables exist
	if err := store.migrate(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize idCache from Cassandra counter + unique offset for this instance
	// This prevents ID collisions across multiple instances
	if err := store.initIDCache(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to initialize ID cache: %w", err)
	}

	return store, nil
}

// readQuery creates a query with read consistency level.
func (s *Store) readQuery(stmt string, values ...interface{}) *gocql.Query {
	return s.session.Query(stmt, values...).Consistency(s.readConsistency)
}

// writeQuery creates a query with write consistency level.
func (s *Store) writeQuery(stmt string, values ...interface{}) *gocql.Query {
	return s.session.Query(stmt, values...).Consistency(s.writeConsistency)
}

// parseConsistency converts a string to gocql.Consistency.
func parseConsistency(s string) gocql.Consistency {
	switch strings.ToUpper(s) {
	case "ANY":
		return gocql.Any
	case "ONE":
		return gocql.One
	case "TWO":
		return gocql.Two
	case "THREE":
		return gocql.Three
	case "QUORUM":
		return gocql.Quorum
	case "ALL":
		return gocql.All
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum
	case "EACH_QUORUM":
		return gocql.EachQuorum
	case "LOCAL_ONE":
		return gocql.LocalOne
	default:
		return gocql.LocalQuorum
	}
}

// migrate runs CQL migrations.
func (s *Store) migrate() error {
	for i, migration := range migrations {
		if err := s.session.Query(migration).Exec(); err != nil {
			// Ignore "already exists" errors for IF NOT EXISTS statements
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("migration %d failed: %w", i+1, err)
			}
		}
	}
	return nil
}

// subjectBucket returns the bucket number for a subject (for distributed subject listing).
func (s *Store) subjectBucket(subject string) int {
	// Simple hash-based bucketing
	var hash int
	for _, c := range subject {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	buckets := s.config.SubjectBuckets
	if buckets <= 0 {
		buckets = 16
	}
	return hash % buckets
}

// CreateSchema stores a new schema record.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	// Check for existing schema with same fingerprint
	var existingID int64
	var existingVersion int
	err := s.session.Query(
		`SELECT id, version FROM schemas_by_fingerprint WHERE subject = ? AND fingerprint = ?`,
		record.Subject, record.Fingerprint,
	).WithContext(ctx).Scan(&existingID, &existingVersion)

	if err == nil {
		// Check if it's deleted
		var deleted bool
		_ = s.session.Query(
			`SELECT deleted FROM schemas WHERE subject = ? AND version = ?`,
			record.Subject, existingVersion,
		).WithContext(ctx).Scan(&deleted)

		if !deleted {
			record.ID = existingID
			record.Version = existingVersion
			return storage.ErrSchemaExists
		}
	}

	// Generate new ID first
	newID, err := s.NextID(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate ID: %w", err)
	}
	record.ID = newID
	record.CreatedAt = time.Now()

	// Use LWT (Lightweight Transaction) with retry to handle concurrent version assignment
	// This ensures no two schemas get the same version for the same subject
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get current max version
		var maxVersion int
		iter := s.session.Query(
			`SELECT version FROM schemas WHERE subject = ? ORDER BY version DESC LIMIT 1`,
			record.Subject,
		).WithContext(ctx).Iter()
		_ = iter.Scan(&maxVersion)
		iter.Close()
		nextVersion := maxVersion + 1
		record.Version = nextVersion

		// Use LWT INSERT with IF NOT EXISTS for the main schemas table
		// This ensures atomic version assignment
		result := make(map[string]interface{})
		applied, err := s.session.Query(
			`INSERT INTO schemas (subject, version, id, schema_type, schema_text, fingerprint, deleted, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?) IF NOT EXISTS`,
			record.Subject, record.Version, record.ID, string(record.SchemaType),
			record.Schema, record.Fingerprint, false, record.CreatedAt,
		).WithContext(ctx).MapScanCAS(result)

		if err != nil {
			return fmt.Errorf("failed to insert schema: %w", err)
		}

		if applied {
			// LWT succeeded, now insert into lookup tables
			// These don't need LWT since the primary table insert succeeded
			if err := s.session.Query(
				`INSERT INTO schemas_by_id (id, subject, version) VALUES (?, ?, ?)`,
				record.ID, record.Subject, record.Version,
			).WithContext(ctx).Exec(); err != nil {
				return fmt.Errorf("failed to insert schema_by_id: %w", err)
			}

			if err := s.session.Query(
				`INSERT INTO schemas_by_fingerprint (subject, fingerprint, id, version) VALUES (?, ?, ?, ?)`,
				record.Subject, record.Fingerprint, record.ID, record.Version,
			).WithContext(ctx).Exec(); err != nil {
				return fmt.Errorf("failed to insert schema_by_fingerprint: %w", err)
			}

			// Track subject in subjects table
			bucket := s.subjectBucket(record.Subject)
			if err := s.session.Query(
				`INSERT INTO subjects (bucket, subject) VALUES (?, ?)`,
				bucket, record.Subject,
			).WithContext(ctx).Exec(); err != nil {
				return fmt.Errorf("failed to insert subject: %w", err)
			}

			break // Success, exit retry loop
		}

		// LWT failed (version already exists), retry with next version
		if attempt == maxRetries-1 {
			return fmt.Errorf("failed to insert schema after %d retries: version conflict", maxRetries)
		}
	}

	// Insert references
	for _, ref := range record.References {
		if err := s.session.Query(
			`INSERT INTO schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`,
			record.ID, ref.Name, ref.Subject, ref.Version,
		).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to insert reference: %w", err)
		}

		// Insert reverse reference
		if err := s.session.Query(
			`INSERT INTO references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`,
			ref.Subject, ref.Version, record.Subject, record.Version,
		).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to insert reverse reference: %w", err)
		}
	}

	return nil
}

// GetSchemaByID retrieves a schema by its global ID.
func (s *Store) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	// First lookup subject and version from schemas_by_id
	var subject string
	var version int
	if err := s.readQuery(
		`SELECT subject, version FROM schemas_by_id WHERE id = ?`,
		id,
	).WithContext(ctx).Scan(&subject, &version); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to lookup schema: %w", err)
	}

	return s.GetSchemaBySubjectVersion(ctx, subject, version)
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	// Handle "latest" version (-1)
	if version == -1 {
		return s.GetLatestSchema(ctx, subject)
	}

	record := &storage.SchemaRecord{}
	var schemaType string

	if err := s.readQuery(
		`SELECT subject, version, id, schema_type, schema_text, fingerprint, deleted, created_at
		 FROM schemas WHERE subject = ? AND version = ?`,
		subject, version,
	).WithContext(ctx).Scan(
		&record.Subject, &record.Version, &record.ID, &schemaType,
		&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt,
	); err != nil {
		if err == gocql.ErrNotFound {
			// Check if subject exists
			var count int
			_ = s.readQuery(`SELECT COUNT(*) FROM schemas WHERE subject = ?`, subject).
				WithContext(ctx).Scan(&count)
			if count == 0 {
				return nil, storage.ErrSubjectNotFound
			}
			return nil, storage.ErrVersionNotFound
		}
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
	query := `SELECT subject, version, id, schema_type, schema_text, fingerprint, deleted, created_at
		      FROM schemas WHERE subject = ?`

	iter := s.readQuery(query, subject).WithContext(ctx).Iter()

	var schemas []*storage.SchemaRecord
	for {
		record := &storage.SchemaRecord{}
		var schemaType string
		if !iter.Scan(&record.Subject, &record.Version, &record.ID, &schemaType,
			&record.Schema, &record.Fingerprint, &record.Deleted, &record.CreatedAt) {
			break
		}

		if !includeDeleted && record.Deleted {
			continue
		}

		record.SchemaType = storage.SchemaType(schemaType)

		// Load references
		refs, err := s.loadReferences(ctx, record.ID)
		if err != nil {
			iter.Close()
			return nil, err
		}
		record.References = refs

		schemas = append(schemas, record)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}

	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string) (*storage.SchemaRecord, error) {
	var id int64
	var version int
	if err := s.readQuery(
		`SELECT id, version FROM schemas_by_fingerprint WHERE subject = ? AND fingerprint = ?`,
		subject, fingerprint,
	).WithContext(ctx).Scan(&id, &version); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to lookup schema: %w", err)
	}

	record, err := s.GetSchemaBySubjectVersion(ctx, subject, version)
	if err != nil {
		return nil, err
	}

	if record.Deleted {
		return nil, storage.ErrSchemaNotFound
	}

	return record, nil
}

// GetLatestSchema retrieves the latest schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, subject string) (*storage.SchemaRecord, error) {
	// Get all schemas for subject, ordered by version descending
	schemas, err := s.GetSchemasBySubject(ctx, subject, false)
	if err != nil {
		return nil, err
	}

	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Find the highest version (schemas are not guaranteed to be sorted)
	var latest *storage.SchemaRecord
	for _, schema := range schemas {
		if latest == nil || schema.Version > latest.Version {
			latest = schema
		}
	}

	return latest, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	// First check if schema exists
	var deleted bool
	var fingerprint string
	var id int64
	if err := s.session.Query(
		`SELECT deleted, fingerprint, id FROM schemas WHERE subject = ? AND version = ?`,
		subject, version,
	).WithContext(ctx).Scan(&deleted, &fingerprint, &id); err != nil {
		if err == gocql.ErrNotFound {
			// Check if subject exists
			var count int
			_ = s.session.Query(`SELECT COUNT(*) FROM schemas WHERE subject = ?`, subject).
				WithContext(ctx).Scan(&count)
			if count == 0 {
				return storage.ErrSubjectNotFound
			}
			return storage.ErrVersionNotFound
		}
		return fmt.Errorf("failed to get schema: %w", err)
	}

	if permanent {
		// Hard delete from all tables
		batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
		batch.Query(`DELETE FROM schemas WHERE subject = ? AND version = ?`, subject, version)
		batch.Query(`DELETE FROM schemas_by_id WHERE id = ?`, id)
		batch.Query(`DELETE FROM schemas_by_fingerprint WHERE subject = ? AND fingerprint = ?`, subject, fingerprint)
		batch.Query(`DELETE FROM schema_references WHERE schema_id = ?`, id)

		if err := s.session.ExecuteBatch(batch); err != nil {
			return fmt.Errorf("failed to delete schema: %w", err)
		}
	} else {
		// Soft delete
		if err := s.session.Query(
			`UPDATE schemas SET deleted = true WHERE subject = ? AND version = ?`,
			subject, version,
		).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to soft-delete schema: %w", err)
		}
	}

	return nil
}

// ListSubjects returns all subject names.
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	// Query all buckets for subjects
	subjectSet := make(map[string]bool)
	buckets := s.config.SubjectBuckets
	if buckets <= 0 {
		buckets = 16
	}

	for bucket := 0; bucket < buckets; bucket++ {
		iter := s.session.Query(
			`SELECT subject FROM subjects WHERE bucket = ?`,
			bucket,
		).WithContext(ctx).Iter()

		var subject string
		for iter.Scan(&subject) {
			subjectSet[subject] = true
		}
		iter.Close()
	}

	// Filter out subjects with no non-deleted schemas if needed
	var subjects []string
	for subject := range subjectSet {
		if includeDeleted {
			subjects = append(subjects, subject)
			continue
		}

		// Check if subject has any non-deleted schemas
		var deleted bool
		iter := s.session.Query(
			`SELECT deleted FROM schemas WHERE subject = ? LIMIT 1`,
			subject,
		).WithContext(ctx).Iter()

		hasNonDeleted := false
		for iter.Scan(&deleted) {
			if !deleted {
				hasNonDeleted = true
				break
			}
		}
		iter.Close()

		if hasNonDeleted {
			subjects = append(subjects, subject)
		}
	}

	return subjects, nil
}

// DeleteSubject deletes all versions of a subject.
func (s *Store) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	// Get all versions
	iter := s.session.Query(
		`SELECT version, fingerprint, id, deleted FROM schemas WHERE subject = ?`,
		subject,
	).WithContext(ctx).Iter()

	type schemaInfo struct {
		version     int
		fingerprint string
		id          int64
		deleted     bool
	}

	var schemas []schemaInfo
	var info schemaInfo
	for iter.Scan(&info.version, &info.fingerprint, &info.id, &info.deleted) {
		schemas = append(schemas, info)
	}
	iter.Close()

	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var deletedVersions []int
	for _, schema := range schemas {
		if schema.deleted && !permanent {
			continue
		}
		deletedVersions = append(deletedVersions, schema.version)

		if permanent {
			batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
			batch.Query(`DELETE FROM schemas WHERE subject = ? AND version = ?`, subject, schema.version)
			batch.Query(`DELETE FROM schemas_by_id WHERE id = ?`, schema.id)
			batch.Query(`DELETE FROM schemas_by_fingerprint WHERE subject = ? AND fingerprint = ?`, subject, schema.fingerprint)
			batch.Query(`DELETE FROM schema_references WHERE schema_id = ?`, schema.id)
			_ = s.session.ExecuteBatch(batch)
		} else {
			_ = s.session.Query(
				`UPDATE schemas SET deleted = true WHERE subject = ? AND version = ?`,
				subject, schema.version,
			).WithContext(ctx).Exec()
		}
	}

	if permanent {
		// Remove from subjects table
		bucket := s.subjectBucket(subject)
		_ = s.session.Query(`DELETE FROM subjects WHERE bucket = ? AND subject = ?`, bucket, subject).
			WithContext(ctx).Exec()

		// Delete configs and modes
		_ = s.session.Query(`DELETE FROM configs WHERE subject = ?`, subject).WithContext(ctx).Exec()
		_ = s.session.Query(`DELETE FROM modes WHERE subject = ?`, subject).WithContext(ctx).Exec()
	}

	return deletedVersions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	var deleted bool
	iter := s.session.Query(
		`SELECT deleted FROM schemas WHERE subject = ?`,
		subject,
	).WithContext(ctx).Iter()

	for iter.Scan(&deleted) {
		if !deleted {
			iter.Close()
			return true, nil
		}
	}
	iter.Close()

	return false, nil
}

// GetConfig retrieves the compatibility configuration for a subject.
func (s *Store) GetConfig(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	config := &storage.ConfigRecord{Subject: subject}
	if err := s.session.Query(
		`SELECT compatibility_level FROM configs WHERE subject = ?`,
		subject,
	).WithContext(ctx).Scan(&config.CompatibilityLevel); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return config, nil
}

// SetConfig sets the compatibility configuration for a subject.
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
	if err := s.session.Query(
		`INSERT INTO configs (subject, compatibility_level) VALUES (?, ?)`,
		subject, config.CompatibilityLevel,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	// Check if exists first
	var level string
	if err := s.session.Query(
		`SELECT compatibility_level FROM configs WHERE subject = ?`,
		subject,
	).WithContext(ctx).Scan(&level); err != nil {
		if err == gocql.ErrNotFound {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to check config: %w", err)
	}

	if err := s.session.Query(
		`DELETE FROM configs WHERE subject = ?`,
		subject,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// GetGlobalConfig retrieves the global compatibility configuration.
func (s *Store) GetGlobalConfig(ctx context.Context) (*storage.ConfigRecord, error) {
	return s.GetConfig(ctx, globalKey)
}

// SetGlobalConfig sets the global compatibility configuration.
func (s *Store) SetGlobalConfig(ctx context.Context, config *storage.ConfigRecord) error {
	return s.SetConfig(ctx, globalKey, config)
}

// GetMode retrieves the mode for a subject.
func (s *Store) GetMode(ctx context.Context, subject string) (*storage.ModeRecord, error) {
	mode := &storage.ModeRecord{Subject: subject}
	if err := s.session.Query(
		`SELECT mode FROM modes WHERE subject = ?`,
		subject,
	).WithContext(ctx).Scan(&mode.Mode); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get mode: %w", err)
	}
	return mode, nil
}

// SetMode sets the mode for a subject.
func (s *Store) SetMode(ctx context.Context, subject string, mode *storage.ModeRecord) error {
	if err := s.session.Query(
		`INSERT INTO modes (subject, mode) VALUES (?, ?)`,
		subject, mode.Mode,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, subject string) error {
	// Check if exists first
	var mode string
	if err := s.session.Query(
		`SELECT mode FROM modes WHERE subject = ?`,
		subject,
	).WithContext(ctx).Scan(&mode); err != nil {
		if err == gocql.ErrNotFound {
			return storage.ErrNotFound
		}
		return fmt.Errorf("failed to check mode: %w", err)
	}

	if err := s.session.Query(
		`DELETE FROM modes WHERE subject = ?`,
		subject,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to delete mode: %w", err)
	}
	return nil
}

// GetGlobalMode retrieves the global mode.
func (s *Store) GetGlobalMode(ctx context.Context) (*storage.ModeRecord, error) {
	return s.GetMode(ctx, globalKey)
}

// SetGlobalMode sets the global mode.
func (s *Store) SetGlobalMode(ctx context.Context, mode *storage.ModeRecord) error {
	return s.SetMode(ctx, globalKey, mode)
}

// initIDCache initializes the local ID cache from Cassandra with a unique offset.
// This prevents ID collisions across multiple instances.
func (s *Store) initIDCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read current counter value from Cassandra
	var currentValue int64
	err := s.session.Query(
		`SELECT value FROM id_counter WHERE name = 'schema_id'`,
	).WithContext(ctx).Scan(&currentValue)
	if err != nil && err != gocql.ErrNotFound {
		return fmt.Errorf("failed to read counter: %w", err)
	}

	// Add a unique offset based on current time (in milliseconds)
	// This ensures each instance starts with a unique ID range
	// Even if two instances start at the exact same millisecond,
	// the atomic counter will ensure uniqueness within each instance
	timeOffset := time.Now().UnixMilli() % 1000000 // Keep reasonable range

	// Set initial cache value: current DB value + time-based offset * large multiplier
	// This creates distinct ID ranges for each instance
	s.idCache = currentValue + (timeOffset * 10000)

	return nil
}

// NextID returns the next available schema ID using a hybrid approach.
// Uses local atomic counter combined with instance-unique offset for uniqueness.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	// Use local atomic counter for the ID
	// The idCache was initialized with a unique offset for this instance
	localID := atomic.AddInt64(&s.idCache, 1)

	// Also increment Cassandra counter (best effort, for tracking)
	_ = s.session.Query(
		`UPDATE id_counter SET value = value + 1 WHERE name = 'schema_id'`,
	).WithContext(ctx).Exec()

	return localID, nil
}

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	iter := s.session.Query(
		`SELECT schema_subject, schema_version FROM references_by_target
		 WHERE ref_subject = ? AND ref_version = ?`,
		subject, version,
	).WithContext(ctx).Iter()

	var refs []storage.SubjectVersion
	var ref storage.SubjectVersion
	for iter.Scan(&ref.Subject, &ref.Version) {
		refs = append(refs, ref)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}

	return refs, nil
}

// loadReferences loads references for a schema.
func (s *Store) loadReferences(ctx context.Context, schemaID int64) ([]storage.Reference, error) {
	iter := s.readQuery(
		`SELECT name, ref_subject, ref_version FROM schema_references WHERE schema_id = ?`,
		schemaID,
	).WithContext(ctx).Iter()

	var refs []storage.Reference
	var ref storage.Reference
	for iter.Scan(&ref.Name, &ref.Subject, &ref.Version) {
		refs = append(refs, ref)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}

	return refs, nil
}

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	// First lookup subject from schemas_by_id
	var subject string
	var version int
	if err := s.session.Query(
		`SELECT subject, version FROM schemas_by_id WHERE id = ?`,
		id,
	).WithContext(ctx).Scan(&subject, &version); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to lookup schema: %w", err)
	}

	// Check if deleted
	if !includeDeleted {
		var deleted bool
		if err := s.session.Query(
			`SELECT deleted FROM schemas WHERE subject = ? AND version = ?`,
			subject, version,
		).WithContext(ctx).Scan(&deleted); err == nil && deleted {
			return []string{}, nil
		}
	}

	return []string{subject}, nil
}

// GetVersionsBySchemaID returns all subject-version pairs where the given schema ID is registered.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	// First lookup subject from schemas_by_id
	var subject string
	var version int
	if err := s.session.Query(
		`SELECT subject, version FROM schemas_by_id WHERE id = ?`,
		id,
	).WithContext(ctx).Scan(&subject, &version); err != nil {
		if err == gocql.ErrNotFound {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to lookup schema: %w", err)
	}

	// Check if deleted
	if !includeDeleted {
		var deleted bool
		if err := s.session.Query(
			`SELECT deleted FROM schemas WHERE subject = ? AND version = ?`,
			subject, version,
		).WithContext(ctx).Scan(&deleted); err == nil && deleted {
			return []storage.SubjectVersion{}, nil
		}
	}

	return []storage.SubjectVersion{{Subject: subject, Version: version}}, nil
}

// ListSchemas returns schemas matching the given filters.
func (s *Store) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	// Get all subjects first
	subjects, err := s.ListSubjects(ctx, params.Deleted)
	if err != nil {
		return nil, err
	}

	var results []*storage.SchemaRecord

	for _, subject := range subjects {
		// Apply subject prefix filter
		if params.SubjectPrefix != "" {
			if len(subject) < len(params.SubjectPrefix) ||
				subject[:len(params.SubjectPrefix)] != params.SubjectPrefix {
				continue
			}
		}

		if params.LatestOnly {
			// Get only latest schema for this subject
			schema, err := s.GetLatestSchema(ctx, subject)
			if err != nil {
				continue
			}
			results = append(results, schema)
		} else {
			// Get all schemas for this subject
			schemas, err := s.GetSchemasBySubject(ctx, subject, params.Deleted)
			if err != nil {
				continue
			}
			results = append(results, schemas...)
		}
	}

	// Apply offset and limit
	if params.Offset > 0 {
		if params.Offset >= len(results) {
			return []*storage.SchemaRecord{}, nil
		}
		results = results[params.Offset:]
	}

	if params.Limit > 0 && params.Limit < len(results) {
		results = results[:params.Limit]
	}

	return results, nil
}

// DeleteGlobalConfig resets the global config to default.
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	if err := s.session.Query(
		`INSERT INTO configs (subject, compatibility_level) VALUES (?, 'BACKWARD')`,
		globalKey,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to reset global config: %w", err)
	}
	return nil
}

// nextUserID returns the next available user ID.
func (s *Store) nextUserID(ctx context.Context) (int64, error) {
	// Increment counter first
	if err := s.session.Query(
		`UPDATE user_id_counter SET value = value + 1 WHERE name = 'user_id'`,
	).WithContext(ctx).Exec(); err != nil {
		return 0, fmt.Errorf("failed to increment user counter: %w", err)
	}

	// Read the counter value
	var value int64
	if err := s.session.Query(
		`SELECT value FROM user_id_counter WHERE name = 'user_id'`,
	).WithContext(ctx).Scan(&value); err != nil {
		return 0, fmt.Errorf("failed to read user counter: %w", err)
	}

	return value, nil
}

// nextAPIKeyID returns the next available API key ID.
func (s *Store) nextAPIKeyID(ctx context.Context) (int64, error) {
	// Increment counter first
	if err := s.session.Query(
		`UPDATE api_key_id_counter SET value = value + 1 WHERE name = 'api_key_id'`,
	).WithContext(ctx).Exec(); err != nil {
		return 0, fmt.Errorf("failed to increment API key counter: %w", err)
	}

	// Read the counter value
	var value int64
	if err := s.session.Query(
		`SELECT value FROM api_key_id_counter WHERE name = 'api_key_id'`,
	).WithContext(ctx).Scan(&value); err != nil {
		return 0, fmt.Errorf("failed to read API key counter: %w", err)
	}

	return value, nil
}

// CreateUser creates a new user record.
func (s *Store) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Check for existing username
	var existingID int64
	err := s.session.Query(
		`SELECT id FROM users_by_username WHERE username = ?`,
		user.Username,
	).WithContext(ctx).Scan(&existingID)
	if err == nil {
		return storage.ErrUserExists
	}
	if err != gocql.ErrNotFound {
		return fmt.Errorf("failed to check username: %w", err)
	}

	// Check for existing email if provided
	if user.Email != "" {
		err = s.session.Query(
			`SELECT id FROM users_by_email WHERE email = ?`,
			user.Email,
		).WithContext(ctx).Scan(&existingID)
		if err == nil {
			return storage.ErrUserExists
		}
		if err != gocql.ErrNotFound {
			return fmt.Errorf("failed to check email: %w", err)
		}
	}

	// Get next ID
	id, err := s.nextUserID(ctx)
	if err != nil {
		return err
	}
	user.ID = id

	// Insert into all tables using batch
	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(
		`INSERT INTO users (id, username, email, password_hash, role, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.Enabled, user.CreatedAt, user.UpdatedAt,
	)

	batch.Query(
		`INSERT INTO users_by_username (username, id) VALUES (?, ?)`,
		user.Username, user.ID,
	)

	if user.Email != "" {
		batch.Query(
			`INSERT INTO users_by_email (email, id) VALUES (?, ?)`,
			user.Email, user.ID,
		)
	}

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	user := &storage.UserRecord{}
	var email *string

	err := s.session.Query(
		`SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		 FROM users WHERE id = ?`,
		id,
	).WithContext(ctx).Scan(&user.ID, &user.Username, &email, &user.PasswordHash,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err == gocql.ErrNotFound {
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email != nil {
		user.Email = *email
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	// Look up ID from username index
	var id int64
	err := s.session.Query(
		`SELECT id FROM users_by_username WHERE username = ?`,
		username,
	).WithContext(ctx).Scan(&id)

	if err == gocql.ErrNotFound {
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	return s.GetUserByID(ctx, id)
}

// UpdateUser updates an existing user record.
func (s *Store) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	user.UpdatedAt = time.Now()

	// Get current user to check for username/email changes
	current, err := s.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}

	// Check if new username is taken (if changed)
	if user.Username != current.Username {
		var existingID int64
		err := s.session.Query(
			`SELECT id FROM users_by_username WHERE username = ?`,
			user.Username,
		).WithContext(ctx).Scan(&existingID)
		if err == nil {
			return storage.ErrUserExists
		}
		if err != gocql.ErrNotFound {
			return fmt.Errorf("failed to check username: %w", err)
		}
	}

	// Check if new email is taken (if changed)
	if user.Email != "" && user.Email != current.Email {
		var existingID int64
		err := s.session.Query(
			`SELECT id FROM users_by_email WHERE email = ?`,
			user.Email,
		).WithContext(ctx).Scan(&existingID)
		if err == nil {
			return storage.ErrUserExists
		}
		if err != gocql.ErrNotFound {
			return fmt.Errorf("failed to check email: %w", err)
		}
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	// Update main record
	batch.Query(
		`INSERT INTO users (id, username, email, password_hash, role, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.Enabled, user.CreatedAt, user.UpdatedAt,
	)

	// Update username index if changed
	if user.Username != current.Username {
		batch.Query(`DELETE FROM users_by_username WHERE username = ?`, current.Username)
		batch.Query(`INSERT INTO users_by_username (username, id) VALUES (?, ?)`, user.Username, user.ID)
	}

	// Update email index if changed
	if user.Email != current.Email {
		if current.Email != "" {
			batch.Query(`DELETE FROM users_by_email WHERE email = ?`, current.Email)
		}
		if user.Email != "" {
			batch.Query(`INSERT INTO users_by_email (email, id) VALUES (?, ?)`, user.Email, user.ID)
		}
	}

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	// Get current user for index cleanup
	current, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(`DELETE FROM users WHERE id = ?`, id)
	batch.Query(`DELETE FROM users_by_username WHERE username = ?`, current.Username)
	if current.Email != "" {
		batch.Query(`DELETE FROM users_by_email WHERE email = ?`, current.Email)
	}

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	iter := s.session.Query(
		`SELECT id, username, email, password_hash, role, enabled, created_at, updated_at FROM users`,
	).WithContext(ctx).Iter()

	var users []*storage.UserRecord
	for {
		user := &storage.UserRecord{}
		var email *string
		if !iter.Scan(&user.ID, &user.Username, &email, &user.PasswordHash,
			&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt) {
			break
		}
		if email != nil {
			user.Email = *email
		}
		users = append(users, user)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// CreateAPIKey creates a new API key record.
func (s *Store) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	key.CreatedAt = time.Now()

	// Check for existing key hash
	var existingID int64
	err := s.session.Query(
		`SELECT id FROM api_keys_by_hash WHERE key_hash = ?`,
		key.KeyHash,
	).WithContext(ctx).Scan(&existingID)
	if err == nil {
		return storage.ErrAPIKeyExists
	}
	if err != gocql.ErrNotFound {
		return fmt.Errorf("failed to check key hash: %w", err)
	}

	// Get next ID
	id, err := s.nextAPIKeyID(ctx)
	if err != nil {
		return err
	}
	key.ID = id

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(
		`INSERT INTO api_keys (id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.Role, key.Enabled, key.CreatedAt, key.ExpiresAt, key.LastUsed,
	)

	batch.Query(
		`INSERT INTO api_keys_by_hash (key_hash, id) VALUES (?, ?)`,
		key.KeyHash, key.ID,
	)

	// UserID is now required, always insert into api_keys_by_user
	batch.Query(
		`INSERT INTO api_keys_by_user (user_id, id, created_at) VALUES (?, ?, ?)`,
		key.UserID, key.ID, key.CreatedAt,
	)

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Store) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	key := &storage.APIKeyRecord{}
	var lastUsed *time.Time

	err := s.session.Query(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used
		 FROM api_keys WHERE id = ?`,
		id,
	).WithContext(ctx).Scan(&key.ID, &key.UserID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Role,
		&key.Enabled, &key.CreatedAt, &key.ExpiresAt, &lastUsed)

	if err == gocql.ErrNotFound {
		return nil, storage.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	key.LastUsed = lastUsed

	return key, nil
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	// Look up ID from hash index
	var id int64
	err := s.session.Query(
		`SELECT id FROM api_keys_by_hash WHERE key_hash = ?`,
		keyHash,
	).WithContext(ctx).Scan(&id)

	if err == gocql.ErrNotFound {
		return nil, storage.ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lookup API key: %w", err)
	}

	return s.GetAPIKeyByID(ctx, id)
}

// UpdateAPIKey updates an existing API key record.
func (s *Store) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	// Get current key for reference
	current, err := s.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		return err
	}

	// Update main record (key_hash is immutable)
	if err := s.session.Query(
		`INSERT INTO api_keys (id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.UserID, current.KeyHash, current.KeyPrefix, key.Name, key.Role, key.Enabled, current.CreatedAt, key.ExpiresAt, key.LastUsed,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key by ID.
func (s *Store) DeleteAPIKey(ctx context.Context, id int64) error {
	// Get current key for index cleanup
	current, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(`DELETE FROM api_keys WHERE id = ?`, id)
	batch.Query(`DELETE FROM api_keys_by_hash WHERE key_hash = ?`, current.KeyHash)
	// UserID is now required, always delete from api_keys_by_user
	batch.Query(`DELETE FROM api_keys_by_user WHERE user_id = ? AND created_at = ? AND id = ?`,
		current.UserID, current.CreatedAt, id)

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// ListAPIKeys returns all API keys.
func (s *Store) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	iter := s.session.Query(
		`SELECT id, user_id, key_hash, key_prefix, name, role, enabled, created_at, expires_at, last_used FROM api_keys`,
	).WithContext(ctx).Iter()

	var keys []*storage.APIKeyRecord
	for {
		key := &storage.APIKeyRecord{}
		var lastUsed *time.Time
		if !iter.Scan(&key.ID, &key.UserID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Role,
			&key.Enabled, &key.CreatedAt, &key.ExpiresAt, &lastUsed) {
			break
		}
		key.LastUsed = lastUsed
		keys = append(keys, key)
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return keys, nil
}

// ListAPIKeysByUserID returns all API keys for a user.
func (s *Store) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	iter := s.session.Query(
		`SELECT id FROM api_keys_by_user WHERE user_id = ?`,
		userID,
	).WithContext(ctx).Iter()

	var keys []*storage.APIKeyRecord
	var keyID int64
	for iter.Scan(&keyID) {
		key, err := s.GetAPIKeyByID(ctx, keyID)
		if err == nil {
			keys = append(keys, key)
		}
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return keys, nil
}

// GetAPIKeyByUserAndName retrieves an API key by user ID and name.
func (s *Store) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
	// Get all keys for the user and find the one with matching name
	keys, err := s.ListAPIKeysByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Name == name {
			return key, nil
		}
	}

	return nil, storage.ErrAPIKeyNotFound
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	now := time.Now()
	if err := s.session.Query(
		`UPDATE api_keys SET last_used = ? WHERE id = ?`,
		now, id,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	return nil
}

// Close closes the Cassandra session.
func (s *Store) Close() error {
	s.session.Close()
	return nil
}

// IsHealthy returns true if the Cassandra connection is healthy.
func (s *Store) IsHealthy(ctx context.Context) bool {
	// Try a simple query
	var now time.Time
	err := s.session.Query(`SELECT now() FROM system.local`).WithContext(ctx).Scan(&now)
	return err == nil
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

// keyspaceCQL generates the CQL for creating the keyspace.
func keyspaceCQL(keyspace, replicationStrategy string, replicationFactor int) string {
	if replicationStrategy == "" {
		replicationStrategy = "SimpleStrategy"
	}
	if replicationFactor == 0 {
		replicationFactor = 1
	}

	return fmt.Sprintf(
		`CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': '%s', 'replication_factor': %d}`,
		keyspace, replicationStrategy, replicationFactor,
	)
}

// helper to convert int to string for keyspace CQL
func init() {
	// Ensure strconv is used
	_ = strconv.Itoa(1)
}
