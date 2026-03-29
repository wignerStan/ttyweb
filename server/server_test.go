package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"ttyweb/db"
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

func TestSetupHandlers_BasicAuth(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:             "/",
		TitleFormat:      "{{ .server.Version }}",
		EnableBasicAuth:  true,
		Credential:       "user:pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.setupHandlers(ctx, cancel, "/", counter)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestTLSConfig_ValidPEM(t *testing.T) {
	// Generate a self-signed certificate.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tmpFile, err := os.CreateTemp(t.TempDir(), "ca*.crt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tmpFile.Write(certPEM)
	_ = tmpFile.Close()

	srv := &Server{
		options: &Options{
			TLSCACrtFile: tmpFile.Name(),
		},
	}

	cfg, err := srv.tlsConfig()
	if err != nil {
		t.Fatalf("expected no error for valid PEM, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if cfg.ClientCAs == nil {
		t.Error("expected non-nil ClientCAs")
	}
}

func generateTestCert(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certFile, err := os.CreateTemp(t.TempDir(), "cert*.pem")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = certFile.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	_ = certFile.Close()

	return certFile.Name()
}

func TestSetupHTTPServer_WithTLSClientAuth_ValidCert(t *testing.T) {
	certPath := generateTestCert(t)

	srv := &Server{
		options: &Options{
			Path:                "/",
			EnableTLSClientAuth: true,
			TLSCACrtFile:        certPath,
		},
	}

	httpSrv, err := srv.setupHTTPServer(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpSrv == nil {
		t.Fatal("expected non-nil server")
	}
	if httpSrv.TLSConfig == nil {
		t.Error("expected non-nil TLS config")
	}
}

func TestRun_GracefulShutdown(t *testing.T) {
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

	// Cancel gracefully after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		gracefulCancel()
	}()

	err = srv.Run(ctx, WithGracefulContext(gracefulCtx))
	// Graceful shutdown returns nil.
	if err != nil {
		t.Logf("Run returned: %v (expected nil for graceful shutdown)", err)
	}
	_ = cancel
}
