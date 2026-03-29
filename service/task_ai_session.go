// Package service provides in-memory business-logic services.
package service

import (
	"fmt"
	"sync"

	"ttyweb/db"
	"ttyweb/pkg/randomstring"
)

// requireFields validates that label/value pairs are non-empty, returning an
// error for the first blank value encountered. Call as:
//
//	requireFields("task_id", taskID, "ai_session_id", aiSessionID)
func requireFields(pairs ...string) error {
	if len(pairs)%2 != 0 {
		panic("requireFields: must receive an even number of arguments (label, value pairs)")
	}
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			return fmt.Errorf("%s is required", pairs[i])
		}
	}
	return nil
}

// TaskAISessionService manages many-to-many links between tasks and AI sessions.
type TaskAISessionService struct {
	mu    sync.RWMutex
	links []db.TaskAISession
}

// NewTaskAISessionService creates a ready-to-use TaskAISessionService.
func NewTaskAISessionService() *TaskAISessionService {
	return &TaskAISessionService{}
}

// LinkSession creates an association between a task and an AI session.
func (s *TaskAISessionService) LinkSession(taskID, aiSessionID string) (*db.TaskAISession, error) {
	if err := requireFields("task_id", taskID, "ai_session_id", aiSessionID); err != nil {
		return nil, err
	}

	link := db.TaskAISession{
		ID:          randomstring.Generate(32),
		TaskID:      taskID,
		AISessionID: aiSessionID,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.links = append(s.links, link)

	return &link, nil
}

// UnlinkSession removes the association between a task and an AI session.
func (s *TaskAISessionService) UnlinkSession(taskID, aiSessionID string) error {
	if err := requireFields("task_id", taskID, "ai_session_id", aiSessionID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, link := range s.links {
		if link.TaskID == taskID && link.AISessionID == aiSessionID {
			s.links = append(s.links[:i], s.links[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("link not found")
}

// ListLinkedSessions returns all AI sessions linked to the given task.
func (s *TaskAISessionService) ListLinkedSessions(taskID string) ([]db.TaskAISession, error) {
	if err := requireFields("task_id", taskID); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]db.TaskAISession, 0)
	for _, link := range s.links {
		if link.TaskID == taskID {
			result = append(result, link)
		}
	}
	return result, nil
}
