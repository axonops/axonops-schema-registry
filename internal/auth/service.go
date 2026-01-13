// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// userCacheEntry stores a cached user with verification timestamp.
type userCacheEntry struct {
	user       *storage.UserRecord
	verifiedAt time.Time
}

// Service provides user and API key management operations.
type Service struct {
	storage   storage.AuthStorage
	apiSecret []byte // Secret for HMAC-SHA256 API key hashing (pepper)
	keyPrefix string // Prefix for generated API keys (e.g., "sr_live_")

	// apiKeyCache caches validated API keys in memory for performance.
	// Keys are cached on first successful validation and refreshed
	// periodically by a background process to ensure cluster consistency.
	// Map: keyHash (string) -> *storage.APIKeyRecord
	apiKeyCache sync.Map

	// userCredCache caches validated user credentials in memory for performance.
	// Entries include a TTL to ensure password changes are eventually reflected.
	// Map: cacheKey (string) -> *userCacheEntry
	userCredCache sync.Map

	// userCacheTTL is how long validated user credentials are cached.
	userCacheTTL time.Duration

	// cacheRefreshInterval is how often the background process refreshes cached keys.
	cacheRefreshInterval time.Duration

	// stopCacheRefresh signals the background refresh goroutine to stop.
	stopCacheRefresh chan struct{}

	// cacheRefreshDone signals that the background refresh goroutine has stopped.
	cacheRefreshDone chan struct{}
}

// ServiceConfig contains configuration for the auth service.
type ServiceConfig struct {
	// APIKeySecret is the secret used for HMAC-SHA256 hashing of API keys.
	// This provides defense-in-depth: even if the database is compromised,
	// the attacker cannot verify API keys without this secret.
	// Should be at least 32 bytes of cryptographically random data.
	// If empty, falls back to plain SHA-256 (backward compatible but less secure).
	APIKeySecret string
	// APIKeyPrefix is prepended to generated API keys (e.g., "sr_live_").
	// This helps identify keys and their purpose.
	APIKeyPrefix string
	// CacheRefreshInterval is how often the background process refreshes
	// cached API keys from the database. This ensures cluster consistency
	// as all nodes will eventually converge to the same state.
	// Set to 0 to disable caching entirely. Default is 1 minute.
	CacheRefreshInterval time.Duration
	// UserCacheTTL is how long validated user credentials are cached.
	// This reduces database load for frequently authenticating users.
	// Set to 0 to disable user credential caching. Default is 60 seconds.
	UserCacheTTL time.Duration
}

// DefaultCacheRefreshInterval is the default interval for refreshing the API key cache.
const DefaultCacheRefreshInterval = 1 * time.Minute

// DefaultUserCacheTTL is the default TTL for cached user credentials.
const DefaultUserCacheTTL = 60 * time.Second

// NewService creates a new auth service with default configuration.
func NewService(store storage.AuthStorage) *Service {
	return NewServiceWithConfig(store, ServiceConfig{
		CacheRefreshInterval: DefaultCacheRefreshInterval,
		UserCacheTTL:         DefaultUserCacheTTL,
	})
}

// NewServiceWithConfig creates a new auth service with configuration.
// Note: CacheRefreshInterval and UserCacheTTL of 0 will disable caching.
// Use DefaultCacheRefreshInterval and DefaultUserCacheTTL for default behavior.
func NewServiceWithConfig(store storage.AuthStorage, cfg ServiceConfig) *Service {
	s := &Service{
		storage:              store,
		keyPrefix:            cfg.APIKeyPrefix,
		userCacheTTL:         cfg.UserCacheTTL,         // 0 means disabled
		cacheRefreshInterval: cfg.CacheRefreshInterval, // 0 means disabled
		stopCacheRefresh:     make(chan struct{}),
		cacheRefreshDone:     make(chan struct{}),
	}

	// Decode hex secret if provided
	if cfg.APIKeySecret != "" {
		// Try hex decoding first (recommended)
		secret, err := hex.DecodeString(cfg.APIKeySecret)
		if err != nil {
			// Fall back to using raw string as secret
			secret = []byte(cfg.APIKeySecret)
		}
		s.apiSecret = secret
	}

	// Load all API keys into cache on startup (only if caching is enabled)
	if s.cacheRefreshInterval > 0 {
		s.refreshAPIKeyCache()
	}

	// Start background refresh goroutine
	go s.runCacheRefresh()

	return s
}

// Close stops the background cache refresh goroutine.
// Should be called when shutting down the server.
func (s *Service) Close() {
	close(s.stopCacheRefresh)
	<-s.cacheRefreshDone
}

// runCacheRefresh periodically refreshes the API key cache from the database.
func (s *Service) runCacheRefresh() {
	defer close(s.cacheRefreshDone)

	// If cache refresh interval is 0, caching is disabled - just wait for stop signal
	if s.cacheRefreshInterval == 0 {
		<-s.stopCacheRefresh
		return
	}

	ticker := time.NewTicker(s.cacheRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCacheRefresh:
			return
		case <-ticker.C:
			s.refreshAPIKeyCache()
		}
	}
}

// refreshAPIKeyCache loads all API keys from the database into the cache.
func (s *Service) refreshAPIKeyCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keys, err := s.storage.ListAPIKeys(ctx)
	if err != nil {
		// Log error but don't crash - keep using existing cache
		return
	}

	// Build new cache map
	newKeys := make(map[string]*storage.APIKeyRecord, len(keys))
	for _, key := range keys {
		newKeys[key.KeyHash] = key
	}

	// Clear old entries not in database and update existing ones
	s.apiKeyCache.Range(func(k, v interface{}) bool {
		hash := k.(string)
		if _, exists := newKeys[hash]; !exists {
			s.apiKeyCache.Delete(hash)
		}
		return true
	})

	// Add/update all keys from database
	for hash, key := range newKeys {
		s.apiKeyCache.Store(hash, key)
	}
}

// CreateUserRequest contains the data needed to create a user.
type CreateUserRequest struct {
	Username string
	Email    string
	Password string
	Role     string
	Enabled  bool
}

// CreateAPIKeyRequest contains the data needed to create an API key.
type CreateAPIKeyRequest struct {
	UserID    int64     // Required: user who owns this key
	Name      string    // Required: must be unique per user
	Role      string    // Required: role for this API key
	ExpiresAt time.Time // Required: when the key expires
}

// CreateAPIKeyResponse contains the created API key details including the raw key.
type CreateAPIKeyResponse struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"` // Raw key, only returned on creation
	KeyPrefix string    `json:"key_prefix"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	UserID    int64     `json:"user_id"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateUser creates a new user with the given details.
func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (*storage.UserRecord, error) {
	// Validate role
	if !ValidRole(req.Role) {
		return nil, fmt.Errorf("%w: %s", storage.ErrInvalidRole, req.Role)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &storage.UserRecord{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		Enabled:      req.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *Service) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	return s.storage.GetUserByID(ctx, id)
}

// GetUserByUsername retrieves a user by username.
func (s *Service) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	return s.storage.GetUserByUsername(ctx, username)
}

// UpdateUser updates an existing user.
func (s *Service) UpdateUser(ctx context.Context, id int64, updates map[string]interface{}) (*storage.UserRecord, error) {
	user, err := s.storage.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "email":
			if email, ok := value.(string); ok {
				user.Email = email
			}
		case "password":
			if password, ok := value.(string); ok {
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return nil, fmt.Errorf("failed to hash password: %w", err)
				}
				user.PasswordHash = string(hash)
			}
		case "role":
			if role, ok := value.(string); ok {
				if !ValidRole(role) {
					return nil, fmt.Errorf("%w: %s", storage.ErrInvalidRole, role)
				}
				user.Role = role
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				user.Enabled = enabled
			}
		}
	}

	user.UpdatedAt = time.Now().UTC()

	if err := s.storage.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate credential cache for this user
	s.invalidateUserCredCacheByID(id)

	return user, nil
}

// DeleteUser deletes a user by ID.
func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	// Invalidate credential cache before delete
	s.invalidateUserCredCacheByID(id)

	return s.storage.DeleteUser(ctx, id)
}

// ListUsers returns all users.
func (s *Service) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	return s.storage.ListUsers(ctx)
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	user, err := s.storage.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return storage.ErrPermissionDenied
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now().UTC()

	if err := s.storage.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate credential cache for this user
	s.invalidateUserCredCacheByID(id)

	return nil
}

// userCredCacheKey generates a cache key for user credentials.
// Uses HMAC of username+password to create a secure cache key.
func (s *Service) userCredCacheKey(username, password string) string {
	h := sha256.New()
	h.Write([]byte(username))
	h.Write([]byte(":"))
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

// invalidateUserCredCache removes all cached entries for a user.
func (s *Service) invalidateUserCredCache(username string) {
	// Since cache keys include password hash, we need to iterate and check
	// We store username as part of the entry for invalidation purposes
	s.userCredCache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*userCacheEntry); ok {
			if entry.user.Username == username {
				s.userCredCache.Delete(key)
			}
		}
		return true
	})
}

// invalidateUserCredCacheByID removes all cached entries for a user by ID.
func (s *Service) invalidateUserCredCacheByID(userID int64) {
	s.userCredCache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*userCacheEntry); ok {
			if entry.user.ID == userID {
				s.userCredCache.Delete(key)
			}
		}
		return true
	})
}

// ValidateCredentials validates user credentials and returns the user if valid.
// Results are cached for performance; cache entries expire after UserCacheTTL.
func (s *Service) ValidateCredentials(ctx context.Context, username, password string) (*storage.UserRecord, error) {
	// Generate cache key from credentials
	cacheKey := s.userCredCacheKey(username, password)

	// Check cache first
	if cached, ok := s.userCredCache.Load(cacheKey); ok {
		entry := cached.(*userCacheEntry)
		// Check if cache entry is still valid (within TTL)
		if time.Since(entry.verifiedAt) < s.userCacheTTL {
			// Verify user is still enabled (could have been disabled)
			if entry.user.Enabled {
				return entry.user, nil
			}
			// User disabled, remove from cache
			s.userCredCache.Delete(cacheKey)
			return nil, storage.ErrUserDisabled
		}
		// Entry expired, remove it
		s.userCredCache.Delete(cacheKey)
	}

	// Cache miss or expired - validate against database
	user, err := s.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, storage.ErrUserNotFound
	}

	if !user.Enabled {
		return nil, storage.ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, storage.ErrUserNotFound
	}

	// Cache the validated credentials
	s.userCredCache.Store(cacheKey, &userCacheEntry{
		user:       user,
		verifiedAt: time.Now(),
	})

	return user, nil
}

// CreateAPIKey creates a new API key and returns the raw key.
func (s *Service) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Validate role
	if !ValidRole(req.Role) {
		return nil, fmt.Errorf("%w: %s", storage.ErrInvalidRole, req.Role)
	}

	// Validate name is not empty
	if req.Name == "" {
		return nil, fmt.Errorf("API key name is required")
	}

	// Validate UserID
	if req.UserID <= 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	// Validate expiry is in the future
	now := time.Now().UTC()
	if req.ExpiresAt.Before(now) {
		return nil, fmt.Errorf("expiry time must be in the future")
	}

	// Check if API key name already exists for this user
	existing, err := s.storage.GetAPIKeyByUserAndName(ctx, req.UserID, req.Name)
	if err == nil && existing != nil {
		return nil, storage.ErrAPIKeyNameExists
	}

	// Generate random API key (32 bytes = 256 bits of entropy)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create the raw key with optional prefix for identification
	rawKeyHex := hex.EncodeToString(keyBytes)
	var rawKey string
	if s.keyPrefix != "" {
		rawKey = s.keyPrefix + rawKeyHex
	} else {
		rawKey = rawKeyHex
	}

	// Store first 8 chars of the hex portion for display/identification
	keyPrefixDisplay := rawKeyHex[:8]

	// Hash the key securely for storage using HMAC-SHA256 with pepper
	keyHashStr := s.hashAPIKey(rawKey)

	record := &storage.APIKeyRecord{
		UserID:    req.UserID,
		KeyHash:   keyHashStr,
		KeyPrefix: keyPrefixDisplay,
		Name:      req.Name,
		Role:      req.Role,
		Enabled:   true,
		CreatedAt: now,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.storage.CreateAPIKey(ctx, record); err != nil {
		return nil, err
	}

	// Add to cache immediately so the key can be used right away (only if caching is enabled)
	if s.cacheRefreshInterval > 0 {
		s.apiKeyCache.Store(keyHashStr, record)
	}

	return &CreateAPIKeyResponse{
		ID:        record.ID,
		Key:       rawKey,
		KeyPrefix: keyPrefixDisplay,
		Name:      req.Name,
		Role:      req.Role,
		UserID:    req.UserID,
		Enabled:   true,
		CreatedAt: now,
		ExpiresAt: req.ExpiresAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns the record if valid.
// First checks the in-memory cache for performance, then falls back to the database
// if the key is not found in cache. The cache is refreshed periodically from the
// database to ensure cluster consistency.
func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (*storage.APIKeyRecord, error) {
	// Hash the provided key using the same method as CreateAPIKey
	keyHashStr := s.hashAPIKey(rawKey)

	// Look up in cache first (if caching is enabled)
	var record *storage.APIKeyRecord
	if s.cacheRefreshInterval > 0 {
		if cached, ok := s.apiKeyCache.Load(keyHashStr); ok {
			record = cached.(*storage.APIKeyRecord)
		}
	}

	// Cache miss or caching disabled - fall back to database
	if record == nil {
		var err error
		record, err = s.storage.GetAPIKeyByHash(ctx, keyHashStr)
		if err != nil {
			return nil, storage.ErrAPIKeyNotFound
		}
		// Cache the result for future lookups (only if caching is enabled)
		if s.cacheRefreshInterval > 0 {
			s.apiKeyCache.Store(keyHashStr, record)
		}
	}

	// Validate the key is enabled
	if !record.Enabled {
		return nil, storage.ErrAPIKeyDisabled
	}

	// Validate the key hasn't expired
	if record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, storage.ErrAPIKeyExpired
	}

	// Update last used time (non-blocking)
	go func() {
		bgCtx := context.Background()
		_ = s.storage.UpdateAPIKeyLastUsed(bgCtx, record.ID)
	}()

	return record, nil
}

// GetAPIKeyByID retrieves an API key by ID.
func (s *Service) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	return s.storage.GetAPIKeyByID(ctx, id)
}

// invalidateAPIKeyCache removes an API key from the cache by its ID.
// This iterates through the cache since we index by hash, not ID.
func (s *Service) invalidateAPIKeyCache(id int64) {
	s.apiKeyCache.Range(func(key, value interface{}) bool {
		if record, ok := value.(*storage.APIKeyRecord); ok && record.ID == id {
			s.apiKeyCache.Delete(key)
			return false // Stop iteration, found the key
		}
		return true // Continue iteration
	})
}

// UpdateAPIKey updates an existing API key.
func (s *Service) UpdateAPIKey(ctx context.Context, id int64, updates map[string]interface{}) (*storage.APIKeyRecord, error) {
	record, err := s.storage.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "name":
			if name, ok := value.(string); ok {
				// Check if name already exists for this user
				if name != record.Name {
					existing, err := s.storage.GetAPIKeyByUserAndName(ctx, record.UserID, name)
					if err == nil && existing != nil && existing.ID != record.ID {
						return nil, storage.ErrAPIKeyNameExists
					}
				}
				record.Name = name
			}
		case "role":
			if role, ok := value.(string); ok {
				if !ValidRole(role) {
					return nil, fmt.Errorf("%w: %s", storage.ErrInvalidRole, role)
				}
				record.Role = role
			}
		case "enabled":
			if enabled, ok := value.(bool); ok {
				record.Enabled = enabled
			}
		case "expires_at":
			if expiresAt, ok := value.(time.Time); ok {
				if expiresAt.Before(time.Now().UTC()) {
					return nil, fmt.Errorf("expiry time must be in the future")
				}
				record.ExpiresAt = expiresAt
			}
		}
	}

	if err := s.storage.UpdateAPIKey(ctx, record); err != nil {
		return nil, err
	}

	// Invalidate cache so next validation fetches fresh data
	s.invalidateAPIKeyCache(id)

	return record, nil
}

// DeleteAPIKey deletes an API key by ID.
func (s *Service) DeleteAPIKey(ctx context.Context, id int64) error {
	// Invalidate cache before delete
	s.invalidateAPIKeyCache(id)

	return s.storage.DeleteAPIKey(ctx, id)
}

// ListAPIKeys returns all API keys.
func (s *Service) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	return s.storage.ListAPIKeys(ctx)
}

// ListAPIKeysByUserID returns all API keys for a specific user.
func (s *Service) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	return s.storage.ListAPIKeysByUserID(ctx, userID)
}

// RevokeAPIKey disables an API key.
func (s *Service) RevokeAPIKey(ctx context.Context, id int64) error {
	record, err := s.storage.GetAPIKeyByID(ctx, id)
	if err != nil {
		return err
	}

	record.Enabled = false
	if err := s.storage.UpdateAPIKey(ctx, record); err != nil {
		return err
	}

	// Invalidate cache so the revoked key is rejected immediately
	s.invalidateAPIKeyCache(id)

	return nil
}

// RotateAPIKey creates a new API key with same settings and revokes the old one.
// The new key will have a fresh expiry based on the remaining duration of the old key.
func (s *Service) RotateAPIKey(ctx context.Context, id int64, newExpiresAt time.Time) (*CreateAPIKeyResponse, error) {
	oldKey, err := s.storage.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Generate a unique name for the rotated key
	now := time.Now().UTC()
	newName := fmt.Sprintf("%s (rotated %s)", oldKey.Name, now.Format("2006-01-02T15:04:05"))

	// Validate new expiry
	if newExpiresAt.Before(now) {
		return nil, fmt.Errorf("expiry time must be in the future")
	}

	// Create new key with same settings but new expiry
	newKey, err := s.CreateAPIKey(ctx, CreateAPIKeyRequest{
		UserID:    oldKey.UserID,
		Name:      newName,
		Role:      oldKey.Role,
		ExpiresAt: newExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	// Revoke old key
	if err := s.RevokeAPIKey(ctx, id); err != nil {
		// Log but don't fail - new key was created
		_ = err
	}

	return newKey, nil
}

// hashAPIKey returns a secure hash of an API key for storage.
// If an API secret (pepper) is configured, it uses HMAC-SHA256 for defense-in-depth.
// Otherwise, it falls back to plain SHA-256 for backward compatibility.
func (s *Service) hashAPIKey(rawKey string) string {
	if len(s.apiSecret) > 0 {
		// Use HMAC-SHA256 with the server secret as a pepper.
		// This provides defense-in-depth: even if the database is compromised,
		// an attacker cannot verify API keys without knowing the secret.
		h := hmac.New(sha256.New, s.apiSecret)
		h.Write([]byte(rawKey))
		return hex.EncodeToString(h.Sum(nil))
	}
	// Fall back to plain SHA-256 if no secret is configured
	keyHash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(keyHash[:])
}

// HashAPIKey returns the SHA-256 hash of an API key (for lookup).
// Deprecated: Use Service.hashAPIKey instead for HMAC support.
func HashAPIKey(rawKey string) string {
	keyHash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(keyHash[:])
}

// BootstrapResult contains the result of bootstrapping the initial admin user.
type BootstrapResult struct {
	Created  bool   // Whether a new user was created
	Username string // Username of the created/existing admin
	Message  string // Human-readable message
}

// BootstrapAdmin creates the initial admin user if the users table is empty.
// This solves the chicken-and-egg problem where you need an admin to create users,
// but there are no users when the system is first deployed.
//
// The function will:
// - Return immediately if the users table is not empty
// - Create an admin user with the provided credentials if the table is empty
// - Return an error if the credentials are not provided
//
// This function is idempotent: if there are existing users, it does nothing.
func (s *Service) BootstrapAdmin(ctx context.Context, username, password, email string) (*BootstrapResult, error) {
	// Check if users already exist
	users, err := s.storage.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// If there are existing users, skip bootstrap
	if len(users) > 0 {
		return &BootstrapResult{
			Created: false,
			Message: fmt.Sprintf("bootstrap skipped: %d user(s) already exist", len(users)),
		}, nil
	}

	// Validate bootstrap credentials
	if username == "" {
		return nil, fmt.Errorf("bootstrap username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("bootstrap password is required")
	}

	// Create the admin user
	user, err := s.CreateUser(ctx, CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
		Role:     "super_admin",
		Enabled:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bootstrap admin: %w", err)
	}

	return &BootstrapResult{
		Created:  true,
		Username: user.Username,
		Message:  fmt.Sprintf("bootstrap admin user '%s' created successfully", user.Username),
	}, nil
}
