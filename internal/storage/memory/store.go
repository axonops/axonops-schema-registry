// Package memory provides an in-memory storage implementation.
package memory

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// DefaultContext is the default registry context name.
const DefaultContext = "."

// subjectVersionInfo stores info about a schema registered under a subject.
type subjectVersionInfo struct {
	schemaID  int64
	version   int
	deleted   bool
	createdAt time.Time
	metadata  *storage.Metadata
	ruleSet   *storage.RuleSet
}

// contextStore holds all per-context data. Each registry context (namespace)
// has its own schemas, subjects, IDs, configs, and modes.
type contextStore struct {
	// schemas stores schema content by ID (deduplicated per-context by fingerprint)
	schemas map[int64]*storage.SchemaRecord

	// subjectVersions stores version info by subject (subject → version → info)
	subjectVersions map[string]map[int]*subjectVersionInfo

	// nextSubjectVersion tracks the next version number for each subject (monotonically increasing)
	nextSubjectVersion map[string]int

	// fingerprints maps fingerprint to schema ID for per-context deduplication
	fingerprints map[string]int64

	// idToSubjectVersions maps schema ID to all subject-versions using it
	idToSubjectVersions map[int64][]storage.SubjectVersion

	// configs stores compatibility configurations by subject
	configs map[string]*storage.ConfigRecord

	// modes stores mode configurations by subject
	modes map[string]*storage.ModeRecord

	// globalConfig is the context-level compatibility configuration (applies to all subjects in context)
	globalConfig *storage.ConfigRecord

	// globalMode is the context-level mode configuration (applies to all subjects in context)
	globalMode *storage.ModeRecord

	// nextID is the next schema ID to assign within this context
	nextID int64
}

// newContextStore creates a new initialized context store.
func newContextStore() *contextStore {
	return &contextStore{
		schemas:             make(map[int64]*storage.SchemaRecord),
		subjectVersions:     make(map[string]map[int]*subjectVersionInfo),
		nextSubjectVersion:  make(map[string]int),
		fingerprints:        make(map[string]int64),
		idToSubjectVersions: make(map[int64][]storage.SubjectVersion),
		configs:             make(map[string]*storage.ConfigRecord),
		modes:               make(map[string]*storage.ModeRecord),
		globalConfig:        &storage.ConfigRecord{Subject: "", CompatibilityLevel: "BACKWARD"},
		globalMode:          &storage.ModeRecord{Subject: "", Mode: "READWRITE"},
		nextID:              1,
	}
}

// Store implements the storage.Storage interface using in-memory data structures.
// All schema, subject, config, mode, and ID operations are scoped to a registry context.
type Store struct {
	mu sync.RWMutex

	// contexts maps registry context name to its per-context store
	contexts map[string]*contextStore

	// users stores user records by ID (global, not per-context)
	users map[int64]*storage.UserRecord

	// usersByUsername maps username to user ID (global)
	usersByUsername map[string]int64

	// nextUserID is the next user ID to assign (global)
	nextUserID int64

	// apiKeys stores API key records by ID (global)
	apiKeys map[int64]*storage.APIKeyRecord

	// apiKeysByHash maps key hash to API key ID (global)
	apiKeysByHash map[string]int64

	// nextAPIKeyID is the next API key ID to assign (global)
	nextAPIKeyID int64
}

// NewStore creates a new in-memory store with the default context initialized.
func NewStore() *Store {
	s := &Store{
		contexts:       make(map[string]*contextStore),
		users:          make(map[int64]*storage.UserRecord),
		usersByUsername: make(map[string]int64),
		apiKeys:        make(map[int64]*storage.APIKeyRecord),
		apiKeysByHash:  make(map[string]int64),
		nextUserID:     1,
		nextAPIKeyID:   1,
	}
	// Default context is always present
	s.contexts[DefaultContext] = newContextStore()
	return s
}

// getOrCreateContext returns the context store, creating it if it doesn't exist.
// Must be called with s.mu held (write lock).
func (s *Store) getOrCreateContext(registryCtx string) *contextStore {
	cs, exists := s.contexts[registryCtx]
	if !exists {
		cs = newContextStore()
		s.contexts[registryCtx] = cs
	}
	return cs
}

// getContext returns the context store, or nil if it doesn't exist.
// Must be called with s.mu held (read or write lock).
func (s *Store) getContext(registryCtx string) *contextStore {
	return s.contexts[registryCtx]
}

// CreateSchema stores a new schema record.
// Uses per-context fingerprint deduplication: same schema content = same ID within a context.
func (s *Store) CreateSchema(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)

	// Initialize subject's version map if needed
	if cs.subjectVersions[record.Subject] == nil {
		cs.subjectVersions[record.Subject] = make(map[int]*subjectVersionInfo)
	}

	// Check if this fingerprint already exists in this subject (exact duplicate)
	for _, info := range cs.subjectVersions[record.Subject] {
		if !info.deleted {
			existingSchema := cs.schemas[info.schemaID]
			if existingSchema != nil && existingSchema.Fingerprint == record.Fingerprint {
				// Same schema already registered under this subject
				record.ID = info.schemaID
				record.Version = info.version
				return storage.ErrSchemaExists
			}
		}
	}

	// Check for per-context fingerprint (same schema in any subject within this context)
	var schemaID int64
	if existingID, exists := cs.fingerprints[record.Fingerprint]; exists {
		// Reuse the existing schema ID (per-context deduplication)
		schemaID = existingID
	} else {
		// New schema, assign new ID within this context
		schemaID = cs.nextID
		cs.nextID++
		cs.fingerprints[record.Fingerprint] = schemaID

		// Store the schema content (first time seeing this fingerprint in this context)
		cs.schemas[schemaID] = &storage.SchemaRecord{
			ID:          schemaID,
			SchemaType:  record.SchemaType,
			Schema:      record.Schema,
			References:  record.References,
			Fingerprint: record.Fingerprint,
		}
	}

	// Determine version for this subject (monotonically increasing)
	cs.nextSubjectVersion[record.Subject]++
	version := cs.nextSubjectVersion[record.Subject]

	// Store the subject-version mapping
	cs.subjectVersions[record.Subject][version] = &subjectVersionInfo{
		schemaID:  schemaID,
		version:   version,
		deleted:   false,
		createdAt: time.Now(),
		metadata:  record.Metadata,
		ruleSet:   record.RuleSet,
	}

	// Update idToSubjectVersions
	cs.idToSubjectVersions[schemaID] = append(cs.idToSubjectVersions[schemaID], storage.SubjectVersion{
		Subject: record.Subject,
		Version: version,
	})

	// Update the record with assigned values
	record.ID = schemaID
	record.Version = version
	record.CreatedAt = time.Now()

	return nil
}

// GetSchemaByID retrieves a schema by its ID within a context.
func (s *Store) GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSchemaNotFound
	}

	schema, exists := cs.schemas[id]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	return schema, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version within a context.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSubjectNotFound
	}

	subjectVersionMap := cs.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Handle "latest" version (-1)
	if version == -1 {
		// Find the latest non-deleted version
		latestVersion := 0
		for v, info := range subjectVersionMap {
			if !info.deleted && v > latestVersion {
				latestVersion = v
			}
		}
		if latestVersion == 0 {
			return nil, storage.ErrSubjectNotFound
		}
		version = latestVersion
	}

	info, exists := subjectVersionMap[version]
	if !exists {
		return nil, storage.ErrVersionNotFound
	}

	if info.deleted {
		return nil, storage.ErrVersionNotFound
	}

	schema := cs.schemas[info.schemaID]
	if schema == nil {
		return nil, storage.ErrSchemaNotFound
	}

	// Return a copy with the subject and version filled in
	return &storage.SchemaRecord{
		ID:          schema.ID,
		Subject:     subject,
		Version:     version,
		SchemaType:  schema.SchemaType,
		Schema:      schema.Schema,
		References:  schema.References,
		Metadata:    info.metadata,
		RuleSet:     info.ruleSet,
		Fingerprint: schema.Fingerprint,
		Deleted:     info.deleted,
		CreatedAt:   info.createdAt,
	}, nil
}

// GetSchemasBySubject retrieves all schemas for a subject within a context.
func (s *Store) GetSchemasBySubject(ctx context.Context, registryCtx string, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSubjectNotFound
	}

	subjectVersionMap := cs.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var schemas []*storage.SchemaRecord
	for version, info := range subjectVersionMap {
		if !includeDeleted && info.deleted {
			continue
		}
		schema := cs.schemas[info.schemaID]
		if schema != nil {
			schemas = append(schemas, &storage.SchemaRecord{
				ID:          schema.ID,
				Subject:     subject,
				Version:     version,
				SchemaType:  schema.SchemaType,
				Schema:      schema.Schema,
				References:  schema.References,
				Metadata:    info.metadata,
				RuleSet:     info.ruleSet,
				Fingerprint: schema.Fingerprint,
				Deleted:     info.deleted,
				CreatedAt:   info.createdAt,
			})
		}
	}

	// If no schemas matched (all were deleted and includeDeleted=false), return not found
	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Sort by version
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Version < schemas[j].Version
	})

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint within a context.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, registryCtx string, subject, fingerprint string, includeDeleted bool) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSubjectNotFound
	}

	subjectVersionMap, exists := cs.subjectVersions[subject]
	if !exists || len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Check if subject has any non-deleted versions (if not including deleted)
	if !includeDeleted {
		hasActive := false
		for _, info := range subjectVersionMap {
			if !info.deleted {
				hasActive = true
				break
			}
		}
		if !hasActive {
			return nil, storage.ErrSubjectNotFound
		}
	}

	// Find a version in this subject with the matching fingerprint
	for version, info := range subjectVersionMap {
		if info.deleted && !includeDeleted {
			continue
		}
		schema := cs.schemas[info.schemaID]
		if schema != nil && schema.Fingerprint == fingerprint {
			return &storage.SchemaRecord{
				ID:          schema.ID,
				Subject:     subject,
				Version:     version,
				SchemaType:  schema.SchemaType,
				Schema:      schema.Schema,
				References:  schema.References,
				Metadata:    info.metadata,
				RuleSet:     info.ruleSet,
				Fingerprint: schema.Fingerprint,
				Deleted:     info.deleted,
				CreatedAt:   info.createdAt,
			}, nil
		}
	}

	return nil, storage.ErrSchemaNotFound
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint within a context.
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, registryCtx string, fingerprint string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSchemaNotFound
	}

	id, exists := cs.fingerprints[fingerprint]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	schema := cs.schemas[id]
	if schema == nil {
		return nil, storage.ErrSchemaNotFound
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest schema for a subject within a context.
func (s *Store) GetLatestSchema(ctx context.Context, registryCtx string, subject string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSubjectNotFound
	}

	subjectVersionMap := cs.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Find the latest non-deleted version
	latestVersion := 0
	var latestInfo *subjectVersionInfo
	for v, info := range subjectVersionMap {
		if !info.deleted && v > latestVersion {
			latestVersion = v
			latestInfo = info
		}
	}

	if latestInfo == nil {
		return nil, storage.ErrSubjectNotFound
	}

	schema := cs.schemas[latestInfo.schemaID]
	if schema == nil {
		return nil, storage.ErrSchemaNotFound
	}

	return &storage.SchemaRecord{
		ID:          schema.ID,
		Subject:     subject,
		Version:     latestVersion,
		SchemaType:  schema.SchemaType,
		Schema:      schema.Schema,
		References:  schema.References,
		Metadata:    latestInfo.metadata,
		RuleSet:     latestInfo.ruleSet,
		Fingerprint: schema.Fingerprint,
		Deleted:     latestInfo.deleted,
		CreatedAt:   latestInfo.createdAt,
	}, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version within a context.
func (s *Store) DeleteSchema(ctx context.Context, registryCtx string, subject string, version int, permanent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return storage.ErrSubjectNotFound
	}

	subjectVersionMap := cs.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return storage.ErrSubjectNotFound
	}

	info, exists := subjectVersionMap[version]
	if !exists {
		return storage.ErrVersionNotFound
	}

	if permanent && !info.deleted {
		return storage.ErrVersionNotSoftDeleted
	}

	if permanent {
		// Remove from subject versions
		delete(subjectVersionMap, version)

		// Remove from idToSubjectVersions
		svs := cs.idToSubjectVersions[info.schemaID]
		newSvs := make([]storage.SubjectVersion, 0, len(svs))
		for _, sv := range svs {
			if sv.Subject != subject || sv.Version != version {
				newSvs = append(newSvs, sv)
			}
		}
		if len(newSvs) == 0 {
			// No more references to this schema, can delete it
			schema := cs.schemas[info.schemaID]
			if schema != nil {
				delete(cs.fingerprints, schema.Fingerprint)
			}
			delete(cs.schemas, info.schemaID)
			delete(cs.idToSubjectVersions, info.schemaID)
		} else {
			cs.idToSubjectVersions[info.schemaID] = newSvs
		}
	} else {
		info.deleted = true
	}

	return nil
}

// ListSubjects returns all subject names within a context.
func (s *Store) ListSubjects(ctx context.Context, registryCtx string, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return []string{}, nil
	}

	var subjects []string
	for subject, versionMap := range cs.subjectVersions {
		if includeDeleted {
			subjects = append(subjects, subject)
			continue
		}

		// Check if subject has any non-deleted schemas
		for _, info := range versionMap {
			if !info.deleted {
				subjects = append(subjects, subject)
				break
			}
		}
	}

	sort.Strings(subjects)
	return subjects, nil
}

// DeleteSubject deletes all versions of a subject within a context.
func (s *Store) DeleteSubject(ctx context.Context, registryCtx string, subject string, permanent bool) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSubjectNotFound
	}

	subjectVersionMap := cs.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Check if all versions are already soft-deleted
	allDeleted := true
	for _, info := range subjectVersionMap {
		if !info.deleted {
			allDeleted = false
			break
		}
	}

	if permanent {
		// For permanent delete, verify all versions are already soft-deleted
		if !allDeleted {
			return nil, storage.ErrSubjectNotSoftDeleted
		}
	} else {
		// For soft-delete, if all versions already soft-deleted → subject is already deleted
		if allDeleted {
			return nil, storage.ErrSubjectDeleted
		}
	}

	var deletedVersions []int
	for version, info := range subjectVersionMap {
		if info.deleted && !permanent {
			continue
		}

		deletedVersions = append(deletedVersions, version)

		if permanent {
			// Remove from idToSubjectVersions
			svs := cs.idToSubjectVersions[info.schemaID]
			newSvs := make([]storage.SubjectVersion, 0, len(svs))
			for _, sv := range svs {
				if sv.Subject != subject || sv.Version != version {
					newSvs = append(newSvs, sv)
				}
			}
			if len(newSvs) == 0 {
				// No more references to this schema
				schema := cs.schemas[info.schemaID]
				if schema != nil {
					delete(cs.fingerprints, schema.Fingerprint)
				}
				delete(cs.schemas, info.schemaID)
				delete(cs.idToSubjectVersions, info.schemaID)
			} else {
				cs.idToSubjectVersions[info.schemaID] = newSvs
			}
		} else {
			info.deleted = true
		}
	}

	// Sort deleted versions
	sort.Ints(deletedVersions)

	if permanent {
		delete(cs.subjectVersions, subject)
		delete(cs.nextSubjectVersion, subject)
		delete(cs.configs, subject)
		delete(cs.modes, subject)
	}

	return deletedVersions, nil
}

// SubjectExists checks if a subject exists within a context.
func (s *Store) SubjectExists(ctx context.Context, registryCtx string, subject string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return false, nil
	}

	subjectVersionMap := cs.subjectVersions[subject]
	for _, info := range subjectVersionMap {
		if !info.deleted {
			return true, nil
		}
	}

	return false, nil
}

// GetConfig retrieves the compatibility configuration for a subject within a context.
func (s *Store) GetConfig(ctx context.Context, registryCtx string, subject string) (*storage.ConfigRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrNotFound
	}

	config, exists := cs.configs[subject]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return config, nil
}

// SetConfig sets the compatibility configuration for a subject within a context.
func (s *Store) SetConfig(ctx context.Context, registryCtx string, subject string, config *storage.ConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	config.Subject = subject
	cs.configs[subject] = config
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject within a context.
func (s *Store) DeleteConfig(ctx context.Context, registryCtx string, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return storage.ErrNotFound
	}

	if _, exists := cs.configs[subject]; !exists {
		return storage.ErrNotFound
	}

	delete(cs.configs, subject)
	return nil
}

// GetGlobalConfig retrieves the global compatibility configuration for a context.
func (s *Store) GetGlobalConfig(ctx context.Context, registryCtx string) (*storage.ConfigRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		// Return default for non-existent context
		return &storage.ConfigRecord{Subject: "", CompatibilityLevel: "BACKWARD"}, nil
	}

	if cs.globalConfig == nil {
		return nil, storage.ErrNotFound
	}
	return cs.globalConfig, nil
}

// SetGlobalConfig sets the global compatibility configuration for a context.
func (s *Store) SetGlobalConfig(ctx context.Context, registryCtx string, config *storage.ConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	config.Subject = ""
	cs.globalConfig = config
	return nil
}

// GetMode retrieves the mode for a subject within a context.
func (s *Store) GetMode(ctx context.Context, registryCtx string, subject string) (*storage.ModeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrNotFound
	}

	mode, exists := cs.modes[subject]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return mode, nil
}

// SetMode sets the mode for a subject within a context.
func (s *Store) SetMode(ctx context.Context, registryCtx string, subject string, mode *storage.ModeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	mode.Subject = subject
	cs.modes[subject] = mode
	return nil
}

// DeleteMode deletes the mode for a subject within a context.
func (s *Store) DeleteMode(ctx context.Context, registryCtx string, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return storage.ErrNotFound
	}

	if _, exists := cs.modes[subject]; !exists {
		return storage.ErrNotFound
	}

	delete(cs.modes, subject)
	return nil
}

// GetGlobalMode retrieves the global mode for a context.
func (s *Store) GetGlobalMode(ctx context.Context, registryCtx string) (*storage.ModeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		// Return default for non-existent context
		return &storage.ModeRecord{Subject: "", Mode: "READWRITE"}, nil
	}

	if cs.globalMode == nil {
		return nil, storage.ErrNotFound
	}
	return cs.globalMode, nil
}

// SetGlobalMode sets the global mode for a context.
func (s *Store) SetGlobalMode(ctx context.Context, registryCtx string, mode *storage.ModeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	mode.Subject = ""
	cs.globalMode = mode
	return nil
}

// NextID returns the next available schema ID for a context.
func (s *Store) NextID(ctx context.Context, registryCtx string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	id := cs.nextID
	cs.nextID++
	return id, nil
}

// GetMaxSchemaID returns the highest schema ID currently assigned in a context.
func (s *Store) GetMaxSchemaID(ctx context.Context, registryCtx string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return 0, nil
	}

	return cs.nextID - 1, nil
}

// ImportSchema inserts a schema with a specified ID (for migration) within a context.
// Returns ErrSchemaIDConflict if the ID already exists with different content.
func (s *Store) ImportSchema(ctx context.Context, registryCtx string, record *storage.SchemaRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)

	// Check if schema ID already exists in this context
	existingSchema, idExists := cs.schemas[record.ID]
	if idExists {
		// If same content (fingerprint), allow associating with new subject.
		// If different content, reject — can't overwrite a schema ID.
		if existingSchema.Fingerprint != record.Fingerprint {
			return storage.ErrSchemaIDConflict
		}
	}

	// Initialize subject's version map if needed
	if cs.subjectVersions[record.Subject] == nil {
		cs.subjectVersions[record.Subject] = make(map[int]*subjectVersionInfo)
	}

	// Check if version already exists for this subject
	if _, exists := cs.subjectVersions[record.Subject][record.Version]; exists {
		return storage.ErrSchemaExists
	}

	// Store the schema content (or update if same ID/fingerprint)
	if !idExists {
		cs.schemas[record.ID] = &storage.SchemaRecord{
			ID:          record.ID,
			SchemaType:  record.SchemaType,
			Schema:      record.Schema,
			References:  record.References,
			Fingerprint: record.Fingerprint,
		}
	}

	// Update per-context fingerprint mapping
	cs.fingerprints[record.Fingerprint] = record.ID

	// Store the subject-version mapping
	cs.subjectVersions[record.Subject][record.Version] = &subjectVersionInfo{
		schemaID:  record.ID,
		version:   record.Version,
		deleted:   false,
		createdAt: time.Now(),
		metadata:  record.Metadata,
		ruleSet:   record.RuleSet,
	}

	// Update idToSubjectVersions
	cs.idToSubjectVersions[record.ID] = append(cs.idToSubjectVersions[record.ID], storage.SubjectVersion{
		Subject: record.Subject,
		Version: record.Version,
	})

	record.CreatedAt = time.Now()

	return nil
}

// SetNextID sets the ID sequence to start from the given value for a context.
// Used after import to prevent ID conflicts.
func (s *Store) SetNextID(ctx context.Context, registryCtx string, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreateContext(registryCtx)
	cs.nextID = id
	return nil
}

// GetReferencedBy returns subjects/versions that reference the given schema within a context.
func (s *Store) GetReferencedBy(ctx context.Context, registryCtx string, subject string, version int) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, nil
	}

	var refs []storage.SubjectVersion

	// Check all subject-versions in this context for references to this subject/version
	for subj, versionMap := range cs.subjectVersions {
		for ver, info := range versionMap {
			if info.deleted {
				continue
			}
			schema := cs.schemas[info.schemaID]
			if schema == nil {
				continue
			}
			for _, ref := range schema.References {
				if ref.Subject == subject && ref.Version == version {
					refs = append(refs, storage.SubjectVersion{
						Subject: subj,
						Version: ver,
					})
					break
				}
			}
		}
	}

	return refs, nil
}

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered within a context.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSchemaNotFound
	}

	if _, exists := cs.schemas[id]; !exists {
		return nil, storage.ErrSchemaNotFound
	}

	svs := cs.idToSubjectVersions[id]
	if len(svs) == 0 {
		return []string{}, nil
	}

	// Collect unique subjects, filtering by deleted status
	subjectSet := make(map[string]bool)
	for _, sv := range svs {
		if subjectVersionMap, ok := cs.subjectVersions[sv.Subject]; ok {
			if info, ok := subjectVersionMap[sv.Version]; ok {
				if includeDeleted || !info.deleted {
					subjectSet[sv.Subject] = true
				}
			}
		}
	}

	subjects := make([]string, 0, len(subjectSet))
	for subj := range subjectSet {
		subjects = append(subjects, subj)
	}
	sort.Strings(subjects)

	return subjects, nil
}

// GetVersionsBySchemaID returns all subject-version pairs where the given schema ID is registered within a context.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil, storage.ErrSchemaNotFound
	}

	if _, exists := cs.schemas[id]; !exists {
		return nil, storage.ErrSchemaNotFound
	}

	svs := cs.idToSubjectVersions[id]
	if len(svs) == 0 {
		return []storage.SubjectVersion{}, nil
	}

	// Filter by deleted status
	var result []storage.SubjectVersion
	for _, sv := range svs {
		if subjectVersionMap, ok := cs.subjectVersions[sv.Subject]; ok {
			if info, ok := subjectVersionMap[sv.Version]; ok {
				if includeDeleted || !info.deleted {
					result = append(result, sv)
				}
			}
		}
	}

	return result, nil
}

// ListSchemas returns schemas matching the given filters within a context.
func (s *Store) ListSchemas(ctx context.Context, registryCtx string, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return []*storage.SchemaRecord{}, nil
	}

	var results []*storage.SchemaRecord

	// Track latest versions per subject if needed
	latestVersions := make(map[string]int)
	if params.LatestOnly {
		for subject, versionMap := range cs.subjectVersions {
			latestVersion := 0
			for v, info := range versionMap {
				if (params.Deleted || !info.deleted) && v > latestVersion {
					latestVersion = v
				}
			}
			if latestVersion > 0 {
				latestVersions[subject] = latestVersion
			}
		}
	}

	// Collect matching schemas from all subject-versions in this context
	for subject, versionMap := range cs.subjectVersions {
		// Apply subject prefix filter
		if params.SubjectPrefix != "" {
			if len(subject) < len(params.SubjectPrefix) ||
				subject[:len(params.SubjectPrefix)] != params.SubjectPrefix {
				continue
			}
		}

		for version, info := range versionMap {
			// Apply deleted filter
			if !params.Deleted && info.deleted {
				continue
			}

			// Apply latestOnly filter
			if params.LatestOnly {
				if latestVersion, ok := latestVersions[subject]; ok {
					if version != latestVersion {
						continue
					}
				} else {
					continue
				}
			}

			schema := cs.schemas[info.schemaID]
			if schema == nil {
				continue
			}

			results = append(results, &storage.SchemaRecord{
				ID:          schema.ID,
				Subject:     subject,
				Version:     version,
				SchemaType:  schema.SchemaType,
				Schema:      schema.Schema,
				References:  schema.References,
				Metadata:    info.metadata,
				RuleSet:     info.ruleSet,
				Fingerprint: schema.Fingerprint,
				Deleted:     info.deleted,
				CreatedAt:   info.createdAt,
			})
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

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

// ListContexts returns all registry context names, sorted alphabetically.
func (s *Store) ListContexts(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contexts := make([]string, 0, len(s.contexts))
	for name := range s.contexts {
		contexts = append(contexts, name)
	}
	sort.Strings(contexts)
	return contexts, nil
}

// DeleteGlobalConfig resets the global config to default for a context.
func (s *Store) DeleteGlobalConfig(ctx context.Context, registryCtx string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getContext(registryCtx)
	if cs == nil {
		return nil
	}

	cs.globalConfig = &storage.ConfigRecord{Subject: "", CompatibilityLevel: "BACKWARD"}
	return nil
}

// Close closes the store.
func (s *Store) Close() error {
	return nil
}

// IsHealthy returns true if the store is healthy.
func (s *Store) IsHealthy(ctx context.Context) bool {
	return true
}

// CreateUser creates a new user.
func (s *Store) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing username
	if _, exists := s.usersByUsername[user.Username]; exists {
		return storage.ErrUserExists
	}

	// Assign ID if not set
	if user.ID == 0 {
		user.ID = atomic.AddInt64(&s.nextUserID, 1) - 1
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Store the user
	s.users[user.ID] = user
	s.usersByUsername[user.Username] = user.ID

	return nil
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, storage.ErrUserNotFound
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.usersByUsername[username]
	if !exists {
		return nil, storage.ErrUserNotFound
	}

	user := s.users[id]
	if user == nil {
		return nil, storage.ErrUserNotFound
	}

	return user, nil
}

// UpdateUser updates an existing user.
func (s *Store) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.users[user.ID]
	if !exists {
		return storage.ErrUserNotFound
	}

	// If username changed, update lookup map
	if existing.Username != user.Username {
		// Check if new username is taken
		if _, taken := s.usersByUsername[user.Username]; taken {
			return storage.ErrUserExists
		}
		delete(s.usersByUsername, existing.Username)
		s.usersByUsername[user.Username] = user.ID
	}

	user.UpdatedAt = time.Now()
	s.users[user.ID] = user

	return nil
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return storage.ErrUserNotFound
	}

	delete(s.usersByUsername, user.Username)
	delete(s.users, id)

	return nil
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*storage.UserRecord, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	// Sort by ID for consistent ordering
	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	return users, nil
}

// CreateAPIKey creates a new API key.
func (s *Store) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing key hash
	if _, exists := s.apiKeysByHash[key.KeyHash]; exists {
		return storage.ErrAPIKeyExists
	}

	// Assign ID if not set
	if key.ID == 0 {
		key.ID = atomic.AddInt64(&s.nextAPIKeyID, 1) - 1
	}

	key.CreatedAt = time.Now()

	// Store the key
	s.apiKeys[key.ID] = key
	s.apiKeysByHash[key.KeyHash] = key.ID

	return nil
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Store) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.apiKeys[id]
	if !exists {
		return nil, storage.ErrAPIKeyNotFound
	}

	return key, nil
}

// GetAPIKeyByHash retrieves an API key by key hash.
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.apiKeysByHash[keyHash]
	if !exists {
		return nil, storage.ErrAPIKeyNotFound
	}

	key := s.apiKeys[id]
	if key == nil {
		return nil, storage.ErrAPIKeyNotFound
	}

	return key, nil
}

// UpdateAPIKey updates an existing API key.
func (s *Store) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.apiKeys[key.ID]
	if !exists {
		return storage.ErrAPIKeyNotFound
	}

	// If key hash changed, update lookup map
	if existing.KeyHash != key.KeyHash {
		// Check if new hash is taken
		if _, taken := s.apiKeysByHash[key.KeyHash]; taken {
			return storage.ErrAPIKeyExists
		}
		delete(s.apiKeysByHash, existing.KeyHash)
		s.apiKeysByHash[key.KeyHash] = key.ID
	}

	s.apiKeys[key.ID] = key

	return nil
}

// DeleteAPIKey deletes an API key by ID.
func (s *Store) DeleteAPIKey(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.apiKeys[id]
	if !exists {
		return storage.ErrAPIKeyNotFound
	}

	delete(s.apiKeysByHash, key.KeyHash)
	delete(s.apiKeys, id)

	return nil
}

// ListAPIKeys returns all API keys.
func (s *Store) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*storage.APIKeyRecord, 0, len(s.apiKeys))
	for _, key := range s.apiKeys {
		keys = append(keys, key)
	}

	// Sort by ID for consistent ordering
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].ID < keys[j].ID
	})

	return keys, nil
}

// ListAPIKeysByUserID returns all API keys for a user.
func (s *Store) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []*storage.APIKeyRecord
	for _, key := range s.apiKeys {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].ID < keys[j].ID
	})

	return keys, nil
}

// GetAPIKeyByUserAndName retrieves an API key by user ID and name.
func (s *Store) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, key := range s.apiKeys {
		if key.UserID == userID && key.Name == name {
			return key, nil
		}
	}

	return nil, storage.ErrAPIKeyNotFound
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.apiKeys[id]
	if !exists {
		return storage.ErrAPIKeyNotFound
	}

	now := time.Now()
	key.LastUsed = &now

	return nil
}
