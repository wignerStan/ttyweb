package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStressConcurrentMixedOperations(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sem := NewOperationSemaphore(4)
	ctx := context.Background()

	const numWorktrees = 8
	paths := make([]string, numWorktrees)

	// Phase 1: Create all worktrees concurrently.
	var wg sync.WaitGroup
	var createSuccess atomic.Int32

	for i := 0; i < numWorktrees; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			branch := "stress-" + string(rune('A'+idx))
			p, err := CreateWorktree(ctx, repo, branch, "main", true)
			if err != nil {
				t.Errorf("CreateWorktree %d: %v", idx, err)
				return
			}
			paths[idx] = p
			createSuccess.Add(1)
		}(i)
	}
	wg.Wait()

	if got := createSuccess.Load(); got != numWorktrees {
		t.Fatalf("created %d/%d worktrees", got, numWorktrees)
	}

	// Phase 2: Mixed status + commit operations concurrently.
	var opsSuccess atomic.Int64
	const numOps = 50

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			opCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			guard, err := sem.Acquire(opCtx)
			if err != nil {
				return
			}
			defer guard.Release()

			wtIdx := idx % numWorktrees

			switch idx % 3 {
			case 0:
				if _, err := GetWorktreeStatus(opCtx, paths[wtIdx]); err != nil {
					t.Errorf("GetWorktreeStatus %d: %v", idx, err)
					return
				}
			case 1:
				f := filepath.Join(paths[wtIdx], "stress.txt")
				_ = os.WriteFile(f, []byte("data\n"), 0o644)
				if err := CommitWorktree(opCtx, paths[wtIdx], "stress commit"); err != nil {
					if !strings.Contains(err.Error(), "nothing to commit") {
						t.Errorf("CommitWorktree %d: %v", idx, err)
						return
					}
				}
			case 2:
				if _, err := ListWorktrees(opCtx, repo); err != nil {
					t.Errorf("ListWorktrees %d: %v", idx, err)
					return
				}
			}
			opsSuccess.Add(1)
		}(i)
	}
	wg.Wait()

	if got := opsSuccess.Load(); got < int64(numOps)-5 {
		t.Errorf("only %d/%d operations succeeded", got, numOps)
	}

	// Phase 3: Remove all worktrees concurrently.
	var removeSuccess atomic.Int32
	for i := 0; i < numWorktrees; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			opCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			guard, err := sem.Acquire(opCtx)
			if err != nil {
				return
			}
			defer guard.Release()

			if err := RemoveWorktree(opCtx, repo, paths[idx], false); err != nil {
				t.Errorf("RemoveWorktree %d: %v", idx, err)
				return
			}
			removeSuccess.Add(1)
		}(i)
	}
	wg.Wait()

	if got := removeSuccess.Load(); got != numWorktrees {
		t.Errorf("removed %d/%d worktrees", got, numWorktrees)
	}
}
