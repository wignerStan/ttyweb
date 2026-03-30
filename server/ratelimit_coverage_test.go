package server

import (
	"sync"
	"testing"

	"golang.org/x/time/rate"
)

// TestGetLimiter tests that getLimiter returns existing or creates new limiters.
func TestGetLimiter(t *testing.T) {
	vl := newVisitorLimiter(rate.Limit(10), 5)

	limiter1 := vl.getLimiter("10.0.0.1")
	if limiter1 == nil {
		t.Fatal("expected a limiter")
	}

	// Same IP should return the same limiter.
	limiter2 := vl.getLimiter("10.0.0.1")
	if limiter1 != limiter2 {
		t.Error("expected same limiter for same IP")
	}

	// Different IP should return a different limiter.
	limiter3 := vl.getLimiter("10.0.0.2")
	if limiter1 == limiter3 {
		t.Error("expected different limiter for different IP")
	}
}

// TestVisitorLimiter_ConcurrentAccess tests concurrent access to the visitor limiter.
func TestVisitorLimiter_ConcurrentAccess(t *testing.T) {
	vl := newVisitorLimiter(rate.Limit(100), 100)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = vl.getLimiter("10.0.0.1")
			_ = vl.getLimiter("10.0.0.2")
		}()
	}
	wg.Wait()
}
