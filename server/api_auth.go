package server

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// handleAuthCheck checks whether the client is authenticated.
// If basic auth is enabled, it validates the Authorization header.
// If basic auth is not configured, it always returns success.
//
// @Summary Check auth status
// @Description Returns whether basic auth is enabled and if the client is authenticated
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Router /auth/check [get]
func (server *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If basic auth is not enabled, always succeed.
	if !server.options.EnableBasicAuth {
		writeAPISuccess(w, map[string]any{
			"authenticated": true,
			"auth_enabled":  false,
		})
		return
	}

	// Basic auth is enabled: validate the Authorization header.
	token := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(token) != 2 || strings.ToLower(token[0]) != "basic" {
		w.Header().Set("WWW-Authenticate", `Basic realm="ttyweb"`)
		writeAPIError(w, http.StatusUnauthorized, "basic auth required")
		return
	}

	payload, err := base64.StdEncoding.DecodeString(token[1])
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	if subtle.ConstantTimeCompare([]byte(server.options.Credential), payload) != 1 {
		w.Header().Set("WWW-Authenticate", `Basic realm="ttyweb"`)
		writeAPIError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeAPISuccess(w, map[string]any{
		"authenticated": true,
		"auth_enabled":  true,
	})
}
