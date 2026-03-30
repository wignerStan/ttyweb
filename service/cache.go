package service

import (
	"sync"
	"time"
)

// CacheEntry represents an item in the shared cache.
type CacheEntry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// SharedCache provides a thread-safe in-memory cache with TTL expiration.
// Designed to be shared between backend instances.
type SharedCache[T any] struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry[T]
	ttl     time.Duration
}

// NewSharedCache creates a cache with the given TTL.
func NewSharedCache[T any](ttl time.Duration) *SharedCache[T] {
	return &SharedCache[T]{
		entries: make(map[string]CacheEntry[T]),
		ttl:     ttl,
	}
}

// Get retrieves a value from the cache. Returns the value and true if found and not expired.
func (c *SharedCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		var zero T
		return zero, false
	}
	return entry.Value, true
}

// Set stores a value in the cache with the default TTL.
func (c *SharedCache[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *SharedCache[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = CacheEntry[T]{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache.
func (c *SharedCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *SharedCache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]CacheEntry[T])
}

// Len returns the number of entries in the cache (including expired).
func (c *SharedCache[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Cleanup removes expired entries. Call periodically.
func (c *SharedCache[T]) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	count := 0
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
			count++
		}
	}
	return count
}
