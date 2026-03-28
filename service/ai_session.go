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

// AISessionStore provides in-memory CRUD for AI sessions.
type AISessionStore struct {
	mu       sync.RWMutex
	records  []AISessionRecord
	nextID   int
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
			updated := existing
			updated.ProjectPath = session.ProjectPath
			updated.FilePath = session.FilePath
			updated.Model = session.Model
			updated.Title = session.Title
			updated.LastMessageAt = session.LastMessageAt
			updated.MessageCount = session.MessageCount
			updated.AssistantMessageCount = session.AssistantMessageCount
			updated.FileModTime = session.FileModTime
			updated.FileSize = session.FileSize
			s.records[i] = updated
			return updated
		}
	}

	record := AISessionRecord{
		ID:           s.nextID,
		SessionID:    session.SessionID,
		Type:         session.Type,
		ProjectPath:  session.ProjectPath,
		FilePath:     session.FilePath,
		Model:        session.Model,
		Title:        session.Title,
		LastMessageAt: session.LastMessageAt,
		MessageCount: session.MessageCount,
		AssistantMessageCount: session.AssistantMessageCount,
		FileModTime:  session.FileModTime,
		FileSize:     session.FileSize,
	}
	if !session.SessionStartedAt.IsZero() {
		record.SessionStartedAt = session.SessionStartedAt
	} else {
		record.SessionStartedAt = session.FileModTime
	}

	s.nextID++
	s.records = append(s.records, record)
	return record
}

// List returns all stored sessions, sorted by last message time (newest first).
func (s *AISessionStore) List(projectPath string) []AISessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []AISessionRecord
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

// Delete removes a session by its database ID.
func (s *AISessionStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.records {
		if r.ID == id {
			s.records = append(s.records[:i], s.records[i+1:]...)
			return true
		}
	}
	return false
}

// DeleteByFilePath removes sessions whose file path is not in the provided set.
func (s *AISessionStore) DeleteMissingFiles(existingPaths map[string]bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	newRecords := make([]AISessionRecord, 0, len(s.records))
	for _, r := range s.records {
		if existingPaths[r.FilePath] {
			newRecords = append(newRecords, r)
		} else {
			removed++
		}
	}
	s.records = newRecords
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
	sessions, err := scanSessionByFilePath(record.Type, record.FilePath, record.ProjectPath)
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
	records := svc.store.List("")
	paths := make(map[string]bool, len(records))
	for _, r := range records {
		paths[r.FilePath] = true
	}

	// Re-scan to find which files still exist.
	var allSessions []ai.AISession
	claudeProjects, err := ai.ScanClaudeProjects()
	if err == nil {
		for _, project := range claudeProjects {
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

	existing := make(map[string]bool, len(allSessions))
	for _, s := range allSessions {
		existing[s.FilePath] = true
	}

	return svc.store.DeleteMissingFiles(existing)
}

// scanSessionByFilePath re-scans a specific session file by type.
func scanSessionByFilePath(sessionType, filePath, projectPath string) ([]ai.AISession, error) {
	if projectPath != "" {
		sessions, err := ai.ScanClaudeSessions(projectPath)
		if err != nil {
			return nil, err
		}
		for _, s := range sessions {
			if s.FilePath == filePath {
				return []ai.AISession{s}, nil
			}
		}
	}

	sessions, err := ai.ScanCodexSessions()
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if s.FilePath == filePath {
			return []ai.AISession{s}, nil
		}
	}

	return nil, nil
}
