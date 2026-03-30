package server

import (
	"testing"
)

// TestBuiltinRoleIDMap verifies the role ID map has expected entries.
func TestBuiltinRoleIDMap(t *testing.T) {
	if builtinRoleIDMap["cli"] != 1 {
		t.Error("expected cli -> 1")
	}
	if builtinRoleIDMap["ops"] != 7 {
		t.Error("expected ops -> 7")
	}
	if builtinRoleIDMap["prompt"] != 5 {
		t.Error("expected prompt -> 5")
	}
}
