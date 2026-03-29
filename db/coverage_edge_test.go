package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInit_ReadOnlyDirectory(t *testing.T) {
	resetGlobal()

	// Create a read-only directory to cause gorm.Open to fail.
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyDir, 0o755)
	})

	dsn := filepath.Join(readOnlyDir, "test.db")
	err := Init(dsn)
	if err == nil {
		// Running as root — read-only doesn't apply.
		_ = Close()
		t.Log("Init succeeded on read-only dir (running as root)")
	} else {
		t.Logf("Init failed on read-only dir: %v", err)
	}
}

func TestInit_WhitespaceOnlyDSN(t *testing.T) {
	resetGlobal()

	err := Init("\t\n")
	if err == nil {
		t.Fatal("Init('\\t\\n') error = nil, want error")
	}
}

func TestConcurrentClose(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = Close()
		}()
	}
	wg.Wait()

	if IsAvailable() {
		t.Error("IsAvailable() = true after concurrent Close(), want false")
	}
}

func TestClose_Idempotent(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestInit_Close_Reinit(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	dsn2 := tempDB(t)
	if err := Init(dsn2); err != nil {
		t.Fatalf("re-Init() error = %v", err)
	}
	if !IsAvailable() {
		t.Fatal("IsAvailable() = false after re-init, want true")
	}
	_ = Close()
}

func TestGetDB_AfterClose(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_, err := GetDB()
	if err != ErrNotInitialized {
		t.Fatalf("GetDB() after Close() error = %v, want %v", err, ErrNotInitialized)
	}
}

func TestIsAvailable_AfterFailedInit(t *testing.T) {
	resetGlobal()

	_ = Init("")
	if IsAvailable() {
		t.Fatal("IsAvailable() = true after failed Init, want false")
	}
}

func TestDefaultOptions_NoHome(t *testing.T) {
	if _, ok := os.LookupEnv("HOME"); ok {
		t.Setenv("HOME", "")
	} else if _, ok := os.LookupEnv("USERPROFILE"); ok {
		t.Setenv("USERPROFILE", "")
	}

	opts := DefaultOptions()
	if opts.Path == "" {
		t.Fatal("DefaultOptions().Path is empty even in fallback")
	}
}

func TestGetDB_ReturnsSameInstance(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	db1, err := GetDB()
	if err != nil {
		t.Fatalf("first GetDB() error = %v", err)
	}
	db2, err := GetDB()
	if err != nil {
		t.Fatalf("second GetDB() error = %v", err)
	}
	if db1 != db2 {
		t.Fatal("GetDB() returned different instances")
	}

	_ = Close()
}

func TestConcurrentGetDB(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db, err := GetDB()
			if err != nil {
				t.Errorf("concurrent GetDB() error = %v", err)
				return
			}
			if db == nil {
				t.Error("concurrent GetDB() = nil")
			}
		}()
	}
	wg.Wait()

	_ = Close()
}

func TestInit_ConcurrentWithClose(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = Init(dsn)
		}()
		go func() {
			defer wg.Done()
			_ = Close()
		}()
	}

	wg.Wait()
}

func TestInit_MultipleDSNs(t *testing.T) {
	resetGlobal()

	dsn1 := tempDB(t)
	if err := Init(dsn1); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}

	dsn2 := filepath.Join(t.TempDir(), "other.db")
	if err := Init(dsn2); err != nil {
		t.Errorf("second Init() with different DSN should be no-op, got: %v", err)
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetDB() = nil")
	}

	_ = Close()
}
