package homedir

import (
	"os"
)

func Expand(path string) string {
	if len(path) >= 2 && path[0:2] == "~/" {
		return os.Getenv("HOME") + path[1:]
	} else {
		return path
	}
}
