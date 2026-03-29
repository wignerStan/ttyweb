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
		unlock := rl.Lock(path, context.Background())
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
		unlock := rl.Lock(path, context.Background())
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
	unlock1 := rl.Lock("/repo/a", ctx)
	unlock2 := rl.Lock("/repo/b", ctx)

	unlock1()
	unlock2()
}

func TestRepoLock_LockRespectsContext(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/locked"

	unlock := rl.Lock(path, context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		unlockFn := rl.Lock(path, ctx)
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

func TestRepoLock_RLockConcurrency(t *testing.T) {
	t.Parallel()

	rl := NewRepoLock()
	path := "/repo/main"

	ctx := context.Background()
	unlock1 := rl.RLock(path, ctx)
	unlock2 := rl.RLock(path, ctx)

	unlock1()
	unlock2()
}
