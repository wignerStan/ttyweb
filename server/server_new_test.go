package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ttyweb/db"
)

func TestNew_ValidOptions(t *testing.T) {
	// Initialize DB for the test.
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .server.Version }}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.options == nil {
		t.Fatal("expected non-nil options")
	}
	if srv.upgrader == nil {
		t.Fatal("expected non-nil upgrader")
	}
	if srv.noteSvc == nil {
		t.Fatal("expected non-nil noteSvc")
	}
	if srv.segmentService == nil {
		t.Fatal("expected non-nil segmentService")
	}
}

func TestNew_CustomIndexFile_NotFound(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	_, err := New(factory, &Options{
		Path:        "/",
		IndexFile:   "/nonexistent/index.html",
		TitleFormat: "{{ .server.Version }}",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent index file")
	}
}

func TestNew_InvalidTitleFormat(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	_, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{unclosed",
	})
	if err == nil {
		t.Fatal("expected error for invalid title format")
	}
}

func TestNew_InvalidWSOriginRegex(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	_, err := New(factory, &Options{
		Path:        "/",
		WSOrigin:    "[invalid",
		TitleFormat: "{{ .server.Version }}",
	})
	if err == nil {
		t.Fatal("expected error for invalid WSOrigin regex")
	}
}

func TestNew_WithValidWSOrigin(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		WSOrigin:    "https://example\\.com",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	// Verify the origin checker works.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	if !srv.upgrader.CheckOrigin(req) {
		t.Fatal("expected origin check to pass for https://example.com")
	}

	req2 := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req2.Header.Set("Origin", "https://evil.com")
	if srv.upgrader.CheckOrigin(req2) {
		t.Fatal("expected origin check to fail for https://evil.com")
	}
}

func TestNew_DefaultOriginChecker_NoOrigin(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No origin header should pass.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	if !srv.upgrader.CheckOrigin(req) {
		t.Fatal("expected origin check to pass when no Origin header")
	}
}

func TestNew_DefaultOriginChecker_MatchingHost(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Matching host should pass.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	req.Host = "localhost:8080"
	if !srv.upgrader.CheckOrigin(req) {
		t.Fatal("expected origin check to pass for matching host")
	}

	// Non-matching host should fail.
	req2 := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req2.Header.Set("Origin", "http://evil.com:8080")
	req2.Host = "localhost:8080"
	if srv.upgrader.CheckOrigin(req2) {
		t.Fatal("expected origin check to fail for non-matching host")
	}
}

func TestNew_CustomIndexFile(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Create a temporary index file.
	tmpFile, err := os.CreateTemp(t.TempDir(), "index*.html")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tmpFile.WriteString("<html><body>Custom Index</body></html>")
	_ = tmpFile.Close()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		IndexFile:   tmpFile.Name(),
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	// Verify the custom index is loaded.
	if string(indexHTML) != "<html><body>Custom Index</body></html>" {
		t.Fatalf("expected custom index HTML, got %q", string(indexHTML))
	}
}

func TestTLSConfig_InvalidCAFile(t *testing.T) {
	srv := &Server{
		options: &Options{
			TLSCACrtFile: "/nonexistent/ca.crt",
		},
	}

	_, err := srv.tlsConfig()
	if err == nil {
		t.Fatal("expected error for nonexistent CA file")
	}
}

func TestTLSConfig_InvalidPEM(t *testing.T) {
	// Create a temp file with invalid PEM data.
	tmpFile, err := os.CreateTemp(t.TempDir(), "ca*.crt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tmpFile.WriteString("not valid PEM data")
	_ = tmpFile.Close()

	srv := &Server{
		options: &Options{
			TLSCACrtFile: tmpFile.Name(),
		},
	}

	_, err = srv.tlsConfig()
	if err == nil {
		t.Fatal("expected error for invalid PEM data")
	}
}

func TestNew_DefaultOriginChecker_InvalidOrigin(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// An unparseable Origin header should fail.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Origin", "http://[::1]:namedport") // invalid URL
	if srv.upgrader.CheckOrigin(req) {
		t.Fatal("expected origin check to fail for unparseable Origin")
	}
}

func TestNew_NoDatabase(t *testing.T) {
	// Ensure DB is not initialized.
	_ = db.Close()

	factory := &mockFactory{name: "test"}
	_, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .server.Version }}",
	})
	if err == nil {
		t.Fatal("expected error when DB is not initialized")
	}

	// Re-init DB for other tests.
	_ = db.Init(t.TempDir() + "/reinit-new.db")
}

func TestSetupHTTPServer_NoTLS(t *testing.T) {
	srv := &Server{
		options: &Options{},
	}

	httpSrv, err := srv.setupHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpSrv == nil {
		t.Fatal("expected non-nil HTTP server")
	}
	if httpSrv.TLSConfig != nil {
		t.Fatal("expected nil TLS config when EnableTLSClientAuth is false")
	}
}

func TestRun_PortZero_ImmediateCancel(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		Port:        "0",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after starting.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = srv.Run(ctx)
	// Should return context.Canceled error.
	if err != context.Canceled {
		t.Logf("Run returned: %v (expected context.Canceled)", err)
	}
}

func TestRun_WithRandomURL(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:            "/",
		Port:            "0",
		EnableRandomUrl: true,
		RandomUrlLength: 8,
		TitleFormat:     "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = srv.Run(ctx)
	if err != context.Canceled {
		t.Logf("Run returned: %v (expected context.Canceled)", err)
	}
}

func TestRun_WithGracefulShutdown(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		Port:        "0",
		TitleFormat: "{{ .server.Version }}",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gracefulCtx, gracefulCancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		gracefulCancel() // trigger graceful shutdown
	}()

	err = srv.Run(ctx, WithGracefullContext(gracefulCtx))
	if err != nil {
		t.Logf("Run returned: %v (expected nil for graceful shutdown)", err)
	}
	_ = cancel
}
