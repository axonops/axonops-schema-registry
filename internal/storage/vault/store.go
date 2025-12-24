// Package vault provides HashiCorp Vault storage for authentication data.
package vault

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Config holds Vault connection configuration.
type Config struct {
	Address       string
	Token         string
	Namespace     string
	MountPath     string // KV v2 mount path (default: "secret")
	BasePath      string // Base path for data (default: "schema-registry")
	TLSCertFile   string
	TLSKeyFile    string
	TLSCAFile     string
	TLSSkipVerify bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Address:   "http://localhost:8200",
		MountPath: "secret",
		BasePath:  "schema-registry",
	}
}

// Store implements storage.AuthStorage using HashiCorp Vault.
type Store struct {
	client    *api.Client
	config    Config
	mu        sync.RWMutex
	userIDSeq int64
	keyIDSeq  int64
}

// NewStore creates a new Vault auth store.
func NewStore(config Config) (*Store, error) {
	// Apply defaults
	if config.MountPath == "" {
		config.MountPath = "secret"
	}
	if config.BasePath == "" {
		config.BasePath = "schema-registry"
	}

	// Create Vault client config
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.Address

	// Configure TLS if needed
	if config.TLSCertFile != "" || config.TLSCAFile != "" || config.TLSSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLSSkipVerify,
		}

		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		vaultConfig.HttpClient.Transport = transport
	}

	// Create client
	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set token
	if config.Token != "" {
		client.SetToken(config.Token)
	}

	// Set namespace if provided (Vault Enterprise)
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	store := &Store{
		client: client,
		config: config,
	}

	// Initialize ID sequences
	if err := store.initSequences(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize sequences: %w", err)
	}

	return store, nil
}

// initSequences initializes the ID sequences from Vault.
func (s *Store) initSequences(ctx context.Context) error {
	// Read user ID sequence
	userSeq, err := s.readSequence(ctx, "user_id_seq")
	if err != nil {
		return err
	}
	s.userIDSeq = userSeq

	// Read API key ID sequence
	keySeq, err := s.readSequence(ctx, "apikey_id_seq")
	if err != nil {
		return err
	}
	s.keyIDSeq = keySeq

	return nil
}

func (s *Store) readSequence(ctx context.Context, name string) (int64, error) {
	path := s.kvPath("sequences/" + name)
	secret, err := s.client.KVv2(s.config.MountPath).Get(ctx, path)
	if err != nil {
		// If not found, start at 0
		if isNotFoundError(err) {
			return 0, nil
		}
		return 0, err
	}

	if secret == nil || secret.Data == nil {
		return 0, nil
	}

	if val, ok := secret.Data["value"]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v), nil
		case json.Number:
			return v.Int64()
		case string:
			return strconv.ParseInt(v, 10, 64)
		}
	}

	return 0, nil
}

func (s *Store) writeSequence(ctx context.Context, name string, value int64) error {
	path := s.kvPath("sequences/" + name)
	_, err := s.client.KVv2(s.config.MountPath).Put(ctx, path, map[string]interface{}{
		"value": value,
	})
	return err
}

func (s *Store) nextUserID(ctx context.Context) (int64, error) {
	id := atomic.AddInt64(&s.userIDSeq, 1)
	if err := s.writeSequence(ctx, "user_id_seq", id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) nextAPIKeyID(ctx context.Context) (int64, error) {
	id := atomic.AddInt64(&s.keyIDSeq, 1)
	if err := s.writeSequence(ctx, "apikey_id_seq", id); err != nil {
		return 0, err
	}
	return id, nil
}

// kvPath returns the full path for a key in KV v2.
func (s *Store) kvPath(key string) string {
	return s.config.BasePath + "/" + key
}

// CreateUser creates a new user record.
func (s *Store) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username already exists
	existing, _ := s.getUserByUsernameInternal(ctx, user.Username)
	if existing != nil {
		return storage.ErrUserExists
	}

	// Check if email already exists (if provided)
	if user.Email != "" {
		users, _ := s.listUsersInternal(ctx)
		for _, u := range users {
			if u.Email == user.Email {
				return storage.ErrUserExists
			}
		}
	}

	// Generate ID
	id, err := s.nextUserID(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate user ID: %w", err)
	}
	user.ID = id

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Store user
	if err := s.writeUser(ctx, user); err != nil {
		return err
	}

	// Store username index
	return s.writeUsernameIndex(ctx, user.Username, user.ID)
}

func (s *Store) writeUser(ctx context.Context, user *storage.UserRecord) error {
	path := s.kvPath(fmt.Sprintf("users/%d", user.ID))
	data := map[string]interface{}{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"role":          user.Role,
		"enabled":       user.Enabled,
		"created_at":    user.CreatedAt.Format(time.RFC3339),
		"updated_at":    user.UpdatedAt.Format(time.RFC3339),
	}
	_, err := s.client.KVv2(s.config.MountPath).Put(ctx, path, data)
	return err
}

func (s *Store) writeUsernameIndex(ctx context.Context, username string, id int64) error {
	path := s.kvPath("indexes/users/username/" + username)
	_, err := s.client.KVv2(s.config.MountPath).Put(ctx, path, map[string]interface{}{
		"id": id,
	})
	return err
}

func (s *Store) deleteUsernameIndex(ctx context.Context, username string) error {
	path := s.kvPath("indexes/users/username/" + username)
	return s.client.KVv2(s.config.MountPath).Delete(ctx, path)
}

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	path := s.kvPath(fmt.Sprintf("users/%d", id))
	secret, err := s.client.KVv2(s.config.MountPath).Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return parseUserRecord(secret.Data)
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	return s.getUserByUsernameInternal(ctx, username)
}

func (s *Store) getUserByUsernameInternal(ctx context.Context, username string) (*storage.UserRecord, error) {
	// Look up ID from username index
	path := s.kvPath("indexes/users/username/" + username)
	secret, err := s.client.KVv2(s.config.MountPath).Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, storage.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	id, err := parseID(secret.Data, "id")
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, id)
}

// UpdateUser updates an existing user record.
func (s *Store) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current user
	current, err := s.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}

	// Check if new username is taken (if changed)
	if user.Username != current.Username {
		existing, _ := s.getUserByUsernameInternal(ctx, user.Username)
		if existing != nil {
			return storage.ErrUserExists
		}
	}

	user.UpdatedAt = time.Now().UTC()

	// Update user record
	if err := s.writeUser(ctx, user); err != nil {
		return err
	}

	// Update username index if changed
	if user.Username != current.Username {
		if err := s.deleteUsernameIndex(ctx, current.Username); err != nil {
			return err
		}
		return s.writeUsernameIndex(ctx, user.Username, user.ID)
	}

	return nil
}

// DeleteUser deletes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current user for index cleanup
	current, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete username index
	if err := s.deleteUsernameIndex(ctx, current.Username); err != nil {
		return err
	}

	// Delete user record
	path := s.kvPath(fmt.Sprintf("users/%d", id))
	return s.client.KVv2(s.config.MountPath).Delete(ctx, path)
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	return s.listUsersInternal(ctx)
}

func (s *Store) listUsersInternal(ctx context.Context) ([]*storage.UserRecord, error) {
	path := s.config.BasePath + "/users"
	secret, err := s.client.Logical().ListWithContext(ctx, s.config.MountPath+"/metadata/"+path)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []*storage.UserRecord{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []*storage.UserRecord{}, nil
	}

	var users []*storage.UserRecord
	for _, key := range keys {
		idStr, ok := key.(string)
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		user, err := s.GetUserByID(ctx, id)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// CreateAPIKey creates a new API key record.
func (s *Store) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if key hash already exists
	existing, _ := s.getAPIKeyByHashInternal(ctx, key.KeyHash)
	if existing != nil {
		return storage.ErrAPIKeyExists
	}

	// Generate ID
	id, err := s.nextAPIKeyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate API key ID: %w", err)
	}
	key.ID = id
	key.CreatedAt = time.Now().UTC()

	// Store API key
	if err := s.writeAPIKey(ctx, key); err != nil {
		return err
	}

	// Store hash index
	return s.writeAPIKeyHashIndex(ctx, key.KeyHash, key.ID)
}

func (s *Store) writeAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	path := s.kvPath(fmt.Sprintf("apikeys/%d", key.ID))
	data := map[string]interface{}{
		"id":         key.ID,
		"user_id":    key.UserID,
		"key_hash":   key.KeyHash,
		"key_prefix": key.KeyPrefix,
		"name":       key.Name,
		"role":       key.Role,
		"enabled":    key.Enabled,
		"created_at": key.CreatedAt.Format(time.RFC3339),
		"expires_at": key.ExpiresAt.Format(time.RFC3339),
	}
	if key.LastUsed != nil {
		data["last_used"] = key.LastUsed.Format(time.RFC3339)
	}
	_, err := s.client.KVv2(s.config.MountPath).Put(ctx, path, data)
	return err
}

func (s *Store) writeAPIKeyHashIndex(ctx context.Context, hash string, id int64) error {
	path := s.kvPath("indexes/apikeys/hash/" + hash)
	_, err := s.client.KVv2(s.config.MountPath).Put(ctx, path, map[string]interface{}{
		"id": id,
	})
	return err
}

func (s *Store) deleteAPIKeyHashIndex(ctx context.Context, hash string) error {
	path := s.kvPath("indexes/apikeys/hash/" + hash)
	return s.client.KVv2(s.config.MountPath).Delete(ctx, path)
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Store) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	path := s.kvPath(fmt.Sprintf("apikeys/%d", id))
	secret, err := s.client.KVv2(s.config.MountPath).Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, storage.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return parseAPIKeyRecord(secret.Data)
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	return s.getAPIKeyByHashInternal(ctx, keyHash)
}

func (s *Store) getAPIKeyByHashInternal(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	path := s.kvPath("indexes/apikeys/hash/" + keyHash)
	secret, err := s.client.KVv2(s.config.MountPath).Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, storage.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to lookup API key: %w", err)
	}

	id, err := parseID(secret.Data, "id")
	if err != nil {
		return nil, err
	}

	return s.GetAPIKeyByID(ctx, id)
}

// GetAPIKeyByUserAndName retrieves an API key by user ID and name.
func (s *Store) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
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

// UpdateAPIKey updates an existing API key record.
func (s *Store) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify key exists
	_, err := s.GetAPIKeyByID(ctx, key.ID)
	if err != nil {
		return err
	}

	return s.writeAPIKey(ctx, key)
}

// DeleteAPIKey deletes an API key by ID.
func (s *Store) DeleteAPIKey(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current key for index cleanup
	current, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete hash index
	if err := s.deleteAPIKeyHashIndex(ctx, current.KeyHash); err != nil {
		return err
	}

	// Delete API key record
	path := s.kvPath(fmt.Sprintf("apikeys/%d", id))
	return s.client.KVv2(s.config.MountPath).Delete(ctx, path)
}

// ListAPIKeys returns all API keys.
func (s *Store) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	path := s.config.BasePath + "/apikeys"
	secret, err := s.client.Logical().ListWithContext(ctx, s.config.MountPath+"/metadata/"+path)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []*storage.APIKeyRecord{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []*storage.APIKeyRecord{}, nil
	}

	var apiKeys []*storage.APIKeyRecord
	for _, key := range keys {
		idStr, ok := key.(string)
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		apiKey, err := s.GetAPIKeyByID(ctx, id)
		if err != nil {
			continue
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// ListAPIKeysByUserID returns all API keys for a user.
func (s *Store) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	allKeys, err := s.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	var userKeys []*storage.APIKeyRecord
	for _, key := range allKeys {
		if key.UserID == userID {
			userKeys = append(userKeys, key)
		}
	}

	return userKeys, nil
}

// UpdateAPIKeyLastUsed updates the last_used timestamp for an API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	key.LastUsed = &now

	return s.writeAPIKey(ctx, key)
}

// Close closes the Vault client connection.
func (s *Store) Close() error {
	// Vault client doesn't need explicit closing
	return nil
}

// IsHealthy returns true if the Vault connection is healthy.
func (s *Store) IsHealthy(ctx context.Context) bool {
	health, err := s.client.Sys().Health()
	if err != nil {
		return false
	}
	return health.Initialized && !health.Sealed
}

// Helper functions

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Vault returns 404 for missing secrets
	if respErr, ok := err.(*api.ResponseError); ok {
		return respErr.StatusCode == 404
	}
	return false
}

func parseID(data map[string]interface{}, key string) (int64, error) {
	val, ok := data[key]
	if !ok {
		return 0, fmt.Errorf("missing %s", key)
	}

	switch v := val.(type) {
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("invalid %s type", key)
	}
}

func parseUserRecord(data map[string]interface{}) (*storage.UserRecord, error) {
	user := &storage.UserRecord{}

	id, err := parseID(data, "id")
	if err != nil {
		return nil, err
	}
	user.ID = id

	if v, ok := data["username"].(string); ok {
		user.Username = v
	}
	if v, ok := data["email"].(string); ok {
		user.Email = v
	}
	if v, ok := data["password_hash"].(string); ok {
		user.PasswordHash = v
	}
	if v, ok := data["role"].(string); ok {
		user.Role = v
	}
	if v, ok := data["enabled"].(bool); ok {
		user.Enabled = v
	}
	if v, ok := data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			user.CreatedAt = t
		}
	}
	if v, ok := data["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			user.UpdatedAt = t
		}
	}

	return user, nil
}

func parseAPIKeyRecord(data map[string]interface{}) (*storage.APIKeyRecord, error) {
	key := &storage.APIKeyRecord{}

	id, err := parseID(data, "id")
	if err != nil {
		return nil, err
	}
	key.ID = id

	if userID, err := parseID(data, "user_id"); err == nil {
		key.UserID = userID
	}
	if v, ok := data["key_hash"].(string); ok {
		key.KeyHash = v
	}
	if v, ok := data["key_prefix"].(string); ok {
		key.KeyPrefix = v
	}
	if v, ok := data["name"].(string); ok {
		key.Name = v
	}
	if v, ok := data["role"].(string); ok {
		key.Role = v
	}
	if v, ok := data["enabled"].(bool); ok {
		key.Enabled = v
	}
	if v, ok := data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			key.CreatedAt = t
		}
	}
	if v, ok := data["expires_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			key.ExpiresAt = t
		}
	}
	if v, ok := data["last_used"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			key.LastUsed = &t
		}
	}

	return key, nil
}

// Ensure Store implements storage.AuthStorage
var _ storage.AuthStorage = (*Store)(nil)
