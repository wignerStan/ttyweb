package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/swaggo/swag"
)

func TestSwaggerDocsHandler_IndexHTML(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/docs/index.html", nil)
	rec := httptest.NewRecorder()

	handleSwaggerDocs(rec, req)

	resp := rec.Result()
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %s", ct)
	}
}

func TestSwaggerJSONHandler(t *testing.T) {
	doc, err := swag.ReadDoc()
	if err != nil {
		t.Fatalf("failed to read swagger doc: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(doc), &body); err != nil {
		t.Fatalf("failed to decode swagger JSON: %v", err)
	}

	if _, ok := body["swagger"]; !ok {
		if _, ok := body["openapi"]; !ok {
			t.Error("response JSON must contain 'swagger' or 'openapi' field")
		}
	}

	if body["info"] == nil {
		t.Error("response JSON must contain 'info' field")
	}
}
