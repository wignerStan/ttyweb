package worktree

import (
	"context"
	"sync"
	"time"
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

// Remove removes the lock entry for a path, freeing the map entry.
// The caller must ensure no lock is held for the path when calling Remove.
func (rl *RepoLock) Remove(path string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.locks, path)
}

// Lock acquires an exclusive (write) lock for the repository at path.
// Returns an unlock function, or nil if ctx is cancelled before the lock is acquired.
func (rl *RepoLock) Lock(path string, ctx context.Context) func() {
	entry := rl.getEntry(path)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if entry.mu.TryLock() {
			return entry.mu.Unlock
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// RLock acquires a shared (read) lock for the repository at path.
// Multiple readers can hold RLock simultaneously, but writers block.
// Returns an unlock function. The caller must invoke it when done.
func (rl *RepoLock) RLock(path string, ctx context.Context) func() {
	entry := rl.getEntry(path)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if entry.mu.TryRLock() {
			return entry.mu.RUnlock
		}
		time.Sleep(1 * time.Millisecond)
	}
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
