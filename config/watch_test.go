package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func resetGlobalConfig() {
	globalConfig = nil
	configOnce = sync.Once{}
}

func TestReload(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"llm":{"apiKey":"old"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.APIKey != "old" {
		t.Fatalf("expected old key, got: %s", cfg.LLM.APIKey)
	}

	// Set initial global config so Get() doesn't trigger LoadOrDefault.
	globalConfig = cfg
	configOnce.Do(func() {})

	// Update file.
	if err := os.WriteFile(cfgPath, []byte(`{"llm":{"apiKey":"new"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Reload(cfgPath); err != nil {
		t.Fatal(err)
	}

	updated := Get()
	if updated.LLM.APIKey != "new" {
		t.Errorf("expected new key after reload, got: %s", updated.LLM.APIKey)
	}
}

func TestWatchNotifiesOnChange(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"llm":{"apiKey":"initial"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ch, stop, err := Watch(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	if err := os.WriteFile(cfgPath, []byte(`{"llm":{"apiKey":"changed"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case cfg := <-ch:
		if cfg.LLM.APIKey != "changed" {
			t.Errorf("expected changed key, got: %s", cfg.LLM.APIKey)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for config change notification")
	}
}

func TestGetReturnsZeroValueWhenUnset(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get should never return nil")
	}
	// Get() triggers LoadOrDefault which loads defaults with defaults filled in.
	// APIKey is empty in DefaultConfig, so it should be empty (unless env var is set).
	if cfg.LLM.APIKey != "" {
		t.Errorf("expected empty API key, got: %s", cfg.LLM.APIKey)
	}
}

func TestReloadBadFile(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	// Set globalConfig so Reload has something to work with.
	globalConfig = DefaultConfig()
	configOnce.Do(func() {})

	err := Reload("/nonexistent/path/config.json")
	// Load returns defaults for missing files, so Reload should succeed.
	if err != nil {
		t.Errorf("Reload on missing file should succeed (returns defaults): %v", err)
	}
}

func TestReloadInvalidJSON(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(cfgPath, []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	globalConfig = DefaultConfig()
	configOnce.Do(func() {})

	err := Reload(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON in Reload")
	}
}

func TestWatchNonexistentPath(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	_, _, err := Watch("/nonexistent/path/config.json")
	// fsnotify returns an error when adding a non-existent path.
	if err == nil {
		t.Fatal("expected error when watching non-existent path")
	}
}

func TestWatchStopClosesChannel(t *testing.T) {
	resetGlobalConfig()
	defer resetGlobalConfig()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ch, stop, err := Watch(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	stop()

	// Channel should be closed after stop.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after stop()")
	}
}
