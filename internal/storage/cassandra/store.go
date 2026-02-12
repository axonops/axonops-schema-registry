// Package cassandra provides a Cassandra storage implementation.
package cassandra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/hamba/avro/v2"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Config holds Cassandra connection configuration.
type Config struct {
	Hosts            []string      `json:"hosts" yaml:"hosts"`
	Port             int           `json:"port" yaml:"port"`
	Keyspace         string        `json:"keyspace" yaml:"keyspace"`
	Username         string        `json:"username" yaml:"username"`
	Password         string        `json:"password" yaml:"password"`
	LocalDC          string        `json:"local_dc" yaml:"local_dc"`
	Consistency      string        `json:"consistency" yaml:"consistency"`
	ReadConsistency  string        `json:"read_consistency" yaml:"read_consistency"`
	WriteConsistency string        `json:"write_consistency" yaml:"write_consistency"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
	ConnectTimeout   time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	Migrate          bool          `json:"migrate" yaml:"migrate"`
	SubjectBuckets   int           `json:"subject_buckets" yaml:"subject_buckets"`

	// MaxRetries for CAS operations (ID allocation, version allocation)
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
}

// Store implements storage.Storage on Cassandra.
type Store struct {
	cfg              Config
	cluster          *gocql.ClusterConfig
	session          *gocql.Session
	readConsistency  gocql.Consistency
	writeConsistency gocql.Consistency
}

// NewStore connects to Cassandra and optionally runs migrations.
func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	// Apply defaults
	if len(cfg.Hosts) == 0 {
		cfg.Hosts = []string{"127.0.0.1"}
	}
	if cfg.Port == 0 {
		cfg.Port = 9042
	}
	if cfg.Keyspace == "" {
		cfg.Keyspace = "axonops_schema_registry"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.SubjectBuckets <= 0 {
		cfg.SubjectBuckets = 16
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 50
	}

	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = cfg.Port
	cluster.Keyspace = cfg.Keyspace
	cluster.Timeout = cfg.Timeout
	cluster.ConnectTimeout = cfg.ConnectTimeout

	if cfg.LocalDC != "" {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(cfg.LocalDC)
	}
	if cfg.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: cfg.Username, Password: cfg.Password}
	}

	// Parse consistency levels
	defaultConsistency := gocql.LocalQuorum
	if cfg.Consistency != "" {
		c, err := parseConsistency(cfg.Consistency)
		if err != nil {
			return nil, err
		}
		defaultConsistency = c
	}
	cluster.Consistency = defaultConsistency

	readConsistency := defaultConsistency
	writeConsistency := defaultConsistency
	if cfg.ReadConsistency != "" {
		c, err := parseConsistency(cfg.ReadConsistency)
		if err != nil {
			return nil, err
		}
		readConsistency = c
	}
	if cfg.WriteConsistency != "" {
		c, err := parseConsistency(cfg.WriteConsistency)
		if err != nil {
			return nil, err
		}
		writeConsistency = c
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Cassandra: %w", err)
	}

	s := &Store{
		cfg:              cfg,
		cluster:          cluster,
		session:          session,
		readConsistency:  readConsistency,
		writeConsistency: writeConsistency,
	}

	if cfg.Migrate {
		if err := Migrate(session, cfg.Keyspace); err != nil {
			session.Close()
			return nil, err
		}
	}

	return s, nil
}

// Close closes the Cassandra session.
func (s *Store) Close() error {
	if s.session != nil {
		s.session.Close()
	}
	return nil
}

// IsHealthy checks if the Cassandra connection is healthy.
func (s *Store) IsHealthy(ctx context.Context) bool {
	var now time.Time
	err := s.session.Query(`SELECT now() FROM system.local`).WithContext(ctx).Scan(&now)
	return err == nil
}

// readQuery creates a query with read consistency.
func (s *Store) readQuery(stmt string, values ...interface{}) *gocql.Query {
	return s.session.Query(stmt, values...).Consistency(s.readConsistency)
}

// writeQuery creates a query with write consistency.
func (s *Store) writeQuery(stmt string, values ...interface{}) *gocql.Query {
	return s.session.Query(stmt, values...).Consistency(s.writeConsistency)
}

// ---------- ID Allocation ----------

// NextID returns a new globally unique schema ID using sequential allocation.
// Uses Cassandra LWT (Lightweight Transactions) to guarantee strict sequential
// IDs across all instances, matching Confluent Schema Registry behavior.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	for attempt := 0; attempt < s.cfg.MaxRetries; attempt++ {
		var current int
		if err := s.session.Query(
			fmt.Sprintf(`SELECT next_id FROM %s.id_alloc WHERE name = ?`, qident(s.cfg.Keyspace)),
			"schema_id",
		).WithContext(ctx).Scan(&current); err != nil {
			if errors.Is(err, gocql.ErrNotFound) {
				// Initialize if not exists
				applied, err := casApplied(
					s.session.Query(
						fmt.Sprintf(`INSERT INTO %s.id_alloc (name, next_id) VALUES (?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
						"schema_id", 2, // Start at 2, return 1 as first ID
					).WithContext(ctx),
				)
				if err != nil {
					return 0, err
				}
				if applied {
					return 1, nil
				}
				continue // Someone else initialized, retry
			}
			return 0, err
		}

		// Atomically increment: SET next_id = current + 1 IF next_id = current
		applied, err := casApplied(
			s.session.Query(
				fmt.Sprintf(`UPDATE %s.id_alloc SET next_id = ? WHERE name = ? IF next_id = ?`, qident(s.cfg.Keyspace)),
				current+1, "schema_id", current,
			).WithContext(ctx),
		)
		if err != nil {
			return 0, err
		}
		if !applied {
			continue // Contention, retry
		}
		return int64(current), nil
	}
	return 0, errors.New("failed to allocate schema ID: too much contention")
}

// GetMaxSchemaID returns the highest schema ID currently assigned.
func (s *Store) GetMaxSchemaID(ctx context.Context) (int64, error) {
	var nextID int
	if err := s.session.Query(
		fmt.Sprintf(`SELECT next_id FROM %s.id_alloc WHERE name = ?`, qident(s.cfg.Keyspace)),
		"schema_id",
	).WithContext(ctx).Scan(&nextID); err != nil {
		return 0, fmt.Errorf("failed to get max schema ID: %w", err)
	}
	// next_id is the next ID to assign, so max assigned is next_id - 1
	return int64(nextID - 1), nil
}

// ImportSchema inserts a schema with a specified ID (for migration).
// Returns ErrSchemaIDConflict if the ID already exists.
func (s *Store) ImportSchema(ctx context.Context, record *storage.SchemaRecord) error {
	if record == nil {
		return errors.New("schema is nil")
	}
	if record.ID <= 0 {
		return errors.New("schema ID is required for import")
	}
	if record.Subject == "" {
		return errors.New("subject is required")
	}
	if record.Version <= 0 {
		return errors.New("version is required")
	}
	if record.SchemaType == "" {
		record.SchemaType = storage.SchemaTypeAvro
	}

	canonical := canonicalize(string(record.SchemaType), record.Schema)
	fp := record.Fingerprint
	if fp == "" {
		fp = fingerprint(canonical)
	}
	record.Fingerprint = fp

	// Check if schema ID already exists
	var existingType string
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(record.ID),
	).WithContext(ctx).Scan(&existingType)
	if err == nil {
		return storage.ErrSchemaIDConflict
	}
	if !errors.Is(err, gocql.ErrNotFound) {
		return err
	}

	// Check if version already exists for this subject
	var existingSchemaID int
	err = s.readQuery(
		fmt.Sprintf(`SELECT schema_id FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		record.Subject, record.Version,
	).WithContext(ctx).Scan(&existingSchemaID)
	if err == nil {
		return storage.ErrSchemaExists
	}
	if !errors.Is(err, gocql.ErrNotFound) {
		return err
	}

	createdUUID := gocql.TimeUUID()

	// Insert into schemas_by_id
	err = s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.schemas_by_id (schema_id, schema_type, fingerprint, schema_text, canonical_text, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		int(record.ID), string(record.SchemaType), fp, record.Schema, canonical, createdUUID,
	).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to insert schema_by_id: %w", err)
	}

	// Insert into schemas_by_fingerprint
	err = s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.schemas_by_fingerprint (fingerprint, schema_id, schema_type, schema_text, canonical_text, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		fp, int(record.ID), string(record.SchemaType), record.Schema, canonical, createdUUID,
	).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to insert schemas_by_fingerprint: %w", err)
	}

	// Insert into subject_versions
	err = s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.subject_versions (subject, version, schema_id, deleted, created_at)
			VALUES (?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		record.Subject, record.Version, int(record.ID), false, createdUUID,
	).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to insert subject_versions: %w", err)
	}

	// Update subject_latest if this is the highest version
	latestVersion, _, exists, err := s.getSubjectLatest(ctx, record.Subject)
	if err != nil {
		return err
	}
	if !exists || record.Version > latestVersion {
		err = s.writeQuery(
			fmt.Sprintf(`INSERT INTO %s.subject_latest (subject, latest_version, latest_schema_id, updated_at)
				VALUES (?, ?, ?, now())`, qident(s.cfg.Keyspace)),
			record.Subject, record.Version, int(record.ID),
		).WithContext(ctx).Exec()
		if err != nil {
			return fmt.Errorf("failed to update subject_latest: %w", err)
		}
	}

	// Track subject in bucketed table
	bucket := s.subjectBucket(record.Subject)
	_ = s.session.Query(
		fmt.Sprintf(`INSERT INTO %s.subjects (bucket, subject) VALUES (?, ?)`, qident(s.cfg.Keyspace)),
		bucket, record.Subject,
	).WithContext(ctx).Exec()

	// Write schema references if any
	if len(record.References) > 0 {
		for _, ref := range record.References {
			_ = s.writeQuery(
				fmt.Sprintf(`INSERT INTO %s.schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
				int(record.ID), ref.Name, ref.Subject, ref.Version,
			).WithContext(ctx).Exec()
			_ = s.writeQuery(
				fmt.Sprintf(`INSERT INTO %s.references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
				ref.Subject, ref.Version, record.Subject, record.Version,
			).WithContext(ctx).Exec()
		}
	}

	record.CreatedAt = createdUUID.Time()
	record.Deleted = false

	return nil
}

// SetNextID sets the ID sequence to start from the given value.
// Used after import to prevent ID conflicts.
func (s *Store) SetNextID(ctx context.Context, id int64) error {
	// Use LWT to set the next_id value
	for attempt := 0; attempt < s.cfg.MaxRetries; attempt++ {
		var current int
		err := s.session.Query(
			fmt.Sprintf(`SELECT next_id FROM %s.id_alloc WHERE name = ?`, qident(s.cfg.Keyspace)),
			"schema_id",
		).WithContext(ctx).Scan(&current)

		if errors.Is(err, gocql.ErrNotFound) {
			// Initialize if not exists
			applied, err := casApplied(
				s.session.Query(
					fmt.Sprintf(`INSERT INTO %s.id_alloc (name, next_id) VALUES (?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
					"schema_id", int(id),
				).WithContext(ctx),
			)
			if err != nil {
				return err
			}
			if applied {
				return nil
			}
			continue
		}
		if err != nil {
			return err
		}

		// CAS update
		applied, err := casApplied(
			s.session.Query(
				fmt.Sprintf(`UPDATE %s.id_alloc SET next_id = ? WHERE name = ? IF next_id = ?`, qident(s.cfg.Keyspace)),
				int(id), "schema_id", current,
			).WithContext(ctx),
		)
		if err != nil {
			return err
		}
		if applied {
			return nil
		}
		// Contention, retry
	}
	return errors.New("failed to set next ID: too much contention")
}

// ---------- Schema Operations ----------

// CreateSchema registers a schema under a subject.
// Implements global schema deduplication: same canonical schema => same ID.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	if record == nil {
		return errors.New("schema is nil")
	}
	if record.Subject == "" {
		return errors.New("subject is required")
	}
	if record.SchemaType == "" {
		return errors.New("schema_type is required")
	}
	if record.Schema == "" {
		return errors.New("schema is required")
	}

	canonical := canonicalize(string(record.SchemaType), record.Schema)
	fp := record.Fingerprint
	if fp == "" {
		fp = fingerprint(canonical)
	}

	// Ensure global schema exists (deduplication)
	schemaID, _, err := s.ensureGlobalSchema(ctx, string(record.SchemaType), record.Schema, canonical, fp)
	if err != nil {
		return err
	}

	// Check if subject already has this schema
	if existing, err := s.findSchemaInSubject(ctx, record.Subject, schemaID); err == nil && existing != nil {
		// Schema already exists in subject, update record with existing info
		record.ID = existing.ID
		record.Version = existing.Version
		record.Fingerprint = fp
		record.CreatedAt = existing.CreatedAt
		return storage.ErrSchemaExists
	}

	// Allocate next subject version and persist atomically.
	// Strategy: Write data FIRST, then CAS update subject_latest to "publish" it.
	// This ensures no version gaps - if CAS fails, we retry with same data.
	for attempt := 0; attempt < s.cfg.MaxRetries; attempt++ {
		latestVersion, latestSchemaID, exists, err := s.getSubjectLatest(ctx, record.Subject)
		if err != nil {
			return err
		}

		newVersion := 1
		if exists {
			newVersion = latestVersion + 1
		}

		createdUUID := gocql.TimeUUID()

		// Step 1: Write subject_versions FIRST (idempotent with IF NOT EXISTS)
		// This ensures the data exists before we "publish" via subject_latest
		applied, err := casApplied(
			s.session.Query(
				fmt.Sprintf(`INSERT INTO %s.subject_versions (subject, version, schema_id, deleted, created_at)
					VALUES (?, ?, ?, ?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
				record.Subject, newVersion, int(schemaID), false, createdUUID,
			).WithContext(ctx),
		)
		if err != nil {
			return err
		}
		if !applied {
			// Version already exists - either we're retrying or there's contention
			// Check if it has our schema_id (retry case) or different (contention)
			var existingSchemaID int
			err := s.session.Query(
				fmt.Sprintf(`SELECT schema_id FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
				record.Subject, newVersion,
			).WithContext(ctx).Scan(&existingSchemaID)
			if err != nil {
				continue // Retry with new version
			}
			if int64(existingSchemaID) != schemaID {
				continue // Different schema claimed this version, retry with next
			}
			// Our schema already has this version, proceed to publish
		}

		// Step 2: CAS update subject_latest to "publish" this version
		if !exists {
			applied, err = casApplied(
				s.session.Query(
					fmt.Sprintf(`INSERT INTO %s.subject_latest (subject, latest_version, latest_schema_id, updated_at)
						VALUES (?, ?, ?, now()) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
					record.Subject, newVersion, int(schemaID),
				).WithContext(ctx),
			)
		} else {
			applied, err = casApplied(
				s.session.Query(
					fmt.Sprintf(`UPDATE %s.subject_latest SET latest_version = ?, latest_schema_id = ?, updated_at = now()
						WHERE subject = ? IF latest_version = ? AND latest_schema_id = ?`, qident(s.cfg.Keyspace)),
					newVersion, int(schemaID), record.Subject, latestVersion, latestSchemaID,
				).WithContext(ctx),
			)
		}
		if err != nil {
			return err
		}
		if !applied {
			// subject_latest was updated by another writer, retry
			continue
		}

		// Step 3: Track subject in bucketed table (best effort, not critical)
		bucket := s.subjectBucket(record.Subject)
		_ = s.session.Query(
			fmt.Sprintf(`INSERT INTO %s.subjects (bucket, subject) VALUES (?, ?)`, qident(s.cfg.Keyspace)),
			bucket, record.Subject,
		).WithContext(ctx).Exec()

		// Step 4: Write schema references if any
		if len(record.References) > 0 {
			for _, ref := range record.References {
				_ = s.writeQuery(
					fmt.Sprintf(`INSERT INTO %s.schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
					int(schemaID), ref.Name, ref.Subject, ref.Version,
				).WithContext(ctx).Exec()
				_ = s.writeQuery(
					fmt.Sprintf(`INSERT INTO %s.references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
					ref.Subject, ref.Version, record.Subject, newVersion,
				).WithContext(ctx).Exec()
			}
		}

		// Update record with assigned values
		record.ID = schemaID
		record.Version = newVersion
		record.Fingerprint = fp
		record.Deleted = false
		record.CreatedAt = createdUUID.Time()
		return nil
	}

	return errors.New("failed to create schema due to contention (too many retries)")
}

func (s *Store) ensureGlobalSchema(ctx context.Context, schemaType, schemaText, canonical, fp string) (schemaID int64, createdAt time.Time, err error) {
	// Try by fingerprint first (global dedup)
	var existingID int
	var createdUUID gocql.UUID
	err = s.readQuery(
		fmt.Sprintf(`SELECT schema_id, created_at FROM %s.schemas_by_fingerprint WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
		fp,
	).WithContext(ctx).Scan(&existingID, &createdUUID)
	if err == nil {
		// Schema exists in fingerprint table, ensure it also exists in schemas_by_id
		// (handles case where previous write to schemas_by_id failed)
		if err := s.ensureSchemaByIDExists(ctx, int64(existingID), schemaType, schemaText, canonical, fp, createdUUID); err != nil {
			return 0, time.Time{}, err
		}
		return int64(existingID), createdUUID.Time(), nil
	}
	if !errors.Is(err, gocql.ErrNotFound) {
		return 0, time.Time{}, err
	}

	// Allocate new ID and try to insert with IF NOT EXISTS
	newID, err := s.NextID(ctx)
	if err != nil {
		return 0, time.Time{}, err
	}
	createdUUID = gocql.TimeUUID()

	applied, err := casApplied(
		s.session.Query(
			fmt.Sprintf(`INSERT INTO %s.schemas_by_fingerprint (fingerprint, schema_id, schema_type, schema_text, canonical_text, created_at)
				VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
			fp, int(newID), schemaType, schemaText, canonical, createdUUID,
		).WithContext(ctx),
	)
	if err != nil {
		return 0, time.Time{}, err
	}
	if !applied {
		// Lost race: read existing and ensure schemas_by_id exists
		err = s.readQuery(
			fmt.Sprintf(`SELECT schema_id, created_at FROM %s.schemas_by_fingerprint WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
			fp,
		).WithContext(ctx).Scan(&existingID, &createdUUID)
		if err != nil {
			if errors.Is(err, gocql.ErrNotFound) {
				return 0, time.Time{}, errors.New("schema fingerprint contention: row not found after failed CAS")
			}
			return 0, time.Time{}, err
		}
		if err := s.ensureSchemaByIDExists(ctx, int64(existingID), schemaType, schemaText, canonical, fp, createdUUID); err != nil {
			return 0, time.Time{}, err
		}
		return int64(existingID), createdUUID.Time(), nil
	}

	// We won: insert into schemas_by_id too
	if err := s.ensureSchemaByIDExists(ctx, newID, schemaType, schemaText, canonical, fp, createdUUID); err != nil {
		return 0, time.Time{}, err
	}

	return newID, createdUUID.Time(), nil
}

// ensureSchemaByIDExists ensures the schema exists in schemas_by_id table.
// Uses INSERT IF NOT EXISTS to be idempotent.
func (s *Store) ensureSchemaByIDExists(ctx context.Context, schemaID int64, schemaType, schemaText, canonical, fp string, createdUUID gocql.UUID) error {
	// Use IF NOT EXISTS to make this idempotent
	_, err := casApplied(
		s.session.Query(
			fmt.Sprintf(`INSERT INTO %s.schemas_by_id (schema_id, schema_type, fingerprint, schema_text, canonical_text, created_at)
				VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
			int(schemaID), schemaType, fp, schemaText, canonical, createdUUID,
		).WithContext(ctx),
	)
	// We don't care if it was applied or not - either we inserted it, or it already exists
	return err
}

func (s *Store) findSchemaInSubject(ctx context.Context, subject string, schemaID int64) (*storage.SchemaRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, schema_id, deleted, created_at FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()

	var version, versionSchemaID int
	var deleted bool
	var createdUUID gocql.UUID
	for iter.Scan(&version, &versionSchemaID, &deleted, &createdUUID) {
		if deleted {
			continue
		}
		if int64(versionSchemaID) == schemaID {
			iter.Close()
			return s.GetSchemaBySubjectVersion(ctx, subject, version)
		}
	}
	iter.Close()
	return nil, nil
}

// GetSchemaByID retrieves a schema by its global ID.
func (s *Store) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	var schemaType, schemaText string
	var createdUUID gocql.UUID

	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type, schema_text, created_at FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Scan(&schemaType, &schemaText, &createdUUID)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, err
	}

	rec := &storage.SchemaRecord{
		ID:         id,
		SchemaType: storage.SchemaType(schemaType),
		Schema:     schemaText,
		CreatedAt:  createdUUID.Time(),
	}

	// Load references for this schema
	refIter := s.readQuery(
		fmt.Sprintf(`SELECT name, ref_subject, ref_version FROM %s.schema_references WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Iter()
	var refName, refSubject string
	var refVersion int
	for refIter.Scan(&refName, &refSubject, &refVersion) {
		rec.References = append(rec.References, storage.Reference{
			Name:    refName,
			Subject: refSubject,
			Version: refVersion,
		})
	}
	refIter.Close()

	// Look up the first subject/version that uses this schema ID
	versions, err := s.GetVersionsBySchemaID(ctx, id, false)
	if err == nil && len(versions) > 0 {
		rec.Subject = versions[0].Subject
		rec.Version = versions[0].Version
	}

	return rec, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	if subject == "" {
		return nil, storage.ErrVersionNotFound
	}

	// Handle "latest" version (-1)
	if version == -1 {
		return s.GetLatestSchema(ctx, subject)
	}

	if version <= 0 {
		return nil, storage.ErrVersionNotFound
	}

	var schemaID int
	var deleted bool
	var createdUUID gocql.UUID
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, deleted, created_at FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Scan(&schemaID, &deleted, &createdUUID)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			// Check if subject exists at all to distinguish ErrSubjectNotFound vs ErrVersionNotFound
			_, _, subjectExists, subErr := s.getSubjectLatest(ctx, subject)
			if subErr != nil {
				return nil, subErr
			}
			if !subjectExists {
				return nil, storage.ErrSubjectNotFound
			}
			return nil, storage.ErrVersionNotFound
		}
		return nil, err
	}

	if deleted {
		return nil, storage.ErrVersionNotFound
	}

	rec, err := s.GetSchemaByID(ctx, int64(schemaID))
	if err != nil {
		return nil, err
	}
	rec.Subject = subject
	rec.Version = version
	rec.Deleted = deleted
	rec.CreatedAt = createdUUID.Time()
	return rec, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, schema_id, deleted, created_at FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()

	type versionInfo struct {
		version   int
		schemaID  int
		deleted   bool
		createdAt gocql.UUID
	}
	var entries []versionInfo
	var vi versionInfo
	hasAnyRows := false
	for iter.Scan(&vi.version, &vi.schemaID, &vi.deleted, &vi.createdAt) {
		hasAnyRows = true
		if includeDeleted || !vi.deleted {
			entries = append(entries, vi)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	if !hasAnyRows {
		return nil, storage.ErrSubjectNotFound
	}

	if len(entries) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].version < entries[j].version })
	out := make([]*storage.SchemaRecord, 0, len(entries))
	for _, e := range entries {
		rec, err := s.GetSchemaByID(ctx, int64(e.schemaID))
		if err != nil {
			continue
		}
		rec.Subject = subject
		rec.Version = e.version
		rec.Deleted = e.deleted
		rec.CreatedAt = e.createdAt.Time()
		out = append(out, rec)
	}
	return out, nil
}

// GetSchemaByFingerprint retrieves a schema by fingerprint within a subject.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fp string, includeDeleted bool) (*storage.SchemaRecord, error) {
	// First get schema by global fingerprint
	globalRec, err := s.GetSchemaByGlobalFingerprint(ctx, fp)
	if err != nil {
		return nil, err
	}

	// Find version in subject
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, schema_id, deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()

	var version, schemaID int
	var deleted bool
	for iter.Scan(&version, &schemaID, &deleted) {
		if (includeDeleted || !deleted) && int64(schemaID) == globalRec.ID {
			iter.Close()
			// Build result directly instead of calling GetSchemaBySubjectVersion,
			// which rejects deleted versions even when includeDeleted=true.
			rec, err := s.GetSchemaByID(ctx, globalRec.ID)
			if err != nil {
				return nil, err
			}
			rec.Subject = subject
			rec.Version = version
			rec.Deleted = deleted
			rec.Fingerprint = fp
			return rec, nil
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	// Check if subject exists at all
	_, _, subjectExists, subErr := s.getSubjectLatest(ctx, subject)
	if subErr != nil {
		return nil, subErr
	}
	if !subjectExists {
		return nil, storage.ErrSubjectNotFound
	}

	return nil, storage.ErrSchemaNotFound
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint (global lookup).
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, fp string) (*storage.SchemaRecord, error) {
	var schemaID int
	var schemaType, schemaText string
	var createdUUID gocql.UUID

	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, schema_type, schema_text, created_at FROM %s.schemas_by_fingerprint WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
		fp,
	).WithContext(ctx).Scan(&schemaID, &schemaType, &schemaText, &createdUUID)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, err
	}

	return &storage.SchemaRecord{
		ID:          int64(schemaID),
		SchemaType:  storage.SchemaType(schemaType),
		Schema:      schemaText,
		Fingerprint: fp,
		CreatedAt:   createdUUID.Time(),
	}, nil
}

// GetLatestSchema retrieves the latest non-deleted schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, subject string) (*storage.SchemaRecord, error) {
	_, _, ok, err := s.getSubjectLatest(ctx, subject)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, storage.ErrSubjectNotFound
	}
	// Scan all versions to find the latest non-deleted one
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()
	latestVersion := 0
	var version int
	var deleted bool
	for iter.Scan(&version, &deleted) {
		if !deleted && version > latestVersion {
			latestVersion = version
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	if latestVersion == 0 {
		return nil, storage.ErrSubjectNotFound
	}
	return s.GetSchemaBySubjectVersion(ctx, subject, latestVersion)
}

func (s *Store) getSubjectLatest(ctx context.Context, subject string) (latestVersion int, latestSchemaID int, exists bool, err error) {
	var v, sid int
	var updated gocql.UUID
	err = s.readQuery(
		fmt.Sprintf(`SELECT latest_version, latest_schema_id, updated_at FROM %s.subject_latest WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Scan(&v, &sid, &updated)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return 0, 0, false, nil
		}
		return 0, 0, false, err
	}
	return v, sid, true, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	if subject == "" || version <= 0 {
		return storage.ErrVersionNotFound
	}

	// Check if the version exists
	var existingSchemaID int
	var deleted bool
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, deleted FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Scan(&existingSchemaID, &deleted)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			// Check if subject exists at all
			_, _, subjectExists, subErr := s.getSubjectLatest(ctx, subject)
			if subErr != nil {
				return subErr
			}
			if !subjectExists {
				return storage.ErrSubjectNotFound
			}
			return storage.ErrVersionNotFound
		}
		return err
	}

	if permanent {
		if !deleted {
			return storage.ErrVersionNotSoftDeleted
		}
		if err := s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
			subject, version,
		).WithContext(ctx).Exec(); err != nil {
			return err
		}
		// Check if any subject_versions still reference this schema_id.
		// If none, clean up schemas_by_id and schemas_by_fingerprint.
		s.cleanupOrphanedSchema(ctx, existingSchemaID)
		return nil
	}
	return s.writeQuery(
		fmt.Sprintf(`UPDATE %s.subject_versions SET deleted = true WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Exec()
}

// cleanupOrphanedSchema removes a schema from schemas_by_id and schemas_by_fingerprint
// if no subject_versions reference it anymore.
func (s *Store) cleanupOrphanedSchema(ctx context.Context, schemaID int) {
	// Check all subjects to see if any still reference this schema_id
	allSubjects, err := s.ListSubjects(ctx, true)
	if err != nil {
		return
	}
	for _, subject := range allSubjects {
		iter := s.readQuery(
			fmt.Sprintf(`SELECT schema_id FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
			subject,
		).WithContext(ctx).Iter()
		var sid int
		for iter.Scan(&sid) {
			if sid == schemaID {
				iter.Close()
				return // Still referenced, don't clean up
			}
		}
		iter.Close()
	}

	// No subject_versions reference this schema_id — clean up
	var fingerprint string
	if err := s.readQuery(
		fmt.Sprintf(`SELECT fingerprint FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Scan(&fingerprint); err == nil && fingerprint != "" {
		_ = s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.schemas_by_fingerprint WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
			fingerprint,
		).WithContext(ctx).Exec()
	}
	_ = s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Exec()
	// Also clean up references
	_ = s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.schema_references WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Exec()
}

// ---------- Subject Operations ----------

// ListSubjects returns all subjects.
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	subjectSet := make(map[string]bool)

	for bucket := 0; bucket < s.cfg.SubjectBuckets; bucket++ {
		iter := s.readQuery(
			fmt.Sprintf(`SELECT subject FROM %s.subjects WHERE bucket = ?`, qident(s.cfg.Keyspace)),
			bucket,
		).WithContext(ctx).Iter()

		var subject string
		for iter.Scan(&subject) {
			subjectSet[subject] = true
		}
		iter.Close()
	}

	// If not including deleted, filter out subjects with no non-deleted versions
	subjects := make([]string, 0, len(subjectSet))
	for subject := range subjectSet {
		if !includeDeleted {
			// Check if subject has any non-deleted versions
			hasActive := false
			iter := s.readQuery(
				fmt.Sprintf(`SELECT deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
				subject,
			).WithContext(ctx).Iter()
			var deleted bool
			for iter.Scan(&deleted) {
				if !deleted {
					hasActive = true
					break
				}
			}
			iter.Close()
			if !hasActive {
				continue
			}
		}
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	return subjects, nil
}

// DeleteSubject soft-deletes or permanently deletes all versions of a subject.
func (s *Store) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	if subject == "" {
		return nil, nil
	}

	iter := s.session.Query(
		fmt.Sprintf(`SELECT version, deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()

	type versionInfo struct {
		version int
		deleted bool
	}
	var allVersions []versionInfo
	var version int
	var deleted bool
	for iter.Scan(&version, &deleted) {
		allVersions = append(allVersions, versionInfo{version, deleted})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	if len(allVersions) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var deletedVersions []int
	if permanent {
		// Check that all versions are soft-deleted first
		for _, vi := range allVersions {
			if !vi.deleted {
				return nil, storage.ErrSubjectNotSoftDeleted
			}
			deletedVersions = append(deletedVersions, vi.version)
		}
	} else {
		// Soft-delete: collect non-deleted versions
		allSoftDeleted := true
		for _, vi := range allVersions {
			if !vi.deleted {
				allSoftDeleted = false
				deletedVersions = append(deletedVersions, vi.version)
			}
		}
		if allSoftDeleted {
			return nil, storage.ErrSubjectDeleted
		}
	}

	if len(deletedVersions) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	for _, v := range deletedVersions {
		if err := s.DeleteSchema(ctx, subject, v, permanent); err != nil {
			return nil, err
		}
	}

	if permanent {
		// Remove from subject_latest
		_ = s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.subject_latest WHERE subject = ?`, qident(s.cfg.Keyspace)),
			subject,
		).WithContext(ctx).Exec()
		// Remove from subjects bucket
		bucket := s.subjectBucket(subject)
		_ = s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.subjects WHERE bucket = ? AND subject = ?`, qident(s.cfg.Keyspace)),
			bucket, subject,
		).WithContext(ctx).Exec()
	}

	sort.Ints(deletedVersions)
	return deletedVersions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	_, _, exists, err := s.getSubjectLatest(ctx, subject)
	if err != nil || !exists {
		return false, err
	}
	// Check if there are any non-deleted versions
	iter := s.readQuery(
		fmt.Sprintf(`SELECT deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()
	var deleted bool
	for iter.Scan(&deleted) {
		if !deleted {
			iter.Close()
			return true, nil
		}
	}
	if err := iter.Close(); err != nil {
		return false, err
	}
	return false, nil
}

func (s *Store) subjectBucket(subject string) int {
	var hash int
	for _, c := range subject {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % s.cfg.SubjectBuckets
}

// ---------- Schema References ----------

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT schema_subject, schema_version FROM %s.references_by_target WHERE ref_subject = ? AND ref_version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Iter()

	var refs []storage.SubjectVersion
	var ref storage.SubjectVersion
	for iter.Scan(&ref.Subject, &ref.Version) {
		refs = append(refs, ref)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return refs, nil
}

// GetSubjectsBySchemaID returns subjects using the given schema ID.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	// Verify schema ID exists first (direct query to avoid recursion with GetSchemaByID)
	var dummy string
	if err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Scan(&dummy); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, err
	}

	// Need to scan all subjects to find which use this schema ID
	allSubjects, err := s.ListSubjects(ctx, true)
	if err != nil {
		return nil, err
	}

	subjectSet := make(map[string]bool)
	for _, subject := range allSubjects {
		iter := s.readQuery(
			fmt.Sprintf(`SELECT schema_id, deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
			subject,
		).WithContext(ctx).Iter()

		var schemaID int
		var deleted bool
		for iter.Scan(&schemaID, &deleted) {
			if int64(schemaID) == id && (includeDeleted || !deleted) {
				subjectSet[subject] = true
				break
			}
		}
		iter.Close()
	}

	subjects := make([]string, 0, len(subjectSet))
	for subject := range subjectSet {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	return subjects, nil
}

// GetVersionsBySchemaID returns subject-version pairs using the given schema ID.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	// Verify schema ID exists first (direct query to avoid recursion with GetSchemaByID)
	var dummy string
	if err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Scan(&dummy); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrSchemaNotFound
		}
		return nil, err
	}

	allSubjects, err := s.ListSubjects(ctx, true)
	if err != nil {
		return nil, err
	}

	var results []storage.SubjectVersion
	for _, subject := range allSubjects {
		iter := s.readQuery(
			fmt.Sprintf(`SELECT version, schema_id, deleted FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
			subject,
		).WithContext(ctx).Iter()

		var version, schemaID int
		var deleted bool
		for iter.Scan(&version, &schemaID, &deleted) {
			if int64(schemaID) == id && (includeDeleted || !deleted) {
				results = append(results, storage.SubjectVersion{Subject: subject, Version: version})
			}
		}
		iter.Close()
	}
	return results, nil
}

// ListSchemas lists schemas with optional filtering.
func (s *Store) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	if params == nil {
		params = &storage.ListSchemasParams{}
	}

	subjects, err := s.ListSubjects(ctx, params.Deleted)
	if err != nil {
		return nil, err
	}

	var results []*storage.SchemaRecord
	for _, subject := range subjects {
		if params.SubjectPrefix != "" && !strings.HasPrefix(subject, params.SubjectPrefix) {
			continue
		}

		if params.LatestOnly {
			rec, err := s.GetLatestSchema(ctx, subject)
			if err == nil {
				results = append(results, rec)
			}
		} else {
			recs, err := s.GetSchemasBySubject(ctx, subject, params.Deleted)
			if err == nil {
				results = append(results, recs...)
			}
		}
	}

	// Apply offset and limit
	if params.Offset > 0 {
		if params.Offset >= len(results) {
			return []*storage.SchemaRecord{}, nil
		}
		results = results[params.Offset:]
	}
	if params.Limit > 0 && len(results) > params.Limit {
		results = results[:params.Limit]
	}

	return results, nil
}

// ---------- Config Operations ----------

// GetConfig retrieves the compatibility config for a subject.
func (s *Store) GetConfig(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	if subject == "" {
		return s.GetGlobalConfig(ctx)
	}
	var compat string
	err := s.readQuery(
		fmt.Sprintf(`SELECT compatibility FROM %s.subject_configs WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Scan(&compat)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ConfigRecord{Subject: subject, CompatibilityLevel: compat}, nil
}

// SetConfig sets the compatibility config for a subject.
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
	if config == nil {
		return errors.New("config is nil")
	}
	compat := normalizeCompat(config.CompatibilityLevel)
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.subject_configs (subject, compatibility, updated_at) VALUES (?, ?, now())`, qident(s.cfg.Keyspace)),
		subject, compat,
	).WithContext(ctx).Exec()
}

// DeleteConfig deletes a compatibility config for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	if subject == "" {
		return nil
	}
	// Check if config exists first (Cassandra DELETE is idempotent)
	_, err := s.GetConfig(ctx, subject)
	if err != nil {
		return err
	}
	return s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.subject_configs WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Exec()
}

// GetGlobalConfig retrieves the global compatibility config.
func (s *Store) GetGlobalConfig(ctx context.Context) (*storage.ConfigRecord, error) {
	var compat string
	err := s.readQuery(
		fmt.Sprintf(`SELECT compatibility FROM %s.global_config WHERE key = ?`, qident(s.cfg.Keyspace)),
		"global",
	).WithContext(ctx).Scan(&compat)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ConfigRecord{Subject: "", CompatibilityLevel: compat}, nil
}

// SetGlobalConfig sets the global compatibility config.
func (s *Store) SetGlobalConfig(ctx context.Context, config *storage.ConfigRecord) error {
	compat := "BACKWARD"
	if config != nil {
		compat = normalizeCompat(config.CompatibilityLevel)
	}
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.global_config (key, compatibility, updated_at) VALUES (?, ?, now())`, qident(s.cfg.Keyspace)),
		"global", compat,
	).WithContext(ctx).Exec()
}

// DeleteGlobalConfig resets the global config to the default (BACKWARD).
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.global_config (key, compatibility, updated_at) VALUES (?, ?, now())`, qident(s.cfg.Keyspace)),
		"global", "BACKWARD",
	).WithContext(ctx).Exec()
}

// ---------- Mode Operations ----------

// GetMode retrieves the mode for a subject.
func (s *Store) GetMode(ctx context.Context, subject string) (*storage.ModeRecord, error) {
	if subject == "" {
		return s.GetGlobalMode(ctx)
	}
	var mode string
	err := s.readQuery(
		fmt.Sprintf(`SELECT mode FROM %s.modes WHERE key = ?`, qident(s.cfg.Keyspace)),
		"subject:"+subject,
	).WithContext(ctx).Scan(&mode)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ModeRecord{Subject: subject, Mode: mode}, nil
}

// SetMode sets the mode for a subject.
func (s *Store) SetMode(ctx context.Context, subject string, mode *storage.ModeRecord) error {
	if mode == nil {
		return errors.New("mode is nil")
	}
	key := "global"
	if subject != "" {
		key = "subject:" + subject
	}
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.modes (key, mode, updated_at) VALUES (?, ?, now())`, qident(s.cfg.Keyspace)),
		key, mode.Mode,
	).WithContext(ctx).Exec()
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, subject string) error {
	// Check if mode exists first (Cassandra DELETE is idempotent)
	_, err := s.GetMode(ctx, subject)
	if err != nil {
		return err
	}
	key := "global"
	if subject != "" {
		key = "subject:" + subject
	}
	return s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.modes WHERE key = ?`, qident(s.cfg.Keyspace)),
		key,
	).WithContext(ctx).Exec()
}

// GetGlobalMode retrieves the global mode.
func (s *Store) GetGlobalMode(ctx context.Context) (*storage.ModeRecord, error) {
	var mode string
	err := s.readQuery(
		fmt.Sprintf(`SELECT mode FROM %s.modes WHERE key = ?`, qident(s.cfg.Keyspace)),
		"global",
	).WithContext(ctx).Scan(&mode)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ModeRecord{Subject: "", Mode: mode}, nil
}

// SetGlobalMode sets the global mode.
func (s *Store) SetGlobalMode(ctx context.Context, mode *storage.ModeRecord) error {
	m := "READWRITE"
	if mode != nil {
		m = mode.Mode
	}
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.modes (key, mode, updated_at) VALUES (?, ?, now())`, qident(s.cfg.Keyspace)),
		"global", m,
	).WithContext(ctx).Exec()
}

// ---------- User Operations ----------

// CreateUser creates a new user.
func (s *Store) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if user.Username == "" {
		return errors.New("username is required")
	}

	// Check for existing user with same username
	existing, err := s.GetUserByUsername(ctx, user.Username)
	if err == nil && existing != nil {
		return storage.ErrUserExists
	}

	// Generate ID if not set
	if user.ID == 0 {
		id, err := s.NextID(ctx)
		if err != nil {
			return err
		}
		user.ID = id
	}

	nowUUID := gocql.TimeUUID()
	now := nowUUID.Time()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	createdUUID := gocql.UUIDFromTime(user.CreatedAt)
	updatedUUID := gocql.UUIDFromTime(user.UpdatedAt)

	// Use username as key in users_by_email (repurposed as users_by_username)
	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.users_by_id (user_id, email, name, password_hash, roles, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		user.ID, user.Email, user.Username, user.PasswordHash, []string{user.Role}, user.Enabled, createdUUID, updatedUUID,
	)
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.users_by_email (email, user_id, name, password_hash, roles, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		user.Username, user.ID, user.Username, user.PasswordHash, []string{user.Role}, user.Enabled, createdUUID, updatedUUID,
	)
	return s.session.ExecuteBatch(batch)
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	var email, name, pw string
	var roles []string
	var enabled bool
	var createdUUID, updatedUUID gocql.UUID
	err := s.readQuery(
		fmt.Sprintf(`SELECT email, name, password_hash, roles, enabled, created_at, updated_at FROM %s.users_by_id WHERE user_id = ?`, qident(s.cfg.Keyspace)),
		id,
	).WithContext(ctx).Scan(&email, &name, &pw, &roles, &enabled, &createdUUID, &updatedUUID)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrUserNotFound
		}
		return nil, err
	}

	role := ""
	if len(roles) > 0 {
		role = roles[0]
	}

	return &storage.UserRecord{
		ID:           id,
		Username:     name,
		Email:        email,
		PasswordHash: pw,
		Role:         role,
		Enabled:      enabled,
		CreatedAt:    createdUUID.Time(),
		UpdatedAt:    updatedUUID.Time(),
	}, nil
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	if username == "" {
		return nil, storage.ErrUserNotFound
	}
	var userID int64
	err := s.readQuery(
		fmt.Sprintf(`SELECT user_id FROM %s.users_by_email WHERE email = ?`, qident(s.cfg.Keyspace)),
		username,
	).WithContext(ctx).Scan(&userID)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrUserNotFound
		}
		return nil, err
	}

	// Fetch the full user record to get all fields
	return s.GetUserByID(ctx, userID)
}

// UpdateUser updates a user.
func (s *Store) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if user.ID == 0 {
		return errors.New("user id is required")
	}

	// Verify user exists
	existing, err := s.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}

	// Check for duplicate username if username is changing
	if existing.Username != user.Username {
		existingByName, err := s.GetUserByUsername(ctx, user.Username)
		if err == nil && existingByName.ID != user.ID {
			return storage.ErrUserExists
		}
	}

	user.UpdatedAt = time.Now()
	updatedUUID := gocql.UUIDFromTime(user.UpdatedAt)
	createdUUID := gocql.UUIDFromTime(user.CreatedAt)

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	// Update users_by_id
	batch.Query(
		fmt.Sprintf(`UPDATE %s.users_by_id SET email = ?, name = ?, password_hash = ?, roles = ?, enabled = ?, updated_at = ? WHERE user_id = ?`, qident(s.cfg.Keyspace)),
		user.Email, user.Username, user.PasswordHash, []string{user.Role}, user.Enabled, updatedUUID, user.ID,
	)

	// If username changed, delete old entry and create new one in users_by_email
	if existing.Username != user.Username {
		batch.Query(
			fmt.Sprintf(`DELETE FROM %s.users_by_email WHERE email = ?`, qident(s.cfg.Keyspace)),
			existing.Username,
		)
	}

	// Upsert users_by_email with current username
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.users_by_email (email, user_id, name, password_hash, roles, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		user.Username, user.ID, user.Username, user.PasswordHash, []string{user.Role}, user.Enabled, createdUUID, updatedUUID,
	)

	return s.session.ExecuteBatch(batch)
}

// DeleteUser deletes a user.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	u, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(fmt.Sprintf(`DELETE FROM %s.users_by_id WHERE user_id = ?`, qident(s.cfg.Keyspace)), id)
	batch.Query(fmt.Sprintf(`DELETE FROM %s.users_by_email WHERE email = ?`, qident(s.cfg.Keyspace)), u.Username)
	return s.session.ExecuteBatch(batch)
}

// ListUsers retrieves all users.
func (s *Store) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT user_id FROM %s.users_by_id`, qident(s.cfg.Keyspace)),
	).WithContext(ctx).Iter()

	var userID int64
	var out []*storage.UserRecord
	for iter.Scan(&userID) {
		u, err := s.GetUserByID(ctx, userID)
		if err == nil {
			out = append(out, u)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ---------- API Key Operations ----------

// CreateAPIKey creates a new API key.
func (s *Store) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	if key == nil {
		return errors.New("api key is nil")
	}
	if key.UserID == 0 {
		return errors.New("user_id is required")
	}
	if key.KeyHash == "" {
		return errors.New("key_hash is required")
	}

	// Generate ID if not set
	if key.ID == 0 {
		id, err := s.NextID(ctx)
		if err != nil {
			return err
		}
		key.ID = id
	}

	createdUUID := gocql.TimeUUID()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = createdUUID.Time()
	} else {
		createdUUID = gocql.UUIDFromTime(key.CreatedAt)
	}

	// Default enabled to true if not set
	if !key.Enabled {
		key.Enabled = true
	}

	// Check for duplicate hash
	if _, err := s.GetAPIKeyByHash(ctx, key.KeyHash); err == nil {
		return storage.ErrAPIKeyExists
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.api_keys_by_id (api_key_id, user_id, name, api_key_hash, key_prefix, role, enabled, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		key.ID, key.UserID, key.Name, key.KeyHash, key.KeyPrefix, key.Role, key.Enabled, createdUUID, key.ExpiresAt,
	)
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.api_keys_by_user (user_id, api_key_id, name, api_key_hash, key_prefix, role, enabled, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		key.UserID, key.ID, key.Name, key.KeyHash, key.KeyPrefix, key.Role, key.Enabled, createdUUID, key.ExpiresAt,
	)
	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.api_keys_by_hash (api_key_hash, api_key_id, user_id, name, key_prefix, role, enabled, created_at, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		key.KeyHash, key.ID, key.UserID, key.Name, key.KeyPrefix, key.Role, key.Enabled, createdUUID, key.ExpiresAt,
	)
	return s.session.ExecuteBatch(batch)
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Store) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	var userID int64
	var name, hash, keyPrefix, role string
	var enabled bool
	var createdUUID gocql.UUID
	var expiresAt, lastUsed time.Time
	err := s.readQuery(
		fmt.Sprintf(`SELECT user_id, name, api_key_hash, key_prefix, role, enabled, created_at, expires_at, last_used FROM %s.api_keys_by_id WHERE api_key_id = ?`, qident(s.cfg.Keyspace)),
		id,
	).WithContext(ctx).Scan(&userID, &name, &hash, &keyPrefix, &role, &enabled, &createdUUID, &expiresAt, &lastUsed)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrAPIKeyNotFound
		}
		return nil, err
	}

	rec := &storage.APIKeyRecord{
		ID:        id,
		UserID:    userID,
		Name:      name,
		KeyHash:   hash,
		KeyPrefix: keyPrefix,
		Role:      role,
		Enabled:   enabled,
		CreatedAt: createdUUID.Time(),
		ExpiresAt: expiresAt,
	}
	if !lastUsed.IsZero() {
		rec.LastUsed = &lastUsed
	}
	return rec, nil
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	var keyID, userID int64
	var name, keyPrefix, role string
	var enabled bool
	var createdUUID gocql.UUID
	var expiresAt, lastUsed time.Time
	err := s.readQuery(
		fmt.Sprintf(`SELECT api_key_id, user_id, name, key_prefix, role, enabled, created_at, expires_at, last_used FROM %s.api_keys_by_hash WHERE api_key_hash = ?`, qident(s.cfg.Keyspace)),
		keyHash,
	).WithContext(ctx).Scan(&keyID, &userID, &name, &keyPrefix, &role, &enabled, &createdUUID, &expiresAt, &lastUsed)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrAPIKeyNotFound
		}
		return nil, err
	}

	rec := &storage.APIKeyRecord{
		ID:        keyID,
		UserID:    userID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Role:      role,
		Enabled:   enabled,
		CreatedAt: createdUUID.Time(),
		ExpiresAt: expiresAt,
	}
	if !lastUsed.IsZero() {
		rec.LastUsed = &lastUsed
	}
	return rec, nil
}

// GetAPIKeyByUserAndName retrieves an API key by user ID and name.
func (s *Store) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
	keys, err := s.ListAPIKeysByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if k.Name == name {
			return k, nil
		}
	}
	return nil, storage.ErrAPIKeyNotFound
}

// UpdateAPIKey updates an API key.
func (s *Store) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	if key == nil {
		return errors.New("api key is nil")
	}
	if key.ID == 0 {
		return errors.New("api key id is required")
	}

	// Verify API key exists
	existing, err := s.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		return err
	}

	createdUUID := gocql.UUIDFromTime(key.CreatedAt)

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	// Update api_keys_by_id
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_id SET name = ?, key_prefix = ?, role = ?, enabled = ?, expires_at = ? WHERE api_key_id = ?`, qident(s.cfg.Keyspace)),
		key.Name, key.KeyPrefix, key.Role, key.Enabled, key.ExpiresAt, key.ID,
	)

	// Update api_keys_by_user
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_user SET name = ?, key_prefix = ?, role = ?, enabled = ?, expires_at = ? WHERE user_id = ? AND api_key_id = ?`, qident(s.cfg.Keyspace)),
		key.Name, key.KeyPrefix, key.Role, key.Enabled, key.ExpiresAt, key.UserID, key.ID,
	)

	// If hash changed, delete old entry and create new one in api_keys_by_hash
	if existing.KeyHash != key.KeyHash {
		batch.Query(
			fmt.Sprintf(`DELETE FROM %s.api_keys_by_hash WHERE api_key_hash = ?`, qident(s.cfg.Keyspace)),
			existing.KeyHash,
		)
		batch.Query(
			fmt.Sprintf(`INSERT INTO %s.api_keys_by_hash (api_key_hash, api_key_id, user_id, name, key_prefix, role, enabled, created_at, expires_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
			key.KeyHash, key.ID, key.UserID, key.Name, key.KeyPrefix, key.Role, key.Enabled, createdUUID, key.ExpiresAt,
		)
	} else {
		// Hash unchanged, just update existing entry
		batch.Query(
			fmt.Sprintf(`UPDATE %s.api_keys_by_hash SET name = ?, key_prefix = ?, role = ?, enabled = ?, expires_at = ? WHERE api_key_hash = ?`, qident(s.cfg.Keyspace)),
			key.Name, key.KeyPrefix, key.Role, key.Enabled, key.ExpiresAt, key.KeyHash,
		)
	}

	return s.session.ExecuteBatch(batch)
}

// DeleteAPIKey deletes an API key.
func (s *Store) DeleteAPIKey(ctx context.Context, id int64) error {
	rec, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(fmt.Sprintf(`DELETE FROM %s.api_keys_by_id WHERE api_key_id = ?`, qident(s.cfg.Keyspace)), id)
	batch.Query(fmt.Sprintf(`DELETE FROM %s.api_keys_by_user WHERE user_id = ? AND api_key_id = ?`, qident(s.cfg.Keyspace)), rec.UserID, id)
	batch.Query(fmt.Sprintf(`DELETE FROM %s.api_keys_by_hash WHERE api_key_hash = ?`, qident(s.cfg.Keyspace)), rec.KeyHash)
	return s.session.ExecuteBatch(batch)
}

// ListAPIKeys retrieves all API keys.
func (s *Store) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT api_key_id FROM %s.api_keys_by_id`, qident(s.cfg.Keyspace)),
	).WithContext(ctx).Iter()

	var keyID int64
	var out []*storage.APIKeyRecord
	for iter.Scan(&keyID) {
		rec, err := s.GetAPIKeyByID(ctx, keyID)
		if err == nil {
			out = append(out, rec)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAPIKeysByUserID retrieves all API keys for a user.
func (s *Store) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT api_key_id FROM %s.api_keys_by_user WHERE user_id = ?`, qident(s.cfg.Keyspace)),
		userID,
	).WithContext(ctx).Iter()

	var keyID int64
	var out []*storage.APIKeyRecord
	for iter.Scan(&keyID) {
		rec, err := s.GetAPIKeyByID(ctx, keyID)
		if err == nil {
			out = append(out, rec)
		}
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp for an API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	// Get the API key to find the hash and user ID for updating all tables
	key, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_id SET last_used = ? WHERE api_key_id = ?`, qident(s.cfg.Keyspace)),
		now, id,
	)
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_user SET last_used = ? WHERE user_id = ? AND api_key_id = ?`, qident(s.cfg.Keyspace)),
		now, key.UserID, id,
	)
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_hash SET last_used = ? WHERE api_key_hash = ?`, qident(s.cfg.Keyspace)),
		now, key.KeyHash,
	)
	return s.session.ExecuteBatch(batch)
}

// ---------- Helpers ----------

func casApplied(q *gocql.Query) (bool, error) {
	m := map[string]interface{}{}
	return q.MapScanCAS(m)
}

func parseConsistency(v string) (gocql.Consistency, error) {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "ANY":
		return gocql.Any, nil
	case "ONE":
		return gocql.One, nil
	case "TWO":
		return gocql.Two, nil
	case "THREE":
		return gocql.Three, nil
	case "QUORUM":
		return gocql.Quorum, nil
	case "ALL":
		return gocql.All, nil
	case "LOCAL_ONE":
		return gocql.LocalOne, nil
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum, nil
	case "EACH_QUORUM":
		return gocql.EachQuorum, nil
	default:
		return 0, fmt.Errorf("unknown cassandra consistency: %q", v)
	}
}

func normalizeCompat(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return "BACKWARD"
	}
	return v
}

func canonicalize(schemaType string, schemaText string) string {
	st := strings.ToUpper(strings.TrimSpace(schemaType))
	switch st {
	case "AVRO":
		sc, err := avro.Parse(schemaText)
		if err != nil {
			return strings.TrimSpace(schemaText)
		}
		return sc.String()
	default:
		return strings.TrimSpace(schemaText)
	}
}

func fingerprint(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}
