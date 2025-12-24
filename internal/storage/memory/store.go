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

// Store implements the storage.Storage interface using in-memory data structures.
type Store struct {
	mu sync.RWMutex

	// schemas stores all schema records by ID
	schemas map[int64]*storage.SchemaRecord

	// subjectSchemas stores schema IDs by subject, ordered by version
	subjectSchemas map[string][]int64

	// fingerprints maps subject+fingerprint to schema ID for deduplication
	fingerprints map[string]int64

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
		schemas:         make(map[int64]*storage.SchemaRecord),
		subjectSchemas:  make(map[string][]int64),
		fingerprints:    make(map[string]int64),
		configs:         make(map[string]*storage.ConfigRecord),
		modes:           make(map[string]*storage.ModeRecord),
		users:           make(map[int64]*storage.UserRecord),
		usersByUsername: make(map[string]int64),
		apiKeys:         make(map[int64]*storage.APIKeyRecord),
		apiKeysByHash:   make(map[string]int64),
		globalConfig: &storage.ConfigRecord{
			CompatibilityLevel: "BACKWARD",
		},
		globalMode: &storage.ModeRecord{
			Mode: "READWRITE",
		},
		nextID:       1,
		nextUserID:   1,
		nextAPIKeyID: 1,
	}
}

// fingerprintKey generates a key for the fingerprint map.
func fingerprintKey(subject, fingerprint string) string {
	return subject + ":" + fingerprint
}

// CreateSchema stores a new schema record.
func (s *Store) CreateSchema(ctx context.Context, record *storage.SchemaRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing schema with same fingerprint
	key := fingerprintKey(record.Subject, record.Fingerprint)
	if existingID, exists := s.fingerprints[key]; exists {
		existing := s.schemas[existingID]
		if existing != nil && !existing.Deleted {
			record.ID = existing.ID
			record.Version = existing.Version
			return storage.ErrSchemaExists
		}
	}

	// Assign ID if not set
	if record.ID == 0 {
		record.ID = atomic.AddInt64(&s.nextID, 1) - 1
	}

	// Determine version
	versions := s.subjectSchemas[record.Subject]
	record.Version = len(versions) + 1
	record.CreatedAt = time.Now()

	// Store the schema
	s.schemas[record.ID] = record
	s.subjectSchemas[record.Subject] = append(versions, record.ID)
	s.fingerprints[key] = record.ID

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

	versions := s.subjectSchemas[subject]
	if len(versions) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Handle "latest" version (-1)
	if version == -1 {
		version = len(versions)
	}

	if version < 1 || version > len(versions) {
		return nil, storage.ErrVersionNotFound
	}

	schema := s.schemas[versions[version-1]]
	if schema == nil {
		return nil, storage.ErrSchemaNotFound
	}

	if schema.Deleted {
		return nil, storage.ErrVersionNotFound
	}

	return schema, nil
}

// GetSchemasBySubject retrieves all schemas for a subject.
func (s *Store) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.subjectSchemas[subject]
	if len(ids) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var schemas []*storage.SchemaRecord
	for _, id := range ids {
		schema := s.schemas[id]
		if schema != nil && (includeDeleted || !schema.Deleted) {
			schemas = append(schemas, schema)
		}
	}

	return schemas, nil
}

// GetSchemaByFingerprint retrieves a schema by subject and fingerprint.
func (s *Store) GetSchemaByFingerprint(ctx context.Context, subject, fingerprint string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fingerprintKey(subject, fingerprint)
	id, exists := s.fingerprints[key]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	schema := s.schemas[id]
	if schema == nil || schema.Deleted {
		return nil, storage.ErrSchemaNotFound
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest schema for a subject.
func (s *Store) GetLatestSchema(ctx context.Context, subject string) (*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.subjectSchemas[subject]
	if len(ids) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// Find the latest non-deleted schema
	for i := len(ids) - 1; i >= 0; i-- {
		schema := s.schemas[ids[i]]
		if schema != nil && !schema.Deleted {
			return schema, nil
		}
	}

	return nil, storage.ErrSubjectNotFound
}

// DeleteSchema soft-deletes or permanently deletes a schema version.
func (s *Store) DeleteSchema(ctx context.Context, subject string, version int, permanent bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := s.subjectSchemas[subject]
	if len(ids) == 0 {
		return storage.ErrSubjectNotFound
	}

	if version < 1 || version > len(ids) {
		return storage.ErrVersionNotFound
	}

	schema := s.schemas[ids[version-1]]
	if schema == nil {
		return storage.ErrVersionNotFound
	}

	if permanent {
		// Remove from fingerprints
		key := fingerprintKey(subject, schema.Fingerprint)
		delete(s.fingerprints, key)
		delete(s.schemas, schema.ID)
	} else {
		schema.Deleted = true
	}

	return nil
}

// ListSubjects returns all subject names.
func (s *Store) ListSubjects(ctx context.Context, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subjects []string
	for subject, ids := range s.subjectSchemas {
		if includeDeleted {
			subjects = append(subjects, subject)
			continue
		}

		// Check if subject has any non-deleted schemas
		for _, id := range ids {
			schema := s.schemas[id]
			if schema != nil && !schema.Deleted {
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

	ids := s.subjectSchemas[subject]
	if len(ids) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	var deletedVersions []int
	for _, id := range ids {
		schema := s.schemas[id]
		if schema == nil {
			continue
		}

		if schema.Deleted && !permanent {
			continue
		}

		deletedVersions = append(deletedVersions, schema.Version)

		if permanent {
			key := fingerprintKey(subject, schema.Fingerprint)
			delete(s.fingerprints, key)
			delete(s.schemas, id)
		} else {
			schema.Deleted = true
		}
	}

	if permanent {
		delete(s.subjectSchemas, subject)
		delete(s.configs, subject)
		delete(s.modes, subject)
	}

	return deletedVersions, nil
}

// SubjectExists checks if a subject exists.
func (s *Store) SubjectExists(ctx context.Context, subject string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.subjectSchemas[subject]
	for _, id := range ids {
		schema := s.schemas[id]
		if schema != nil && !schema.Deleted {
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

// GetReferencedBy returns subjects/versions that reference the given schema.
func (s *Store) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var refs []storage.SubjectVersion

	// Check all schemas for references to this subject/version
	for _, schema := range s.schemas {
		if schema.Deleted {
			continue
		}
		for _, ref := range schema.References {
			if ref.Subject == subject && ref.Version == version {
				refs = append(refs, storage.SubjectVersion{
					Subject: schema.Subject,
					Version: schema.Version,
				})
				break
			}
		}
	}

	return refs, nil
}

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered.
func (s *Store) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, exists := s.schemas[id]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	// In this implementation, a schema ID is unique to a subject
	// Return the subject if not deleted (or if includeDeleted)
	if !includeDeleted && schema.Deleted {
		return []string{}, nil
	}

	return []string{schema.Subject}, nil
}

// GetVersionsBySchemaID returns all subject-version pairs where the given schema ID is registered.
func (s *Store) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, exists := s.schemas[id]
	if !exists {
		return nil, storage.ErrSchemaNotFound
	}

	// In this implementation, a schema ID maps to exactly one subject-version
	if !includeDeleted && schema.Deleted {
		return []storage.SubjectVersion{}, nil
	}

	return []storage.SubjectVersion{
		{Subject: schema.Subject, Version: schema.Version},
	}, nil
}

// ListSchemas returns schemas matching the given filters.
func (s *Store) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*storage.SchemaRecord

	// Track latest versions per subject if needed
	latestVersions := make(map[string]int)
	if params.LatestOnly {
		for subject, ids := range s.subjectSchemas {
			for i := len(ids) - 1; i >= 0; i-- {
				schema := s.schemas[ids[i]]
				if schema != nil && (params.Deleted || !schema.Deleted) {
					latestVersions[subject] = schema.Version
					break
				}
			}
		}
	}

	// Collect matching schemas
	for _, schema := range s.schemas {
		// Apply deleted filter
		if !params.Deleted && schema.Deleted {
			continue
		}

		// Apply subject prefix filter
		if params.SubjectPrefix != "" {
			if len(schema.Subject) < len(params.SubjectPrefix) ||
				schema.Subject[:len(params.SubjectPrefix)] != params.SubjectPrefix {
				continue
			}
		}

		// Apply latestOnly filter
		if params.LatestOnly {
			if latestVersion, ok := latestVersions[schema.Subject]; ok {
				if schema.Version != latestVersion {
					continue
				}
			} else {
				continue
			}
		}

		results = append(results, schema)
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

	s.globalConfig = &storage.ConfigRecord{
		CompatibilityLevel: "BACKWARD",
	}
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
