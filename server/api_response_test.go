package server

import (
	"bytes"
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func compareGolden(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateGolden {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		_ = os.WriteFile(golden, got, 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestGolden_EmptyProfiles(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/profiles", nil)
	srv.handleProfiles(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ProfileCreateError(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"name":"Missing Key"}`))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", body)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ProfileCreateSuccess(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"profile_key":"golden-pk","name":"Golden Profile","sort_order":1}`))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", body)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ProfileMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/profiles", nil)
	srv.handleProfiles(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
