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
	defer os.RemoveAll(wtPath)

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s", wtPath)
	}

	// Verify .git is a file with gitdir prefix.
	gitFile, err := os.ReadFile(filepath.Join(wtPath, ".git"))
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
