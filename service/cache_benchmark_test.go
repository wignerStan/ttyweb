package service

import (
	"fmt"
	"testing"
)

func BenchmarkSharedCache_Get(b *testing.B) {
	cache := NewSharedCache[string](0)
	cache.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("key")
	}
}

func BenchmarkSharedCache_Set(b *testing.B) {
	cache := NewSharedCache[string](0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", fmt.Sprintf("value-%d", i))
	}
}

func BenchmarkSharedCache_ConcurrentGet(b *testing.B) {
	cache := NewSharedCache[string](0)
	cache.Set("key", "value")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get("key")
		}
	})
}
