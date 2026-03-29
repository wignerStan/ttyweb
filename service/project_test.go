package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// initGitRepo creates a bare git repository at the given path with an initial commit.
func initGitRepo(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create repo dir: %v", err)
	}

	runGit(t, path, "init")
	runGit(t, path, "config", "user.email", "test@example.com")
	runGit(t, path, "config", "user.name", "Test User")

	// Create an initial commit so HEAD is valid.
	readme := filepath.Join(path, "README.md")
	if err := os.WriteFile(readme, []byte("test"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, path, "add", "README.md")
	runGit(t, path, "commit", "-m", "initial")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s (err: %v)", args, string(out), err)
	}
}

func TestAddProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	project, err := svc.AddProject("test-project", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	if project.ID == "" {
		t.Error("expected non-empty ID")
	}
	if project.Name != "test-project" {
		t.Errorf("expected name 'test-project', got %q", project.Name)
	}
	if project.Path != repoDir {
		t.Errorf("expected path %q, got %q", repoDir, project.Path)
	}
	if project.Priority != 0 {
		t.Errorf("expected priority 0, got %d", project.Priority)
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestAddProject_WithOptions(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	now := time.Now().UTC()

	project, err := svc.AddProject("my-project", repoDir,
		WithDescription("A test project"),
		WithDefaultBranch("develop"),
		WithWorktreeBasePath("/tmp/worktrees"),
		WithPriority(5),
	)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	if project.Description != "A test project" {
		t.Errorf("expected description 'A test project', got %q", project.Description)
	}
	if project.DefaultBranch != "develop" {
		t.Errorf("expected default branch 'develop', got %q", project.DefaultBranch)
	}
	if project.WorktreeBasePath != "/tmp/worktrees" {
		t.Errorf("expected worktree base path '/tmp/worktrees', got %q", project.WorktreeBasePath)
	}
	if project.Priority != 5 {
		t.Errorf("expected priority 5, got %d", project.Priority)
	}

	// Verify it was persisted.
	found, err := svc.GetProject(project.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if found.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", found.Name)
	}
	if !found.CreatedAt.Before(now.Add(time.Second)) {
		t.Error("CreatedAt should be set")
	}
}

func TestAddProject_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.AddProject("", "/some/path")
	if err != ErrProjectNameRequired {
		t.Errorf("expected ErrProjectNameRequired, got %v", err)
	}
}

func TestAddProject_EmptyPath(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.AddProject("name", "")
	if err != ErrProjectPathRequired {
		t.Errorf("expected ErrProjectPathRequired, got %v", err)
	}
}

func TestAddProject_NonGitRepo(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	dir := t.TempDir()

	_, err := svc.AddProject("bad-project", dir)
	if err != ErrInvalidProjectPath {
		t.Errorf("expected ErrInvalidProjectPath, got %v", err)
	}
}

func TestAddProject_RelativePath(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	// Change to the parent directory and use a relative path.
	parentDir := filepath.Dir(repoDir)
	basename := filepath.Base(repoDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(parentDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	project, err := svc.AddProject("rel-project", basename)
	if err != nil {
		t.Fatalf("AddProject with relative path: %v", err)
	}

	// Path should have been converted to absolute.
	if !filepath.IsAbs(project.Path) {
		t.Errorf("expected absolute path, got %q", project.Path)
	}
	if project.Path != repoDir {
		t.Errorf("expected path %q, got %q", repoDir, project.Path)
	}
}

func TestAddProject_DuplicatePath(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	_, err := svc.AddProject("first", repoDir)
	if err != nil {
		t.Fatalf("AddProject first: %v", err)
	}

	_, err = svc.AddProject("second", repoDir)
	if err != ErrProjectAlreadyExists {
		t.Errorf("expected ErrProjectAlreadyExists, got %v", err)
	}
}

func TestListProjects(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir1 := t.TempDir()
	initGitRepo(t, repoDir1)
	repoDir2 := t.TempDir()
	initGitRepo(t, repoDir2)

	p1, err := svc.AddProject("alpha", repoDir1, WithPriority(1))
	if err != nil {
		t.Fatalf("AddProject alpha: %v", err)
	}

	_, err = svc.AddProject("beta", repoDir2)
	if err != nil {
		t.Fatalf("AddProject beta: %v", err)
	}

	projects, err := svc.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// alpha has higher priority, should be first.
	if projects[0].ID != p1.ID {
		t.Errorf("expected alpha first (higher priority), got %q", projects[0].Name)
	}
}

func TestGetProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("get-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	found, err := svc.GetProject(created.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if found.Name != "get-test" {
		t.Errorf("expected name 'get-test', got %q", found.Name)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.GetProject("nonexistent")
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestUpdateProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("update-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	name := "renamed"
	desc := "updated description"
	priority := 3
	req := UpdateProjectRequest{
		Name:        &name,
		Description: &desc,
		Priority:    &priority,
	}

	updated, err := svc.UpdateProject(created.ID, req)
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	if updated.Name != "renamed" {
		t.Errorf("expected name 'renamed', got %q", updated.Name)
	}
	if updated.Description != "updated description" {
		t.Errorf("expected description 'updated description', got %q", updated.Description)
	}
	if updated.Priority != 3 {
		t.Errorf("expected priority 3, got %d", updated.Priority)
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	name := "nope"
	_, err := svc.UpdateProject("nonexistent", UpdateProjectRequest{Name: &name})
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestUpdateProject_MassAssignmentBlocked(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("mass-assign-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	// Only name and description are in the request. Path, ID, RemoteURL,
	// DefaultBranch, LastSyncAt are NOT part of UpdateProjectRequest,
	// so they cannot be set via UpdateProject even if an attacker tries
	// to inject extra JSON fields (they will be ignored during decode).
	name := "updated-name"
	desc := "updated-desc"
	req := UpdateProjectRequest{
		Name:        &name,
		Description: &desc,
	}

	updated, err := svc.UpdateProject(created.ID, req)
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	// Whitelisted fields should be updated.
	if updated.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got %q", updated.Name)
	}
	if updated.Description != "updated-desc" {
		t.Errorf("expected description 'updated-desc', got %q", updated.Description)
	}

	// Non-whitelisted fields must remain unchanged.
	if updated.Path != created.Path {
		t.Errorf("path should not have changed, got %q", updated.Path)
	}
	if updated.ID != created.ID {
		t.Errorf("ID should not have changed, got %q", updated.ID)
	}
	if updated.RemoteURL != created.RemoteURL {
		t.Errorf("remote_url should not have changed, got %q", updated.RemoteURL)
	}
	if updated.DefaultBranch != created.DefaultBranch {
		t.Errorf("default_branch should not have changed, got %q", updated.DefaultBranch)
	}
}

func TestUpdateProject_EmptyRequest(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("empty-req-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	// Empty request (all nil fields) should return the project unchanged.
	result, err := svc.UpdateProject(created.ID, UpdateProjectRequest{})
	if err != nil {
		t.Fatalf("UpdateProject with empty request: %v", err)
	}
	if result.Name != created.Name {
		t.Errorf("expected name %q unchanged, got %q", created.Name, result.Name)
	}
}

func TestUpdateProject_PartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	desc := "original description"
	created, err := svc.AddProject("partial-test", repoDir, WithDescription(desc), WithPriority(1))
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	// Update only priority; description should remain unchanged.
	priority := 10
	req := UpdateProjectRequest{Priority: &priority}

	updated, err := svc.UpdateProject(created.ID, req)
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	if updated.Priority != 10 {
		t.Errorf("expected priority 10, got %d", updated.Priority)
	}
	if updated.Description != "original description" {
		t.Errorf("expected description unchanged, got %q", updated.Description)
	}
}

func TestUpdateProject_WorktreeBasePath(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("worktree-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	wtPath := "/tmp/my-worktrees"
	req := UpdateProjectRequest{WorktreeBasePath: &wtPath}

	updated, err := svc.UpdateProject(created.ID, req)
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	if updated.WorktreeBasePath != "/tmp/my-worktrees" {
		t.Errorf("expected worktree_base_path '/tmp/my-worktrees', got %q", updated.WorktreeBasePath)
	}
}

func TestToUpdates_OnlyNonNilFields(t *testing.T) {
	name := "name-only"
	req := UpdateProjectRequest{Name: &name}

	updates := req.toUpdates()

	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d: %v", len(updates), updates)
	}
	if updates["name"] != "name-only" {
		t.Errorf("expected name 'name-only', got %v", updates["name"])
	}
	if _, ok := updates["description"]; ok {
		t.Error("description should not be in updates when nil")
	}
	if _, ok := updates["priority"]; ok {
		t.Error("priority should not be in updates when nil")
	}
	if _, ok := updates["worktree_base_path"]; ok {
		t.Error("worktree_base_path should not be in updates when nil")
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	created, err := svc.AddProject("delete-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	err = svc.DeleteProject(created.ID)
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	// Verify it's gone.
	_, err = svc.GetProject(created.ID)
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound after delete, got %v", err)
	}

	// Verify list is empty.
	projects, err := svc.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects after delete, got %d", len(projects))
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	err := svc.DeleteProject("nonexistent")
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestSyncProject(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	// Add a remote so SyncProject can detect it.
	runGit(t, repoDir, "remote", "add", "origin", "https://github.com/example/repo.git")

	project, err := svc.AddProject("sync-test", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	synced, err := svc.SyncProject(project.ID)
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	if synced.RemoteURL != "https://github.com/example/repo.git" {
		t.Errorf("expected remote URL, got %q", synced.RemoteURL)
	}
	if synced.DefaultBranch != "main" {
		t.Errorf("expected default branch 'main', got %q", synced.DefaultBranch)
	}
	if synced.LastSyncAt == nil {
		t.Error("expected LastSyncAt to be set")
	}
}

func TestSyncProject_NoRemote(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	project, err := svc.AddProject("no-remote", repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	synced, err := svc.SyncProject(project.ID)
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}

	if synced.RemoteURL != "" {
		t.Errorf("expected empty remote URL, got %q", synced.RemoteURL)
	}
}

func TestSyncProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProjectService(db)

	_, err := svc.SyncProject("nonexistent")
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestGenerateID_InProjectContext(t *testing.T) {
	id1, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	if len(id1) != 16 {
		t.Errorf("expected 16-char ID, got %d chars: %q", len(id1), id1)
	}

	id2, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}
