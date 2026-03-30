package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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

func TestHeadCommitMessage(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	msg := headCommitMessage(context.Background(), repo)
	if msg == "" {
		t.Error("expected non-empty commit message for repo with commits")
	}
}

func TestHeadCommitMessage_EmptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Init a bare repo with no commits.
	mustRun(t, dir, "init", "--bare")
	msg := headCommitMessage(context.Background(), dir)
	if msg != "" {
		t.Errorf("expected empty message for bare repo with no commits, got %q", msg)
	}
}

func TestKindError_Error(t *testing.T) {
	t.Parallel()
	if ErrNotFound.Error() != "not_found" {
		t.Errorf("expected 'not_found', got %q", ErrNotFound.Error())
	}
	if ErrConflict.Error() != "conflict" {
		t.Errorf("expected 'conflict', got %q", ErrConflict.Error())
	}
	if ErrWorktreeLocked.Error() != "worktree_locked" {
		t.Errorf("expected 'worktree_locked', got %q", ErrWorktreeLocked.Error())
	}
}

func TestEqualPathSame(t *testing.T) {
	t.Parallel()
	if !EqualPath("/tmp/a", "/tmp/a") {
		t.Error("EqualPath should return true for identical paths")
	}
}

func TestEqualPathTrailingSlash(t *testing.T) {
	t.Parallel()
	if !EqualPath("/tmp/a/", "/tmp/a") {
		t.Error("EqualPath should handle trailing slashes")
	}
}

func TestEqualPathDifferent(t *testing.T) {
	t.Parallel()
	if EqualPath("/tmp/a", "/tmp/b") {
		t.Error("EqualPath should return false for different paths")
	}
}
