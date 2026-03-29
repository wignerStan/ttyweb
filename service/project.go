// Package service provides domain logic for project workspace management.
package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"ttyweb/db"
)

func init() {
	db.RegisterModel(&Project{})
}

// UpdateProjectRequest contains only the fields that users are allowed to
// update on a project. Any field not present in this struct is ignored,
// preventing mass-assignment of sensitive columns (id, path, remote_url,
// last_sync_at, created_at, updated_at).
type UpdateProjectRequest struct {
	Name             *string `json:"name"`
	Description      *string `json:"description"`
	Priority         *int    `json:"priority"`
	WorktreeBasePath *string `json:"worktree_base_path"`
}

// toUpdates converts the non-nil fields of the request into a GORM updates
// map. Only fields explicitly set in the request will be included.
func (r *UpdateProjectRequest) toUpdates() map[string]interface{} {
	updates := make(map[string]interface{})
	if r.Name != nil {
		updates["name"] = *r.Name
	}
	if r.Description != nil {
		updates["description"] = *r.Description
	}
	if r.Priority != nil {
		updates["priority"] = *r.Priority
	}
	if r.WorktreeBasePath != nil {
		updates["worktree_base_path"] = *r.WorktreeBasePath
	}
	return updates
}

// runGitCommand executes a git command in the given directory and returns its
// trimmed stdout. Returns an error if the command fails.
func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Project represents a tracked git repository with metadata.
type Project struct {
	ID              string     `gorm:"primaryKey;type:text" json:"id"`
	Name            string     `gorm:"type:text;not null;index" json:"name"`
	Path            string     `gorm:"type:text;not null;uniqueIndex" json:"path"`
	Description     string     `gorm:"type:text" json:"description"`
	DefaultBranch   string     `gorm:"type:text" json:"default_branch"`
	WorktreeBasePath string    `gorm:"type:text" json:"worktree_base_path"`
	RemoteURL       string     `gorm:"type:text" json:"remote_url"`
	LastSyncAt      *time.Time `gorm:"type:datetime" json:"last_sync_at"`
	Priority        int        `gorm:"type:integer;not null;default:0" json:"priority"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName maps the GORM model to the projects table.
func (Project) TableName() string {
	return "projects"
}

// Common errors returned by ProjectService methods.
var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrInvalidProjectPath  = errors.New("invalid project path: must be an absolute path to a directory containing .git")
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectPathRequired = errors.New("project path is required")
	ErrProjectAlreadyExists = errors.New("a project with this path already exists")
)

// ProjectOption is a functional option for configuring a new project.
type ProjectOption func(*Project)

// WithDescription sets the project description.
func WithDescription(desc string) ProjectOption {
	return func(p *Project) {
		p.Description = desc
	}
}

// WithDefaultBranch sets the default branch name.
func WithDefaultBranch(branch string) ProjectOption {
	return func(p *Project) {
		p.DefaultBranch = branch
	}
}

// WithWorktreeBasePath sets the worktree base path.
func WithWorktreeBasePath(path string) ProjectOption {
	return func(p *Project) {
		p.WorktreeBasePath = path
	}
}

// WithPriority sets the project priority.
func WithPriority(priority int) ProjectOption {
	return func(p *Project) {
		p.Priority = priority
	}
}

// ProjectService provides CRUD operations for projects.
type ProjectService struct {
	db *gorm.DB
}

// NewProjectService creates a new ProjectService backed by the given GORM database.
func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

// AddProject creates a new project after validating the path exists and is a git repo.
func (s *ProjectService) AddProject(name, path string, opts ...ProjectOption) (*Project, error) {
	if name == "" {
		return nil, ErrProjectNameRequired
	}
	if path == "" {
		return nil, ErrProjectPathRequired
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if err := validateGitRepo(absPath); err != nil {
		return nil, err
	}

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate project ID: %w", err)
	}

	project := &Project{
		ID:   id,
		Name: name,
		Path: absPath,
	}

	for _, opt := range opts {
		opt(project)
	}

	if err := s.db.Create(project).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrProjectAlreadyExists
		}
		return nil, fmt.Errorf("create project: %w", err)
	}

	return project, nil
}

// ListProjects returns all projects ordered by Priority desc, then Name.
func (s *ProjectService) ListProjects() ([]Project, error) {
	var projects []Project
	err := s.db.Order("priority DESC, name ASC").Find(&projects).Error
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

// GetProject returns a single project by ID.
func (s *ProjectService) GetProject(id string) (*Project, error) {
	var project Project
	err := s.db.Where("id = ?", id).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &project, nil
}

// UpdateProject applies partial updates to a project using only the fields
// present in the typed request. This prevents mass-assignment of sensitive
// columns (id, path, remote_url, last_sync_at, created_at, updated_at).
func (s *ProjectService) UpdateProject(id string, req UpdateProjectRequest) (*Project, error) {
	updates := req.toUpdates()
	if len(updates) == 0 {
		return s.GetProject(id)
	}

	return s.updateProjectFields(id, updates)
}

// updateProjectFields applies a GORM updates map to a project. This is an
// internal method used by both UpdateProject (user-facing, whitelisted) and
// SyncProject (system-facing, sets remote_url/default_branch/last_sync_at).
func (s *ProjectService) updateProjectFields(id string, updates map[string]interface{}) (*Project, error) {
	var project Project
	if err := s.db.Where("id = ?", id).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("find project: %w", err)
	}

	if err := s.db.Model(&project).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	// Reload to get the updated state.
	if err := s.db.Where("id = ?", id).First(&project).Error; err != nil {
		return nil, fmt.Errorf("reload project: %w", err)
	}

	return &project, nil
}

// DeleteProject removes a project by ID.
func (s *ProjectService) DeleteProject(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Project{})
	if result.Error != nil {
		return fmt.Errorf("delete project: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// SyncProject detects metadata from the git repository at the project's path.
// It populates RemoteURL, DefaultBranch, and LastSyncAt.
func (s *ProjectService) SyncProject(id string) (*Project, error) {
	project, err := s.GetProject(id)
	if err != nil {
		return nil, err
	}

	remoteURL, err := gitRemoteURL(project.Path)
	if err != nil {
		return nil, fmt.Errorf("detect remote URL: %w", err)
	}

	defaultBranch, err := gitDefaultBranch(project.Path)
	if err != nil {
		return nil, fmt.Errorf("detect default branch: %w", err)
	}

	updates := map[string]interface{}{
		"remote_url":     remoteURL,
		"default_branch": defaultBranch,
		"last_sync_at":   time.Now().UTC(),
	}

	return s.updateProjectFields(id, updates)
}

// validateGitRepo checks that path is an existing directory containing a .git
// entry (file or directory).
func validateGitRepo(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ErrInvalidProjectPath
	}

	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return ErrInvalidProjectPath
	}

	return nil
}

// gitRemoteURL returns the fetch URL of the origin remote, or an empty string
// if no origin remote is configured.
func gitRemoteURL(repoPath string) (string, error) {
	url, err := runGitCommand(repoPath, "remote", "get-url", "origin")
	if err != nil {
		// No origin remote configured is not an error.
		return "", nil
	}
	return url, nil
}

// gitDefaultBranch returns the default branch name (HEAD symbolic ref).
// Returns "main" as a fallback if the symbolic-ref command fails (detached HEAD
// or no commits).
func gitDefaultBranch(repoPath string) (string, error) {
	branch, err := runGitCommand(repoPath, "symbolic-ref", "--short", "HEAD")
	if err != nil || branch == "" {
		// Detached HEAD or no commits; fall back to "main".
		return "main", nil
	}
	return branch, nil
}

// generateID creates a random 16-character hex string using crypto/rand.
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
