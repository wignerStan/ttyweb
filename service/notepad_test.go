package service

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with auto-migration.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&NotePad{}); err != nil {
		t.Fatalf("failed to run auto-migration: %v", err)
	}
	return db
}

func TestCreateNote(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	note, err := svc.CreateNote("Test Note", "Hello world")
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}
	if note.ID == "" {
		t.Error("expected non-empty ID")
	}
	if note.Name != "Test Note" {
		t.Errorf("expected name 'Test Note', got '%s'", note.Name)
	}
	if note.Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got '%s'", note.Content)
	}
	if note.OrderIndex != 0 {
		t.Errorf("expected default order_index 0, got %f", note.OrderIndex)
	}
	if note.ProjectID != nil {
		t.Error("expected nil ProjectID for global note")
	}
	if note.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateNote_WithProjectID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	note, err := svc.CreateNote("Project Note", "content", WithProjectID("proj-123"))
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}
	if note.ProjectID == nil || *note.ProjectID != "proj-123" {
		t.Errorf("expected ProjectID 'proj-123', got %v", note.ProjectID)
	}
}

func TestCreateNote_WithOrderIndex(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	note, err := svc.CreateNote("Ordered Note", "content", WithOrderIndex(5.5))
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}
	if note.OrderIndex != 5.5 {
		t.Errorf("expected order_index 5.5, got %f", note.OrderIndex)
	}
}

func TestCreateNote_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	_, err := svc.CreateNote("", "content")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if err.Error() != "name is required" {
		t.Errorf("expected 'name is required', got '%s'", err.Error())
	}
}

func TestListNotes_Global(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	_, _ = svc.CreateNote("Note A", "a", WithOrderIndex(1))
	_, _ = svc.CreateNote("Note B", "b", WithOrderIndex(0))
	_, _ = svc.CreateNote("Project Note", "p", WithProjectID("proj-1"))

	notes, err := svc.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 global notes, got %d", len(notes))
	}
	// Verify ordering by OrderIndex ASC.
	if notes[0].Name != "Note B" {
		t.Errorf("expected first note 'Note B', got '%s'", notes[0].Name)
	}
	if notes[1].Name != "Note A" {
		t.Errorf("expected second note 'Note A', got '%s'", notes[1].Name)
	}
}

func TestListNotes_ByProjectID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	_, _ = svc.CreateNote("Global", "g")
	_, _ = svc.CreateNote("Proj A", "pa", WithProjectID("proj-1"), WithOrderIndex(0))
	_, _ = svc.CreateNote("Proj B", "pb", WithProjectID("proj-1"), WithOrderIndex(1))
	_, _ = svc.CreateNote("Other Proj", "op", WithProjectID("proj-2"))

	projectID := "proj-1"
	notes, err := svc.ListNotes(&projectID)
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 project notes, got %d", len(notes))
	}
}

func TestListNotes_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	notes, err := svc.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestGetNote(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Get Me", "findable")

	found, err := svc.GetNote(created.ID)
	if err != nil {
		t.Fatalf("GetNote returned error: %v", err)
	}
	if found.Name != "Get Me" {
		t.Errorf("expected name 'Get Me', got '%s'", found.Name)
	}
}

func TestGetNote_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	_, err := svc.GetNote("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent note, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateNote(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Original", "original content")

	newName := "Updated"
	newContent := "new content"
	updated, err := svc.UpdateNote(created.ID, &newName, &newContent)
	if err != nil {
		t.Fatalf("UpdateNote returned error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got '%s'", updated.Name)
	}
	if updated.Content != "new content" {
		t.Errorf("expected content 'new content', got '%s'", updated.Content)
	}
}

func TestUpdateNote_PartialName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Original", "keep this")

	newName := "Renamed"
	updated, err := svc.UpdateNote(created.ID, &newName, nil)
	if err != nil {
		t.Fatalf("UpdateNote returned error: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Errorf("expected name 'Renamed', got '%s'", updated.Name)
	}
	if updated.Content != "keep this" {
		t.Errorf("expected content unchanged 'keep this', got '%s'", updated.Content)
	}
}

func TestUpdateNote_PartialContent(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Keep Name", "old")

	newContent := "new"
	updated, err := svc.UpdateNote(created.ID, nil, &newContent)
	if err != nil {
		t.Fatalf("UpdateNote returned error: %v", err)
	}
	if updated.Name != "Keep Name" {
		t.Errorf("expected name unchanged 'Keep Name', got '%s'", updated.Name)
	}
	if updated.Content != "new" {
		t.Errorf("expected content 'new', got '%s'", updated.Content)
	}
}

func TestUpdateNote_NoChanges(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("No Change", "same")

	updated, err := svc.UpdateNote(created.ID, nil, nil)
	if err != nil {
		t.Fatalf("UpdateNote returned error: %v", err)
	}
	if updated.Name != "No Change" {
		t.Errorf("expected name 'No Change', got '%s'", updated.Name)
	}
}

func TestUpdateNote_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Valid", "content")

	emptyName := ""
	_, err := svc.UpdateNote(created.ID, &emptyName, nil)
	if err == nil {
		t.Fatal("expected error for empty name update, got nil")
	}
}

func TestUpdateNote_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	newName := "Ghost"
	_, err := svc.UpdateNote("nonexistent-id", &newName, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent note, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteNote(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	created, _ := svc.CreateNote("Delete Me", "gone")

	err := svc.DeleteNote(created.ID)
	if err != nil {
		t.Fatalf("DeleteNote returned error: %v", err)
	}

	_, err = svc.GetNote(created.ID)
	if err == nil {
		t.Fatal("expected error when getting deleted note, got nil")
	}
}

func TestDeleteNote_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	err := svc.DeleteNote("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent note, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReorderNotes(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	a, _ := svc.CreateNote("A", "a", WithOrderIndex(0))
	b, _ := svc.CreateNote("B", "b", WithOrderIndex(1))
	c, _ := svc.CreateNote("C", "c", WithOrderIndex(2))

	err := svc.ReorderNotes([]NoteReorder{
		{ID: c.ID, OrderIndex: 0},
		{ID: a.ID, OrderIndex: 0.5},
		{ID: b.ID, OrderIndex: 1},
	})
	if err != nil {
		t.Fatalf("ReorderNotes returned error: %v", err)
	}

	notes, _ := svc.ListNotes(nil)
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	if notes[0].ID != c.ID {
		t.Errorf("expected first note to be C, got ID %s", notes[0].ID)
	}
	if notes[1].ID != a.ID {
		t.Errorf("expected second note to be A, got ID %s", notes[1].ID)
	}
	if notes[2].ID != b.ID {
		t.Errorf("expected third note to be B, got ID %s", notes[2].ID)
	}
}

func TestReorderNotes_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	err := svc.ReorderNotes(nil)
	if err != nil {
		t.Fatalf("ReorderNotes with nil returned error: %v", err)
	}
}

func TestReorderNotes_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNoteService(db)

	err := svc.ReorderNotes([]NoteReorder{
		{ID: "nonexistent", OrderIndex: 1},
	})
	if err == nil {
		t.Fatal("expected error for reordering nonexistent note, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
