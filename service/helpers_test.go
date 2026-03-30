package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with all service models migrated.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := db.AutoMigrate(&NotePad{}, &Project{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	return db
}

func TestGenerateID_Length(t *testing.T) {
	t.Parallel()
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID error: %v", err)
	}
	if len(id) != 16 {
		t.Errorf("expected 16-char ID, got %d chars: %q", len(id), id)
	}
}

func TestGenerateID_HexChars(t *testing.T) {
	t.Parallel()
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID error: %v", err)
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("non-hex char in ID: %c", c)
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	t.Parallel()
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID failed: %v", err)
		}
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}
