package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	globalDB  *gorm.DB
	available bool
	initMu    sync.Mutex
)

// Init opens (or creates) the SQLite database at the given DSN.
// It enables WAL journal mode and creates the parent directory if needed.
// Safe to call multiple times; subsequent calls return nil without side effects.
func Init(dsn string) error {
	initMu.Lock()
	defer initMu.Unlock()

	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("dsn must not be empty")
	}

	if globalDB != nil {
		return nil
	}

	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		available = false
		return fmt.Errorf("create database directory %s: %w", dir, err)
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		available = false
		return fmt.Errorf("open sqlite %s: %w", dsn, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		available = false
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		available = false
		return fmt.Errorf("enable WAL mode: %w", err)
	}

	// Run any pending auto-migrations.
	models := getRegisteredModels()
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			available = false
			return fmt.Errorf("auto-migrate: %w", err)
		}
	}

	globalDB = db
	available = true
	return nil
}

// GetDB returns the global *gorm.DB instance.
// Returns ErrNotInitialized if Init has not been called successfully.
func GetDB() (*gorm.DB, error) {
	initMu.Lock()
	defer initMu.Unlock()

	if globalDB == nil {
		return nil, ErrNotInitialized
	}

	return globalDB, nil
}

// IsAvailable reports whether the database connection is active.
func IsAvailable() bool {
	initMu.Lock()
	defer initMu.Unlock()

	return available
}

// Close closes the underlying database connection.
// Safe to call even if the database was never initialized.
func Close() error {
	initMu.Lock()
	defer initMu.Unlock()

	if globalDB == nil {
		return nil
	}

	sqlDB, err := globalDB.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}

	if cerr := sqlDB.Close(); cerr != nil {
		// Keep globalDB alive so the caller can retry Close().
		return fmt.Errorf("close database: %w", cerr)
	}

	// Clear globals only after successful close.
	globalDB = nil
	available = false
	return nil
}
