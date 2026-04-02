package server

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const rateLimitCleanupInterval = 3 * time.Minute

type visitor struct {
	limiter  *rate.Limiter
	lastSeen int64
}

type visitorLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
	done     chan struct{}
	stopOnce sync.Once
}

func newVisitorLimiter(r rate.Limit, burst int) *visitorLimiter {
	vl := &visitorLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
		done:     make(chan struct{}),
	}
	go vl.cleanupLoop()
	return vl
}

// cleanupLoop periodically removes stale visitor entries to bound memory.
func (vl *visitorLimiter) cleanupLoop() {
	ticker := time.NewTicker(rateLimitCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			vl.mu.Lock()
			now := time.Now().Unix()
			for ip, v := range vl.visitors {
				if now-v.lastSeen > int64(rateLimitCleanupInterval.Seconds()) {
					delete(vl.visitors, ip)
				}
			}
			vl.mu.Unlock()
		case <-vl.done:
			return
		}
	}
}

// stop signals the cleanup goroutine to exit. Safe to call multiple times.
func (vl *visitorLimiter) stop() {
	vl.stopOnce.Do(func() { close(vl.done) })
}

func (vl *visitorLimiter) getLimiter(ip string) *rate.Limiter {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	now := time.Now().Unix()
	existing, ok := vl.visitors[ip]
	if ok {
		existing.lastSeen = now
		return existing.limiter
	}

	limiter := rate.NewLimiter(vl.rate, vl.burst)
	vl.visitors[ip] = &visitor{limiter: limiter, lastSeen: now}
	return limiter
}

func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func rateLimitMiddleware(limiter *visitorLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r.RemoteAddr)
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = fwd
			}
			if !limiter.getLimiter(ip).Allow() {
				http.Error(w, `{"success":false,"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
