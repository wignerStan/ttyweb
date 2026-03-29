package worktree

import (
	"context"
	"sync"
)

// RepoLock serializes write operations on the same repository while allowing
// concurrent reads and operations on different repositories.
type RepoLock struct {
	mu    sync.Mutex
	locks map[string]*repoEntry
}

type repoEntry struct {
	mu sync.RWMutex
}

// NewRepoLock creates a new RepoLock.
func NewRepoLock() *RepoLock {
	return &RepoLock{locks: make(map[string]*repoEntry)}
}

// Lock acquires an exclusive (write) lock for the repository at path.
// Returns an unlock function. The caller must invoke it when done.
func (rl *RepoLock) Lock(path string, ctx context.Context) func() {
	entry := rl.getEntry(path)
	entry.mu.Lock()
	return entry.mu.Unlock
}

// LockContext acquires an exclusive lock, respecting context cancellation.
func (rl *RepoLock) LockContext(path string, ctx context.Context) (func(), error) {
	rl.mu.Lock()
	entry := rl.getOrCreateEntry(path)
	rl.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entry.mu.Lock()
	return entry.mu.Unlock, nil
}

// RLock acquires a shared (read) lock for the repository at path.
// Multiple readers can hold RLock simultaneously, but writers block.
// Returns an unlock function. The caller must invoke it when done.
func (rl *RepoLock) RLock(path string, ctx context.Context) func() {
	entry := rl.getEntry(path)
	entry.mu.RLock()
	return entry.mu.RUnlock
}

func (rl *RepoLock) getEntry(path string) *repoEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.getOrCreateEntry(path)
}

func (rl *RepoLock) getOrCreateEntry(path string) *repoEntry {
	if e, ok := rl.locks[path]; ok {
		return e
	}
	e := &repoEntry{}
	rl.locks[path] = e
	return e
}
