package worktree

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRepoLock_MutualExclusion(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/main"

	var order []string
	var mu sync.Mutex

	done := make(chan string, 2)

	go func() {
		unlock := rl.Lock(context.Background(), path)
		mu.Lock()
		order = append(order, "writer1-start")
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "writer1-end")
		mu.Unlock()
		unlock()
		done <- "w1"
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		unlock := rl.Lock(context.Background(), path)
		mu.Lock()
		order = append(order, "writer2-start")
		mu.Unlock()
		mu.Lock()
		order = append(order, "writer2-end")
		mu.Unlock()
		unlock()
		done <- "w2"
	}()

	<-done
	<-done

	var w1End, w2Start int
	for i, s := range order {
		if s == "writer1-end" {
			w1End = i
		}
		if s == "writer2-start" {
			w2Start = i
		}
	}
	if w2Start < w1End {
		t.Errorf("writer2 started before writer1 finished: %v", order)
	}
}

func TestRepoLock_DifferentRepos(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()

	ctx := context.Background()
	unlock1 := rl.Lock(ctx, "/repo/a")
	unlock2 := rl.Lock(ctx, "/repo/b")

	unlock1()
	unlock2()
}

func TestRepoLock_LockRespectsContext(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/locked"

	unlock := rl.Lock(context.Background(), path)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		unlockFn := rl.Lock(ctx, path)
		if unlockFn != nil {
			t.Error("expected nil unlock from cancelled context")
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Lock blocked despite cancelled context")
	}
	unlock()
}

func TestRepoLock_Remove(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/test"

	unlock := rl.Lock(context.Background(), path)
	unlock()

	rl.Remove(path)

	unlock2 := rl.Lock(context.Background(), path)
	unlock2()

	if len(rl.locks) != 1 {
		t.Errorf("expected 1 entry after Remove+re-Lock, got %d", len(rl.locks))
	}
}

func TestRepoLock_RLockRespectsContext(t *testing.T) {
	t.Parallel()
	rl := NewRepoLock()
	ctx, cancel := context.WithCancel(context.Background())

	// Acquire write lock to block readers.
	unlock := rl.Lock(context.Background(), "/test/path")
	defer unlock()

	// Cancel context before RLock can succeed.
	cancel()

	unlockRL := rl.RLock(ctx, "/test/path")
	if unlockRL != nil {
		t.Error("expected nil unlock func when context is cancelled")
	}
}

func TestRepoLock_RLockConcurrency(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/main"

	ctx := context.Background()
	unlock1 := rl.RLock(ctx, path)
	unlock2 := rl.RLock(ctx, path)

	unlock1()
	unlock2()
}
