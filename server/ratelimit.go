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
}

func newVisitorLimiter(r rate.Limit, burst int) *visitorLimiter {
	return &visitorLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
	}
}

func (vl *visitorLimiter) getLimiter(ip string) *rate.Limiter {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	now := time.Now().Unix()
	for key, v := range vl.visitors {
		if now-v.lastSeen > int64(rateLimitCleanupInterval.Seconds()) {
			delete(vl.visitors, key)
		}
	}

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
