package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAuthCheck_NoAuth(t *testing.T) {
	srv := &Server{options: &Options{EnableBasicAuth: false}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/check", nil)
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool            `json:"success"`
		Data    map[string]bool `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success")
	}
	if !resp.Data["authenticated"] {
		t.Fatal("expected authenticated=true")
	}
	if resp.Data["auth_enabled"] {
		t.Fatal("expected auth_enabled=false")
	}
}

func TestHandleAuthCheck_WithValidCredentials(t *testing.T) {
	cred := "user:pass"
	srv := &Server{options: &Options{EnableBasicAuth: true, Credential: cred}}
	auth := base64.StdEncoding.EncodeToString([]byte(cred))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/check", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool            `json:"success"`
		Data    map[string]bool `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Data["authenticated"] {
		t.Fatal("expected authenticated=true")
	}
	if !resp.Data["auth_enabled"] {
		t.Fatal("expected auth_enabled=true")
	}
}

func TestHandleAuthCheck_InvalidCredentials(t *testing.T) {
	srv := &Server{options: &Options{EnableBasicAuth: true, Credential: "user:pass"}}
	auth := base64.StdEncoding.EncodeToString([]byte("wrong:creds"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/check", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// Verify WWW-Authenticate header.
	if rec.Header().Get("WWW-Authenticate") != `Basic realm="ttyweb"` {
		t.Fatalf("expected WWW-Authenticate header, got %q", rec.Header().Get("WWW-Authenticate"))
	}
}

func TestHandleAuthCheck_MissingAuthorization(t *testing.T) {
	srv := &Server{options: &Options{EnableBasicAuth: true, Credential: "user:pass"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/check", nil)
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuthCheck_InvalidBase64(t *testing.T) {
	srv := &Server{options: &Options{EnableBasicAuth: true, Credential: "user:pass"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/check", nil)
	req.Header.Set("Authorization", "Basic not-valid-base64!!!")
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuthCheck_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/auth/check", nil)
	srv.handleAuthCheck(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
