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

func TestConcurrentCreateWorktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	sem := NewOperationSemaphore(2)

	var wg sync.WaitGroup
	var success atomic.Int32
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			guard, err := sem.Acquire(ctx)
			if err != nil {
				errs[idx] = err
				return
			}
			defer guard.Release()

			branch := "parallel-branch-" + string(rune('A'+idx)) //nolint:gosec // reason: test-only int→rune conversion, values are bounded
			_, err = CreateWorktree(ctx, repo, branch, "main", true)
			if err != nil {
				errs[idx] = err
				return
			}
			success.Add(1)
		}(i)
	}

	wg.Wait()

	if got := success.Load(); got != 5 {
		t.Errorf("created %d/5 worktrees", got)
		for i, e := range errs {
			if e != nil {
				t.Errorf("  branch %d error: %v", i, e)
			}
		}
	}

	infos, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(infos) < 6 {
		t.Errorf("expected >= 6 worktrees, got %d", len(infos))
	}
}

func TestConcurrentRemoveWorktree(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)

	paths := make([]string, 5)
	for i := 0; i < 5; i++ {
		branch := "remove-parallel-" + string(rune('A'+i))
		p, err := CreateWorktree(context.Background(), repo, branch, "main", true)
		if err != nil {
			t.Fatalf("CreateWorktree %d: %v", i, err)
		}
		paths[i] = p
	}

	sem := NewOperationSemaphore(2)
	var wg sync.WaitGroup
	var success atomic.Int32

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			guard, err := sem.Acquire(ctx)
			if err != nil {
				return
			}
			defer guard.Release()

			if err := RemoveWorktree(ctx, repo, paths[idx], false); err != nil {
				t.Errorf("RemoveWorktree %d: %v", idx, err)
				return
			}
			success.Add(1)
		}(i)
	}

	wg.Wait()

	if got := success.Load(); got != 5 {
		t.Errorf("removed %d/5 worktrees", got)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("worktree path still exists: %s", p)
		}
	}
}

func TestConcurrentStatusAndCommit(t *testing.T) {
	t.Parallel()

	repo := initTestRepo(t)
	wtPath, err := CreateWorktree(context.Background(), repo, "status-commit-race", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	sem := NewOperationSemaphore(2)
	var wg sync.WaitGroup
	var statusErrors, commitErrors atomic.Int32
	var wtMu sync.Mutex // serializes writes to the same worktree

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			guard, err := sem.Acquire(ctx)
			if err != nil {
				return
			}
			defer guard.Release()

			if idx%2 == 0 {
				testFile := filepath.Join(wtPath, "race-"+string(rune('0'+idx))+".txt") //nolint:gosec // reason: test-only int→rune conversion, values are bounded
				_ = os.WriteFile(testFile, []byte("data\n"), 0o644)
				wtMu.Lock()
				err := CommitWorktree(ctx, wtPath, "race commit")
				wtMu.Unlock()
				if err != nil && !strings.Contains(err.Error(), "nothing to commit") {
					commitErrors.Add(1)
				}
			} else {
				if _, err := GetWorktreeStatus(ctx, wtPath); err != nil {
					statusErrors.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	if got := statusErrors.Load(); got != 0 {
		t.Errorf("status errors: %d", got)
	}
}
