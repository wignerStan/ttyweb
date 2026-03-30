package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimitMiddleware_AllowsWithinBurst(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(100, 5)
	handler := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksOverBurst(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(0, 2) // 0/s no refill, burst 2
	handler := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_SeparateIPs(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(0, 1)
	var called atomic.Int32
	handler := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	for _, ip := range []string{"10.0.0.1:1234", "10.0.0.2:1234"} {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, rec.Code)
		}
	}
	if called.Load() != 2 {
		t.Errorf("expected 2 calls, got %d", called.Load())
	}
}

func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(0, 1)
	var called atomic.Int32
	handler := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	// First request from real IP
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Second request with same IP but different X-Forwarded-For should use forwarded IP
	req2 := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	req2.Header.Set("X-Forwarded-For", "10.0.0.99")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 (different forwarded IP), got %d", rec2.Code)
	}

	// Third request from forwarded IP should be rate limited
	req3 := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req3.RemoteAddr = "10.0.0.1:1234"
	req3.Header.Set("X-Forwarded-For", "10.0.0.99")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (forwarded IP exhausted), got %d", rec3.Code)
	}
}

func TestExtractIP_NoPort(t *testing.T) {
	t.Parallel()
	ip := extractIP("10.0.0.1") // no port
	if ip != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got '%s'", ip)
	}
}

func TestRateLimitMiddleware_CleansStaleEntries(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(0, 1)
	handler := rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.RemoteAddr = "10.99.99.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	limiter.mu.Lock()
	for _, v := range limiter.visitors {
		v.lastSeen = time.Now().Add(-4 * time.Minute).Unix()
	}
	limiter.mu.Unlock()

	req2 := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req2.RemoteAddr = "10.99.99.2:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	limiter.mu.Lock()
	count := len(limiter.visitors)
	limiter.mu.Unlock()
	if count > 2 {
		t.Errorf("expected <= 2 visitors after cleanup, got %d", count)
	}
}
