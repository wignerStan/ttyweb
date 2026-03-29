// Package service provides a WorktreeService that bridges git operations
// with DB persistence using an in-memory store. It implements the same
// logical operations as CodeKanban's WorktreeService but without GORM.
package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ttyweb/worktree"
)

// WtProject represents a registered git project for worktree management.
type WtProject struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WorktreeRecord represents a persisted worktree.
type WorktreeRecord struct {
	ID                string `json:"id"`
	ProjectID         string `json:"projectId"`
	BranchName        string `json:"branchName"`
	Path              string `json:"path"`
	IsMain            bool   `json:"isMain"`
	HeadCommit        string `json:"headCommit,omitempty"`
	HeadCommitMessage string `json:"headCommitMessage,omitempty"`
	StatusAhead       int    `json:"statusAhead"`
	StatusBehind      int    `json:"statusBehind"`
	StatusModified    int    `json:"statusModified"`
	StatusStaged      int    `json:"statusStaged"`
	StatusUntracked   int    `json:"statusUntracked"`
	StatusConflicts   int    `json:"statusConflicts"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// WorktreeService manages projects and worktrees with in-memory persistence.
type WorktreeService struct {
	mu        sync.RWMutex
	projects  map[string]WtProject
	worktrees map[string]WorktreeRecord
	nextID    atomic.Int64
	repoLock  *worktree.RepoLock
}

// NewWorktreeService creates a WorktreeService with initialized storage.
func NewWorktreeService() *WorktreeService {
	return &WorktreeService{
		projects:  make(map[string]WtProject),
		worktrees: make(map[string]WorktreeRecord),
		repoLock:  worktree.NewRepoLock(),
	}
}

// --- WtProject operations ---

// ListProjects returns all registered projects.
func (s *WorktreeService) ListProjects() []WtProject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]WtProject, 0, len(s.projects))
	for _, p := range s.projects {
		result = append(result, p)
	}
	return result
}

// AddProject validates path is a git repo and registers it.
func (s *WorktreeService) AddProject(path string) (*WtProject, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("path is required")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if !worktree.IsGitRepo(absPath) {
		return nil, errors.New("path is not a git repository")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate by path.
	for _, p := range s.projects {
		if p.Path == absPath {
			return nil, fmt.Errorf("project already registered: %s", absPath)
		}
	}

	now := time.Now()
	id := s.newID()
	name := filepath.Base(absPath)
	project := WtProject{
		ID:        id,
		Path:      absPath,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.projects[id] = project

	return &project, nil
}

// GetProject returns a project by ID.
func (s *WorktreeService) GetProject(id string) (*WtProject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return &p, nil
}

// --- Worktree operations ---

// CreateWorktree creates a new worktree for a project and persists it.
func (s *WorktreeService) CreateWorktree(ctx context.Context, projectID, branchName, baseBranch string, createBranch bool) (*WorktreeRecord, error) {
	if branchName = strings.TrimSpace(branchName); branchName == "" {
		return nil, errors.New("branch name is required")
	}
	if err := worktree.ValidateBranchName(branchName); err != nil {
		return nil, err
	}

	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	unlock := s.repoLock.Lock(project.Path, ctx)
	if unlock == nil {
		return nil, context.Canceled
	}
	defer unlock()

	wtPath, err := worktree.CreateWorktree(ctx, project.Path, branchName, baseBranch, createBranch)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	record := WorktreeRecord{
		ID:         s.newID(),
		ProjectID:  projectID,
		BranchName: branchName,
		Path:       wtPath,
		IsMain:     false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Populate initial status.
	if status, err := worktree.GetWorktreeStatus(ctx, wtPath); err == nil {
		record = record.withStatus(status)
	}

	s.mu.Lock()
	s.worktrees[record.ID] = record
	s.mu.Unlock()

	return &record, nil
}

// ListWorktrees returns worktrees for a project, syncing with git state first.
func (s *WorktreeService) ListWorktrees(ctx context.Context, projectID string) ([]WorktreeRecord, error) {
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	// Sync with git state first (takes its own lock).
	_ = s.syncWorktrees(ctx, project)

	return s.worktreesForProject(projectID), nil
}

// RemoveWorktree removes a worktree by ID.
func (s *WorktreeService) RemoveWorktree(ctx context.Context, projectID, worktreeID string, force bool) error {
	project, err := s.GetProject(projectID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	record, err := s.getWorktreeLocked(worktreeID, projectID)
	s.mu.Unlock()
	if err != nil {
		return err
	}

	if record.IsMain {
		return errors.New("cannot remove main worktree")
	}

	unlock := s.repoLock.Lock(project.Path, ctx)
	if unlock == nil {
		return context.Canceled
	}
	defer unlock()

	if err := worktree.RemoveWorktree(ctx, project.Path, record.Path, force); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.worktrees, worktreeID)
	s.mu.Unlock()

	return nil
}

// RefreshWorktree updates the status of a worktree from git.
func (s *WorktreeService) RefreshWorktree(ctx context.Context, projectID, worktreeID string) (*WorktreeRecord, error) {
	s.mu.RLock()
	record, err := s.getWorktreeLocked(worktreeID, projectID)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	status, err := worktree.GetWorktreeStatus(ctx, record.Path)
	if err != nil {
		return nil, fmt.Errorf("get worktree status: %w", err)
	}

	record = record.withStatus(status)
	record.UpdatedAt = time.Now()

	// Fetch updated commit info from git list.
	infos, err := worktree.ListWorktrees(ctx, filepath.Dir(record.Path))
	if err == nil {
		for _, info := range infos {
			if worktree.EqualPath(info.Path, record.Path) {
				record.HeadCommit = info.HeadCommit
				record.HeadCommitMessage = info.HeadMessage
				record.BranchName = info.Branch
				record.IsMain = info.IsMain
				break
			}
		}
	}

	s.mu.Lock()
	s.worktrees[worktreeID] = record
	s.mu.Unlock()

	return &record, nil
}

// SyncAllWorktrees syncs all worktrees for a project.
func (s *WorktreeService) SyncAllWorktrees(ctx context.Context, projectID string) ([]WorktreeRecord, error) {
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	if err := s.syncWorktrees(ctx, project); err != nil {
		return nil, err
	}

	return s.worktreesForProject(projectID), nil
}

// CommitWorktree stages all and commits in a worktree.
func (s *WorktreeService) CommitWorktree(ctx context.Context, projectID, worktreeID, message string) (*WorktreeRecord, error) {
	s.mu.RLock()
	record, err := s.getWorktreeLocked(worktreeID, projectID)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	unlock := s.repoLock.Lock(project.Path, ctx)
	if unlock == nil {
		return nil, context.Canceled
	}
	defer unlock()

	if err := worktree.CommitWorktree(ctx, record.Path, message); err != nil {
		return nil, err
	}

	return s.RefreshWorktree(ctx, projectID, worktreeID)
}

// --- internal ---

// worktreesForProject returns all worktree records for the given project.
// Caller must hold at least a read lock on s.mu.
func (s *WorktreeService) worktreesForProject(projectID string) []WorktreeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]WorktreeRecord, 0)
	for _, wt := range s.worktrees {
		if wt.ProjectID == projectID {
			result = append(result, wt)
		}
	}
	return result
}

// getWorktreeLocked looks up a worktree by ID and validates project ownership.
// Caller must hold s.mu (at least a read lock).
func (s *WorktreeService) getWorktreeLocked(worktreeID, projectID string) (WorktreeRecord, error) {
	record, ok := s.worktrees[worktreeID]
	if !ok || record.ProjectID != projectID {
		return WorktreeRecord{}, fmt.Errorf("worktree not found: %s", worktreeID)
	}
	return record, nil
}

func (s *WorktreeService) syncWorktrees(ctx context.Context, project *WtProject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	gitWorktrees, err := worktree.ListWorktrees(ctx, project.Path)
	if err != nil {
		return err
	}

	// Build map of existing DB worktrees by normalized path.
	dbByPath := make(map[string]string)
	for id, wt := range s.worktrees {
		if wt.ProjectID == project.ID {
			dbByPath[normalizePath(wt.Path)] = id
		}
	}

	now := time.Now()
	for _, gwt := range gitWorktrees {
		norm := normalizePath(gwt.Path)
		if dbID, exists := dbByPath[norm]; exists {
			// Update existing.
			rec := s.worktrees[dbID]
			rec.BranchName = gwt.Branch
			rec.HeadCommit = gwt.HeadCommit
			rec.HeadCommitMessage = gwt.HeadMessage
			rec.IsMain = gwt.IsMain
			rec.Path = gwt.Path
			rec.UpdatedAt = now
			s.worktrees[dbID] = rec
			delete(dbByPath, norm)
		} else {
			// Insert new.
			record := WorktreeRecord{
				ID:                s.newID(),
				ProjectID:         project.ID,
				BranchName:        gwt.Branch,
				Path:              gwt.Path,
				IsMain:            gwt.IsMain,
				HeadCommit:        gwt.HeadCommit,
				HeadCommitMessage: gwt.HeadMessage,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			s.worktrees[record.ID] = record
		}
	}

	// Remove DB entries not found in git.
	for _, dbID := range dbByPath {
		delete(s.worktrees, dbID)
	}

	return nil
}

func (s *WorktreeService) newID() string {
	return fmt.Sprintf("wt-%d", s.nextID.Add(1))
}

func normalizePath(p string) string {
	return filepath.Clean(p)
}

// withStatus returns a new WorktreeRecord with status fields populated.
func (r WorktreeRecord) withStatus(status *worktree.WorktreeStatus) WorktreeRecord {
	return WorktreeRecord{
		ID:                r.ID,
		ProjectID:         r.ProjectID,
		BranchName:        r.BranchName,
		Path:              r.Path,
		IsMain:            r.IsMain,
		HeadCommit:        r.HeadCommit,
		HeadCommitMessage: r.HeadCommitMessage,
		StatusAhead:       status.Ahead,
		StatusBehind:      status.Behind,
		StatusModified:    status.Modified,
		StatusStaged:      status.Staged,
		StatusUntracked:   status.Untracked,
		StatusConflicts:   status.Conflicts,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}
