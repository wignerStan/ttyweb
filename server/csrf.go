package server

import "net/http"

// csrfMiddleware rejects state-changing requests (POST, PUT, PATCH, DELETE)
// that lack the X-Requested-With: XMLHttpRequest header.
// Safe methods (GET, HEAD, OPTIONS) always pass through.
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			http.Error(w, `{"success":false,"error":"missing required header"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
