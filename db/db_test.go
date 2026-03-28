package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDB_NotInitialized(t *testing.T) {
	// Ensure clean state — if another test initialized the DB, we can't fully
	// reset, so skip if already initialized.
	if globalDB.Load() != nil {
		t.Skip("DB already initialized by another test")
	}

	_, err := GetDB()
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got %v", err)
	}
}

func TestInitClose_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")

	err := Init(dsn)
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if !IsAvailable() {
		t.Error("IsAvailable() should be true after Init()")
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error: %v", err)
	}
	if got == nil {
		t.Error("GetDB() should return a non-nil *gorm.DB")
	}

	// Verify the DB file was created.
	if _, err := os.Stat(dsn); os.IsNotExist(err) {
		t.Error("expected database file to exist after Init()")
	}

	err = Close()
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if IsAvailable() {
		t.Error("IsAvailable() should be false after Close()")
	}
}

func TestClose_NotInitialized(t *testing.T) {
	if globalDB.Load() != nil {
		t.Skip("DB already initialized by another test")
	}

	err := Close()
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got %v", err)
	}
}

func TestDefaultOptions_XDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/testdata")
	opts := DefaultOptions()
	expected := "/tmp/testdata/ttyweb/ttyweb.db"
	if opts.Path != expected {
		t.Errorf("DefaultOptions().Path = %q, want %q", opts.Path, expected)
	}
}

func TestDefaultOptions_NoXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	opts := DefaultOptions()
	expected := filepath.Join(home, ".local", "share", "ttyweb", "ttyweb.db")
	if opts.Path != expected {
		t.Errorf("DefaultOptions().Path = %q, want %q", opts.Path, expected)
	}
}

func TestRegisterModel(t *testing.T) {
	// Reset registered models for test isolation.
	registeredModels = nil

	type SampleModel struct{}
	RegisterModel(SampleModel{})
	RegisterModel(struct{ Name string }{})

	if len(registeredModels) != 2 {
		t.Errorf("expected 2 registered models, got %d", len(registeredModels))
	}
}
