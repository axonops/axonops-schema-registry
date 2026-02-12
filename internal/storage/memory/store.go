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

// subjectVersionInfo stores info about a schema registered under a subject.
type subjectVersionInfo struct {
	schemaID  int64
	version   int
	deleted   bool
	createdAt time.Time
}

// Store implements the storage.Storage interface using in-memory data structures.
type Store struct {
	mu sync.RWMutex

	// schemas stores schema content by ID (deduplicated globally by fingerprint)
	schemas map[int64]*storage.SchemaRecord

	// subjectVersions stores version info by subject (subject → version → info)
	subjectVersions map[string]map[int]*subjectVersionInfo

	// nextSubjectVersion tracks the next version number for each subject (monotonically increasing)
	nextSubjectVersion map[string]int

	// globalFingerprints maps fingerprint to schema ID for global deduplication
	globalFingerprints map[string]int64

	// idToSubjectVersions maps schema ID to all subject-versions using it
	idToSubjectVersions map[int64][]storage.SubjectVersion

	// configs stores compatibility configurations by subject
	configs map[string]*storage.ConfigRecord

	// modes stores mode configurations by subject
	modes map[string]*storage.ModeRecord

	// globalConfig is the global compatibility configuration
	globalConfig *storage.ConfigRecord

	// globalMode is the global mode configuration
	globalMode *storage.ModeRecord

	// nextID is the next schema ID to assign
	nextID int64

	// users stores user records by ID
	users map[int64]*storage.UserRecord

	// usersByUsername maps username to user ID
	usersByUsername map[string]int64

	// nextUserID is the next user ID to assign
	nextUserID int64

	// apiKeys stores API key records by ID
	apiKeys map[int64]*storage.APIKeyRecord

	// apiKeysByHash maps key hash to API key ID
	apiKeysByHash map[string]int64

	// nextAPIKeyID is the next API key ID to assign
	nextAPIKeyID int64
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
	return &Store{
		schemas:             make(map[int64]*storage.SchemaRecord),
		subjectVersions:     make(map[string]map[int]*subjectVersionInfo),
		nextSubjectVersion:  make(map[string]int),
		globalFingerprints:  make(map[string]int64),
		idToSubjectVersions: make(map[int64][]storage.SubjectVersion),
		configs:             make(map[string]*storage.ConfigRecord),
		modes:               make(map[string]*storage.ModeRecord),
		users:               make(map[int64]*storage.UserRecord),
		usersByUsername:     make(map[string]int64),
		apiKeys:             make(map[int64]*storage.APIKeyRecord),
		apiKeysByHash:       make(map[string]int64),
		globalConfig:        &storage.ConfigRecord{Subject: "", CompatibilityLevel: "BACKWARD"},
		globalMode:          &storage.ModeRecord{Subject: "", Mode: "READWRITE"},
		nextID:              1,
		nextUserID:          1,
		nextAPIKeyID:        1,
	}
}

// CreateSchema stores a new schema record.
// Uses global fingerprint deduplication: same schema content = same ID across all subjects.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initialize subject's version map if needed
	if s.subjectVersions[record.Subject] == nil {
		s.subjectVersions[record.Subject] = make(map[int]*subjectVersionInfo)
	}

	// Check if this fingerprint already exists in this subject (exact duplicate)
	for _, info := range s.subjectVersions[record.Subject] {
		if !info.deleted {
			existingSchema := s.schemas[info.schemaID]
			if existingSchema != nil && existingSchema.Fingerprint == record.Fingerprint {
				// Same schema already registered under this subject
				record.ID = info.schemaID
				record.Version = info.version
				return storage.ErrSchemaExists
			}
		}
	}

	// Check for global fingerprint (same schema in any subject)
	var schemaID int64
	if existingID, exists := s.globalFingerprints[record.Fingerprint]; exists {
		// Reuse the existing schema ID (global deduplication)
		schemaID = existingID
	} else {
		// New schema, assign new ID
		schemaID = atomic.AddInt64(&s.nextID, 1) - 1
		s.globalFingerprints[record.Fingerprint] = schemaID

		// Store the schema content (first time seeing this fingerprint)
		s.schemas[schemaID] = &storage.SchemaRecord{
			ID:          schemaID,
			SchemaType:  record.SchemaType,
			Schema:      record.Schema,
			References:  record.References,
			Fingerprint: record.Fingerprint,
		}
	}

	// Determine version for this subject (monotonically increasing)
	s.nextSubjectVersion[record.Subject]++
	version := s.nextSubjectVersion[record.Subject]

	// Store the subject-version mapping
	s.subjectVersions[record.Subject][version] = &subjectVersionInfo{
		schemaID:  schemaID,
		version:   version,
		deleted:   false,
		createdAt: time.Now(),
	}

	// Update idToSubjectVersions
	s.idToSubjectVersions[schemaID] = append(s.idToSubjectVersions[schemaID], storage.SubjectVersion{
		Subject: record.Subject,
		Version: version,
	})

	// Update the record with assigned values
	record.ID = schemaID
	record.Version = version
	record.CreatedAt = time.Now()

	return nil
}

// GetSchemaByID retrieves a schema by its global ID.
func (s *Store) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, exists := s.schemas[id]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	return schema, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (s *Store) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectVersionMap := s.subjectVersions[subject]
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

	schema := s.schemas[info.schemaID]
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
		Fingerprint: schema.Fingerprint,
		Deleted:     info.deleted,
		CreatedAt:   info.createdAt,
	}, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectVersionMap := s.subjectVersions[subject]
	if len(subjectVersionMap) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var schemas []*storage.SchemaRecord
	for version, info := range subjectVersionMap {
		if !includeDeleted && info.deleted {
			continue
		}
		schema := s.schemas[info.schemaID]
		if schema != nil {
			schemas = append(schemas, &storage.SchemaRecord{
				ID:          schema.ID,
				Subject:     subject,
				Version:     version,
				SchemaType:  schema.SchemaType,
				Schema:      schema.Schema,
				References:  schema.References,
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

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string, includeDeleted bool) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectVersionMap, exists := s.subjectVersions[subject]
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
		schema := s.schemas[info.schemaID]
		if schema != nil && schema.Fingerprint == fingerprint {
			return &storage.SchemaRecord{
				ID:          schema.ID,
				Subject:     subject,
				Version:     version,
				SchemaType:  schema.SchemaType,
				Schema:      schema.Schema,
				References:  schema.References,
				Fingerprint: schema.Fingerprint,
				Deleted:     info.deleted,
				CreatedAt:   info.createdAt,
			}, nil
		}
	}

	return nil, storage.ErrSchemaNotFound
}

// GetSchemaByGlobalFingerprint retrieves a schema by fingerprint (global lookup).
func (s *Store) GetSchemaByGlobalFingerprint(ctx context.Context, fingerprint string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.globalFingerprints[fingerprint]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	schema := s.schemas[id]
	if schema == nil {
		return nil, storage.ErrSchemaNotFound
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, subject string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectVersionMap := s.subjectVersions[subject]
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

	schema := s.schemas[latestInfo.schemaID]
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
		Fingerprint: schema.Fingerprint,
		Deleted:     latestInfo.deleted,
		CreatedAt:   latestInfo.createdAt,
	}, nil
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subjectVersionMap := s.subjectVersions[subject]
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
		svs := s.idToSubjectVersions[info.schemaID]
		newSvs := make([]storage.SubjectVersion, 0, len(svs))
		for _, sv := range svs {
			if sv.Subject != subject || sv.Version != version {
				newSvs = append(newSvs, sv)
			}
		}
		if len(newSvs) == 0 {
			// No more references to this schema, can delete it
			schema := s.schemas[info.schemaID]
			if schema != nil {
				delete(s.globalFingerprints, schema.Fingerprint)
			}
			delete(s.schemas, info.schemaID)
			delete(s.idToSubjectVersions, info.schemaID)
		} else {
			s.idToSubjectVersions[info.schemaID] = newSvs
		}
	} else {
		info.deleted = true
	}

	return nil
}

// ListSubjects returns all subject names.
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subjects []string
	for subject, versionMap := range s.subjectVersions {
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

// DeleteSubject deletes all versions of a subject.
func (s *Store) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subjectVersionMap := s.subjectVersions[subject]
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
			svs := s.idToSubjectVersions[info.schemaID]
			newSvs := make([]storage.SubjectVersion, 0, len(svs))
			for _, sv := range svs {
				if sv.Subject != subject || sv.Version != version {
					newSvs = append(newSvs, sv)
				}
			}
			if len(newSvs) == 0 {
				// No more references to this schema
				schema := s.schemas[info.schemaID]
				if schema != nil {
					delete(s.globalFingerprints, schema.Fingerprint)
				}
				delete(s.schemas, info.schemaID)
				delete(s.idToSubjectVersions, info.schemaID)
			} else {
				s.idToSubjectVersions[info.schemaID] = newSvs
			}
		} else {
			info.deleted = true
		}
	}

	// Sort deleted versions
	sort.Ints(deletedVersions)

	if permanent {
		delete(s.subjectVersions, subject)
		delete(s.nextSubjectVersion, subject)
		delete(s.configs, subject)
		delete(s.modes, subject)
	}

	return deletedVersions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subjectVersionMap := s.subjectVersions[subject]
	for _, info := range subjectVersionMap {
		if !info.deleted {
			return true, nil
		}
	}

	return false, nil
}

// GetConfig retrieves the compatibility configuration for a subject.
func (s *Store) GetConfig(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.configs[subject]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return config, nil
}

// SetConfig sets the compatibility configuration for a subject.
func (s *Store) SetConfig(ctx context.Context, subject string, config *storage.ConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.Subject = subject
	s.configs[subject] = config
	return nil
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (s *Store) DeleteConfig(ctx context.Context, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[subject]; !exists {
		return storage.ErrNotFound
	}

	delete(s.configs, subject)
	return nil
}

// GetGlobalConfig retrieves the global compatibility configuration.
func (s *Store) GetGlobalConfig(ctx context.Context) (*storage.ConfigRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.globalConfig == nil {
		return nil, storage.ErrNotFound
	}
	return s.globalConfig, nil
}

// SetGlobalConfig sets the global compatibility configuration.
func (s *Store) SetGlobalConfig(ctx context.Context, config *storage.ConfigRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.Subject = ""
	s.globalConfig = config
	return nil
}

// GetMode retrieves the mode for a subject.
func (s *Store) GetMode(ctx context.Context, subject string) (*storage.ModeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mode, exists := s.modes[subject]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return mode, nil
}

// SetMode sets the mode for a subject.
func (s *Store) SetMode(ctx context.Context, subject string, mode *storage.ModeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode.Subject = subject
	s.modes[subject] = mode
	return nil
}

// DeleteMode deletes the mode for a subject.
func (s *Store) DeleteMode(ctx context.Context, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.modes[subject]; !exists {
		return storage.ErrNotFound
	}

	delete(s.modes, subject)
	return nil
}

// GetGlobalMode retrieves the global mode.
func (s *Store) GetGlobalMode(ctx context.Context) (*storage.ModeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.globalMode == nil {
		return nil, storage.ErrNotFound
	}
	return s.globalMode, nil
}

// SetGlobalMode sets the global mode.
func (s *Store) SetGlobalMode(ctx context.Context, mode *storage.ModeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode.Subject = ""
	s.globalMode = mode
	return nil
}

// NextID returns the next available schema ID.
func (s *Store) NextID(ctx context.Context) (int64, error) {
	return atomic.AddInt64(&s.nextID, 1) - 1, nil
}

// GetMaxSchemaID returns the highest schema ID currently assigned.
func (s *Store) GetMaxSchemaID(ctx context.Context) (int64, error) {
	return atomic.LoadInt64(&s.nextID) - 1, nil
}

// ImportSchema inserts a schema with a specified ID (for migration).
// Returns ErrSchemaIDConflict if the ID already exists.
func (s *Store) ImportSchema(ctx context.Context, record *storage.SchemaRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if schema ID already exists
	if _, exists := s.schemas[record.ID]; exists {
		return storage.ErrSchemaIDConflict
	}

	// Initialize subject's version map if needed
	if s.subjectVersions[record.Subject] == nil {
		s.subjectVersions[record.Subject] = make(map[int]*subjectVersionInfo)
	}

	// Check if version already exists for this subject
	if _, exists := s.subjectVersions[record.Subject][record.Version]; exists {
		return storage.ErrSchemaExists
	}

	// Store the schema content
	s.schemas[record.ID] = &storage.SchemaRecord{
		ID:          record.ID,
		SchemaType:  record.SchemaType,
		Schema:      record.Schema,
		References:  record.References,
		Fingerprint: record.Fingerprint,
	}

	// Update global fingerprint mapping
	s.globalFingerprints[record.Fingerprint] = record.ID

	// Store the subject-version mapping
	s.subjectVersions[record.Subject][record.Version] = &subjectVersionInfo{
		schemaID:  record.ID,
		version:   record.Version,
		deleted:   false,
		createdAt: time.Now(),
	}

	// Update idToSubjectVersions
	s.idToSubjectVersions[record.ID] = append(s.idToSubjectVersions[record.ID], storage.SubjectVersion{
		Subject: record.Subject,
		Version: record.Version,
	})

	record.CreatedAt = time.Now()

	return nil
}

// SetNextID sets the ID sequence to start from the given value.
// Used after import to prevent ID conflicts.
func (s *Store) SetNextID(ctx context.Context, id int64) error {
	atomic.StoreInt64(&s.nextID, id)
	return nil
}

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var refs []storage.SubjectVersion

	// Check all subject-versions for references to this subject/version
	for subj, versionMap := range s.subjectVersions {
		for ver, info := range versionMap {
			if info.deleted {
				continue
			}
			schema := s.schemas[info.schemaID]
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

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.schemas[id]; !exists {
		return nil, storage.ErrSchemaNotFound
	}

	svs := s.idToSubjectVersions[id]
	if len(svs) == 0 {
		return []string{}, nil
	}

	// Collect unique subjects, filtering by deleted status
	subjectSet := make(map[string]bool)
	for _, sv := range svs {
		if subjectVersionMap, ok := s.subjectVersions[sv.Subject]; ok {
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

// GetVersionsBySchemaID returns all subject-version pairs where the given schema ID is registered.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.schemas[id]; !exists {
		return nil, storage.ErrSchemaNotFound
	}

	svs := s.idToSubjectVersions[id]
	if len(svs) == 0 {
		return []storage.SubjectVersion{}, nil
	}

	// Filter by deleted status
	var result []storage.SubjectVersion
	for _, sv := range svs {
		if subjectVersionMap, ok := s.subjectVersions[sv.Subject]; ok {
			if info, ok := subjectVersionMap[sv.Version]; ok {
				if includeDeleted || !info.deleted {
					result = append(result, sv)
				}
			}
		}
	}

	return result, nil
}

// ListSchemas returns schemas matching the given filters.
func (s *Store) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*storage.SchemaRecord

	// Track latest versions per subject if needed
	latestVersions := make(map[string]int)
	if params.LatestOnly {
		for subject, versionMap := range s.subjectVersions {
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

	// Collect matching schemas from all subject-versions
	for subject, versionMap := range s.subjectVersions {
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

			schema := s.schemas[info.schemaID]
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

// DeleteGlobalConfig resets the global config to default.
func (s *Store) DeleteGlobalConfig(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalConfig = &storage.ConfigRecord{Subject: "", CompatibilityLevel: "BACKWARD"}
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
