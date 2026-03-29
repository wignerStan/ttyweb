package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetGlobal() {
	initMu.Lock()
	defer initMu.Unlock()

	if globalDB != nil {
		sqlDB, _ := globalDB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		globalDB = nil
		available = false
	}

	migrateMu.Lock()
	registered = nil
	migrateMu.Unlock()
}

func tempDB(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	return filepath.Join(dir, "test.db")
}

func TestInitCreatesDatabase(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	err := Init(dsn)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if !IsAvailable() {
		t.Fatal("IsAvailable() = false, want true")
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetDB() = nil, want non-nil")
	}

	if err := Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if IsAvailable() {
		t.Fatal("IsAvailable() = true after Close(), want false")
	}
}

func TestInitCreatesDirectory(t *testing.T) {
	resetGlobal()

	dir := filepath.Join(t.TempDir(), "sub", "dir")
	dsn := filepath.Join(dir, "test.db")

	err := Init(dsn)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("database directory was not created")
	}

	_ = Close()
}

func TestInitIdempotent(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)

	if err := Init(dsn); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	if err := Init(dsn); err != nil {
		t.Fatalf("second Init() error = %v", err)
	}

	_ = Close()
}

func TestGetDBBeforeInit(t *testing.T) {
	resetGlobal()

	_, err := GetDB()
	if err != ErrNotInitialized {
		t.Fatalf("GetDB() error = %v, want %v", err, ErrNotInitialized)
	}
}

func TestCloseBeforeInit(t *testing.T) {
	resetGlobal()

	if err := Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWALMode(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}

	sqlDB, err := got.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}

	var mode string
	row := sqlDB.QueryRow("PRAGMA journal_mode")
	if err := row.Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode scan error = %v", err)
	}

	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}

	_ = Close()
}

func TestRegisterModel(t *testing.T) {
	resetGlobal()

	type TestModel struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:255"`
	}

	RegisterModel(&TestModel{})

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}

	// Verify table was created.
	if !got.Migrator().HasTable("test_models") {
		t.Fatal("table test_models not created")
	}

	_ = Close()
}

func TestAutoMigrate(t *testing.T) {
	resetGlobal()

	type ExtraModel struct {
		ID    uint   `gorm:"primaryKey"`
		Value string `gorm:"size:255"`
	}

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	RegisterModel(&ExtraModel{})
	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}

	if !got.Migrator().HasTable("extra_models") {
		t.Fatal("table extra_models not created")
	}

	_ = Close()
}

func TestAutoMigrateBeforeInit(t *testing.T) {
	resetGlobal()

	type StubModel struct {
		ID uint `gorm:"primaryKey"`
	}

	RegisterModel(&StubModel{})
	err := AutoMigrate()
	if err != ErrNotInitialized {
		t.Fatalf("AutoMigrate() error = %v, want %v", err, ErrNotInitialized)
	}
}

func TestAutoMigrateEmpty(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() with no models error = %v", err)
	}

	_ = Close()
}

func TestConcurrentInit(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := Init(dsn); err != nil {
				t.Errorf("concurrent Init() error = %v", err)
			}
		}()
	}

	wg.Wait()

	if !IsAvailable() {
		t.Fatal("IsAvailable() = false after concurrent Init(), want true")
	}

	_ = Close()
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Path == "" {
		t.Fatal("DefaultOptions().Path is empty")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "share", "ttyweb", "ttyweb.db")
	if opts.Path != expected {
		t.Fatalf("DefaultOptions().Path = %q, want %q", opts.Path, expected)
	}
}

func TestAutoMigrate_NotInitialized(t *testing.T) {
	resetGlobal()

	type PlaceholderModel struct {
		ID uint `gorm:"primaryKey"`
	}

	RegisterModel(&PlaceholderModel{})
	err := AutoMigrate()
	if err != ErrNotInitialized {
		t.Fatalf("AutoMigrate() error = %v, want %v", err, ErrNotInitialized)
	}
}

func TestAutoMigrate_Success(t *testing.T) {
	resetGlobal()

	type VerifyModel struct {
		ID    uint   `gorm:"primaryKey"`
		Label string `gorm:"size:255"`
	}

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	RegisterModel(&VerifyModel{})
	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB() error = %v", err)
	}

	// Verify table exists by querying sqlite_master directly.
	var name string
	row := got.Raw(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='verify_models'",
	).Row()
	if err := row.Scan(&name); err != nil {
		t.Fatalf("sqlite_master query error = %v", err)
	}
	if name != "verify_models" {
		t.Fatalf("table name = %q, want verify_models", name)
	}

	_ = Close()
}

func TestAutoMigrate_NoModels(t *testing.T) {
	resetGlobal()

	dsn := tempDB(t)
	if err := Init(dsn); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// No models registered — AutoMigrate should be a no-op.
	err := AutoMigrate()
	if err != nil {
		t.Fatalf("AutoMigrate() with no models error = %v, want nil", err)
	}

	_ = Close()
}

func TestInit_EmptyDSN(t *testing.T) {
	resetGlobal()

	err := Init("")
	if err == nil {
		t.Fatal("Init('') error = nil, want error")
	}

	err = Init("   ")
	if err == nil {
		t.Fatal("Init('   ') error = nil, want error")
	}
}

func TestInit_InvalidDSN(t *testing.T) {
	resetGlobal()

	// Path to a non-existent directory that MkdirAll cannot create.
	err := Init("/nonexistent/path/db.sqlite")
	if err == nil {
		t.Fatal("Init() with invalid path error = nil, want error")
	}
}
