// Package db provides GORM database initialization and migration.
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ttyweb/service"
)

// Open opens (or creates) the SQLite database at the default path and
// runs auto-migration for all registered models.
// Returns the *gorm.DB handle or an error.
func Open() (*gorm.DB, error) {
	dbPath := defaultDBPath()

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

// migrate registers all models for auto-migration.
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&service.Project{},
	)
}

// defaultDBPath returns the path to the SQLite database file.
// It lives in the user's config directory.
func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "ttyweb", "ttyweb.db")
}
