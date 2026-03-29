package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rec, status: 200}

	lrw.WriteHeader(http.StatusNotFound)

	if lrw.status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", lrw.status)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected recorder code 404, got %d", rec.Code)
	}
}

func TestLogResponseWriter_DefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := &logResponseWriter{ResponseWriter: rec, status: 200}

	// If WriteHeader is never called, status should be 200.
	if lrw.status != 200 {
		t.Fatalf("expected default status 200, got %d", lrw.status)
	}
}
