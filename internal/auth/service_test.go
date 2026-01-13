package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// mockAuthStorage is a simple mock for testing auth service caching.
type mockAuthStorage struct {
	users         map[string]*storage.UserRecord
	apiKeys       map[string]*storage.APIKeyRecord
	getUserCalls  int64
	listKeysCalls int64
}

func newMockAuthStorage() *mockAuthStorage {
	return &mockAuthStorage{
		users:   make(map[string]*storage.UserRecord),
		apiKeys: make(map[string]*storage.APIKeyRecord),
	}
}

func (m *mockAuthStorage) CreateUser(ctx context.Context, user *storage.UserRecord) error {
	m.users[user.Username] = user
	return nil
}

func (m *mockAuthStorage) GetUserByID(ctx context.Context, id int64) (*storage.UserRecord, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, storage.ErrUserNotFound
}

func (m *mockAuthStorage) GetUserByUsername(ctx context.Context, username string) (*storage.UserRecord, error) {
	atomic.AddInt64(&m.getUserCalls, 1)
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, storage.ErrUserNotFound
}

func (m *mockAuthStorage) UpdateUser(ctx context.Context, user *storage.UserRecord) error {
	m.users[user.Username] = user
	return nil
}

func (m *mockAuthStorage) DeleteUser(ctx context.Context, id int64) error {
	for username, u := range m.users {
		if u.ID == id {
			delete(m.users, username)
			return nil
		}
	}
	return storage.ErrUserNotFound
}

func (m *mockAuthStorage) ListUsers(ctx context.Context) ([]*storage.UserRecord, error) {
	var users []*storage.UserRecord
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

func (m *mockAuthStorage) CreateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	m.apiKeys[key.KeyHash] = key
	return nil
}

func (m *mockAuthStorage) GetAPIKeyByID(ctx context.Context, id int64) (*storage.APIKeyRecord, error) {
	for _, k := range m.apiKeys {
		if k.ID == id {
			return k, nil
		}
	}
	return nil, storage.ErrAPIKeyNotFound
}

func (m *mockAuthStorage) GetAPIKeyByHash(ctx context.Context, keyHash string) (*storage.APIKeyRecord, error) {
	if key, ok := m.apiKeys[keyHash]; ok {
		return key, nil
	}
	return nil, storage.ErrAPIKeyNotFound
}

func (m *mockAuthStorage) GetAPIKeyByUserAndName(ctx context.Context, userID int64, name string) (*storage.APIKeyRecord, error) {
	for _, k := range m.apiKeys {
		if k.UserID == userID && k.Name == name {
			return k, nil
		}
	}
	return nil, storage.ErrAPIKeyNotFound
}

func (m *mockAuthStorage) UpdateAPIKey(ctx context.Context, key *storage.APIKeyRecord) error {
	m.apiKeys[key.KeyHash] = key
	return nil
}

func (m *mockAuthStorage) DeleteAPIKey(ctx context.Context, id int64) error {
	for hash, k := range m.apiKeys {
		if k.ID == id {
			delete(m.apiKeys, hash)
			return nil
		}
	}
	return storage.ErrAPIKeyNotFound
}

func (m *mockAuthStorage) ListAPIKeys(ctx context.Context) ([]*storage.APIKeyRecord, error) {
	atomic.AddInt64(&m.listKeysCalls, 1)
	var keys []*storage.APIKeyRecord
	for _, k := range m.apiKeys {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockAuthStorage) ListAPIKeysByUserID(ctx context.Context, userID int64) ([]*storage.APIKeyRecord, error) {
	var keys []*storage.APIKeyRecord
	for _, k := range m.apiKeys {
		if k.UserID == userID {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *mockAuthStorage) UpdateAPIKeyLastUsed(ctx context.Context, id int64) error {
	return nil
}

func TestService_CacheDisabled_UserCredentials(t *testing.T) {
	store := newMockAuthStorage()

	// Create a user with bcrypt password
	password := "testpassword"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store.users["testuser"] = &storage.UserRecord{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hash),
		Role:         "admin",
		Enabled:      true,
	}

	// Create service with UserCacheTTL = 0 (caching disabled)
	svc := NewServiceWithConfig(store, ServiceConfig{
		UserCacheTTL:         0, // Disable user credential caching
		CacheRefreshInterval: 0, // Disable API key cache refresh
	})
	defer svc.Close()

	ctx := context.Background()

	// First validation - should hit database
	_, err = svc.ValidateCredentials(ctx, "testuser", password)
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	firstCallCount := atomic.LoadInt64(&store.getUserCalls)
	if firstCallCount != 1 {
		t.Errorf("expected 1 database call, got %d", firstCallCount)
	}

	// Second validation with same credentials - should still hit database
	// because caching is disabled (TTL=0)
	_, err = svc.ValidateCredentials(ctx, "testuser", password)
	if err != nil {
		t.Fatalf("second validation failed: %v", err)
	}

	secondCallCount := atomic.LoadInt64(&store.getUserCalls)
	if secondCallCount != 2 {
		t.Errorf("expected 2 database calls (cache disabled), got %d", secondCallCount)
	}
}

func TestService_CacheEnabled_UserCredentials(t *testing.T) {
	store := newMockAuthStorage()

	// Create a user with bcrypt password
	password := "testpassword"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store.users["testuser"] = &storage.UserRecord{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hash),
		Role:         "admin",
		Enabled:      true,
	}

	// Create service with caching enabled
	svc := NewServiceWithConfig(store, ServiceConfig{
		UserCacheTTL:         1 * time.Minute, // Enable user credential caching
		CacheRefreshInterval: 0,               // Disable API key cache refresh
	})
	defer svc.Close()

	ctx := context.Background()

	// First validation - should hit database
	_, err = svc.ValidateCredentials(ctx, "testuser", password)
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	firstCallCount := atomic.LoadInt64(&store.getUserCalls)
	if firstCallCount != 1 {
		t.Errorf("expected 1 database call, got %d", firstCallCount)
	}

	// Second validation with same credentials - should use cache
	_, err = svc.ValidateCredentials(ctx, "testuser", password)
	if err != nil {
		t.Fatalf("second validation failed: %v", err)
	}

	// Call count should still be 1 (used cache)
	secondCallCount := atomic.LoadInt64(&store.getUserCalls)
	if secondCallCount != 1 {
		t.Errorf("expected 1 database call (cache hit), got %d", secondCallCount)
	}
}

func TestService_CacheRefreshDisabled(t *testing.T) {
	store := newMockAuthStorage()

	// Create service with cache refresh disabled (interval = 0)
	svc := NewServiceWithConfig(store, ServiceConfig{
		CacheRefreshInterval: 0, // Disable cache refresh
	})

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Close the service
	svc.Close()

	// Should not panic and should shut down cleanly
	// The test passes if we get here without hanging or panicking
}

func TestService_Close(t *testing.T) {
	store := newMockAuthStorage()

	svc := NewServiceWithConfig(store, ServiceConfig{
		CacheRefreshInterval: 100 * time.Millisecond,
	})

	// Close should complete without hanging
	done := make(chan struct{})
	go func() {
		svc.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not complete within timeout")
	}
}

func TestService_CacheDisabled_APIKey(t *testing.T) {
	store := newMockAuthStorage()

	// Create a user first
	store.users["testuser"] = &storage.UserRecord{
		ID:       1,
		Username: "testuser",
		Role:     "admin",
		Enabled:  true,
	}

	// Create service with cache refresh disabled (interval = 0)
	svc := NewServiceWithConfig(store, ServiceConfig{
		CacheRefreshInterval: 0, // Disable caching
	})
	defer svc.Close()

	ctx := context.Background()

	// Create an API key
	createResp, err := svc.CreateAPIKey(ctx, CreateAPIKeyRequest{
		Name:      "test-api-key",
		UserID:    1,
		Role:      "admin",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	rawKey := createResp.Key

	// First validation - should hit database
	_, err = svc.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("first validation failed: %v", err)
	}

	// Check that no caching occurred (cache should be empty)
	cacheCount := 0
	svc.apiKeyCache.Range(func(k, v interface{}) bool {
		cacheCount++
		return true
	})
	if cacheCount != 0 {
		t.Errorf("expected empty cache when caching disabled, but found %d entries", cacheCount)
	}

	// Second validation should still work (hitting database again)
	_, err = svc.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("second validation failed: %v", err)
	}
}

func TestService_ValidateAPIKey_DBFallback(t *testing.T) {
	store := newMockAuthStorage()

	// Create service with cache refresh disabled (so cache won't be populated)
	svc := NewServiceWithConfig(store, ServiceConfig{
		CacheRefreshInterval: 0, // Disable cache refresh
	})
	defer svc.Close()

	// Create an API key directly in storage (bypassing cache)
	keyHash := "test-key-hash"
	store.apiKeys[keyHash] = &storage.APIKeyRecord{
		ID:        1,
		UserID:    1,
		Name:      "test-key",
		KeyHash:   keyHash,
		Role:      "admin",
		Enabled:   true,
		ExpiresAt: time.Now().Add(time.Hour), // Not expired
	}

	ctx := context.Background()

	// Create a user first (required for API key)
	store.users["testuser"] = &storage.UserRecord{
		ID:       1,
		Username: "testuser",
		Role:     "admin",
		Enabled:  true,
	}

	// Create a key using the service (will add to both DB and cache)
	createResp, err := svc.CreateAPIKey(ctx, CreateAPIKeyRequest{
		Name:      "test-api-key",
		UserID:    1,
		Role:      "admin",
		ExpiresAt: time.Now().Add(24 * time.Hour), // Expires in 24 hours
	})
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	// Get the raw key from the response
	rawKey := createResp.Key

	// Validate should work (from cache or DB)
	record, err := svc.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("failed to validate API key: %v", err)
	}
	if record.Name != "test-api-key" {
		t.Errorf("expected name 'test-api-key', got %q", record.Name)
	}

	// Clear the cache manually to test DB fallback
	svc.apiKeyCache = sync.Map{}

	// Validate again - should still work by falling back to DB
	record, err = svc.ValidateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("failed to validate API key after cache clear (DB fallback should work): %v", err)
	}
	if record.Name != "test-api-key" {
		t.Errorf("expected name 'test-api-key', got %q", record.Name)
	}
}
