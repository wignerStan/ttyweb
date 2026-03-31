package service

import (
	"testing"

	"ttyweb/db"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// testGormDB creates an in-memory SQLite DB with all service models migrated.
func testGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	err = gormDB.AutoMigrate(
		&db.ProfileModel{},
		&db.GroupModel{},
		&db.SnippetModel{},
		&NotePad{},
		&Project{},
	)
	if err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return gormDB
}

// TestDeleteProfile_NotFound tests deleting a nonexistent profile.
func TestDeleteProfile_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	err := svc.DeleteProfile(99999)
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

// TestDeleteGroup_NotFound tests deleting a nonexistent group.
func TestDeleteGroup_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	err := svc.DeleteGroup(99999)
	if err == nil {
		t.Error("expected error for nonexistent group")
	}
}

// TestDeleteSnippet_NotFound tests deleting a nonexistent snippet.
func TestDeleteSnippet_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	err := svc.DeleteSnippet(99999)
	if err == nil {
		t.Error("expected error for nonexistent snippet")
	}
}

// TestReindexSnippets_NoSnippets tests reindexing with no snippets.
func TestReindexSnippets_NoSnippets(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	err := svc.ReindexSnippets()
	if err != nil {
		t.Fatalf("ReindexSnippets: %v", err)
	}
}

// TestReindexSnippets_WithGaps tests reindexing snippets with index gaps.
func TestReindexSnippets_WithGaps(t *testing.T) {
	svc := NewPersistService(testGormDB(t))

	_, err := svc.CreateSnippet(db.SnippetModel{Index: 0, Name: "first", Command: "echo 1"})
	if err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}
	_, err = svc.CreateSnippet(db.SnippetModel{Index: 5, Name: "second", Command: "echo 2"})
	if err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}
	_, err = svc.CreateSnippet(db.SnippetModel{Index: 10, Name: "third", Command: "echo 3"})
	if err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}

	err = svc.ReindexSnippets()
	if err != nil {
		t.Fatalf("ReindexSnippets: %v", err)
	}

	snippets, err := svc.ListSnippets()
	if err != nil {
		t.Fatalf("ListSnippets: %v", err)
	}
	expectedIndices := []int{0, 1, 2}
	for i, sn := range snippets {
		if sn.Index != expectedIndices[i] {
			t.Errorf("snippet %d: expected index %d, got %d", i, expectedIndices[i], sn.Index)
		}
	}
}

// TestGetProfile_NotFound tests getting a nonexistent profile.
func TestGetProfile_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.GetProfile(99999)
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

// TestGetGroup_NotFound tests getting a nonexistent group.
func TestGetGroup_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.GetGroup(99999)
	if err == nil {
		t.Error("expected error for nonexistent group")
	}
}

// TestGetSnippet_NotFound tests getting a nonexistent snippet.
func TestGetSnippet_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.GetSnippet(99999)
	if err == nil {
		t.Error("expected error for nonexistent snippet")
	}
}

// TestUpdateProfile_NotFound tests updating a nonexistent profile.
func TestUpdateProfile_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.UpdateProfile(99999, db.ProfileModel{ProfileKey: "test", Name: "Test"})
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

// TestUpdateGroup_NotFound tests updating a nonexistent group.
func TestUpdateGroup_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.UpdateGroup(99999, db.GroupModel{GroupName: "test"})
	if err == nil {
		t.Error("expected error for nonexistent group")
	}
}

// TestUpdateSnippet_NotFound tests updating a nonexistent snippet.
func TestUpdateSnippet_NotFound(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	_, err := svc.UpdateSnippet(99999, db.SnippetModel{Index: 0, Name: "test", Command: "echo"})
	if err == nil {
		t.Error("expected error for nonexistent snippet")
	}
}

// TestListGroupsByProfile_Empty tests listing groups with no matches.
func TestListGroupsByProfile_Empty(t *testing.T) {
	svc := NewPersistService(testGormDB(t))
	groups, err := svc.ListGroupsByProfile("nonexistent-profile")
	if err != nil {
		t.Fatalf("ListGroupsByProfile: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}
