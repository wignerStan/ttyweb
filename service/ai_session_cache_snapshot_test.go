package service

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

var updateCacheGolden = flag.Bool("update-cache", false, "update golden files for AI session cache tests")

func compareCacheGolden(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateCacheGolden {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}
	want, err := os.ReadFile(golden) //nolint:gosec // test file from t.TempDir()
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

// sanitizeCacheJSON replaces dynamic paths and timestamps in cache entry JSON
// with stable placeholders so golden files are deterministic.
func sanitizeCacheJSON(data []byte, dir string) []byte {
	// Replace the temp dir path with a placeholder.
	if dir != "" {
		data = bytes.ReplaceAll(data, []byte(dir), []byte("/tmp/testdir"))
	}
	// Replace ISO 8601 timestamps.
	re := regexp.MustCompile(`"ModTime":\s*"[^"]*"`)
	data = re.ReplaceAll(data, []byte(`"ModTime": "<time>"`))
	return data
}

// --- ScanAndCache snapshot tests ---

func TestSnapshot_ScanAndCache_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAISessionStore()

	store.ScanAndCache(dir, "project-key")

	// The cache should be empty.
	store.mu.RLock()
	count := len(store.cache)
	store.mu.RUnlock()

	if count != 0 {
		t.Fatalf("expected 0 cache entries for empty dir, got %d", count)
	}

	got := []byte(fmt.Sprintf("cache_entries=%d\n", count))
	compareCacheGolden(t, got)
}

func TestSnapshot_ScanAndCache_WithOldFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAISessionStore()

	// Create old .jsonl files (mtime > 24h ago).
	oldTime := time.Now().Add(-48 * time.Hour)
	for _, name := range []string{"session-a.jsonl", "session-b.jsonl", "readme.txt"} {
		path := filepath.Join(dir, name)
		content := fmt.Sprintf(`{"session":%q}`, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		// Set mtime to 48h ago.
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	store.ScanAndCache(dir, "project-key")

	// Only .jsonl files should be cached (readme.txt is skipped).
	store.mu.RLock()
	count := len(store.cache)
	store.mu.RUnlock()

	if count != 2 {
		t.Fatalf("expected 2 cache entries (only .jsonl), got %d", count)
	}

	// Snapshot the cache entries as JSON.
	store.mu.RLock()
	entries := make([]cacheEntry, 0, count)
	for _, e := range store.cache {
		entries = append(entries, e)
	}
	store.mu.RUnlock()

	got, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')
	got = sanitizeCacheJSON(got, dir)
	compareCacheGolden(t, got)
}

// --- IsCacheValid snapshot tests ---

func TestSnapshot_IsCacheValid_NoCache(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()

	valid := store.IsCacheValid("/nonexistent/path.jsonl")
	if valid {
		t.Fatal("expected false for file not in cache")
	}

	got := []byte(fmt.Sprintf("valid=%v\n", valid))
	compareCacheGolden(t, got)
}

func TestSnapshot_IsCacheValid_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAISessionStore()

	// Create a file and populate the cache.
	path := filepath.Join(dir, "valid.jsonl")
	content := `{"session":"valid"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	store.mu.Lock()
	store.cache[cacheKey(path)] = cacheEntry{
		Path:    path,
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}
	store.mu.Unlock()

	valid := store.IsCacheValid(path)
	if !valid {
		t.Fatal("expected true for file matching cache")
	}

	got := []byte(fmt.Sprintf("valid=%v\n", valid))
	compareCacheGolden(t, got)
}

func TestSnapshot_IsCacheValid_Modified(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAISessionStore()

	// Create a file, cache its metadata, then modify it.
	path := filepath.Join(dir, "modified.jsonl")
	if err := os.WriteFile(path, []byte(`{"original":true}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	store.mu.Lock()
	store.cache[cacheKey(path)] = cacheEntry{
		Path:    path,
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}
	store.mu.Unlock()

	// Modify the file (change content, which changes size and potentially mtime).
	if err := os.WriteFile(path, []byte(`{"original":true,"extra":"data"}`), 0o644); err != nil {
		t.Fatalf("write modified: %v", err)
	}

	valid := store.IsCacheValid(path)
	if valid {
		t.Fatal("expected false for modified file")
	}

	got := []byte(fmt.Sprintf("valid=%v\n", valid))
	compareCacheGolden(t, got)
}

func TestSnapshot_IsCacheValid_Deleted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAISessionStore()

	// Create a file, cache its metadata, then delete it.
	path := filepath.Join(dir, "deleted.jsonl")
	if err := os.WriteFile(path, []byte(`{"temp":true}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	store.mu.Lock()
	store.cache[cacheKey(path)] = cacheEntry{
		Path:    path,
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}
	store.mu.Unlock()

	// Delete the file.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	valid := store.IsCacheValid(path)
	if valid {
		t.Fatal("expected false for deleted file")
	}

	got := []byte(fmt.Sprintf("valid=%v\n", valid))
	compareCacheGolden(t, got)
}
