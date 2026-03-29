package server

import (
	"net/http"
	"testing"
)

func TestSetupHTTPServer(t *testing.T) {
	srv := newTestServer()
	httpSrv, err := srv.setupHTTPServer(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpSrv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestSetupHTTPServer_WithTLSClientAuth(t *testing.T) {
	srv := &Server{
		options: &Options{
			Path:                "/",
			EnableTLSClientAuth: true,
			TLSCACrtFile:        "/nonexistent/ca.crt",
		},
	}

	_, err := srv.setupHTTPServer(nil)
	if err == nil {
		t.Fatal("expected error for nonexistent CA cert file")
	}
}

func TestSetupAPIHandlers(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()

	// Should not panic.
	srv.setupAPIHandlers(mux, "/")
}

func TestSetupAPIHandlers_CustomPath(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()

	// Should not panic with custom path prefix.
	srv.setupAPIHandlers(mux, "/custom/")
}
