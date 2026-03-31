package server

import (
	"fmt"
	"sync"
	"time"
)

// Profile represents a saved session profile configuration.
type Profile struct {
	ID         int    `json:"id"`
	ProfileKey string `json:"profile_key"`
	Name       string `json:"name"`
	SortOrder  int    `json:"sort_order"`
}

// SessionGroup represents a group of sessions within a profile.
type SessionGroup struct {
	ID         int    `json:"id"`
	GroupName  string `json:"group_name"`
	SortOrder  int    `json:"sort_order"`
	ProfileKey string `json:"profile_key"`
}

// Snippet represents a saved command snippet.
type Snippet struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

// AiRole represents a custom AI assistant role with a system prompt.
type AiRole struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
}

// TaskEvent represents a conversation event within a task.
type TaskEvent struct {
	ID        int            `json:"id"`
	TaskID    string         `json:"task_id"`
	PaneKey   string         `json:"pane_key"`
	Timestamp time.Time      `json:"ts"`
	Event     string         `json:"event"`
	Data      map[string]any `json:"data,omitempty"`
	Completed bool           `json:"completed"`
}

// MemoryStore provides an in-memory data store with thread-safe CRUD operations.
type MemoryStore struct {
	mu            sync.RWMutex
	profiles      []Profile
	groups        []SessionGroup
	snippets      []Snippet
	roles         []AiRole
	tasks         []TaskEvent
	paneStatuses  map[string]string
	nextProfileID int
	nextGroupID   int
	nextRoleID    int
	nextTaskID    int
}

// NewMemoryStore creates and initializes a MemoryStore with default AI roles.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		profiles:      []Profile{},
		groups:        []SessionGroup{},
		snippets:      []Snippet{},
		roles:         builtinRoles(),
		tasks:         []TaskEvent{},
		paneStatuses:  make(map[string]string),
		nextProfileID: 1,
		nextGroupID:   1,
		nextRoleID:    1,
		nextTaskID:    1,
	}

	// Set next IDs past the default roles.
	s.nextRoleID = len(builtinRoles()) + 1
	return s
}

func builtinRoles() []AiRole {
	return []AiRole{
		{ID: 1, Name: "CLI", Description: "Command-line interface expert", SystemPrompt: "You are a CLI expert. Help users with terminal commands, shell scripting, and command-line tools."},
		{ID: 2, Name: "Operations", Description: "Systems operations expert", SystemPrompt: "You are an operations expert. Help with system administration, deployment, monitoring, and infrastructure."},
		{ID: 3, Name: "Frontend", Description: "Frontend development expert", SystemPrompt: "You are a frontend expert. Help with HTML, CSS, JavaScript, TypeScript, React, Vue, and web development."},
		{ID: 4, Name: "Backend", Description: "Backend development expert", SystemPrompt: "You are a backend expert. Help with server-side development, APIs, databases, and backend architecture."},
		{ID: 5, Name: "Full-Stack", Description: "Full-stack development expert", SystemPrompt: "You are a full-stack expert. Help with both frontend and backend development, architecture decisions, and DevOps."},
		{ID: 6, Name: "Security", Description: "Security expert", SystemPrompt: "You are a security expert. Help with application security, code review, vulnerability assessment, and secure coding practices."},
		{ID: 7, Name: "DevOps", Description: "DevOps and CI/CD expert", SystemPrompt: "You are a DevOps expert. Help with CI/CD pipelines, containerization, orchestration, and infrastructure as code."},
	}
}

// --- Profile CRUD ---

// ListProfiles returns all stored profiles.
func (s *MemoryStore) ListProfiles() []Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Profile, len(s.profiles))
	copy(result, s.profiles)
	return result
}

// CreateProfile adds a new profile and returns it.
func (s *MemoryStore) CreateProfile(p Profile) Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextProfileID
	s.nextProfileID++
	s.profiles = append(s.profiles, p)
	return p
}

// UpdateProfile updates an existing profile by ID.
func (s *MemoryStore) UpdateProfile(id int, p Profile) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.profiles {
		if existing.ID == id {
			p.ID = id
			s.profiles[i] = p
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("profile not found")
}

// DeleteProfile removes a profile by ID.
func (s *MemoryStore) DeleteProfile(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.profiles {
		if existing.ID == id {
			s.profiles = append(s.profiles[:i], s.profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile not found")
}

// --- Group CRUD ---

// ListGroups returns session groups, optionally filtered by profile key.
func (s *MemoryStore) ListGroups(profileKey string) []SessionGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if profileKey == "" {
		result := make([]SessionGroup, len(s.groups))
		copy(result, s.groups)
		return result
	}
	var filtered []SessionGroup
	for _, g := range s.groups {
		if g.ProfileKey == profileKey {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

// CreateGroup adds a new session group and returns it.
func (s *MemoryStore) CreateGroup(g SessionGroup) SessionGroup {
	s.mu.Lock()
	defer s.mu.Unlock()
	g.ID = s.nextGroupID
	s.nextGroupID++
	s.groups = append(s.groups, g)
	return g
}

// UpdateGroup updates an existing group by ID.
func (s *MemoryStore) UpdateGroup(id int, g SessionGroup) (SessionGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.groups {
		if existing.ID == id {
			g.ID = id
			s.groups[i] = g
			return g, nil
		}
	}
	return SessionGroup{}, fmt.Errorf("group not found")
}

// DeleteGroup removes a group by ID.
func (s *MemoryStore) DeleteGroup(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.groups {
		if existing.ID == id {
			s.groups = append(s.groups[:i], s.groups[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("group not found")
}

// --- Snippet CRUD ---

// ListSnippets returns all stored snippets.
func (s *MemoryStore) ListSnippets() []Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Snippet, len(s.snippets))
	copy(result, s.snippets)
	return result
}

// CreateSnippet adds a new snippet and returns it.
func (s *MemoryStore) CreateSnippet(sn Snippet) Snippet {
	s.mu.Lock()
	defer s.mu.Unlock()
	sn.Index = len(s.snippets)
	s.snippets = append(s.snippets, sn)
	return sn
}

// UpdateSnippet updates an existing snippet by index.
func (s *MemoryStore) UpdateSnippet(index int, sn Snippet) (Snippet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.snippets) {
		return Snippet{}, fmt.Errorf("snippet not found at index %d", index)
	}
	sn.Index = index
	s.snippets[index] = sn
	return sn, nil
}

// DeleteSnippet removes a snippet by index.
func (s *MemoryStore) DeleteSnippet(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.snippets) {
		return fmt.Errorf("snippet not found at index %d", index)
	}
	s.snippets = append(s.snippets[:index], s.snippets[index+1:]...)
	// Re-index remaining snippets.
	for i := range s.snippets {
		s.snippets[i].Index = i
	}
	return nil
}

// --- AiRole CRUD ---

// ListRoles returns all stored AI roles.
func (s *MemoryStore) ListRoles() []AiRole {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AiRole, len(s.roles))
	copy(result, s.roles)
	return result
}

// CreateRole adds a new AI role and returns it.
func (s *MemoryStore) CreateRole(r AiRole) AiRole {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.ID = s.nextRoleID
	s.nextRoleID++
	s.roles = append(s.roles, r)
	return r
}

// UpdateRole updates an existing AI role by ID.
func (s *MemoryStore) UpdateRole(id int, r AiRole) (AiRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.roles {
		if existing.ID == id {
			r.ID = id
			s.roles[i] = r
			return r, nil
		}
	}
	return AiRole{}, fmt.Errorf("role not found")
}

// DeleteRole removes an AI role by ID.
func (s *MemoryStore) DeleteRole(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.roles {
		if existing.ID == id {
			s.roles = append(s.roles[:i], s.roles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("role not found")
}

// --- Task CRUD ---

// ListTasks returns task events with pagination.
func (s *MemoryStore) ListTasks(page, limit int) ([]TaskEvent, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := len(s.tasks)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= total {
		return []TaskEvent{}, total
	}
	end := start + limit
	if end > total {
		end = total
	}
	result := make([]TaskEvent, end-start)
	copy(result, s.tasks[start:end])
	return result, total
}

// AddTaskEvent appends a new task event and returns it.
func (s *MemoryStore) AddTaskEvent(t *TaskEvent) TaskEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.ID = s.nextTaskID
	s.nextTaskID++
	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}
	s.tasks = append(s.tasks, *t)
	return *t
}

// CompleteTask marks a task as completed.
func (s *MemoryStore) CompleteTask(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.tasks {
		if t.ID == id {
			updated := s.tasks[i]
			updated.Completed = true
			s.tasks[i] = updated
			return nil
		}
	}
	return fmt.Errorf("task not found")
}

// GetTaskEventsByPane returns task events for a given pane.
func (s *MemoryStore) GetTaskEventsByPane(paneKey string) []TaskEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var events []TaskEvent
	for _, t := range s.tasks {
		if t.PaneKey == paneKey {
			events = append(events, t)
		}
	}
	return events
}

// --- Pane Status ---

// GetPaneStatuses returns all stored pane statuses.
func (s *MemoryStore) GetPaneStatuses() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.paneStatuses))
	for k, v := range s.paneStatuses {
		result[k] = v
	}
	return result
}

// SetPaneStatus stores the status for a pane.
func (s *MemoryStore) SetPaneStatus(paneKey, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paneStatuses[paneKey] = status
}

// --- Pane Mode ---

// GetPaneMode returns the current mode ("pane" or "control") for a pane.
// Returns empty string if no mode is set.
func (s *MemoryStore) GetPaneMode(paneKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paneStatuses["mode:"+paneKey]
}

// SetPaneMode stores the mode for a pane. Only "pane" and "control" are valid.
func (s *MemoryStore) SetPaneMode(paneKey, mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if mode != "pane" && mode != "control" {
		return
	}
	s.paneStatuses["mode:"+paneKey] = mode
}

// store is the global in-memory data store.
var store = NewMemoryStore()

// storeMu protects the global store variable during test resets.
var storeMu sync.Mutex

// resetStore replaces the global store. Used only in tests.
func resetStore() {
	storeMu.Lock()
	defer storeMu.Unlock()
	store = NewMemoryStore()
}
