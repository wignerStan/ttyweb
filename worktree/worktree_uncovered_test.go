package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSetTestEnv(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	custom := []string{"GIT_AUTHOR_NAME=test-bot"}
	SetTestEnv(custom)
	defer SetTestEnv(nil)

	// Verify the env override is applied by creating a git command in a repo
	// and checking that the env is appended. We verify via newGitCmd indirectly.
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	env := cmd.Env
	found := false
	for _, e := range env {
		if e == "GIT_AUTHOR_NAME=test-bot" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected custom env var in git command")
	}
}

func TestSetTestEnv_Nil(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	SetTestEnv(nil)
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	// Should still have standard env vars.
	found := false
	for _, e := range cmd.Env {
		if e == "GIT_TERMINAL_PROMPT=0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected standard env var in git command")
	}
}

func TestSetTestEnv_Empty(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	SetTestEnv([]string{})
	defer SetTestEnv(nil)
	// Empty slice should not be appended (len check > 0).
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	// Standard env should still be present.
	found := false
	for _, e := range cmd.Env {
		if e == "GIT_TERMINAL_PROMPT=0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected standard env var in git command")
	}
}

func TestResolveDefaultBranch_Main(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	branch := resolveDefaultBranch(context.Background(), repo)
	if branch != "main" {
		t.Fatalf("expected 'main', got %q", branch)
	}
}

func TestResolveDefaultBranch_NoBranch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	branch := resolveDefaultBranch(context.Background(), dir)
	if branch != "" {
		t.Fatalf("expected empty string for non-repo, got %q", branch)
	}
}

func TestCollectStatusGoGit(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Modified != 0 || status.Staged != 0 || status.Untracked != 0 {
		t.Errorf("clean repo should have zero status, got: %+v", status)
	}
}

func TestCollectStatusGoGit_WithUntracked(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	mustWriteFile(t, filepath.Join(repo, "untracked.txt"), "new\n")
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Untracked < 1 {
		t.Errorf("expected untracked file, got untracked=%d", status.Untracked)
	}
}

func TestCollectStatusGoGit_WithModification(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("# modified\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Modified < 1 {
		t.Errorf("expected modified file, got modified=%d", status.Modified)
	}
}

func TestCollectStatusGoGit_InvalidPath(t *testing.T) {
	t.Parallel()
	_, err := collectStatusGoGit("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestNewGitCmd_EmptyDir(t *testing.T) {
	t.Parallel()
	cmd := newGitCmd("", "status")
	if cmd.Dir != "" {
		t.Fatalf("expected empty dir, got %q", cmd.Dir)
	}
	if cmd.Path == "" {
		t.Fatal("expected non-empty command path")
	}
}
