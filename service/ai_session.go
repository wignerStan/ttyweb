package service

import (
	"sort"
	"sync"
	"time"

	"ttyweb/ai"
)

// AISessionRecord is the stored representation of an AI session.
type AISessionRecord struct {
	ID                    int
	SessionID             string
	Type                  string
	ProjectPath           string
	FilePath              string
	Model                 string
	Title                 string
	SessionStartedAt      time.Time
	LastMessageAt         *time.Time
	MessageCount          int
	AssistantMessageCount int
	FileModTime           time.Time
	FileSize              int64
}

// toRecord converts an ai.AISession to an AISessionRecord with the given ID.
// If session.SessionStartedAt is zero, it falls back to FileModTime.
func toRecord(id int, session ai.AISession) AISessionRecord {
	startedAt := session.SessionStartedAt
	if startedAt.IsZero() {
		startedAt = session.FileModTime
	}
	return AISessionRecord{
		ID:                    id,
		SessionID:             session.SessionID,
		Type:                  session.Type,
		ProjectPath:           session.ProjectPath,
		FilePath:              session.FilePath,
		Model:                 session.Model,
		Title:                 session.Title,
		SessionStartedAt:      startedAt,
		LastMessageAt:         session.LastMessageAt,
		MessageCount:          session.MessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		FileModTime:           session.FileModTime,
		FileSize:              session.FileSize,
	}
}

// AISessionStore provides in-memory CRUD for AI sessions.
type AISessionStore struct {
	mu      sync.RWMutex
	records []AISessionRecord
	nextID  int
}

// NewAISessionStore creates a new AISessionStore.
func NewAISessionStore() *AISessionStore {
	return &AISessionStore{
		records: make([]AISessionRecord, 0),
		nextID:  1,
	}
}

// Upsert adds or updates a session record.
func (s *AISessionStore) Upsert(session ai.AISession) AISessionRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.records {
		if existing.SessionID == session.SessionID && existing.Type == session.Type {
			updated := toRecord(existing.ID, session)
			s.records[i] = updated
			return updated
		}
	}

	record := toRecord(s.nextID, session)
	s.nextID++
	s.records = append(s.records, record)
	return record
}

// List returns all stored sessions, sorted by last message time (newest first).
func (s *AISessionStore) List(projectPath string) []AISessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]AISessionRecord, 0, len(s.records))
	for _, r := range s.records {
		if projectPath != "" && r.ProjectPath != projectPath {
			continue
		}
		filtered = append(filtered, r)
	}

	sort.Slice(filtered, func(i, j int) bool {
		ti := filtered[i].LastMessageAt
		tj := filtered[j].LastMessageAt
		if ti != nil && tj != nil {
			return ti.After(*tj)
		}
		if ti != nil {
			return true
		}
		if tj != nil {
			return false
		}
		return filtered[i].SessionStartedAt.After(filtered[j].SessionStartedAt)
	})

	return filtered
}

// GetByID returns a session by its database ID.
func (s *AISessionStore) GetByID(id int) (AISessionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.records {
		if r.ID == id {
			return r, true
		}
	}
	return AISessionRecord{}, false
}

// DeleteMissingFiles removes sessions whose file path is not in the provided set.
func (s *AISessionStore) DeleteMissingFiles(existingPaths map[string]bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	kept := make([]AISessionRecord, 0, len(s.records))
	for _, r := range s.records {
		if existingPaths[r.FilePath] {
			kept = append(kept, r)
		} else {
			removed++
		}
	}
	s.records = kept
	return removed
}

// AISessionService provides high-level operations for AI session management.
type AISessionService struct {
	store *AISessionStore
}

// NewAISessionService creates a new AISessionService.
func NewAISessionService() *AISessionService {
	return &AISessionService{
		store: NewAISessionStore(),
	}
}

// Store returns the underlying session store for direct upsert operations.
func (svc *AISessionService) Store() *AISessionStore {
	return svc.store
}

// GetSessions returns all sessions, optionally filtered by project path.
func (svc *AISessionService) GetSessions(projectPath string) []AISessionRecord {
	return svc.store.List(projectPath)
}

// GetSession returns a single session by ID.
func (svc *AISessionService) GetSession(id int) (AISessionRecord, bool) {
	return svc.store.GetByID(id)
}

// GetConversation returns the full conversation for a session.
func (svc *AISessionService) GetConversation(id int) ([]ai.ConversationMessage, error) {
	record, ok := svc.store.GetByID(id)
	if !ok {
		return nil, nil
	}

	switch record.Type {
	case string(ai.AssistantTypeClaudeCode):
		return ai.ParseClaudeConversation(record.FilePath)
	case string(ai.AssistantTypeCodex):
		return ai.ParseCodexConversation(record.FilePath)
	default:
		return nil, nil
	}
}

// RefreshSession re-parses the session file and returns the updated conversation.
func (svc *AISessionService) RefreshSession(id int) ([]ai.ConversationMessage, error) {
	record, ok := svc.store.GetByID(id)
	if !ok {
		return nil, nil
	}

	// Re-scan the file to update metadata.
	sessions, err := scanSessionByFilePath(record.FilePath)
	if err != nil {
		return nil, err
	}
	if len(sessions) > 0 {
		svc.store.Upsert(sessions[0])
	}

	return svc.GetConversation(id)
}

// CleanupStaleSessions removes sessions whose files no longer exist.
func (svc *AISessionService) CleanupStaleSessions() int {
	allSessions := scanAllSessions()

	existing := make(map[string]bool, len(allSessions))
	for _, s := range allSessions {
		existing[s.FilePath] = true
	}

	return svc.store.DeleteMissingFiles(existing)
}

// scanAllSessions scans both Claude and Codex session directories.
func scanAllSessions() []ai.AISession {
	var allSessions []ai.AISession

	projects, err := ai.ScanClaudeProjects()
	if err == nil {
		for _, project := range projects {
			sessions, scanErr := ai.ScanClaudeSessions(project)
			if scanErr == nil {
				allSessions = append(allSessions, sessions...)
			}
		}
	}

	codexSessions, err := ai.ScanCodexSessions()
	if err == nil {
		allSessions = append(allSessions, codexSessions...)
	}

	return allSessions
}

// scanSessionByFilePath re-scans all session directories and returns the
// session whose file path matches, or an empty slice if not found.
func scanSessionByFilePath(filePath string) ([]ai.AISession, error) {
	allSessions := scanAllSessions()
	for _, s := range allSessions {
		if s.FilePath == filePath {
			return []ai.AISession{s}, nil
		}
	}
	return nil, nil
}
