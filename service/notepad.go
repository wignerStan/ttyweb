package service

import (
	"crypto/rand"

	"ttyweb/db"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func init() {
	db.RegisterModel(&NotePad{})
}

// ErrNotFound is returned when a note cannot be found by ID.
var ErrNotFound = errors.New("note not found")

// NotePad is a GORM model for persistent notepad entries.
type NotePad struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	ProjectID  *string   `gorm:"size:256;index" json:"project_id,omitempty"`
	Name       string    `gorm:"size:512;not null" json:"name"`
	Content    string    `gorm:"type:text" json:"content"`
	OrderIndex float64   `gorm:"not null;index;default:0" json:"order_index"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name.
func (NotePad) TableName() string {
	return "notepads"
}

// NoteReorder represents a single reorder operation.
type NoteReorder struct {
	ID         string  `json:"id"`
	OrderIndex float64 `json:"order_index"`
}

// NoteOption is a functional option for CreateNote.
type NoteOption func(*createNoteConfig)

type createNoteConfig struct {
	projectID  *string
	orderIndex float64
}

// WithProjectID sets the project ID for a new note.
func WithProjectID(id string) NoteOption {
	return func(c *createNoteConfig) {
		c.projectID = &id
	}
}

// WithOrderIndex sets the order index for a new note.
func WithOrderIndex(index float64) NoteOption {
	return func(c *createNoteConfig) {
		c.orderIndex = index
	}
}

// NoteService provides CRUD operations for NotePad records.
type NoteService struct {
	db *gorm.DB
}

// NewNoteService creates a new NoteService backed by the given database.
func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

// CreateNote creates a new note with the given name and content.
// Use functional options to set ProjectID and OrderIndex.
func (s *NoteService) CreateNote(name, content string, opts ...NoteOption) (*NotePad, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	cfg := &createNoteConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	note := NotePad{
		ID:         id,
		ProjectID:  cfg.projectID,
		Name:       name,
		Content:    content,
		OrderIndex: cfg.orderIndex,
	}

	result := s.db.Create(&note)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create note: %w", result.Error)
	}

	return &note, nil
}

// ListNotes returns all notes, optionally filtered by projectID.
// Results are ordered by OrderIndex ascending.
func (s *NoteService) ListNotes(projectID *string) ([]NotePad, error) {
	var notes []NotePad

	query := s.db.Order("order_index ASC")
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	} else {
		query = query.Where("project_id IS NULL")
	}

	result := query.Find(&notes)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list notes: %w", result.Error)
	}

	return notes, nil
}

// GetNote retrieves a single note by ID.
func (s *NoteService) GetNote(id string) (*NotePad, error) {
	var note NotePad
	result := s.db.First(&note, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to get note: %w", result.Error)
	}
	return &note, nil
}

// UpdateNote partially updates a note. Only non-nil fields are updated.
func (s *NoteService) UpdateNote(id string, name *string, content *string) (*NotePad, error) {
	note, err := s.GetNote(id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if name != nil {
		if *name == "" {
			return nil, errors.New("name must not be empty")
		}
		updates["name"] = *name
	}
	if content != nil {
		updates["content"] = *content
	}

	if len(updates) == 0 {
		return note, nil
	}

	result := s.db.Model(note).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update note: %w", result.Error)
	}

	// Re-fetch to get updated timestamps.
	return s.GetNote(id)
}

// DeleteNote removes a note by ID.
func (s *NoteService) DeleteNote(id string) error {
	result := s.db.Delete(&NotePad{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete note: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return nil
}

// ReorderNotes updates the OrderIndex of multiple notes in a single transaction.
func (s *NoteService) ReorderNotes(reorders []NoteReorder) error {
	if len(reorders) == 0 {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, r := range reorders {
			result := tx.Model(&NotePad{}).
				Where("id = ?", r.ID).
				Update("order_index", r.OrderIndex)
			if result.Error != nil {
				return fmt.Errorf("failed to reorder note %s: %w", r.ID, result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("%w: %s", ErrNotFound, r.ID)
			}
		}
		return nil
	})
}

// generateID creates a random 16-byte hex ID using crypto/rand.
func generateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
