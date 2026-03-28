package db

import (
	"os"
	"path/filepath"
)

// Options configures the database connection.
type Options struct {
	// Path is the filesystem path to the SQLite database file.
	// The parent directory is created automatically if it does not exist.
	Path string
}

// DefaultOptions returns Options pointing to the standard data directory:
// ~/.local/share/ttyweb/ttyweb.db
func DefaultOptions() Options {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	dir := filepath.Join(home, ".local", "share", "ttyweb")

	return Options{
		Path: filepath.Join(dir, "ttyweb.db"),
	}
}
