package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReload_ValidConfig tests that Reload successfully reloads a valid config file.
func TestReload_ValidConfig(t *testing.T) {
	// Create a temporary config file with a unique model name to detect reload.
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{
		"llm": {
			"api_key": "test-key",
			"api_url": "https://api.openai.com/v1/chat/completions",
			"model": "gpt-4o-reloaded"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := Reload(cfgPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	cfg := Get()
	if cfg.LLM.Model != "gpt-4o-reloaded" {
		t.Errorf("expected Model 'gpt-4o-reloaded', got %q", cfg.LLM.Model)
	}
}

// TestReload_InvalidConfig tests that Reload returns an error for invalid config.
func TestReload_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(cfgPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := Reload(cfgPath)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

// TestReload_NonexistentFile tests that Reload returns defaults
// for missing file (Load returns defaults if file doesn't exist).
func TestReload_NonexistentFile(t *testing.T) {
	// Reload does NOT error for missing files — it returns defaults.
	err := Reload("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("Reload should not fail for nonexistent file, got: %v", err)
	}

	cfg := Get()
	// Should have default values.
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %q", cfg.LLM.Model)
	}
}

// TestWatch_FileChange tests that Watch sends config updates when the file changes.
func TestWatch_FileChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	// Write initial config with a unique model name.
	cfgContent := `{
		"llm": {
			"api_key": "initial-key",
			"api_url": "https://api.openai.com/v1/chat/completions",
			"model": "gpt-4o-watch-test"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ch, stop, err := Watch(cfgPath)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer stop()

	// Modify the config file to trigger a watch event.
	time.Sleep(50 * time.Millisecond)
	updatedContent := `{
		"llm": {
			"api_key": "updated-key",
			"api_url": "https://api.openai.com/v1/chat/completions",
			"model": "gpt-4o-watch-updated"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(updatedContent), 0o644); err != nil {
		t.Fatalf("write updated config: %v", err)
	}

	// Wait for the watch event.
	select {
	case cfg := <-ch:
		if cfg == nil {
			t.Error("expected non-nil config from watch channel")
		} else if cfg.LLM.Model != "gpt-4o-watch-updated" {
			t.Errorf("expected Model 'gpt-4o-watch-updated', got %q", cfg.LLM.Model)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for config update")
	}
}

// TestWatch_Stop tests that the stop function correctly terminates the watcher.
func TestWatch_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfgContent := `{"llm":{"api_key":"key","api_url":"https://api.openai.com/v1/chat/completions","model":"gpt-4o"}}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ch, stop, err := Watch(cfgPath)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Stop the watcher.
	stop()

	// The channel should eventually be closed.
	select {
	case _, ok := <-ch:
		if ok {
			t.Log("received config after stop (acceptable race)")
		}
	case <-time.After(2 * time.Second):
		t.Error("expected channel to be closed after stop")
	}
}

// TestWatch_NonexistentPath tests Watch with a nonexistent file.
func TestWatch_NonexistentPath(t *testing.T) {
	_, _, err := Watch("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}
