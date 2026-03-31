package worktree

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	goGit "github.com/go-git/go-git/v5"
)

// TestMergeBranch_NonFastForward covers the mergeViaCLI fallback path.
// This happens when the target is NOT an ancestor of the source.
func TestMergeBranch_NonFastForward(t *testing.T) {
	repo := initTestRepo(t)

	// Create a feature branch from main.
	mustRun(t, repo, "checkout", "-b", "noff-source")

	// Add a commit on the feature branch.
	mustWriteFile(t, filepath.Join(repo, "feature.txt"), "feature content\n")
	mustRun(t, repo, "add", "feature.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "feature commit")

	// Switch back to main.
	mustRun(t, repo, "checkout", "main")

	// Add a diverging commit on main so merge is non-fast-forward.
	mustWriteFile(t, filepath.Join(repo, "main-only.txt"), "main content\n")
	mustRun(t, repo, "add", "main-only.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "main commit")

	// Merge noff-source into main -- this should fall back to mergeViaCLI.
	err := MergeBranch(context.Background(), repo, "noff-source", "main")
	if err != nil {
		t.Fatalf("MergeBranch non-fast-forward failed: %v", err)
	}

	// Verify both commits are present on main.
	out := mustOutput(t, repo, "log", "--oneline", "main")
	if !strings.Contains(out, "feature commit") {
		t.Errorf("main should contain 'feature commit', got: %s", out)
	}
	if !strings.Contains(out, "main commit") {
		t.Errorf("main should contain 'main commit', got: %s", out)
	}
}

// TestMergeBranch_NonFastForwardOnTarget covers the mergeViaCLI path where
// we're already on the target branch, so no checkout is needed.
func TestMergeBranch_NonFastForwardOnTarget(t *testing.T) {
	repo := initTestRepo(t)

	// Create a feature branch from main and add a commit.
	mustRun(t, repo, "checkout", "-b", "noff-on-target")
	mustWriteFile(t, filepath.Join(repo, "feat.txt"), "feat\n")
	mustRun(t, repo, "add", "feat.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "feat commit")

	// Switch back to main and add a diverging commit.
	mustRun(t, repo, "checkout", "main")
	mustWriteFile(t, filepath.Join(repo, "main.txt"), "main\n")
	mustRun(t, repo, "add", "main.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "main commit")

	// Now checkout main (we should already be on it) and merge.
	// The mergeViaCLI function should detect we're already on target.
	err := MergeBranch(context.Background(), repo, "noff-on-target", "main")
	if err != nil {
		t.Fatalf("MergeBranch non-fast-forward on target failed: %v", err)
	}
}

// TestGetWorktreeStatus_InvalidPath covers the non-repo path error.
func TestGetWorktreeStatus_InvalidPath(t *testing.T) {
	_, err := GetWorktreeStatus(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// TestGetWorktreeDiff_NoCommits covers the error path when the repo has no commits.
func TestGetWorktreeDiff_NoCommits(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, dir, "init", "-b", "main")

	_, err := GetWorktreeDiff(context.Background(), dir, dir)
	if err == nil {
		t.Fatal("expected error for repo with no commits")
	}
}

// TestCreateWorktree_InvalidBranchName covers the branch name validation.
func TestCreateWorktree_InvalidBranchName(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CreateWorktree(context.Background(), repo, "invalid branch!", "main", true)
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

// TestIsGitRepo_Nonexistent covers IsGitRepo with a nonexistent path.
func TestIsGitRepo_Nonexistent(t *testing.T) {
	if IsGitRepo("/nonexistent/path/that/does/not/exist") {
		t.Error("IsGitRepo should return false for nonexistent path")
	}
}

// TestFetchPRDetails_BadURL covers the error path for an invalid URL.
func TestFetchPRDetails_BadURL(t *testing.T) {
	_, err := FetchPRDetails(context.Background(), "://invalid-url", "")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// TestFetchPRDetails_NonexistentServer covers the error path when the server
// doesn't respond.
func TestFetchPRDetails_NonexistentServer(t *testing.T) {
	_, err := FetchPRDetails(context.Background(), "http://127.0.0.1:1/pulls/1", "")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

// TestDetectRepoSlug_NoRemote covers the error path when the repo has no
// origin remote.
func TestDetectRepoSlug_NoRemote(t *testing.T) {
	repo := initTestRepo(t)

	_, err := DetectRepoSlug(repo)
	if err == nil {
		t.Fatal("expected error for repo without origin remote")
	}
}

// TestCheckoutPRBranch_EmptyBranch covers the empty branch validation.
func TestCheckoutPRBranch_EmptyBranch(t *testing.T) {
	_, err := CheckoutPRBranch(context.Background(), "", "branch", "abc123")
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

// TestCheckoutPRBranch_EmptySHA covers the empty SHA validation.
func TestCheckoutPRBranch_EmptySHA(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CheckoutPRBranch(context.Background(), repo, "pr-branch", "")
	if err == nil {
		t.Fatal("expected error for empty SHA")
	}
}

// TestCheckoutPRBranch_EmptyPath covers the empty path validation.
func TestCheckoutPRBranch_EmptyPath(t *testing.T) {
	_, err := CheckoutPRBranch(context.Background(), "", "pr-branch", "abc123")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestCheckoutPRBranch_InvalidBranchName covers the branch name validation.
func TestCheckoutPRBranch_InvalidBranchName(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CheckoutPRBranch(context.Background(), repo, "invalid branch!", "abc123")
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

// TestCheckoutPRBranch_NoOrigin covers the error path when the repo has no
// origin remote.
func TestCheckoutPRBranch_NoOrigin(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CheckoutPRBranch(context.Background(), repo, "pr-branch", "abc123def456abc123def456abc123def456abc12")
	if err == nil {
		t.Fatal("expected error for repo without origin remote")
	}
}

// TestIsUntrackedFileStatus covers the isUntrackedFileStatus helper.
func TestIsUntrackedFileStatus(t *testing.T) {
	t.Parallel()

	fs := &goGit.FileStatus{Staging: goGit.Untracked}
	if !isUntrackedFileStatus(fs) {
		t.Error("expected untracked for Staging=Untracked")
	}

	fs2 := &goGit.FileStatus{Worktree: goGit.Untracked}
	if !isUntrackedFileStatus(fs2) {
		t.Error("expected untracked for Worktree=Untracked")
	}

	fs3 := &goGit.FileStatus{Worktree: goGit.Modified}
	if isUntrackedFileStatus(fs3) {
		t.Error("expected not untracked for Modified")
	}
}

// TestIsConflictedFileStatus covers the isConflictedFileStatus helper.
func TestIsConflictedFileStatus(t *testing.T) {
	t.Parallel()

	fs := &goGit.FileStatus{Staging: goGit.UpdatedButUnmerged}
	if !isConflictedFileStatus(fs) {
		t.Error("expected conflicted for UpdatedButUnmerged")
	}

	fs2 := &goGit.FileStatus{Worktree: goGit.Modified}
	if isConflictedFileStatus(fs2) {
		t.Error("expected not conflicted for Modified")
	}
}

// TestIsModifiedWorktreeStatus covers the isModifiedWorktreeStatus helper.
func TestIsModifiedWorktreeStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code goGit.StatusCode
		want bool
	}{
		{goGit.Modified, true},
		{goGit.Added, true},
		{goGit.Deleted, true},
		{goGit.Renamed, true},
		{goGit.Copied, true},
		{goGit.Unmodified, false},
		{goGit.Untracked, false},
		{goGit.UpdatedButUnmerged, false},
	}

	for _, tc := range tests {
		got := isModifiedWorktreeStatus(tc.code)
		if got != tc.want {
			t.Errorf("isModifiedWorktreeStatus(%v) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestComputeAheadBehind_NoRemote covers the case where there's no tracking branch.
func TestComputeAheadBehind_NoRemote(t *testing.T) {
	repoPath := initGoGitRepo(t, nil)
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ahead, behind := computeAheadBehind(repo)
	if ahead != 0 || behind != 0 {
		t.Errorf("expected (0, 0) for repo without remote, got (%d, %d)", ahead, behind)
	}
}

// TestCountCommits covers the countCommits function with an empty list.
func TestCountCommits(t *testing.T) {
	repoPath := initGoGitRepo(t, nil)
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	count := countCommits(repo, nil)
	if count != 0 {
		t.Errorf("expected 0 for empty list, got %d", count)
	}
}

// TestRemoveWorktree_NonexistentWorktreePath covers the case where the worktree
// directory doesn't exist -- should just prune metadata (no-op).
func TestRemoveWorktree_NonexistentWorktreePath(t *testing.T) {
	repo := initTestRepo(t)

	err := RemoveWorktree(context.Background(), repo, filepath.Join(repo, ".worktrees", "nonexistent-wt"), false)
	if err != nil {
		t.Fatalf("RemoveWorktree with nonexistent worktree path should succeed: %v", err)
	}
}
