// Package config provides JSON configuration loading with environment variable overrides.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return applyEnvOverrides(&cfg), nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return applyEnvOverrides(&cfg), nil
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
	// Return a copy to preserve immutability.
	cp := *globalConfig
	return &cp
}

// Get returns the cached global config (must call LoadOrDefault first).
func Get() *Config {
	if globalConfig == nil {
		return LoadOrDefault()
	}
	cp := *globalConfig
	return &cp
}

// applyEnvOverrides returns a new Config with values replaced by any
// non-empty environment variables.
func applyEnvOverrides(cfg *Config) *Config {
	cp := *cfg

	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cp.LLM.ApiKey = v
	}
	if v := os.Getenv("LLM_API_URL"); v != "" {
		cp.LLM.ApiURL = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cp.LLM.Model = v
	}
	if v := os.Getenv("XFYUN_APP_ID"); v != "" {
		cp.Xunfei.AppID = v
	}
	if v := os.Getenv("XFYUN_API_KEY"); v != "" {
		cp.Xunfei.ApiKey = v
	}
	if v := os.Getenv("XFYUN_API_SECRET"); v != "" {
		cp.Xunfei.ApiSecret = v
	}
	if v := os.Getenv("BUTLER_HOST"); v != "" {
		cp.Butler.Host = v
	}
	if v := os.Getenv("BUTLER_PORT"); v != "" {
		cp.Butler.Port = v
	}

	return &cp
}
