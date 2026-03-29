package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorktreeService_ConcurrentCreateDifferentProjects(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()

	repoDir1 := t.TempDir()
	initGitRepo(t, repoDir1)
	repoDir2 := t.TempDir()
	initGitRepo(t, repoDir2)

	proj1, _ := svc.AddProject(repoDir1)
	proj2, _ := svc.AddProject(repoDir2)

	var wg sync.WaitGroup
	var errs [2]error

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, errs[0] = svc.CreateWorktree(context.Background(), proj1.ID, "branch-a", "main", true)
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = svc.CreateWorktree(context.Background(), proj2.ID, "branch-b", "main", true)
	}()
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("project %d CreateWorktree: %v", i+1, err)
		}
	}
}

func TestWorktreeService_ConcurrentCreateSameProject(t *testing.T) {
	t.Parallel()

	svc := NewWorktreeService()
	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	proj, _ := svc.AddProject(repoDir)

	var wg sync.WaitGroup
	var success atomic.Int32

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			branch := "serialized-" + string(rune('A'+idx))
			_, err := svc.CreateWorktree(context.Background(), proj.ID, branch, "main", true)
			if err != nil {
				t.Errorf("CreateWorktree %d: %v", idx, err)
				return
			}
			success.Add(1)
		}(i)
	}
	wg.Wait()

	if got := success.Load(); got != 5 {
		t.Errorf("created %d/5 worktrees", got)
	}
}

func TestWorktreeService_CreateWorktree_EmptyBranch(t *testing.T) {
	svc := NewWorktreeService()
	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)
	proj, _ := svc.AddProject(repoDir)

	_, err := svc.CreateWorktree(context.Background(), proj.ID, "", "main", true)
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestWorktreeService_CreateWorktree_InvalidBranch(t *testing.T) {
	svc := NewWorktreeService()
	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)
	proj, _ := svc.AddProject(repoDir)

	_, err := svc.CreateWorktree(context.Background(), proj.ID, "invalid branch!", "main", true)
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestWorktreeService_CreateWorktree_NonexistentProject(t *testing.T) {
	svc := NewWorktreeService()

	_, err := svc.CreateWorktree(context.Background(), "nonexistent", "feature-x", "main", true)
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestWorktreeService_RemoveWorktree_MainBranch(t *testing.T) {
	svc := NewWorktreeService()
	repoDir := t.TempDir()
	initTestGitRepo(t, repoDir)
	proj, _ := svc.AddProject(repoDir)

	// Find the main worktree (auto-created by sync).
	wts, err := svc.ListWorktrees(context.Background(), proj.ID)
	if err != nil {
		t.Fatalf("list worktrees: %v", err)
	}
	if len(wts) == 0 {
		t.Fatal("expected at least one worktree (main)")
	}

	mainWT := wts[0]
	err = svc.RemoveWorktree(context.Background(), proj.ID, mainWT.ID, false)
	if err == nil {
		t.Fatal("expected error when removing main worktree")
	}
}
