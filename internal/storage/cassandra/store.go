// Package cassandra provides a Cassandra storage implementation.
//
// Requires Cassandra 5.0+ for Storage Attached Index (SAI) support.
package cassandra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/hamba/avro/v2"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Config holds Cassandra connection configuration.
type Config struct {
	Hosts             []string      `json:"hosts" yaml:"hosts"`
	Port              int           `json:"port" yaml:"port"`
	Keyspace          string        `json:"keyspace" yaml:"keyspace"`
	Username          string        `json:"username" yaml:"username"`
	Password          string        `json:"password" yaml:"password"`
	LocalDC           string        `json:"local_dc" yaml:"local_dc"`
	Consistency       string        `json:"consistency" yaml:"consistency"`
	ReadConsistency   string        `json:"read_consistency" yaml:"read_consistency"`
	WriteConsistency  string        `json:"write_consistency" yaml:"write_consistency"`
	SerialConsistency string        `json:"serial_consistency" yaml:"serial_consistency"`
	Timeout           time.Duration `json:"timeout" yaml:"timeout"`
	ConnectTimeout    time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	Migrate           bool          `json:"migrate" yaml:"migrate"`

	// MaxRetries for CAS operations (ID allocation, version allocation)
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// IDBlockSize is the number of IDs to reserve per LWT call.
	// Higher values reduce LWT frequency but may leave gaps on crash. Default: 50.
	IDBlockSize int `json:"id_block_size" yaml:"id_block_size"`
}

// idAllocator reserves blocks of sequential IDs via a single LWT, then hands
// them out locally. This reduces LWT frequency by ~50x compared to per-ID allocation.
type idAllocator struct {
	mu      sync.Mutex
	current int64
	ceiling int64 // IDs available up to (exclusive)
	block   int64
}

func (a *idAllocator) next(ctx context.Context, s *Store) (int64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.current >= a.ceiling {
		// Reserve a new block via LWT
		base, err := s.reserveIDBlock(ctx, a.block)
		if err != nil {
			return 0, err
		}
		a.current = base
		a.ceiling = base + a.block
	}
	id := a.current
	a.current++
	return id, nil
}

// Store implements storage.Storage on Cassandra.
type Store struct {
	cfg              Config
	cluster          *gocql.ClusterConfig
	session          *gocql.Session
	readConsistency  gocql.Consistency
	writeConsistency gocql.Consistency
	idAlloc          *idAllocator
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
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 50
	}
	if cfg.IDBlockSize <= 0 {
		cfg.IDBlockSize = 50
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

	// Parse serial consistency (for LWT operations: IF NOT EXISTS, IF ... = ?)
	serialConsistency := gocql.LocalSerial
	if cfg.SerialConsistency != "" {
		c, err := parseSerialConsistency(cfg.SerialConsistency)
		if err != nil {
			return nil, err
		}
		serialConsistency = c
	}
	cluster.SerialConsistency = serialConsistency

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
		idAlloc: &idAllocator{
			block: int64(cfg.IDBlockSize),
		},
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

// ---------- ID Allocation (Block-Based) ----------

// NextID returns a new globally unique schema ID using block-based allocation.
// Reserves IDs in blocks via a single LWT, then hands out locally.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	return s.idAlloc.next(ctx, s)
}

// reserveIDBlock atomically reserves a block of IDs via LWT.
// Returns the base ID of the reserved block.
func (s *Store) reserveIDBlock(ctx context.Context, blockSize int64) (int64, error) {
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
						"schema_id", int(blockSize)+1,
					).WithContext(ctx),
				)
				if err != nil {
					return 0, err
				}
				if applied {
					return 1, nil // Base = 1
				}
				continue // Someone else initialized, retry
			}
			return 0, err
		}

		// Atomically reserve block: SET next_id = current + blockSize IF next_id = current
		applied, err := casApplied(
			s.session.Query(
				fmt.Sprintf(`UPDATE %s.id_alloc SET next_id = ? WHERE name = ? IF next_id = ?`, qident(s.cfg.Keyspace)),
				current+int(blockSize), "schema_id", current,
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
	return 0, errors.New("failed to allocate schema ID block: too much contention")
}

// GetMaxSchemaID returns the highest schema ID currently assigned.
// Queries the actual schemas_by_id table rather than the block allocator,
// which would over-report by up to IDBlockSize-1 due to pre-reserved blocks.
func (s *Store) GetMaxSchemaID(ctx context.Context) (int64, error) {
	var maxID int
	if err := s.readQuery(
		fmt.Sprintf(`SELECT MAX(schema_id) FROM %s.schemas_by_id`, qident(s.cfg.Keyspace)),
	).WithContext(ctx).Scan(&maxID); err != nil {
		return 0, fmt.Errorf("failed to get max schema ID: %w", err)
	}
	return int64(maxID), nil
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
				// Reset the local allocator so it doesn't hand out stale IDs
				s.idAlloc.mu.Lock()
				s.idAlloc.current = 0
				s.idAlloc.ceiling = 0
				s.idAlloc.mu.Unlock()
				return nil
			}
			continue
		}
		if err != nil {
			return err
		}

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
			// Reset the local allocator
			s.idAlloc.mu.Lock()
			s.idAlloc.current = 0
			s.idAlloc.ceiling = 0
			s.idAlloc.mu.Unlock()
			return nil
		}
	}
	return errors.New("failed to set next ID: too much contention")
}

// ---------- Schema Operations ----------

// ImportSchema inserts a schema with a specified ID (for migration).
// Returns ErrSchemaIDConflict if the ID already exists with different content.
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
	var existingType, existingFingerprint string
	idExists := false
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type, fingerprint FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(record.ID),
	).WithContext(ctx).Scan(&existingType, &existingFingerprint)
	if err == nil {
		if existingFingerprint != fp {
			return storage.ErrSchemaIDConflict
		}
		idExists = true
	} else if !errors.Is(err, gocql.ErrNotFound) {
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
	metadataStr := marshalJSONText(record.Metadata)
	rulesetStr := marshalJSONText(record.RuleSet)

	// Insert schema content if this is a new ID
	if !idExists {
		// For imports, claim fingerprint but don't reject on conflict — import mode
		// preserves external IDs, so the same schema content (e.g. {"type":"string"})
		// can legitimately appear with different IDs across different subjects/imports.
		// The fingerprint table is primarily for CreateSchema dedup, not import enforcement.
		_, _, _ = s.claimFingerprint(ctx, fp, record.ID)

		if err := s.writeQuery(
			fmt.Sprintf(`INSERT INTO %s.schemas_by_id (schema_id, schema_type, fingerprint, schema_text, canonical_text, created_at, metadata, ruleset)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
			int(record.ID), string(record.SchemaType), fp, record.Schema, canonical, createdUUID, metadataStr, rulesetStr,
		).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("failed to insert schema_by_id: %w", err)
		}
	} else {
		// Schema already exists — ensure fingerprint is claimed (covers pre-migration data)
		_, _, _ = s.claimFingerprint(ctx, fp, record.ID)
	}

	// Batch: subject_versions + subject_latest + references (logged batch for atomicity)
	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(
		fmt.Sprintf(`INSERT INTO %s.subject_versions (subject, version, schema_id, deleted, created_at, metadata, ruleset)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		record.Subject, record.Version, int(record.ID), false, createdUUID, metadataStr, rulesetStr,
	)

	// Update subject_latest if this is the highest version
	latestVersion, _, exists, lerr := s.getSubjectLatest(ctx, record.Subject)
	if lerr != nil {
		return lerr
	}
	if !exists || record.Version > latestVersion {
		batch.Query(
			fmt.Sprintf(`INSERT INTO %s.subject_latest (subject, latest_version, latest_schema_id, updated_at)
				VALUES (?, ?, ?, now())`, qident(s.cfg.Keyspace)),
			record.Subject, record.Version, int(record.ID),
		)
	}

	// Write schema references
	for _, ref := range record.References {
		batch.Query(
			fmt.Sprintf(`INSERT INTO %s.schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
			int(record.ID), ref.Name, ref.Subject, ref.Version,
		)
		batch.Query(
			fmt.Sprintf(`INSERT INTO %s.references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
			ref.Subject, ref.Version, record.Subject, record.Version,
		)
	}

	if err := s.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("failed to import schema batch: %w", err)
	}

	record.CreatedAt = createdUUID.Time()
	record.Deleted = false
	return nil
}

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

	// Ensure global schema exists (deduplication via LWT on schema_fingerprints)
	schemaID, _, err := s.ensureGlobalSchema(ctx, string(record.SchemaType), record.Schema, canonical, fp)
	if err != nil {
		return err
	}

	// Check if subject already has this schema (SAI on schema_id)
	if existing, err := s.findSchemaInSubject(ctx, record.Subject, schemaID); err == nil && existing != nil {
		record.ID = existing.ID
		record.Version = existing.Version
		record.Fingerprint = fp
		record.CreatedAt = existing.CreatedAt
		return storage.ErrSchemaExists
	}

	// Allocate next subject version and persist atomically.
	for attempt := 0; attempt < s.cfg.MaxRetries; attempt++ {
		// Re-check on retry: a concurrent writer may have registered this schema
		if attempt > 0 {
			if existing, err := s.findSchemaInSubject(ctx, record.Subject, schemaID); err == nil && existing != nil {
				record.ID = existing.ID
				record.Version = existing.Version
				record.Fingerprint = fp
				record.CreatedAt = existing.CreatedAt
				return storage.ErrSchemaExists
			}
		}

		latestVersion, latestSchemaID, exists, err := s.getSubjectLatest(ctx, record.Subject)
		if err != nil {
			return err
		}

		newVersion := 1
		if exists {
			newVersion = latestVersion + 1
		}

		createdUUID := gocql.TimeUUID()
		metadataStr := marshalJSONText(record.Metadata)
		rulesetStr := marshalJSONText(record.RuleSet)

		// Step 1: Write subject_versions FIRST (idempotent with IF NOT EXISTS)
		applied, err := casApplied(
			s.session.Query(
				fmt.Sprintf(`INSERT INTO %s.subject_versions (subject, version, schema_id, deleted, created_at, metadata, ruleset)
					VALUES (?, ?, ?, ?, ?, ?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
				record.Subject, newVersion, int(schemaID), false, createdUUID, metadataStr, rulesetStr,
			).WithContext(ctx),
		)
		if err != nil {
			return err
		}
		if !applied {
			// Version already exists — check if it has our schema_id (retry case) or different (contention)
			var existingSchemaID int
			err := s.session.Query(
				fmt.Sprintf(`SELECT schema_id FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
				record.Subject, newVersion,
			).WithContext(ctx).Scan(&existingSchemaID)
			if err != nil {
				continue
			}
			if int64(existingSchemaID) != schemaID {
				continue // Different schema claimed this version, retry with next
			}
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
			continue // subject_latest was updated by another writer, retry
		}

		// Step 3: Write schema references (logged batch for atomicity)
		if len(record.References) > 0 {
			batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
			for _, ref := range record.References {
				batch.Query(
					fmt.Sprintf(`INSERT INTO %s.schema_references (schema_id, name, ref_subject, ref_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
					int(schemaID), ref.Name, ref.Subject, ref.Version,
				)
				batch.Query(
					fmt.Sprintf(`INSERT INTO %s.references_by_target (ref_subject, ref_version, schema_subject, schema_version) VALUES (?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
					ref.Subject, ref.Version, record.Subject, newVersion,
				)
			}
			if err := s.session.ExecuteBatch(batch); err != nil {
				return fmt.Errorf("failed to write schema references: %w", err)
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

// ensureGlobalSchema ensures a schema exists in schemas_by_id, using LWT on
// schema_fingerprints for atomic deduplication.
//
// The schema_fingerprints table has fingerprint as its partition key, so
// INSERT IF NOT EXISTS provides a true compare-and-swap: exactly one writer
// wins the CAS and claims the fingerprint→schema_id mapping. Losers receive
// the winning schema_id in the CAS response without a separate read.
//
// This replaces the previous SAI-based approach where the fingerprint index on
// schemas_by_id was eventually consistent and could not prevent concurrent
// writers from both inserting different schema_ids for the same fingerprint.
func (s *Store) ensureGlobalSchema(ctx context.Context, schemaType, schemaText, canonical, fp string) (schemaID int64, createdAt time.Time, err error) {
	// Fast path: check if fingerprint is already claimed (PK lookup — strongly consistent)
	var existingID int
	err = s.readQuery(
		fmt.Sprintf(`SELECT schema_id FROM %s.schema_fingerprints WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
		fp,
	).WithContext(ctx).Scan(&existingID)
	if err == nil {
		// Fingerprint already claimed — ensure schemas_by_id has the data (crash recovery)
		return s.ensureSchemaData(ctx, int64(existingID), schemaType, schemaText, canonical, fp)
	}
	if !errors.Is(err, gocql.ErrNotFound) {
		return 0, time.Time{}, err
	}

	// Slow path: allocate new ID and claim fingerprint via LWT
	newID, err := s.NextID(ctx)
	if err != nil {
		return 0, time.Time{}, err
	}

	applied, winnerID, err := s.claimFingerprint(ctx, fp, newID)
	if err != nil {
		return 0, time.Time{}, err
	}
	if !applied {
		// Another writer claimed this fingerprint first — use their schema_id
		return s.ensureSchemaData(ctx, winnerID, schemaType, schemaText, canonical, fp)
	}

	// We won the CAS — insert full schema data into schemas_by_id
	return s.ensureSchemaData(ctx, newID, schemaType, schemaText, canonical, fp)
}

// claimFingerprint atomically associates a fingerprint with a schema_id using LWT.
// Returns (true, 0, nil) if we won the CAS (fingerprint newly claimed).
// Returns (false, existingID, nil) if the fingerprint was already claimed by existingID.
func (s *Store) claimFingerprint(ctx context.Context, fp string, schemaID int64) (applied bool, existingID int64, err error) {
	m := map[string]interface{}{}
	applied, err = s.session.Query(
		fmt.Sprintf(`INSERT INTO %s.schema_fingerprints (fingerprint, schema_id) VALUES (?, ?) IF NOT EXISTS`, qident(s.cfg.Keyspace)),
		fp, int(schemaID),
	).WithContext(ctx).MapScanCAS(m)
	if err != nil {
		return false, 0, fmt.Errorf("fingerprint LWT failed: %w", err)
	}
	if !applied {
		// CAS failed — extract the existing schema_id from the result
		if id, ok := m["schema_id"]; ok {
			if v, ok := id.(int); ok {
				return false, int64(v), nil
			}
		}
		return false, 0, fmt.Errorf("fingerprint LWT: CAS failed but could not extract existing schema_id from %v", m)
	}
	return true, 0, nil
}

// ensureSchemaData ensures schemas_by_id has the full schema data for a given schema_id.
// This handles both the normal insert path and crash recovery (where the fingerprint
// was claimed via LWT but the process crashed before writing to schemas_by_id).
func (s *Store) ensureSchemaData(ctx context.Context, schemaID int64, schemaType, schemaText, canonical, fp string) (int64, time.Time, error) {
	// Try to read existing data (common case for dedup hits)
	var createdUUID gocql.UUID
	err := s.readQuery(
		fmt.Sprintf(`SELECT created_at FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(schemaID),
	).WithContext(ctx).Scan(&createdUUID)
	if err == nil {
		return schemaID, createdUUID.Time(), nil
	}
	if !errors.Is(err, gocql.ErrNotFound) {
		return 0, time.Time{}, err
	}

	// Data missing — insert it (first write or crash recovery)
	createdUUID = gocql.TimeUUID()
	if err := s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.schemas_by_id (schema_id, schema_type, fingerprint, schema_text, canonical_text, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, qident(s.cfg.Keyspace)),
		int(schemaID), schemaType, fp, schemaText, canonical, createdUUID,
	).WithContext(ctx).Exec(); err != nil {
		return 0, time.Time{}, err
	}

	return schemaID, createdUUID.Time(), nil
}

// findSchemaInSubject finds a non-deleted version of a schema in a subject using SAI.
func (s *Store) findSchemaInSubject(ctx context.Context, subject string, schemaID int64) (*storage.SchemaRecord, error) {
	// SAI query on subject_versions: partition key (subject) + SAI index (schema_id)
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, deleted, created_at FROM %s.subject_versions WHERE subject = ? AND schema_id = ?`, qident(s.cfg.Keyspace)),
		subject, int(schemaID),
	).WithContext(ctx).Iter()

	var version int
	var deleted bool
	var createdUUID gocql.UUID
	for iter.Scan(&version, &deleted, &createdUUID) {
		if !deleted {
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
	var metadataStr, rulesetStr string
	var createdUUID gocql.UUID

	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_type, schema_text, created_at, metadata, ruleset FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Scan(&schemaType, &schemaText, &createdUUID, &metadataStr, &rulesetStr)
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
		Metadata:   unmarshalJSONText[storage.Metadata](metadataStr),
		RuleSet:    unmarshalJSONText[storage.RuleSet](rulesetStr),
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

	// Look up the first subject/version that uses this schema ID (SAI query)
	var subject string
	var version int
	err = s.readQuery(
		fmt.Sprintf(`SELECT subject, version FROM %s.subject_versions WHERE schema_id = ? AND deleted = false LIMIT 1`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Scan(&subject, &version)
	if err == nil {
		rec.Subject = subject
		rec.Version = version
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
	var metadataStr, rulesetStr string
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, deleted, created_at, metadata, ruleset FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Scan(&schemaID, &deleted, &createdUUID, &metadataStr, &rulesetStr)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
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
	// subject_versions metadata/ruleset override the schemas_by_id values
	if m := unmarshalJSONText[storage.Metadata](metadataStr); m != nil {
		rec.Metadata = m
	}
	if r := unmarshalJSONText[storage.RuleSet](rulesetStr); r != nil {
		rec.RuleSet = r
	}
	return rec, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
// Uses IN-clause batch reads to avoid N+1 queries.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, schema_id, deleted, created_at, metadata, ruleset FROM %s.subject_versions WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()

	type versionInfo struct {
		version     int
		schemaID    int
		deleted     bool
		createdAt   gocql.UUID
		metadataStr string
		rulesetStr  string
	}
	var entries []versionInfo
	var vi versionInfo
	hasAnyRows := false
	for iter.Scan(&vi.version, &vi.schemaID, &vi.deleted, &vi.createdAt, &vi.metadataStr, &vi.rulesetStr) {
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

	// Collect unique schema IDs for IN-clause batch read
	idSet := make(map[int]bool)
	for _, e := range entries {
		idSet[e.schemaID] = true
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	// Batch read all schema content in one query
	type schemaContent struct {
		schemaType string
		schemaText string
		createdAt  gocql.UUID
		metaStr    string
		ruleStr    string
	}
	schemaMap := make(map[int]*schemaContent)
	contentIter := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, schema_type, schema_text, created_at, metadata, ruleset FROM %s.schemas_by_id WHERE schema_id IN ?`, qident(s.cfg.Keyspace)),
		ids,
	).WithContext(ctx).Iter()
	var sid int
	var sc schemaContent
	for contentIter.Scan(&sid, &sc.schemaType, &sc.schemaText, &sc.createdAt, &sc.metaStr, &sc.ruleStr) {
		cp := sc
		schemaMap[sid] = &cp
	}
	contentIter.Close()

	// Batch read all references in one query
	type refEntry struct {
		schemaID   int
		name       string
		refSubject string
		refVersion int
	}
	refMap := make(map[int][]storage.Reference)
	refIter := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, name, ref_subject, ref_version FROM %s.schema_references WHERE schema_id IN ?`, qident(s.cfg.Keyspace)),
		ids,
	).WithContext(ctx).Iter()
	var re refEntry
	for refIter.Scan(&re.schemaID, &re.name, &re.refSubject, &re.refVersion) {
		refMap[re.schemaID] = append(refMap[re.schemaID], storage.Reference{
			Name:    re.name,
			Subject: re.refSubject,
			Version: re.refVersion,
		})
	}
	refIter.Close()

	// Build results
	out := make([]*storage.SchemaRecord, 0, len(entries))
	for _, e := range entries {
		sc, ok := schemaMap[e.schemaID]
		if !ok {
			continue
		}
		rec := &storage.SchemaRecord{
			ID:         int64(e.schemaID),
			Subject:    subject,
			Version:    e.version,
			SchemaType: storage.SchemaType(sc.schemaType),
			Schema:     sc.schemaText,
			Deleted:    e.deleted,
			CreatedAt:  e.createdAt.Time(),
			References: refMap[e.schemaID],
			Metadata:   unmarshalJSONText[storage.Metadata](sc.metaStr),
			RuleSet:    unmarshalJSONText[storage.RuleSet](sc.ruleStr),
		}
		// Overlay per-version metadata/ruleset
		if m := unmarshalJSONText[storage.Metadata](e.metadataStr); m != nil {
			rec.Metadata = m
		}
		if r := unmarshalJSONText[storage.RuleSet](e.rulesetStr); r != nil {
			rec.RuleSet = r
		}
		out = append(out, rec)
	}
	return out, nil
}

// GetSchemaByFingerprint retrieves a schema by fingerprint within a subject.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fp string, includeDeleted bool) (*storage.SchemaRecord, error) {
	// SAI lookup by fingerprint on schemas_by_id
	globalRec, err := s.GetSchemaByGlobalFingerprint(ctx, fp)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			_, _, subjectExists, subErr := s.getSubjectLatest(ctx, subject)
			if subErr != nil {
				return nil, subErr
			}
			if !subjectExists {
				return nil, storage.ErrSubjectNotFound
			}
			return nil, storage.ErrSchemaNotFound
		}
		return nil, err
	}

	// SAI query: find version in subject by schema_id
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version, deleted FROM %s.subject_versions WHERE subject = ? AND schema_id = ?`, qident(s.cfg.Keyspace)),
		subject, int(globalRec.ID),
	).WithContext(ctx).Iter()

	var version int
	var deleted bool
	for iter.Scan(&version, &deleted) {
		if includeDeleted || !deleted {
			iter.Close()
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

	_, _, subjectExists, subErr := s.getSubjectLatest(ctx, subject)
	if subErr != nil {
		return nil, subErr
	}
	if !subjectExists {
		return nil, storage.ErrSubjectNotFound
	}
	return nil, storage.ErrSchemaNotFound
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint (global SAI lookup).
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, fp string) (*storage.SchemaRecord, error) {
	var schemaID int
	var schemaType, schemaText string
	var createdUUID gocql.UUID

	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, schema_type, schema_text, created_at FROM %s.schemas_by_id WHERE fingerprint = ?`, qident(s.cfg.Keyspace)),
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

	// SAI query: get all non-deleted versions, find the max
	iter := s.readQuery(
		fmt.Sprintf(`SELECT version FROM %s.subject_versions WHERE subject = ? AND deleted = false`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Iter()
	latestVersion := 0
	var version int
	for iter.Scan(&version) {
		if version > latestVersion {
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

	var existingSchemaID int
	var deleted bool
	err := s.readQuery(
		fmt.Sprintf(`SELECT schema_id, deleted FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Scan(&existingSchemaID, &deleted)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
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
		// Clean up references_by_target for any schemas this version references
		s.cleanupReferencesByTarget(ctx, existingSchemaID, subject, version)
		if err := s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
			subject, version,
		).WithContext(ctx).Exec(); err != nil {
			return err
		}
		// Check if any subject_versions still reference this schema_id (SAI query)
		s.cleanupOrphanedSchema(ctx, existingSchemaID)
		return nil
	}
	return s.writeQuery(
		fmt.Sprintf(`UPDATE %s.subject_versions SET deleted = true WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
		subject, version,
	).WithContext(ctx).Exec()
}

// cleanupOrphanedSchema removes a schema from schemas_by_id if no subject_versions reference it.
// Uses SAI index on schema_id for O(1) lookup instead of full table scan.
func (s *Store) cleanupOrphanedSchema(ctx context.Context, schemaID int) {
	// SAI query: check if any subject_version still references this schema
	var dummy string
	err := s.readQuery(
		fmt.Sprintf(`SELECT subject FROM %s.subject_versions WHERE schema_id = ? LIMIT 1`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Scan(&dummy)
	if err == nil {
		return // Still referenced, don't clean up
	}

	// Not referenced — clean up schemas_by_id
	if err := s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.schemas_by_id WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Exec(); err != nil {
		slog.Warn("failed to clean up orphaned schema", "schema_id", schemaID, "error", err)
	}

	// Also clean up references
	if err := s.writeQuery(
		fmt.Sprintf(`DELETE FROM %s.schema_references WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Exec(); err != nil {
		slog.Warn("failed to clean up orphaned schema references", "schema_id", schemaID, "error", err)
	}
}

// cleanupReferencesByTarget removes references_by_target entries for a schema
// that is being permanently deleted.
func (s *Store) cleanupReferencesByTarget(ctx context.Context, schemaID int, subject string, version int) {
	refIter := s.readQuery(
		fmt.Sprintf(`SELECT ref_subject, ref_version FROM %s.schema_references WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		schemaID,
	).WithContext(ctx).Iter()
	var refSubject string
	var refVersion int
	for refIter.Scan(&refSubject, &refVersion) {
		if err := s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.references_by_target WHERE ref_subject = ? AND ref_version = ? AND schema_subject = ? AND schema_version = ?`, qident(s.cfg.Keyspace)),
			refSubject, refVersion, subject, version,
		).WithContext(ctx).Exec(); err != nil {
			slog.Warn("failed to clean up reference_by_target", "ref_subject", refSubject, "ref_version", refVersion, "error", err)
		}
	}
	refIter.Close()
}

// ---------- Subject Operations ----------

// ListSubjects returns all subjects.
// Uses subject_latest as the subject listing (replaces old bucketed subjects table).
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	// Scan subject_latest — one row per subject
	iter := s.readQuery(
		fmt.Sprintf(`SELECT subject FROM %s.subject_latest`, qident(s.cfg.Keyspace)),
	).WithContext(ctx).Iter()

	var allSubjects []string
	var subject string
	for iter.Scan(&subject) {
		allSubjects = append(allSubjects, subject)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}

	if includeDeleted {
		sort.Strings(allSubjects)
		return allSubjects, nil
	}

	// Filter: only subjects with at least one non-deleted version (SAI on deleted)
	subjects := make([]string, 0, len(allSubjects))
	for _, subj := range allSubjects {
		var v int
		err := s.readQuery(
			fmt.Sprintf(`SELECT version FROM %s.subject_versions WHERE subject = ? AND deleted = false LIMIT 1`, qident(s.cfg.Keyspace)),
			subj,
		).WithContext(ctx).Scan(&v)
		if err == nil {
			subjects = append(subjects, subj)
		}
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
		for _, vi := range allVersions {
			if !vi.deleted {
				return nil, storage.ErrSubjectNotSoftDeleted
			}
			deletedVersions = append(deletedVersions, vi.version)
		}
	} else {
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

	if permanent {
		// Permanent delete: call DeleteSchema individually (has cross-table cleanup logic)
		for _, v := range deletedVersions {
			if err := s.DeleteSchema(ctx, subject, v, permanent); err != nil {
				return nil, err
			}
		}
		// Remove from subject_latest
		if err := s.writeQuery(
			fmt.Sprintf(`DELETE FROM %s.subject_latest WHERE subject = ?`, qident(s.cfg.Keyspace)),
			subject,
		).WithContext(ctx).Exec(); err != nil {
			slog.Warn("failed to delete subject_latest", "subject", subject, "error", err)
		}
	} else {
		// Soft delete: batch all version updates (same partition = unlogged batch is atomic)
		batch := s.session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
		for _, v := range deletedVersions {
			batch.Query(
				fmt.Sprintf(`UPDATE %s.subject_versions SET deleted = true WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
				subject, v,
			)
		}
		if err := s.session.ExecuteBatch(batch); err != nil {
			return nil, fmt.Errorf("failed to soft-delete versions: %w", err)
		}
	}

	sort.Ints(deletedVersions)
	return deletedVersions, nil
}

// SubjectExists checks if a subject exists (has at least one non-deleted version).
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	_, _, exists, err := s.getSubjectLatest(ctx, subject)
	if err != nil || !exists {
		return false, err
	}
	// SAI query: check for any non-deleted version
	var v int
	err = s.readQuery(
		fmt.Sprintf(`SELECT version FROM %s.subject_versions WHERE subject = ? AND deleted = false LIMIT 1`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Scan(&v)
	if err == nil {
		return true, nil
	}
	return false, nil
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
		// Filter out soft-deleted referrers (consistent with memory store behavior)
		var deleted bool
		if err := s.readQuery(
			fmt.Sprintf(`SELECT deleted FROM %s.subject_versions WHERE subject = ? AND version = ?`, qident(s.cfg.Keyspace)),
			ref.Subject, ref.Version,
		).WithContext(ctx).Scan(&deleted); err != nil || deleted {
			continue
		}
		refs = append(refs, ref)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return refs, nil
}

// GetSubjectsBySchemaID returns subjects using the given schema ID.
// Uses SAI index on subject_versions.schema_id for O(1) lookup.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	// Verify schema ID exists
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

	// SAI query: find all subject_versions with this schema_id
	iter := s.readQuery(
		fmt.Sprintf(`SELECT subject, deleted FROM %s.subject_versions WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Iter()

	subjectSet := make(map[string]bool)
	var subject string
	var deleted bool
	for iter.Scan(&subject, &deleted) {
		if includeDeleted || !deleted {
			subjectSet[subject] = true
		}
	}
	iter.Close()

	subjects := make([]string, 0, len(subjectSet))
	for subj := range subjectSet {
		subjects = append(subjects, subj)
	}
	sort.Strings(subjects)
	return subjects, nil
}

// GetVersionsBySchemaID returns subject-version pairs using the given schema ID.
// Uses SAI index on subject_versions.schema_id for O(1) lookup.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	// Verify schema ID exists
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

	// SAI query: find all subject_versions with this schema_id
	iter := s.readQuery(
		fmt.Sprintf(`SELECT subject, version, deleted FROM %s.subject_versions WHERE schema_id = ?`, qident(s.cfg.Keyspace)),
		int(id),
	).WithContext(ctx).Iter()

	var results []storage.SubjectVersion
	var subject string
	var version int
	var deleted bool
	for iter.Scan(&subject, &version, &deleted) {
		if includeDeleted || !deleted {
			results = append(results, storage.SubjectVersion{Subject: subject, Version: version})
		}
	}
	iter.Close()

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
	var compat, alias string
	var normalize, validateFields *bool
	var compatibilityGroup string
	var defaultMetadataStr, overrideMetadataStr, defaultRulesetStr, overrideRulesetStr string
	err := s.readQuery(
		fmt.Sprintf(`SELECT compatibility, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group FROM %s.subject_configs WHERE subject = ?`, qident(s.cfg.Keyspace)),
		subject,
	).WithContext(ctx).Scan(&compat, &alias, &normalize, &validateFields, &defaultMetadataStr, &overrideMetadataStr, &defaultRulesetStr, &overrideRulesetStr, &compatibilityGroup)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ConfigRecord{
		Subject:            subject,
		CompatibilityLevel: compat,
		Alias:              alias,
		Normalize:          normalize,
		ValidateFields:     validateFields,
		CompatibilityGroup: compatibilityGroup,
		DefaultMetadata:    unmarshalJSONText[storage.Metadata](defaultMetadataStr),
		OverrideMetadata:   unmarshalJSONText[storage.Metadata](overrideMetadataStr),
		DefaultRuleSet:     unmarshalJSONText[storage.RuleSet](defaultRulesetStr),
		OverrideRuleSet:    unmarshalJSONText[storage.RuleSet](overrideRulesetStr),
	}, nil
}

// SetConfig sets the compatibility config for a subject.
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
	if config == nil {
		return errors.New("config is nil")
	}
	compat := normalizeCompat(config.CompatibilityLevel)
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.subject_configs (subject, compatibility, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())`, qident(s.cfg.Keyspace)),
		subject, compat, config.Alias, config.Normalize, config.ValidateFields,
		marshalJSONText(config.DefaultMetadata), marshalJSONText(config.OverrideMetadata),
		marshalJSONText(config.DefaultRuleSet), marshalJSONText(config.OverrideRuleSet),
		config.CompatibilityGroup,
	).WithContext(ctx).Exec()
}

// DeleteConfig deletes a compatibility config for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	if subject == "" {
		return nil
	}
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
	var compat, alias string
	var normalize, validateFields *bool
	var compatibilityGroup string
	var defaultMetadataStr, overrideMetadataStr, defaultRulesetStr, overrideRulesetStr string
	err := s.readQuery(
		fmt.Sprintf(`SELECT compatibility, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group FROM %s.global_config WHERE key = ?`, qident(s.cfg.Keyspace)),
		"global",
	).WithContext(ctx).Scan(&compat, &alias, &normalize, &validateFields, &defaultMetadataStr, &overrideMetadataStr, &defaultRulesetStr, &overrideRulesetStr, &compatibilityGroup)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	return &storage.ConfigRecord{
		Subject:            "",
		CompatibilityLevel: compat,
		Alias:              alias,
		Normalize:          normalize,
		ValidateFields:     validateFields,
		CompatibilityGroup: compatibilityGroup,
		DefaultMetadata:    unmarshalJSONText[storage.Metadata](defaultMetadataStr),
		OverrideMetadata:   unmarshalJSONText[storage.Metadata](overrideMetadataStr),
		DefaultRuleSet:     unmarshalJSONText[storage.RuleSet](defaultRulesetStr),
		OverrideRuleSet:    unmarshalJSONText[storage.RuleSet](overrideRulesetStr),
	}, nil
}

// SetGlobalConfig sets the global compatibility config.
func (s *Store) SetGlobalConfig(ctx context.Context, config *storage.ConfigRecord) error {
	compat := "BACKWARD"
	var alias string
	var normalize, validateFields *bool
	var compatibilityGroup string
	var defaultMetadata, overrideMetadata *storage.Metadata
	var defaultRuleSet, overrideRuleSet *storage.RuleSet
	if config != nil {
		compat = normalizeCompat(config.CompatibilityLevel)
		alias = config.Alias
		normalize = config.Normalize
		validateFields = config.ValidateFields
		compatibilityGroup = config.CompatibilityGroup
		defaultMetadata = config.DefaultMetadata
		overrideMetadata = config.OverrideMetadata
		defaultRuleSet = config.DefaultRuleSet
		overrideRuleSet = config.OverrideRuleSet
	}
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.global_config (key, compatibility, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())`, qident(s.cfg.Keyspace)),
		"global", compat, alias, normalize, validateFields,
		marshalJSONText(defaultMetadata), marshalJSONText(overrideMetadata),
		marshalJSONText(defaultRuleSet), marshalJSONText(overrideRuleSet),
		compatibilityGroup,
	).WithContext(ctx).Exec()
}

// DeleteGlobalConfig resets the global config to the default (BACKWARD).
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	return s.writeQuery(
		fmt.Sprintf(`INSERT INTO %s.global_config (key, compatibility, alias, normalize, validate_fields, default_metadata, override_metadata, default_ruleset, override_ruleset, compatibility_group, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())`, qident(s.cfg.Keyspace)),
		"global", "BACKWARD", "", nil, nil, "", "", "", "", "",
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

	existing, err := s.GetUserByUsername(ctx, user.Username)
	if err == nil && existing != nil {
		return storage.ErrUserExists
	}

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

	existing, err := s.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}

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
	batch.Query(
		fmt.Sprintf(`UPDATE %s.users_by_id SET email = ?, name = ?, password_hash = ?, roles = ?, enabled = ?, updated_at = ? WHERE user_id = ?`, qident(s.cfg.Keyspace)),
		user.Email, user.Username, user.PasswordHash, []string{user.Role}, user.Enabled, updatedUUID, user.ID,
	)

	if existing.Username != user.Username {
		batch.Query(
			fmt.Sprintf(`DELETE FROM %s.users_by_email WHERE email = ?`, qident(s.cfg.Keyspace)),
			existing.Username,
		)
	}

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

	if !key.Enabled {
		key.Enabled = true
	}

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

	existing, err := s.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		return err
	}

	createdUUID := gocql.UUIDFromTime(key.CreatedAt)

	batch := s.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_id SET name = ?, key_prefix = ?, role = ?, enabled = ?, expires_at = ? WHERE api_key_id = ?`, qident(s.cfg.Keyspace)),
		key.Name, key.KeyPrefix, key.Role, key.Enabled, key.ExpiresAt, key.ID,
	)
	batch.Query(
		fmt.Sprintf(`UPDATE %s.api_keys_by_user SET name = ?, key_prefix = ?, role = ?, enabled = ?, expires_at = ? WHERE user_id = ? AND api_key_id = ?`, qident(s.cfg.Keyspace)),
		key.Name, key.KeyPrefix, key.Role, key.Enabled, key.ExpiresAt, key.UserID, key.ID,
	)

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

func parseSerialConsistency(v string) (gocql.Consistency, error) {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "SERIAL":
		return gocql.Serial, nil
	case "LOCAL_SERIAL":
		return gocql.LocalSerial, nil
	default:
		return 0, fmt.Errorf("invalid cassandra serial consistency: %q (must be SERIAL or LOCAL_SERIAL)", v)
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

// marshalJSONText serializes a value to a JSON string for storage in a Cassandra text column.
// Returns empty string if v is nil.
func marshalJSONText(v interface{}) string {
	if v == nil {
		return ""
	}
	// Handle typed nil pointers (e.g., (*storage.Metadata)(nil) passed as interface{})
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// unmarshalJSONText deserializes a JSON text column value into a pointer of type T.
// Returns nil if the input is empty.
func unmarshalJSONText[T any](s string) *T {
	if s == "" {
		return nil
	}
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return &v
}
