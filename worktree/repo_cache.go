package worktree

import (
	"fmt"
	"path/filepath"
	"sync"

	goGit "github.com/go-git/go-git/v5"
)

// RepoCache caches *goGit.Repository instances per path, avoiding repeated
// PlainOpen calls. It is safe for concurrent use.
type RepoCache struct {
	mu    sync.RWMutex
	repos map[string]*goGit.Repository
}

// NewRepoCache creates an empty cache.
func NewRepoCache() *RepoCache {
	return &RepoCache{repos: make(map[string]*goGit.Repository)}
}

// defaultCache is the package-level shared cache.
var defaultCache = NewRepoCache()

// Open returns a cached repository for path, or opens a new one via
// goGit.PlainOpen. Paths are normalized via filepath.Clean.
// Thread-safe via double-checked locking.
func (c *RepoCache) Open(path string) (*goGit.Repository, error) {
	key := filepath.Clean(path)

	// Fast path: read lock to check cache.
	c.mu.RLock()
	r, ok := c.repos[key]
	c.mu.RUnlock()
	if ok {
		return r, nil
	}

	// Slow path: write lock to open and store.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if r, ok := c.repos[key]; ok {
		return r, nil
	}

	r, err := goGit.PlainOpenWithOptions(key, &goGit.PlainOpenOptions{EnableDotGitCommonDir: true})
	if err != nil {
		return nil, fmt.Errorf("open repo %s: %w", key, err)
	}
	c.repos[key] = r
	return r, nil
}

// Remove discards the cached handle for path.
func (c *RepoCache) Remove(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.repos, filepath.Clean(path))
}

// Clear removes all cached entries.
func (c *RepoCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Create a new map to release references to all repositories.
	c.repos = make(map[string]*goGit.Repository)
}
