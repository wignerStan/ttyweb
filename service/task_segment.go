// Package service provides business logic for task segment tracking.
package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ttyweb/db"
)

// init registers task segment models for auto-migration.
func init() {
	db.RegisterModel(&db.TaskSegment{})
	db.RegisterModel(&db.ChatMessage{})
	db.RegisterModel(&db.CommandRecord{})
	db.RegisterModel(&db.TaskSummary{})
}

// Valid task status values.
const (
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
)

// Valid chat message roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// validStatuses holds the set of allowed task status values.
var validStatuses = map[string]bool{
	StatusInProgress: true,
	StatusCompleted:  true,
}

// validRoles holds the set of allowed chat message roles.
var validRoles = map[string]bool{
	RoleUser:      true,
	RoleAssistant: true,
}

// TaskSegmentService provides methods for managing task segments and related data.
type TaskSegmentService struct {
	db *gorm.DB
}

// SegmentDetail aggregates a task segment with its messages, commands, and summary.
type SegmentDetail struct {
	Segment  *db.TaskSegment    `json:"segment"`
	Messages []db.ChatMessage   `json:"messages"`
	Commands []db.CommandRecord `json:"commands"`
	Summary  *db.TaskSummary    `json:"summary"`
}

// NewTaskSegmentService creates a new TaskSegmentService backed by the given database.
func NewTaskSegmentService(database *gorm.DB) *TaskSegmentService {
	return &TaskSegmentService{db: database}
}

// CreateSegment creates a new task segment with the given parameters.
func (s *TaskSegmentService) CreateSegment(sessionName, windowName string, paneIndex int, title string) (*db.TaskSegment, error) {
	if title == "" {
		return nil, fmt.Errorf("task_title is required")
	}

	now := time.Now()
	year, mon := yearMonth(now)

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	segment := &db.TaskSegment{
		ID:          id,
		Year:        year,
		Mon:         mon,
		SessionName: sessionName,
		WindowName:  windowName,
		PaneIndex:   paneIndex,
		TaskTitle:   title,
		TaskStatus:  StatusInProgress,
		StartedAt:   &now,
	}

	if err := s.db.Create(segment).Error; err != nil {
		return nil, fmt.Errorf("failed to create segment: %w", err)
	}

	return segment, nil
}

// ListSegments returns all segments for a given session name.
// If sessionName is empty, returns all segments.
func (s *TaskSegmentService) ListSegments(sessionName string) ([]db.TaskSegment, error) {
	var segments []db.TaskSegment

	query := s.db.Order("created_at DESC")
	if sessionName != "" {
		query = query.Where("session_name = ?", sessionName)
	}

	if err := query.Find(&segments).Error; err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}

	return segments, nil
}

// UpdateSegment updates a segment's title and/or status.
// At least one of title or status must be provided.
func (s *TaskSegmentService) UpdateSegment(id string, title *string, status string) (*db.TaskSegment, error) {
	if title == nil && status == "" {
		return nil, fmt.Errorf("at least one of task_title or task_status is required")
	}

	updates := map[string]interface{}{}

	if title != nil {
		updates["task_title"] = *title
	}

	if status != "" {
		if !validStatuses[status] {
			return nil, fmt.Errorf("invalid task_status: must be in_progress or completed, got %q", status)
		}
		updates["task_status"] = status
		if status == StatusCompleted {
			now := time.Now()
			updates["completed_at"] = &now
		}
	}

	result := s.db.Model(&db.TaskSegment{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update segment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("segment not found: %s", id)
	}

	var segment db.TaskSegment
	if err := s.db.Where("id = ?", id).First(&segment).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch updated segment: %w", err)
	}

	return &segment, nil
}

// GetSegmentDetail returns a segment with all related messages, commands, and summary.
func (s *TaskSegmentService) GetSegmentDetail(id string) (*SegmentDetail, error) {
	var segment db.TaskSegment
	if err := s.db.Where("id = ?", id).First(&segment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("segment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to fetch segment: %w", err)
	}

	messages, err := s.ListMessages(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	commands, err := s.ListCommands(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commands: %w", err)
	}

	var summaryPtr *db.TaskSummary
	var summary db.TaskSummary
	if err := s.db.Where("segment_id = ?", id).First(&summary).Error; err == nil {
		summaryPtr = &summary
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to fetch summary: %w", err)
	}

	return &SegmentDetail{
		Segment:  &segment,
		Messages: messages,
		Commands: commands,
		Summary:  summaryPtr,
	}, nil
}

// ListMessages returns all chat messages for a segment.
func (s *TaskSegmentService) ListMessages(segmentID string) ([]db.ChatMessage, error) {
	var messages []db.ChatMessage
	if err := s.db.Where("segment_id = ?", segmentID).Order("msg_time ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	return messages, nil
}

// ListCommands returns all command records for a segment.
func (s *TaskSegmentService) ListCommands(segmentID string) ([]db.CommandRecord, error) {
	var commands []db.CommandRecord
	if err := s.db.Where("segment_id = ?", segmentID).Order("cmd_time ASC").Find(&commands).Error; err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}
	return commands, nil
}

// AddMessage adds a chat message to a segment.
func (s *TaskSegmentService) AddMessage(segmentID, role, content string) (*db.ChatMessage, error) {
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role: must be user or assistant, got %q", role)
	}

	now := time.Now()
	year, mon := yearMonth(now)

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	message := &db.ChatMessage{
		ID:        id,
		Year:      year,
		Mon:       mon,
		SegmentID: segmentID,
		Role:      role,
		Content:   content,
		MsgTime:   &now,
	}

	if err := s.db.Create(message).Error; err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}

	return message, nil
}

// AddCommandRecord adds a command execution record to a segment.
func (s *TaskSegmentService) AddCommandRecord(segmentID, command string, exitCode int) (*db.CommandRecord, error) {
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	now := time.Now()
	year, mon := yearMonth(now)

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	record := &db.CommandRecord{
		ID:        id,
		Year:      year,
		Mon:       mon,
		SegmentID: segmentID,
		Command:   command,
		CmdTime:   &now,
		ExitCode:  exitCode,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("failed to add command record: %w", err)
	}

	return record, nil
}

// yearMonth returns the current year and month (1-12) as a pair.
func yearMonth(t time.Time) (int, int) {
	return t.Year(), int(t.Month())
}


