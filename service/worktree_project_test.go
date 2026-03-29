package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"ttyweb/worktree"
)

// initTestGitRepo creates a real git repo with an initial commit.
func initTestGitRepo(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	runTestGit(t, path, "init")
	runTestGit(t, path, "config", "user.email", "test@example.com")
	runTestGit(t, path, "config", "user.name", "Test User")

	readme := filepath.Join(path, "README.md")
	if err := os.WriteFile(readme, []byte("test"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runTestGit(t, path, "add", "README.md")
	runTestGit(t, path, "commit", "-m", "initial")
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	cmd.Env = worktree.FilterGitEnv(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s (err: %v)", args, string(out), err)
	}
}

func TestNewWorktreeService(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestWorktreeService_ListProjects_Empty(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	projects := svc.ListProjects()
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestWorktreeService_ListProjects_NonEmpty(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	p1, err := svc.AddProject(repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	projects := svc.ListProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].ID != p1.ID {
		t.Errorf("expected ID %q, got %q", p1.ID, projects[0].ID)
	}
	if projects[0].Name != p1.Name {
		t.Errorf("expected name %q, got %q", p1.Name, projects[0].Name)
	}
}

func TestWorktreeService_AddProject(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, err := svc.AddProject(repoDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if project.ID == "" {
		t.Error("expected non-empty ID")
	}
	if project.Path != repoDir {
		t.Errorf("expected path %q, got %q", repoDir, project.Path)
	}
	if project.Name != filepath.Base(repoDir) {
		t.Errorf("expected name %q, got %q", filepath.Base(repoDir), project.Name)
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestWorktreeService_AddProject_EmptyPath(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.AddProject("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestWorktreeService_AddProject_WhitespacePath(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.AddProject("   ")
	if err == nil {
		t.Fatal("expected error for whitespace path")
	}
}

func TestWorktreeService_AddProject_NonexistentPath(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.AddProject("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestWorktreeService_AddProject_NotGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := NewWorktreeService()
	_, err := svc.AddProject(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if err.Error() != "path is not a git repository" {
		t.Errorf("expected 'path is not a git repository', got %q", err.Error())
	}
}

func TestWorktreeService_AddProject_Duplicate(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	_, err := svc.AddProject(repoDir)
	if err != nil {
		t.Fatalf("first AddProject: %v", err)
	}

	_, err = svc.AddProject(repoDir)
	if err == nil {
		t.Fatal("expected error for duplicate project")
	}
}

func TestWorktreeService_GetProject(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	created, _ := svc.AddProject(repoDir)

	found, err := svc.GetProject(created.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if found.Name != created.Name {
		t.Errorf("expected name %q, got %q", created.Name, found.Name)
	}
}

func TestWorktreeService_GetProject_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.GetProject("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/project", "/home/user/project"},
		{"/home/user/project/", "/home/user/project"},
		{"/home/user/../user/project", "/home/user/project"},
		{".", "."},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWithStatus(t *testing.T) {
	t.Parallel()

	record := WorktreeRecord{
		ID: "wt-1", ProjectID: "proj-1",
		BranchName: "main", Path: "/path/to/repo",
		IsMain: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	status := &worktree.Status{
		Ahead: 2, Behind: 1, Modified: 3,
		Staged: 4, Untracked: 5, Conflicts: 1,
	}

	updated := record.withStatus(status)
	if updated.StatusAhead != 2 || updated.StatusBehind != 1 {
		t.Errorf("expected ahead=2 behind=1, got ahead=%d behind=%d",
			updated.StatusAhead, updated.StatusBehind)
	}
	if updated.StatusModified != 3 || updated.StatusStaged != 4 {
		t.Errorf("expected modified=3 staged=4, got modified=%d staged=%d",
			updated.StatusModified, updated.StatusStaged)
	}
	if updated.StatusUntracked != 5 || updated.StatusConflicts != 1 {
		t.Errorf("expected untracked=5 conflicts=1, got untracked=%d conflicts=%d",
			updated.StatusUntracked, updated.StatusConflicts)
	}
	if updated.ID != record.ID || updated.BranchName != record.BranchName {
		t.Error("original fields should be preserved")
	}
}
