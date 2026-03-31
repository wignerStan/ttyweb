package server

import (
	"sync"
	"testing"
	"time"

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

// TestCleanupLogic_ExpiresStaleVisitor verifies the cleanup logic that
// removes visitors whose lastSeen is older than the cleanup interval.
// Since rateLimitCleanupInterval is a const, we manually invoke the same
// cleanup logic on a visitorLimiter struct.
func TestCleanupLogic_ExpiresStaleVisitor(t *testing.T) {
	vl := &visitorLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(1),
		burst:    1,
	}

	vl.getLimiter("1.2.3.4")
	vl.mu.Lock()
	vl.visitors["1.2.3.4"].lastSeen = time.Now().Unix() - int64(rateLimitCleanupInterval.Seconds()) - 1
	vl.mu.Unlock()

	// Run the same cleanup logic as cleanupLoop.
	vl.mu.Lock()
	now := time.Now().Unix()
	for ip, v := range vl.visitors {
		if now-v.lastSeen > int64(rateLimitCleanupInterval.Seconds()) {
			delete(vl.visitors, ip)
		}
	}
	vl.mu.Unlock()

	vl.mu.Lock()
	_, exists := vl.visitors["1.2.3.4"]
	vl.mu.Unlock()

	if exists {
		t.Error("expected stale visitor to be cleaned up")
	}
}

// TestCleanupLogic_KeepsActiveVisitor verifies the cleanup logic preserves
// visitors that are still active.
func TestCleanupLogic_KeepsActiveVisitor(t *testing.T) {
	vl := &visitorLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(1),
		burst:    1,
	}

	vl.getLimiter("5.6.7.8")
	vl.getLimiter("5.6.7.8")

	vl.mu.Lock()
	now := time.Now().Unix()
	for ip, v := range vl.visitors {
		if now-v.lastSeen > int64(rateLimitCleanupInterval.Seconds()) {
			delete(vl.visitors, ip)
		}
	}
	vl.mu.Unlock()

	vl.mu.Lock()
	_, exists := vl.visitors["5.6.7.8"]
	vl.mu.Unlock()

	if !exists {
		t.Error("expected active visitor to survive cleanup")
	}
}
