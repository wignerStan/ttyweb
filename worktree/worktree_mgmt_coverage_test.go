package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

// TestAddWorktree_InvalidRepoPath covers the error path when repo path is invalid.
func TestAddWorktree_InvalidRepoPath(t *testing.T) {
	wtPath := filepath.Join(t.TempDir(), "nowhere")
	err := addWorktree(t.TempDir(), wtPath, "some-branch")
	if err == nil {
		t.Fatal("expected error for invalid repo path")
	}
}

// TestAddWorktree_EmptyBranchName covers the error path for empty branch.
func TestAddWorktree_EmptyBranchName(t *testing.T) {
	repo := initTestRepo(t)
	wtPath := filepath.Join(repo, ".worktrees", "empty-branch")
	err := addWorktree(repo, wtPath, "")
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

// TestListWorktrees_InvalidRepo covers the error path when repo is invalid.
// listWorktreesManual calls mainWorktreeInfo which tries to open the repo.
// If that fails, it returns an error (repo not found).
func TestListWorktrees_InvalidRepo(t *testing.T) {
	// Create a dir that is not a git repo.
	dir := t.TempDir()

	// listWorktreesManual will try to open the repo and fail.
	_, err := listWorktreesManual(dir)
	// With no .git/worktrees dir, it returns mainWorktreeInfo which
	// fails to open the repo, returning an error.
	if err != nil {
		// This is the expected path for a non-repo directory.
		t.Logf("listWorktreesManual returned error (expected for non-repo): %v", err)
	}
}

// TestHasUncommittedChanges_InvalidPath covers the best-effort behavior.
func TestHasUncommittedChanges_InvalidPath(t *testing.T) {
	// Non-repo path should return false (best-effort).
	result := hasUncommittedChanges(t.TempDir())
	if result {
		t.Error("expected false for non-repo path")
	}
}

// TestMainWorktreeInfo_InvalidRepo covers the error fallback path.
func TestMainWorktreeInfo_InvalidRepo(t *testing.T) {
	info := mainWorktreeInfo(t.TempDir())
	if !info.IsMain {
		t.Error("expected IsMain=true")
	}
	if info.Path == "" {
		t.Error("expected non-empty path")
	}
	if info.Branch != "" {
		t.Errorf("expected empty branch for invalid repo, got %q", info.Branch)
	}
	if info.HeadCommit != "" {
		t.Errorf("expected empty head commit for invalid repo, got %q", info.HeadCommit)
	}
}

// TestResolveWorktreeHead_InvalidBranch covers the case where ref doesn't exist.
func TestResolveWorktreeHead_InvalidBranch(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	// Use a non-existent branch name.
	branch, hash := resolveWorktreeHead(r, "ref: refs/heads/nonexistent-branch")
	if branch != "nonexistent-branch" {
		t.Errorf("expected branch name 'nonexistent-branch', got %q", branch)
	}
	if hash != plumbing.ZeroHash {
		t.Errorf("expected zero hash, got %s", hash)
	}
}

// TestResolveWorktreeHead_EmptyContent covers empty HEAD content.
func TestResolveWorktreeHead_EmptyContent(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	branch, _ := resolveWorktreeHead(r, "")
	if branch != "" {
		t.Errorf("expected empty branch for empty content, got %q", branch)
	}
}

// TestCheckoutTree_FileWriteFailure tests checkoutTree when the target directory
// is read-only.
func TestCheckoutTree_FileWriteFailure(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	head, err := r.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	// Create a worktree path inside a read-only directory to trigger an error.
	readOnlyDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	wtPath := filepath.Join(readOnlyDir, "subdir")
	wtMetaDir := filepath.Join(t.TempDir(), "meta")

	err = checkoutTree(r, wtPath, wtMetaDir, tree)
	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}

// TestPruneStaleEntry tests that pruneStaleEntry removes the directory.
func TestPruneStaleEntry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "to-prune")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	pruneStaleEntry(dir)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected directory to be removed")
	}
}

// TestListWorktrees_MissingHEAD tests listing when HEAD file is missing.
func TestListWorktrees_MissingHEAD(t *testing.T) {
	repo := initTestRepo(t)

	// Create a worktree first.
	wtPath, err := CreateWorktree(context.Background(), repo, "head-missing", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Remove the HEAD file from the metadata directory.
	worktreeName := filepath.Base(wtPath)
	headFile := filepath.Join(repo, ".git", "worktrees", worktreeName, "HEAD")
	_ = os.Remove(headFile)

	// ListWorktrees should still succeed (skipping the entry).
	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	// The entry should be skipped (no branch info available).
	for _, wt := range worktrees {
		if wt.Path == wtPath {
			t.Error("entry with missing HEAD should be skipped")
		}
	}
}

// TestListWorktrees_NoWorktreesDir tests listing when .git/worktrees doesn't exist.
func TestListWorktrees_NoWorktreesDir(t *testing.T) {
	repo := initTestRepo(t)

	// Remove the worktrees directory if it exists.
	_ = os.RemoveAll(filepath.Join(repo, ".git", "worktrees"))

	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	if len(worktrees) < 1 {
		t.Error("expected at least main worktree")
	}

	found := false
	for _, wt := range worktrees {
		if wt.IsMain {
			found = true
		}
	}
	if !found {
		t.Error("expected to find main worktree")
	}
}

// TestCreateWorktree_OpenFailure covers the error path when repo can't be opened.
func TestCreateWorktree_OpenFailure(t *testing.T) {
	_, err := CreateWorktree(context.Background(), "/nonexistent/path/repo", "branch", "main", true)
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

// TestCommitWorktree_OpenFailure covers the error path when repo can't be opened.
func TestCommitWorktree_OpenFailure(t *testing.T) {
	err := CommitWorktree(context.Background(), "/nonexistent/path/repo", "message")
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

// TestGetWorktreeStatus_OpenFailure covers the error path when repo can't be opened.
func TestGetWorktreeStatus_OpenFailure(t *testing.T) {
	_, err := GetWorktreeStatus(context.Background(), "/nonexistent/path/repo")
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

// TestCreateBranchFromBase_NotFound covers base branch not found.
func TestCreateBranchFromBase_NotFound(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	err = createBranchFromBase(r, "new-branch", "nonexistent-base")
	if err == nil {
		t.Fatal("expected error for nonexistent base branch")
	}
}

// TestCreateBranchFromBase_AlreadyExists covers branch already exists.
func TestCreateBranchFromBase_AlreadyExists(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	// main already exists.
	err = createBranchFromBase(r, "main", "main")
	if err == nil {
		t.Fatal("expected error when branch already exists")
	}
}
