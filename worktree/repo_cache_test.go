package worktree

import (
	"path/filepath"
	"sync"
	"testing"

	goGit "github.com/go-git/go-git/v5"
)

func TestRepoCache_OpenCachesInstance(t *testing.T) {
	repo := initTestRepo(t)

	c := NewRepoCache()

	r1, err := c.Open(repo)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}

	r2, err := c.Open(repo)
	if err != nil {
		t.Fatalf("second Open failed: %v", err)
	}

	if r1 != r2 {
		t.Error("expected same pointer for repeated Open calls")
	}

	// Normalized path should also hit the cache.
	dotted := filepath.Join(repo, ".", "..", "work")
	r3, err := c.Open(dotted)
	if err != nil {
		t.Fatalf("Open with non-normalized path failed: %v", err)
	}
	if r1 != r3 {
		t.Error("expected same pointer for non-normalized path variant")
	}
}

func TestRepoCache_Remove(t *testing.T) {
	repo := initTestRepo(t)

	c := NewRepoCache()

	r1, err := c.Open(repo)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	c.Remove(repo)

	r2, err := c.Open(repo)
	if err != nil {
		t.Fatalf("Open after Remove failed: %v", err)
	}

	if r1 == r2 {
		t.Error("expected different pointer after Remove and reopen")
	}
}

func TestRepoCache_OpenNonexistent(t *testing.T) {
	c := NewRepoCache()

	_, err := c.Open("/nonexistent/path/nowhere")
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}

	// Should not be cached.
	_, err = c.Open("/nonexistent/path/nowhere")
	if err == nil {
		t.Fatal("expected error on second call to non-git directory")
	}
}

func TestRepoCache_ConcurrentOpen(t *testing.T) {
	repo := initTestRepo(t)

	c := NewRepoCache()
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	results := make([]*goGit.Repository, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r, err := c.Open(repo)
			if err != nil {
				errs <- err
				return
			}
			results[idx] = r
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Open failed: %v", err)
	}

	// All non-nil results should be the same pointer.
	var first *goGit.Repository
	for _, r := range results {
		if r == nil {
			continue
		}
		if first == nil {
			first = r
			continue
		}
		if r != first {
			t.Error("concurrent opens returned different pointers")
			break
		}
	}
}

func TestRepoCache_Clear(t *testing.T) {
	repo := initTestRepo(t)

	c := NewRepoCache()

	r1, err := c.Open(repo)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	c.Clear()

	r2, err := c.Open(repo)
	if err != nil {
		t.Fatalf("Open after Clear failed: %v", err)
	}

	if r1 == r2 {
		t.Error("expected different pointer after Clear and reopen")
	}
}
