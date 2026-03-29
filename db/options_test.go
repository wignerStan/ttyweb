package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultOptions_ContainsExpectedPath(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	if opts.Path == "" {
		t.Fatal("expected non-empty Path")
	}
	if !strings.HasSuffix(opts.Path, "ttyweb.db") {
		t.Fatalf("expected path ending in ttyweb.db, got %q", opts.Path)
	}
	if !strings.Contains(opts.Path, ".local") {
		t.Fatalf("expected path containing .local, got %q", opts.Path)
	}
	if !strings.Contains(opts.Path, "share") {
		t.Fatalf("expected path containing share, got %q", opts.Path)
	}
	if !strings.Contains(opts.Path, "ttyweb") {
		t.Fatalf("expected path containing ttyweb, got %q", opts.Path)
	}
}

func TestDefaultOptions_AbsolutePath(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	if !filepath.IsAbs(opts.Path) {
		t.Fatalf("expected absolute path, got %q", opts.Path)
	}
}
