package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorktreeService_CreateWorktree(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	wt, err := svc.CreateWorktree(project.ID, "feature-branch", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if wt.ID == "" {
		t.Error("expected non-empty worktree ID")
	}
	if wt.BranchName != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got %q", wt.BranchName)
	}
	if wt.ProjectID != project.ID {
		t.Errorf("expected project ID %q, got %q", project.ID, wt.ProjectID)
	}
	if wt.IsMain {
		t.Error("expected non-main worktree")
	}
}

func TestWorktreeService_CreateWorktree_EmptyBranch(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	_, err := svc.CreateWorktree(project.ID, "", "main", true)
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestWorktreeService_CreateWorktree_InvalidBranch(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	_, err := svc.CreateWorktree(project.ID, "..invalid", "main", true)
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestWorktreeService_CreateWorktree_ProjectNotFound(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.CreateWorktree("nonexistent", "branch", "main", true)
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestWorktreeService_ListWorktrees(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)
	_, _ = svc.CreateWorktree(project.ID, "wt-1", "main", true)

	worktrees, err := svc.ListWorktrees(project.ID)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) < 2 {
		t.Fatalf("expected at least 2 worktrees (main + created), got %d", len(worktrees))
	}
}

func TestWorktreeService_ListWorktrees_ProjectNotFound(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.ListWorktrees("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestWorktreeService_RemoveWorktree(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)
	wt, _ := svc.CreateWorktree(project.ID, "to-remove", "main", true)

	err := svc.RemoveWorktree(project.ID, wt.ID, false)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	worktrees, _ := svc.ListWorktrees(project.ID)
	for _, w := range worktrees {
		if w.ID == wt.ID {
			t.Error("expected worktree to be removed")
		}
	}
}

func TestWorktreeService_RemoveWorktree_Main(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	worktrees, _ := svc.ListWorktrees(project.ID)
	var mainID string
	for _, w := range worktrees {
		if w.IsMain {
			mainID = w.ID
			break
		}
	}
	if mainID == "" {
		t.Fatal("expected to find main worktree")
	}

	err := svc.RemoveWorktree(project.ID, mainID, false)
	if err == nil {
		t.Fatal("expected error when removing main worktree")
	}
	if err.Error() != "cannot remove main worktree" {
		t.Errorf("expected 'cannot remove main worktree', got %q", err.Error())
	}
}

func TestWorktreeService_RemoveWorktree_NotFound(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	err := svc.RemoveWorktree(project.ID, "nonexistent-wt", false)
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
}

func TestWorktreeService_RefreshWorktree(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)
	wt, _ := svc.CreateWorktree(project.ID, "refresh-me", "main", true)

	refreshed, err := svc.RefreshWorktree(project.ID, wt.ID)
	if err != nil {
		t.Fatalf("RefreshWorktree: %v", err)
	}
	if refreshed.ID != wt.ID {
		t.Errorf("expected ID %q, got %q", wt.ID, refreshed.ID)
	}
	if refreshed.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set after refresh")
	}
}

func TestWorktreeService_RefreshWorktree_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	_, err := svc.RefreshWorktree("nonexistent-proj", "nonexistent-wt")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree")
	}
}

func TestWorktreeService_SyncAllWorktrees(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)

	worktrees, err := svc.SyncAllWorktrees(project.ID)
	if err != nil {
		t.Fatalf("SyncAllWorktrees: %v", err)
	}
	if len(worktrees) < 1 {
		t.Fatalf("expected at least 1 worktree (main), got %d", len(worktrees))
	}

	var foundMain bool
	for _, w := range worktrees {
		if w.IsMain {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Error("expected to find main worktree after sync")
	}
}

func TestWorktreeService_CommitWorktree(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)

	svc := NewWorktreeService()
	project, _ := svc.AddProject(repoDir)
	wt, _ := svc.CreateWorktree(project.ID, "commit-wt", "main", true)

	// Create a file in the worktree to commit.
	testFile := filepath.Join(wt.Path, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	record, err := svc.CommitWorktree(project.ID, wt.ID, "test commit")
	if err != nil {
		t.Fatalf("CommitWorktree: %v", err)
	}
	if record.HeadCommit == "" {
		t.Error("expected non-empty HeadCommit after commit")
	}
}
