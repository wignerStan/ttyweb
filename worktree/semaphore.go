package worktree

import (
	"context"
)

// defaultMaxOps is the maximum number of concurrent git CLI operations.
// Chosen empirically: 4 balances parallelism vs mmap contention on shared
// .git/objects files (inspired by worktrunk's HEAVY_OPS_SEMAPHORE).
const defaultMaxOps = 4

// OperationSemaphore bounds concurrent git operations to prevent thrashing
// on shared git internal files (commit-graph, pack files).
type OperationSemaphore struct {
	sem chan struct{}
}

// NewOperationSemaphore creates a semaphore that allows at most n concurrent operations.
func NewOperationSemaphore(n int) *OperationSemaphore {
	return &OperationSemaphore{sem: make(chan struct{}, n)}
}

// Acquire blocks until a permit is available or ctx is done.
// The returned Guard must be released when the operation completes.
func (s *OperationSemaphore) Acquire(ctx context.Context) (Guard, error) {
	select {
	case s.sem <- struct{}{}:
		return Guard{release: func() { <-s.sem }}, nil
	case <-ctx.Done():
		return Guard{}, ctx.Err()
	}
}

// Guard represents a held semaphore permit. Call Release to return it.
type Guard struct {
	release func()
}

// Release returns the permit to the semaphore.
func (g Guard) Release() {
	if g.release != nil {
		g.release()
	}
}

// DefaultSemaphore returns the global shared operation semaphore.
func DefaultSemaphore() *OperationSemaphore {
	return defaultSem
}

var defaultSem = NewOperationSemaphore(defaultMaxOps)
