// Package validate provides input validation utilities for user-supplied
// session names, pane IDs, and other parameters used in command construction.
package validate

import (
	"fmt"
	"math"
	"regexp"
)

var (
	// sessionNameRe allows alphanumeric characters, underscore, dot, and hyphen.
	// Maximum length is 128 characters to prevent buffer issues in command args.
	sessionNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

	// paneIDRe allows alphanumeric characters, underscore, dot, percent,
	// colon, and hyphen. Covers formats like "0", "%0", "1:0", "session:0.1".
	paneIDRe = regexp.MustCompile(`^[a-zA-Z0-9_.%:-]+$`)
)

// SessionName validates a session name against a strict allowlist.
// Returns an error if the name is empty or contains invalid characters.
func SessionName(name string) error {
	if name == "" {
		return fmt.Errorf("session name is required")
	}
	if !sessionNameRe.MatchString(name) {
		return fmt.Errorf(
			"session name contains invalid characters: %q (allowed: alphanumeric, _, ., -)",
			name,
		)
	}
	return nil
}

// PaneID validates a pane identifier.
// Returns an error if the ID is empty or contains invalid characters.
func PaneID(id string) error {
	if id == "" {
		return fmt.Errorf("pane ID is required")
	}
	if !paneIDRe.MatchString(id) {
		return fmt.Errorf("pane ID contains invalid characters: %q", id)
	}
	return nil
}

// Uint16 safely converts an int to uint16, clamping to the valid range.
// Values below 0 become 0; values above math.MaxUint16 become math.MaxUint16.
func Uint16(v int) uint16 {
	if v < 0 {
		return 0
	}
	if v > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(v)
}
