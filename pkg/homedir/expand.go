package homedir

import (
	"os"
)

// Expand replaces a leading ~/ in path with the user's home directory.
func Expand(path string) string {
	if len(path) >= 2 && path[0:2] == "~/" {
		return os.Getenv("HOME") + path[1:]
	}
	return path
}
