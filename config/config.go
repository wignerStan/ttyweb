// Package config provides JSON configuration loading with environment variable overrides.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Config holds all application configuration sections.
type Config struct {
	LLM     LLMConfig     `json:"llm"`
	Xunfei  XunfeiConfig  `json:"xfyun"`
	Butler  ButlerConfig  `json:"butler"`
	DB      DBConfig      `json:"db"`
	Worktree WorktreeConfig `json:"worktree"`
}

// LLMConfig holds settings for the AI/LLM backend.
type LLMConfig struct {
	ApiKey      string `json:"apiKey"`
	ApiURL      string `json:"apiUrl"`
	Model       string `json:"model"`
	DefaultRole string `json:"defaultRole"`
}

// XunfeiConfig holds credentials for Xunfei speech recognition.
type XunfeiConfig struct {
	AppID    string `json:"appId"`
	ApiKey   string `json:"apiKey"`
	ApiSecret string `json:"apiSecret"`
}

// ButlerConfig holds connection details for the butler orchestration proxy.
type ButlerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// DBConfig holds the database file path.
type DBConfig struct {
	Path string `json:"path"`
}

// WorktreeConfig holds the git worktree base path.
type WorktreeConfig struct {
	BasePath string `json:"basePath"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// Load reads and parses a JSON config file at the given path, then applies
// environment variable overrides on top. If the file does not exist, it
// returns DefaultConfig() with env var overrides applied.
func Load(path string) (*Config, error) {
	base := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return applyEnvOverrides(base), nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := *base // copy defaults
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return applyEnvOverrides(&cfg), nil
}

// envOverride maps environment variable names to setter functions.
var envOverride = []struct {
	key string
	set func(*Config, string)
}{
	{"LLM_API_KEY", func(c *Config, v string) { c.LLM.ApiKey = v }},
	{"LLM_API_URL", func(c *Config, v string) { c.LLM.ApiURL = v }},
	{"LLM_MODEL", func(c *Config, v string) { c.LLM.Model = v }},
	{"XFYUN_APP_ID", func(c *Config, v string) { c.Xunfei.AppID = v }},
	{"XFYUN_API_KEY", func(c *Config, v string) { c.Xunfei.ApiKey = v }},
	{"XFYUN_API_SECRET", func(c *Config, v string) { c.Xunfei.ApiSecret = v }},
	{"BUTLER_HOST", func(c *Config, v string) { c.Butler.Host = v }},
	{"BUTLER_PORT", func(c *Config, v string) { c.Butler.Port = v }},
}

// copyConfig returns a shallow copy of cfg.
func copyConfig(cfg *Config) *Config {
	cp := *cfg
	return &cp
}

// LoadOrDefault loads from the default config path, falling back to defaults
// if the file is missing. The result is cached for subsequent calls.
func LoadOrDefault() *Config {
	configOnce.Do(func() {
		cfg, err := Load(DefaultConfigPath())
		if err != nil {
			cfg = applyEnvOverrides(DefaultConfig())
		}
		globalConfig = cfg
	})
	return copyConfig(globalConfig)
}

// Get returns the cached global config (must call LoadOrDefault first).
func Get() *Config {
	if globalConfig == nil {
		return LoadOrDefault()
	}
	return copyConfig(globalConfig)
}

// applyEnvOverrides returns a new Config with values replaced by any
// non-empty environment variables.
func applyEnvOverrides(cfg *Config) *Config {
	cp := copyConfig(cfg)
	for _, override := range envOverride {
		if v := os.Getenv(override.key); v != "" {
			override.set(cp, v)
		}
	}
	return cp
}
