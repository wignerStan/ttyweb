package config

import (
	"os"
	"path/filepath"
)

// DefaultConfigPath returns the standard configuration file path,
// respecting XDG_CONFIG_HOME if set, falling back to ~/.config/ttyweb/config.json.
func DefaultConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, ok := os.UserHomeDir()
		if ok != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "ttyweb", "config.json")
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			APIURL: "https://api.openai.com/v1/chat/completions",
			Model:  "gpt-4o",
		},
		Butler: ButlerConfig{
			Host: "localhost",
			Port: "8215",
		},
	}
}
