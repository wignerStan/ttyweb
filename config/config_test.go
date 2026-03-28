package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.ApiURL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("unexpected default LLM.ApiURL: %s", cfg.LLM.ApiURL)
	}
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("unexpected default LLM.Model: %s", cfg.LLM.Model)
	}
	if cfg.Butler.Host != "localhost" {
		t.Errorf("unexpected default Butler.Host: %s", cfg.Butler.Host)
	}
	if cfg.Butler.Port != "8215" {
		t.Errorf("unexpected default Butler.Port: %s", cfg.Butler.Port)
	}
}

func TestLoadFromJSON(t *testing.T) {
	content := `{
		"llm": {
			"apiKey": "test-key",
			"apiUrl": "https://example.com/v1/chat",
			"model": "my-model",
			"defaultRole": "cli"
		},
		"xfyun": {
			"appId": "xf-app",
			"apiKey": "xf-key",
			"apiSecret": "xf-secret"
		},
		"butler": {
			"host": "butler-host",
			"port": "9000"
		},
		"db": {
			"path": "/tmp/test.db"
		},
		"worktree": {
			"basePath": "/tmp/worktrees"
		}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LLM.ApiKey != "test-key" {
		t.Errorf("LLM.ApiKey = %q, want %q", cfg.LLM.ApiKey, "test-key")
	}
	if cfg.LLM.ApiURL != "https://example.com/v1/chat" {
		t.Errorf("LLM.ApiURL = %q, want %q", cfg.LLM.ApiURL, "https://example.com/v1/chat")
	}
	if cfg.LLM.Model != "my-model" {
		t.Errorf("LLM.Model = %q, want %q", cfg.LLM.Model, "my-model")
	}
	if cfg.LLM.DefaultRole != "cli" {
		t.Errorf("LLM.DefaultRole = %q, want %q", cfg.LLM.DefaultRole, "cli")
	}
	if cfg.Xunfei.AppID != "xf-app" {
		t.Errorf("Xunfei.AppID = %q, want %q", cfg.Xunfei.AppID, "xf-app")
	}
	if cfg.Xunfei.ApiKey != "xf-key" {
		t.Errorf("Xunfei.ApiKey = %q, want %q", cfg.Xunfei.ApiKey, "xf-key")
	}
	if cfg.Xunfei.ApiSecret != "xf-secret" {
		t.Errorf("Xunfei.ApiSecret = %q, want %q", cfg.Xunfei.ApiSecret, "xf-secret")
	}
	if cfg.Butler.Host != "butler-host" {
		t.Errorf("Butler.Host = %q, want %q", cfg.Butler.Host, "butler-host")
	}
	if cfg.Butler.Port != "9000" {
		t.Errorf("Butler.Port = %q, want %q", cfg.Butler.Port, "9000")
	}
	if cfg.DB.Path != "/tmp/test.db" {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, "/tmp/test.db")
	}
	if cfg.Worktree.BasePath != "/tmp/worktrees" {
		t.Errorf("Worktree.BasePath = %q, want %q", cfg.Worktree.BasePath, "/tmp/worktrees")
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil for missing file")
	}
	// Should still have defaults.
	if cfg.LLM.Model == "" {
		t.Error("expected default LLM.Model to be set on missing file")
	}
}

func TestEnvVarOverrides(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
		get    func(*Config) string
	}{
		{"LLM_API_KEY", "LLM_API_KEY", "env-api-key", func(c *Config) string { return c.LLM.ApiKey }},
		{"LLM_API_URL", "LLM_API_URL", "https://env.example.com/v1", func(c *Config) string { return c.LLM.ApiURL }},
		{"LLM_MODEL", "LLM_MODEL", "env-model", func(c *Config) string { return c.LLM.Model }},
		{"XFYUN_APP_ID", "XFYUN_APP_ID", "env-app-id", func(c *Config) string { return c.Xunfei.AppID }},
		{"XFYUN_API_KEY", "XFYUN_API_KEY", "env-xf-key", func(c *Config) string { return c.Xunfei.ApiKey }},
		{"XFYUN_API_SECRET", "XFYUN_API_SECRET", "env-xf-secret", func(c *Config) string { return c.Xunfei.ApiSecret }},
		{"BUTLER_HOST", "BUTLER_HOST", "env-butler-host", func(c *Config) string { return c.Butler.Host }},
		{"BUTLER_PORT", "BUTLER_PORT", "9999", func(c *Config) string { return c.Butler.Port }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)
			cfg := applyEnvOverrides(DefaultConfig())
			if got := tt.get(cfg); got != tt.envVal {
				t.Errorf("%s = %q, want %q", tt.envKey, got, tt.envVal)
			}
		})
	}
}

func TestEnvVarOverridesJSONValues(t *testing.T) {
	content := `{
		"llm": {"apiKey": "json-key", "model": "json-model"},
		"butler": {"host": "json-host", "port": "3000"}
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LLM_API_KEY", "override-key")
	t.Setenv("BUTLER_PORT", "override-port")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LLM.ApiKey != "override-key" {
		t.Errorf("LLM.ApiKey = %q, want env override %q", cfg.LLM.ApiKey, "override-key")
	}
	if cfg.LLM.Model != "json-model" {
		t.Errorf("LLM.Model = %q, want JSON value %q (not overridden)", cfg.LLM.Model, "json-model")
	}
	if cfg.Butler.Port != "override-port" {
		t.Errorf("Butler.Port = %q, want env override %q", cfg.Butler.Port, "override-port")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("DefaultConfigPath returned empty string")
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("expected filename config.json, got %s", filepath.Base(path))
	}
}

func TestDefaultConfigPathWithXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	path := DefaultConfigPath()
	expected := filepath.Join("/tmp/xdg-test", "ttyweb", "config.json")
	if path != expected {
		t.Errorf("DefaultConfigPath with XDG = %q, want %q", path, expected)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json{{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed for empty JSON: %v", err)
	}
	// Empty JSON object should preserve defaults.
	if cfg.LLM.ApiURL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("expected default LLM.ApiURL, got %q", cfg.LLM.ApiURL)
	}
}

func TestGetReturnsCopy(t *testing.T) {
	// Reset global state for this test.
	globalConfig = nil
	configOnce = sync.Once{}

	defer func() {
		globalConfig = nil
		configOnce = sync.Once{}
	}()

	cfg1 := Get()
	cfg1.LLM.Model = "mutated-model"

	cfg2 := Get()
	if cfg2.LLM.Model == "mutated-model" {
		t.Error("Get() did not return a copy; mutation leaked between calls")
	}
}
