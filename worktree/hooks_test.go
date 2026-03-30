package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHookEvent_String(t *testing.T) {
	tests := []struct {
		event HookEvent
		want  string
	}{
		{HookEventPreCreate, "pre-create"},
		{HookEventPostCreate, "post-create"},
		{HookEventPreMerge, "pre-merge"},
		{HookEventPostMerge, "post-merge"},
		{HookEventPreRemove, "pre-remove"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.event.String(); got != tt.want {
				t.Errorf("HookEvent.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHookContext_EnvVars(t *testing.T) {
	ctx := HookContext{
		Event:        HookEventPreCreate,
		WorktreePath: "/tmp/my-worktree",
		Branch:       "feature/test",
	}
	env := ctx.EnvVars()

	envMap := make(map[string]string, len(env))
	for _, e := range env {
		key, val, ok := cutEnvVar(e)
		if !ok {
			t.Fatalf("malformed env var: %q", e)
		}
		envMap[key] = val
	}

	if envMap["TTYWEB_EVENT"] != "pre-create" {
		t.Errorf("TTYWEB_EVENT = %q, want %q", envMap["TTYWEB_EVENT"], "pre-create")
	}
	if envMap["TTYWEB_WORKTREE_PATH"] != "/tmp/my-worktree" {
		t.Errorf("TTYWEB_WORKTREE_PATH = %q, want %q", envMap["TTYWEB_WORKTREE_PATH"], "/tmp/my-worktree")
	}
	if envMap["TTYWEB_BRANCH"] != "feature/test" {
		t.Errorf("TTYWEB_BRANCH = %q, want %q", envMap["TTYWEB_BRANCH"], "feature/test")
	}

	if len(env) != 3 {
		t.Errorf("EnvVars() returned %d vars, want 3", len(env))
	}
}

func TestRunHook_ScriptNotFound(t *testing.T) {
	// Hooks are optional — missing hook dir returns nil, not an error.
	tmpDir := t.TempDir()
	ctx := HookContext{
		Event:        HookEventPreCreate,
		WorktreePath: filepath.Join(tmpDir, "worktree"),
		Branch:       "test-branch",
	}

	err := RunHook(HookEventPreCreate, tmpDir, ctx)
	if err != nil {
		t.Errorf("RunHook with nonexistent hooks dir should return nil, got: %v", err)
	}
}

func TestRunHook_ScriptExecutes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ttyweb/hooks/pre-create script.
	hooksDir := filepath.Join(tmpDir, ".ttyweb", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "hook-ran-marker")
	script := "#!/bin/sh\ntouch " + markerFile + "\n"
	scriptPath := filepath.Join(hooksDir, "pre-create")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write hook script: %v", err)
	}

	ctx := HookContext{
		Event:        HookEventPreCreate,
		WorktreePath: filepath.Join(tmpDir, "worktree"),
		Branch:       "feature/test",
	}

	err := RunHook(HookEventPreCreate, tmpDir, ctx)
	if err != nil {
		t.Fatalf("RunHook should succeed: %v", err)
	}

	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("hook script did not execute — marker file not created")
	}
}

func TestRunHook_ScriptFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ttyweb/hooks/pre-merge script that exits with error.
	hooksDir := filepath.Join(tmpDir, ".ttyweb", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}

	script := "#!/bin/sh\nexit 1\n"
	scriptPath := filepath.Join(hooksDir, "pre-merge")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write hook script: %v", err)
	}

	ctx := HookContext{
		Event:        HookEventPreMerge,
		WorktreePath: filepath.Join(tmpDir, "worktree"),
		Branch:       "feature/merge-test",
	}

	err := RunHook(HookEventPreMerge, tmpDir, ctx)
	if err == nil {
		t.Error("RunHook should return error when script exits with 1")
	}
}

// cutEnvVar splits an env var string on the first "=".
func cutEnvVar(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}
