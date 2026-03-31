package db

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSetTestDB_SetsGlobal(t *testing.T) {
	// Reset any existing state.
	initMu.Lock()
	globalDB = nil
	available = false
	initMu.Unlock()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	SetTestDB(gormDB)

	if !IsAvailable() {
		t.Error("expected IsAvailable() = true after SetTestDB")
	}
	got, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB after SetTestDB: %v", err)
	}
	if got != gormDB {
		t.Error("GetDB should return the same instance set by SetTestDB")
	}
}

func TestSetTestDB_Nil(t *testing.T) {
	initMu.Lock()
	globalDB = nil
	available = false
	initMu.Unlock()

	SetTestDB(nil)

	if IsAvailable() {
		t.Error("expected IsAvailable() = false after SetTestDB(nil)")
	}
	_, err := GetDB()
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got %v", err)
	}
}
