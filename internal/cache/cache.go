// Package cache provides caching functionality for the schema registry.
package cache

import (
	"sync"
	"time"
)

// Cache is a simple in-memory cache with LRU eviction.
type Cache struct {
	capacity int
	ttl      time.Duration
	mu       sync.RWMutex
	items    map[string]*cacheItem
	order    []string // For LRU tracking
}

// cacheItem represents a cached item.
type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// New creates a new cache with the specified capacity and TTL.
func New(capacity int, ttl time.Duration) *Cache {
	return &Cache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*cacheItem),
		order:    make([]string, 0, capacity),
	}
}

// Get retrieves an item from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		c.Delete(key)
		return nil, false
	}

	// Move to end of order list (most recently used)
	c.mu.Lock()
	c.moveToEnd(key)
	c.mu.Unlock()

	return item.value, true
}

// Set stores an item in the cache.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if _, exists := c.items[key]; exists {
		c.items[key] = &cacheItem{
			value:     value,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.moveToEnd(key)
		return
	}

	// Evict if at capacity
	if len(c.items) >= c.capacity && c.capacity > 0 {
		c.evict()
	}

	// Add new item
	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.order = append(c.order, key)
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	c.removeFromOrder(key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.order = make([]string, 0, c.capacity)
}

// Size returns the number of items in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evict removes the least recently used item.
func (c *Cache) evict() {
	if len(c.order) == 0 {
		return
	}

	// Remove oldest (first in order)
	oldest := c.order[0]
	c.order = c.order[1:]
	delete(c.items, oldest)
}

// moveToEnd moves a key to the end of the order list.
func (c *Cache) moveToEnd(key string) {
	c.removeFromOrder(key)
	c.order = append(c.order, key)
}

// removeFromOrder removes a key from the order list.
func (c *Cache) removeFromOrder(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// CleanupExpired removes all expired items from the cache.
func (c *Cache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
			c.removeFromOrder(key)
			removed++
		}
	}
	return removed
}

// Stats returns cache statistics.
type Stats struct {
	Size     int
	Capacity int
}

// Stats returns the current cache statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Stats{
		Size:     len(c.items),
		Capacity: c.capacity,
	}
}

// SchemaCache provides specialized caching for parsed schemas.
type SchemaCache struct {
	cache *Cache
}

// NewSchemaCache creates a new schema cache.
func NewSchemaCache(capacity int, ttl time.Duration) *SchemaCache {
	return &SchemaCache{
		cache: New(capacity, ttl),
	}
}

// Get retrieves a parsed schema from the cache.
func (c *SchemaCache) Get(schemaType, fingerprint string) (interface{}, bool) {
	key := schemaType + ":" + fingerprint
	return c.cache.Get(key)
}

// Set stores a parsed schema in the cache.
func (c *SchemaCache) Set(schemaType, fingerprint string, parsed interface{}) {
	key := schemaType + ":" + fingerprint
	c.cache.Set(key, parsed)
}

// Size returns the cache size.
func (c *SchemaCache) Size() int {
	return c.cache.Size()
}

// Clear clears the cache.
func (c *SchemaCache) Clear() {
	c.cache.Clear()
}

// CompatibilityCache provides specialized caching for compatibility results.
type CompatibilityCache struct {
	cache *Cache
}

// NewCompatibilityCache creates a new compatibility cache.
func NewCompatibilityCache(capacity int, ttl time.Duration) *CompatibilityCache {
	return &CompatibilityCache{
		cache: New(capacity, ttl),
	}
}

// CompatibilityResult represents a cached compatibility result.
type CompatibilityResult struct {
	Compatible bool
	Messages   []string
}

// Get retrieves a compatibility result from the cache.
func (c *CompatibilityCache) Get(mode, schemaType, newFingerprint, existingFingerprint string) (*CompatibilityResult, bool) {
	key := mode + ":" + schemaType + ":" + newFingerprint + ":" + existingFingerprint
	result, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	return result.(*CompatibilityResult), true
}

// Set stores a compatibility result in the cache.
func (c *CompatibilityCache) Set(mode, schemaType, newFingerprint, existingFingerprint string, result *CompatibilityResult) {
	key := mode + ":" + schemaType + ":" + newFingerprint + ":" + existingFingerprint
	c.cache.Set(key, result)
}

// Size returns the cache size.
func (c *CompatibilityCache) Size() int {
	return c.cache.Size()
}

// Clear clears the cache.
func (c *CompatibilityCache) Clear() {
	c.cache.Clear()
}
