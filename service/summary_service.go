package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ttyweb/ai"
	"ttyweb/db"
)

// summarySystemPrompt is the system prompt for the LLM summarizer.
const summarySystemPrompt = "You are a concise task summarizer. Given a task title and the commands run + AI conversation during the task, produce a one-line summary (max 200 chars). Output ONLY the summary text."

// SummaryService provides AI-powered task summary generation.
type SummaryService struct {
	db       *gorm.DB
	aiClient *ai.Client
}

// NewSummaryService creates a new SummaryService.
func NewSummaryService(gormDB *gorm.DB, aiClient *ai.Client) *SummaryService {
	return &SummaryService{db: gormDB, aiClient: aiClient}
}

// GenerateSummary creates an AI-generated summary for the given segment.
// It fetches the segment's commands and messages, builds a prompt, calls the LLM,
// truncates the result, and persists the summary to the database.
func (s *SummaryService) GenerateSummary(ctx context.Context, segmentID string) (*db.TaskSummary, error) {
	// Fetch the segment.
	var segment db.TaskSegment
	if err := s.db.Where("id = ?", segmentID).First(&segment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("segment not found: %s", segmentID)
		}
		return nil, fmt.Errorf("failed to fetch segment: %w", err)
	}

	// Fetch commands ordered by creation time.
	var commands []db.CommandRecord
	if err := s.db.Where("segment_id = ?", segmentID).Order("created_at ASC").Find(&commands).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch commands: %w", err)
	}

	// Fetch messages ordered by creation time.
	var messages []db.ChatMessage
	if err := s.db.Where("segment_id = ?", segmentID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Build the user prompt.
	userPrompt := buildPrompt(segment.TaskTitle, commands, messages)

	// Call the LLM.
	summaryText, err := s.aiClient.ChatCompletion(ctx, summarySystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Truncate to 500 chars.
	if len(summaryText) > 500 {
		summaryText = summaryText[:500]
	}

	// Persist the summary.
	now := time.Now()
	year, mon := yearMonth(now)

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	summary := &db.TaskSummary{
		ID:             id,
		Year:           year,
		Mon:            mon,
		SegmentID:      segmentID,
		SessionName:    segment.SessionName,
		WindowIndex:    segment.WindowIndex,
		WindowName:     segment.WindowName,
		CommandSummary: summaryText,
		SummaryStatus:  "completed",
		GeneratedAt:    &now,
	}

	if err := s.db.Create(summary).Error; err != nil {
		return nil, fmt.Errorf("failed to persist summary: %w", err)
	}

	return summary, nil
}

// GetSummary retrieves a summary by segment ID.
func (s *SummaryService) GetSummary(segmentID string) (*db.TaskSummary, error) {
	var summary db.TaskSummary
	if err := s.db.Where("segment_id = ?", segmentID).First(&summary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch summary: %w", err)
	}
	return &summary, nil
}

// ListSummaries returns all summaries, ordered by most recent first, limited to 100.
func (s *SummaryService) ListSummaries() ([]db.TaskSummary, error) {
	var summaries []db.TaskSummary
	if err := s.db.Order("generated_at DESC").Limit(100).Find(&summaries).Error; err != nil {
		return nil, fmt.Errorf("failed to list summaries: %w", err)
	}
	return summaries, nil
}

// buildPrompt constructs the user prompt for the LLM from task data.
func buildPrompt(taskTitle string, commands []db.CommandRecord, messages []db.ChatMessage) string {
	var b strings.Builder

	b.WriteString("Task: ")
	b.WriteString(taskTitle)
	b.WriteString("\n\n")

	if len(commands) > 0 {
		b.WriteString("Commands executed:\n")
		for _, cmd := range commands {
			b.WriteString("- ")
			b.WriteString(cmd.Command)
			if cmd.ExitCode != 0 {
				fmt.Fprintf(&b, " (exit %d)", cmd.ExitCode)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(messages) > 0 {
		b.WriteString("AI Conversation:\n")
		for _, msg := range messages {
			b.WriteString(msg.Role)
			b.WriteString(": ")
			b.WriteString(msg.Content)
			b.WriteString("\n")
		}
	}

	return b.String()
}
