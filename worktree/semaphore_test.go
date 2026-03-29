package worktree

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOperationSemaphore_BoundedConcurrency(t *testing.T) {
	t.Parallel()

	sem := NewOperationSemaphore(2)
	var running atomic.Int32
	var maxRunning atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			guard, err := sem.Acquire(ctx)
			if err != nil {
				t.Errorf("Acquire failed: %v", err)
				return
			}
			defer guard.Release()

			cur := running.Add(1)
			for {
				old := maxRunning.Load()
				if cur <= old || maxRunning.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
		}()
	}

	wg.Wait()

	if got := maxRunning.Load(); got > 2 {
		t.Errorf("max concurrent operations = %d, want <= 2", got)
	}
}

func TestOperationSemaphore_Cancel(t *testing.T) {
	t.Parallel()

	sem := NewOperationSemaphore(1)

	guard1, err := sem.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer guard1.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = sem.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestGuard_DoubleReleaseSafe(t *testing.T) {
	t.Parallel()

	sem := NewOperationSemaphore(1)

	guard, err := sem.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	guard.Release()
	done := make(chan struct{})
	go func() {
		guard.Release()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second Release() blocked (deadlock)")
	}
}

func TestGuard_NilGuardReleaseSafe(t *testing.T) {
	t.Parallel()

	var g Guard
	g.Release()
}

func TestDefaultSemaphore(t *testing.T) {
	t.Parallel()

	sem := DefaultSemaphore()
	if sem == nil {
		t.Fatal("DefaultSemaphore returned nil")
	}

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		guard, err := sem.Acquire(ctx)
		cancel()
		if err != nil {
			t.Fatalf("DefaultSemaphore Acquire %d: %v", i, err)
		}
		guard.Release()
	}
}
