package db

import (
	"os"
	"path/filepath"

	"ttyweb/pkg/homedir"
)

// Options holds configuration for the database connection.
type Options struct {
	Path string
}

// DefaultOptions returns Options with the database path resolved to
// $XDG_DATA_HOME/ttyweb/ttyweb.db (falls back to ~/.local/share/ttyweb/ttyweb.db).
func DefaultOptions() Options {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = homedir.Expand("~/.local/share")
	}
	return Options{
		Path: filepath.Join(dataHome, "ttyweb", "ttyweb.db"),
	}
}
