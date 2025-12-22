package cache

import (
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	c := New(10, time.Hour)

	// Test Set and Get
	c.Set("key1", "value1")
	val, ok := c.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Test missing key
	_, ok = c.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(10, 50*time.Millisecond)

	c.Set("key1", "value1")

	// Should exist immediately
	_, ok := c.Get("key1")
	if !ok {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = c.Get("key1")
	if ok {
		t.Error("Expected key1 to be expired")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	c := New(3, time.Hour)

	// Fill cache
	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	// Access key1 to make it recently used
	c.Get("key1")

	// Add another item, should evict key2 (oldest not accessed)
	c.Set("key4", "value4")

	if c.Size() != 3 {
		t.Errorf("Expected size 3, got %d", c.Size())
	}

	// key1 should still exist (was accessed)
	_, ok := c.Get("key1")
	if !ok {
		t.Error("Expected key1 to still exist")
	}

	// key4 should exist (just added)
	_, ok = c.Get("key4")
	if !ok {
		t.Error("Expected key4 to exist")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New(10, time.Hour)

	c.Set("key1", "value1")
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(10, time.Hour)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Clear()

	if c.Size() != 0 {
		t.Errorf("Expected empty cache, got size %d", c.Size())
	}
}

func TestCache_CleanupExpired(t *testing.T) {
	c := New(10, 50*time.Millisecond)

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	removed := c.CleanupExpired()
	if removed != 2 {
		t.Errorf("Expected 2 items removed, got %d", removed)
	}

	if c.Size() != 0 {
		t.Errorf("Expected empty cache after cleanup, got size %d", c.Size())
	}
}

func TestCache_Stats(t *testing.T) {
	c := New(10, time.Hour)

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	stats := c.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}
	if stats.Capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", stats.Capacity)
	}
}

func TestCache_UpdateExisting(t *testing.T) {
	c := New(10, time.Hour)

	c.Set("key1", "value1")
	c.Set("key1", "value2")

	val, ok := c.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	if c.Size() != 1 {
		t.Errorf("Expected size 1, got %d", c.Size())
	}
}

func TestSchemaCache(t *testing.T) {
	c := NewSchemaCache(10, time.Hour)

	type mockParsed struct {
		name string
	}

	parsed := &mockParsed{name: "test"}
	c.Set("AVRO", "abc123", parsed)

	result, ok := c.Get("AVRO", "abc123")
	if !ok {
		t.Error("Expected to find cached schema")
	}
	if result.(*mockParsed).name != "test" {
		t.Error("Expected cached schema to match")
	}

	_, ok = c.Get("AVRO", "nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent fingerprint")
	}
}

func TestCompatibilityCache(t *testing.T) {
	c := NewCompatibilityCache(10, time.Hour)

	result := &CompatibilityResult{
		Compatible: true,
		Messages:   nil,
	}
	c.Set("BACKWARD", "AVRO", "new123", "old123", result)

	cached, ok := c.Get("BACKWARD", "AVRO", "new123", "old123")
	if !ok {
		t.Error("Expected to find cached result")
	}
	if !cached.Compatible {
		t.Error("Expected compatible to be true")
	}

	_, ok = c.Get("BACKWARD", "AVRO", "different", "old123")
	if ok {
		t.Error("Expected not to find different fingerprint")
	}
}

func TestCache_ZeroCapacity(t *testing.T) {
	// Zero capacity means unlimited
	c := New(0, time.Hour)

	c.Set("key1", "value1")
	c.Set("key2", "value2")

	if c.Size() != 2 {
		t.Errorf("Expected size 2, got %d", c.Size())
	}
}

func TestCache_Concurrent(t *testing.T) {
	c := New(100, time.Hour)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a'+n)) + string(rune('0'+j%10))
				c.Set(key, j)
				c.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and size should be reasonable
	if c.Size() > 100 {
		t.Errorf("Expected size <= 100, got %d", c.Size())
	}
}
