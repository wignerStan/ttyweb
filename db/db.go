package db

import (
	"log"
	"sync/atomic"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	globalDB atomic.Pointer[gorm.DB]
)

// Init opens a SQLite database at the given DSN, configures WAL mode,
// sets connection limits, and stores the handle for later retrieval.
// Safe to call only once; subsequent calls return an error.
func Init(dsn string) error {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return err
	}

	// Connection management.
	sqlDB.SetMaxOpenConns(1) // SQLite only supports one writer at a time.
	sqlDB.SetMaxIdleConns(1)

	globalDB.Store(db)
	log.Printf("[db] SQLite database initialized at %s", dsn)
	return nil
}

// GetDB returns the global GORM database handle.
// Returns ErrNotInitialized if Init has not been called.
func GetDB() (*gorm.DB, error) {
	db := globalDB.Load()
	if db == nil {
		return nil, ErrNotInitialized
	}
	return db, nil
}

// IsAvailable reports whether the database has been initialized and is usable.
func IsAvailable() bool {
	db := globalDB.Load()
	if db == nil {
		return false
	}
	sqlDB, err := db.DB()
	if err != nil {
		return false
	}
	return sqlDB.Ping() == nil
}

// Close closes the underlying SQL connection.
// Returns ErrNotInitialized if the database was never opened.
func Close() error {
	db := globalDB.Load()
	if db == nil {
		return ErrNotInitialized
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if cerr := sqlDB.Close(); cerr != nil {
		return cerr
	}

	globalDB.Store(nil)
	log.Println("[db] Database connection closed")
	return nil
}
