package server

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig holds the CORS policy configuration.
// When AllowedOrigins is empty, all cross-origin requests are denied (same-origin only).
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func corsMiddleware(config *CORSConfig) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, o := range config.AllowedOrigins {
		allowedSet[o] = struct{}{}
	}
	methods := strings.Join(config.AllowedMethods, ", ")
	headers := strings.Join(config.AllowedHeaders, ", ")
	expose := strings.Join(config.ExposeHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			_, allowed := allowedSet[origin]
			if !allowed {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if expose != "" {
				w.Header().Set("Access-Control-Expose-Headers", expose)
			}
			if r.Method == http.MethodOptions {
				if methods != "" {
					w.Header().Set("Access-Control-Allow-Methods", methods)
				}
				if headers != "" {
					w.Header().Set("Access-Control-Allow-Headers", headers)
				}
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
