package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorktree_Native(t *testing.T) {
	repo := initTestRepo(t)
	wtPath, err := CreateWorktree(context.Background(), repo, "feature-native", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	defer os.RemoveAll(wtPath) //nolint:errcheck // test cleanup

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s", wtPath)
	}

	// Verify .git is a file with gitdir prefix.
	gitFile, err := os.ReadFile(filepath.Join(wtPath, ".git")) //nolint:gosec // test code
	if err != nil {
		t.Fatalf("read .git file: %v", err)
	}
	if !strings.HasPrefix(string(gitFile), "gitdir:") {
		t.Errorf("expected .git file with gitdir prefix, got %q", string(gitFile))
	}

	// Verify files checked out from branch.
	if _, err := os.Stat(filepath.Join(wtPath, "README.md")); err != nil {
		t.Error("expected README.md checked out in worktree")
	}

	// Verify it appears in ListWorktrees.
	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	var found bool
	for _, wt := range worktrees {
		if wt.Branch == "feature-native" {
			found = true
			if wt.IsMain {
				t.Error("feature worktree should not be main")
			}
			if wt.HeadCommit == "" {
				t.Error("worktree should have head commit")
			}
		}
	}
	if !found {
		t.Error("created worktree not found in list")
	}
}

func TestRemoveWorktree_Native(t *testing.T) {
	repo := initTestRepo(t)
	wtPath, err := CreateWorktree(context.Background(), repo, "to-remove", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	if err := RemoveWorktree(context.Background(), repo, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed")
	}
	worktrees, _ := ListWorktrees(context.Background(), repo)
	for _, wt := range worktrees {
		if wt.Branch == "to-remove" {
			t.Error("removed worktree should not appear in list")
		}
	}
}

func TestWorktreeRoundTrip_Native(t *testing.T) {
	repo := initTestRepo(t)
	wtPath, err := CreateWorktree(context.Background(), repo, "roundtrip", "main", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	mustWriteFile(t, filepath.Join(wtPath, "test.txt"), "hello\n")
	if err := CommitWorktree(context.Background(), wtPath, "add test file"); err != nil {
		t.Fatalf("commit: %v", err)
	}
	status, err := GetWorktreeStatus(context.Background(), wtPath)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Modified != 0 || status.Untracked != 0 || status.Staged != 0 {
		t.Errorf("expected clean status, got: %+v", status)
	}
	if err := RemoveWorktree(context.Background(), repo, wtPath, false); err != nil {
		t.Fatalf("remove: %v", err)
	}
}

func TestCreateWorktree_ExistingBranch_Native(t *testing.T) {
	repo := initTestRepo(t)
	mustRun(t, repo, "branch", "pre-existing")
	_, err := CreateWorktree(context.Background(), repo, "pre-existing", "main", true)
	if err == nil {
		t.Fatal("expected error for existing branch with createBranch=true")
	}
	wtPath, err := CreateWorktree(context.Background(), repo, "pre-existing", "main", false)
	if err != nil {
		t.Fatalf("createBranch=false should work: %v", err)
	}
	_ = RemoveWorktree(context.Background(), repo, wtPath, false)
}

func TestAddWorktree_BranchNotFound(t *testing.T) {
	repo := initTestRepo(t)
	wtPath := filepath.Join(repo, ".worktrees", "nobranch")
	err := addWorktree(repo, wtPath, "nonexistent-branch")
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

func TestAddWorktree_Subdirectory(t *testing.T) {
	repo := initTestRepo(t)
	// Create a commit with a file in a subdirectory.
	mustWriteFile(t, filepath.Join(repo, "sub", "dir.txt"), "nested\n")
	mustRun(t, repo, "add", ".")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "add subdirectory")

	wtPath, err := CreateWorktree(context.Background(), repo, "subdir-test", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree with subdirectory files failed: %v", err)
	}
	defer os.RemoveAll(wtPath) //nolint:errcheck // test cleanup

	if _, err := os.Stat(filepath.Join(wtPath, "sub", "dir.txt")); err != nil {
		t.Error("expected subdirectory file to be checked out")
	}
}

func TestListWorktrees_StaleEntry(t *testing.T) {
	repo := initTestRepo(t)

	// Create a worktree.
	wtPath, err := CreateWorktree(context.Background(), repo, "stale-wt", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Manually remove the worktree directory but leave metadata.
	_ = os.RemoveAll(wtPath)

	// ListWorktrees should prune the stale entry automatically.
	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	for _, wt := range worktrees {
		if wt.Branch == "stale-wt" {
			t.Error("stale worktree should have been pruned from list")
		}
	}
}

func TestListWorktrees_NonDirectoryEntry(t *testing.T) {
	repo := initTestRepo(t)

	// Create a non-directory file in .git/worktrees/ to test skip logic.
	wtDir := filepath.Join(repo, ".git", "worktrees")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtDir, "not-a-dir"), []byte("junk"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees with non-dir entry failed: %v", err)
	}
	// Should only have main worktree, not the junk entry.
	for _, wt := range worktrees {
		if wt.Branch == "not-a-dir" {
			t.Error("non-directory entry should be skipped")
		}
	}
}

func TestListWorktrees_MissingGitdir(t *testing.T) {
	repo := initTestRepo(t)

	// Create a metadata directory with no gitdir file.
	wtDir := filepath.Join(repo, ".git", "worktrees", "no-gitdir")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	for _, wt := range worktrees {
		if wt.Branch == "no-gitdir" {
			t.Error("entry without gitdir should be pruned")
		}
	}
}

func TestRemoveWorktree_UncommittedChanges(t *testing.T) {
	repo := initTestRepo(t)

	wtPath, err := CreateWorktree(context.Background(), repo, "dirty-wt", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Write an uncommitted file.
	mustWriteFile(t, filepath.Join(wtPath, "dirty.txt"), "uncommitted\n")

	// Non-force remove should fail.
	err = RemoveWorktree(context.Background(), repo, wtPath, false)
	if err == nil {
		t.Fatal("expected error when removing worktree with uncommitted changes")
	}

	// Force remove should succeed.
	err = RemoveWorktree(context.Background(), repo, wtPath, true)
	if err != nil {
		t.Fatalf("force RemoveWorktree failed: %v", err)
	}
}

func TestRemoveWorktree_AlreadyGone(t *testing.T) {
	repo := initTestRepo(t)

	wtPath, err := CreateWorktree(context.Background(), repo, "gone-wt", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Remove the worktree directory directly.
	_ = os.RemoveAll(wtPath)

	// RemoveWorktree should prune metadata without error.
	err = RemoveWorktree(context.Background(), repo, wtPath, false)
	if err != nil {
		t.Fatalf("RemoveWorktree for already-removed path failed: %v", err)
	}
}

func TestComputeAheadBehind(t *testing.T) {
	repo := initTestRepo(t)

	// Set up a fake origin remote pointing at a separate clone so we can
	// create a divergence (local ahead, remote behind).
	originDir := t.TempDir()
	mustRun(t, originDir, "clone", "file://"+repo, "origin-repo")
	originRepo := filepath.Join(originDir, "origin-repo")

	// Push the initial commit to origin.
	mustRun(t, repo, "remote", "add", "origin", "file://"+originRepo)
	mustRun(t, repo, "push", "origin", "main")

	// Create a second commit on local main so we're ahead of origin/main.
	mustWriteFile(t, filepath.Join(repo, "ahead.txt"), "ahead\n")
	mustRun(t, repo, "add", "ahead.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "ahead commit")

	// Fetch to update origin/main ref.
	mustRun(t, repo, "fetch", "origin")

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ahead, behind := computeAheadBehind(r)
	if ahead != 1 {
		t.Errorf("expected ahead=1, got %d", ahead)
	}
	if behind != 0 {
		t.Errorf("expected behind=0, got %d", behind)
	}
}

func TestComputeAheadBehind_NoTrackingBranch(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ahead, behind := computeAheadBehind(r)
	// No remote tracking branch configured, so (0, 0).
	if ahead != 0 || behind != 0 {
		t.Errorf("expected (0, 0) without tracking branch, got (%d, %d)", ahead, behind)
	}
}

func TestResolveWorktreeHead_Detached(t *testing.T) {
	repo := initTestRepo(t)

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	// Get current HEAD hash for detached HEAD test.
	head, err := r.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}

	// Simulate detached HEAD content.
	branch, hash := resolveWorktreeHead(r, head.Hash().String())
	if branch != "" {
		t.Errorf("expected empty branch for detached HEAD, got %q", branch)
	}
	if hash != head.Hash() {
		t.Errorf("expected hash %s, got %s", head.Hash(), hash)
	}
}

func TestMainWorktreeInfo_Detached(t *testing.T) {
	repo := initTestRepo(t)

	// Detach HEAD.
	mustRun(t, repo, "checkout", "--detach", "HEAD")

	info := mainWorktreeInfo(repo)
	if !info.IsMain {
		t.Error("main worktree should have IsMain=true")
	}
	if info.Branch != "" {
		t.Errorf("expected empty branch for detached HEAD, got %q", info.Branch)
	}
	if info.HeadCommit == "" {
		t.Error("main worktree should still have a head commit")
	}
}

func TestCreateWorktree_BaseBranchNotFound(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CreateWorktree(context.Background(), repo, "no-base", "nonexistent-base", true)
	if err == nil {
		t.Fatal("expected error when base branch does not exist")
	}
}

func TestCreateWorktree_WithStagedChanges(t *testing.T) {
	repo := initTestRepo(t)

	// Stage a file and commit.
	mustWriteFile(t, filepath.Join(repo, "staged.txt"), "content\n")
	mustRun(t, repo, "add", "staged.txt")
	if err := CommitWorktree(context.Background(), repo, "stage and commit"); err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// Create a worktree and verify it sees the commit.
	wtPath, err := CreateWorktree(context.Background(), repo, "after-staged", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	defer os.RemoveAll(wtPath) //nolint:errcheck // test cleanup

	if _, err := os.Stat(filepath.Join(wtPath, "staged.txt")); err != nil {
		t.Error("worktree should contain committed file")
	}
}

func TestCollectStatusGoGit_WithStagedAndModified(t *testing.T) {
	repo := initTestRepo(t)

	// Stage a file.
	mustWriteFile(t, filepath.Join(repo, "staged.txt"), "staged\n")
	mustRun(t, repo, "add", "staged.txt")

	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Staged < 1 {
		t.Errorf("expected staged file, got staged=%d", status.Staged)
	}

	// Modify the staged file (now both staged and modified).
	if err := os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	status, err = collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit after modify failed: %v", err)
	}
	if status.Modified < 1 {
		t.Errorf("expected modified file, got modified=%d", status.Modified)
	}
}

func TestComputeAheadBehind_Diverged(t *testing.T) {
	repo := initTestRepo(t)

	// Create a bare clone as origin.
	originDir := t.TempDir()
	mustRun(t, originDir, "clone", "--bare", "file://"+repo, "origin-bare")
	originPath := filepath.Join(originDir, "origin-bare")

	// Push main to origin.
	mustRun(t, repo, "remote", "add", "origin", "file://"+originPath)
	mustRun(t, repo, "push", "origin", "main")

	// Create a commit on local main (ahead).
	mustWriteFile(t, filepath.Join(repo, "local.txt"), "local\n")
	mustRun(t, repo, "add", "local.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "local commit")

	// Now simulate origin/main being ahead by pushing a different commit from the clone.
	cloneDir := t.TempDir()
	mustRun(t, cloneDir, "clone", "file://"+originPath, "clone-repo")
	cloneRepo := filepath.Join(cloneDir, "clone-repo")
	mustWriteFile(t, filepath.Join(cloneRepo, "remote.txt"), "remote\n")
	mustRun(t, cloneRepo, "add", "remote.txt")
	mustRun(t, cloneRepo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "remote commit")
	mustRun(t, cloneRepo, "push", "origin", "main")

	// Fetch to update origin/main in our repo.
	mustRun(t, repo, "fetch", "origin")

	r, err := defaultCache.Open(repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ahead, behind := computeAheadBehind(r)
	if ahead < 1 {
		t.Errorf("expected ahead >= 1, got %d", ahead)
	}
	if behind < 1 {
		t.Errorf("expected behind >= 1, got %d", behind)
	}
}

func TestCheckoutTree_WithSymlink(t *testing.T) {
	repo := initTestRepo(t)

	// Create a symlink in the repo.
	mustRun(t, repo, "config", "core.symlinks", "true")
	_ = os.Symlink("README.md", filepath.Join(repo, "link.md"))
	mustRun(t, repo, "add", "link.md")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "add symlink")

	wtPath, err := CreateWorktree(context.Background(), repo, "symlink-test", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	defer os.RemoveAll(wtPath) //nolint:errcheck // test cleanup

	// Verify the symlink was checked out.
	target, err := os.Readlink(filepath.Join(wtPath, "link.md"))
	if err != nil {
		t.Fatalf("readlink failed: %v", err)
	}
	if target != "README.md" {
		t.Errorf("expected symlink target 'README.md', got %q", target)
	}
}
