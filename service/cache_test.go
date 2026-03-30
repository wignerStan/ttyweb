package service

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedCache_SetAndGet(t *testing.T) {
	cache := NewSharedCache[string](5 * time.Minute)

	cache.Set("key1", "value1")
	val, ok := cache.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestSharedCache_Expired(t *testing.T) {
	cache := NewSharedCache[string](1 * time.Millisecond)

	cache.Set("key1", "value1")
	time.Sleep(10 * time.Millisecond)

	_, ok := cache.Get("key1")
	assert.False(t, ok, "expired entry should not be found")
}

func TestSharedCache_Delete(t *testing.T) {
	cache := NewSharedCache[string](5 * time.Minute)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	assert.False(t, ok, "deleted entry should not be found")
}

func TestSharedCache_Clear(t *testing.T) {
	cache := NewSharedCache[string](5 * time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	assert.Equal(t, 3, cache.Len())

	cache.Clear()
	assert.Equal(t, 0, cache.Len())

	_, ok := cache.Get("key1")
	assert.False(t, ok)
}

func TestSharedCache_SetWithTTL(t *testing.T) {
	cache := NewSharedCache[string](5 * time.Minute)

	// Set with short custom TTL
	cache.SetWithTTL("short", "value", 1*time.Millisecond)
	// Set with long custom TTL
	cache.SetWithTTL("long", "value", 1*time.Hour)

	time.Sleep(10 * time.Millisecond)

	_, shortOk := cache.Get("short")
	assert.False(t, shortOk, "short TTL entry should be expired")

	_, longOk := cache.Get("long")
	assert.True(t, longOk, "long TTL entry should still be present")
}

func TestSharedCache_Cleanup(t *testing.T) {
	cache := NewSharedCache[int](5 * time.Minute)

	// Set entries: some already expired, some not
	cache.SetWithTTL("expired1", 1, 1*time.Millisecond)
	cache.SetWithTTL("expired2", 2, 1*time.Millisecond)
	cache.SetWithTTL("valid1", 3, 1*time.Hour)
	cache.SetWithTTL("valid2", 4, 1*time.Hour)

	time.Sleep(10 * time.Millisecond)

	removed := cache.Cleanup()
	assert.Equal(t, 2, removed, "should have removed 2 expired entries")
	assert.Equal(t, 2, cache.Len(), "should have 2 remaining entries")

	_, ok := cache.Get("valid1")
	assert.True(t, ok)
	_, ok = cache.Get("valid2")
	assert.True(t, ok)
}

func TestSharedCache_ConcurrentAccess(t *testing.T) {
	cache := NewSharedCache[int](5 * time.Minute)

	var wg sync.WaitGroup
	const goroutines = 50
	const opsPerGoroutine = 100

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := string(rune('a' + (id%26))) + "-" + string(rune('0'+(j%10)))
				cache.Set(key, id*opsPerGoroutine+j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := string(rune('a' + (id%26))) + "-" + string(rune('0'+(j%10)))
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify the cache is in a consistent state
	assert.Greater(t, cache.Len(), 0)
}
