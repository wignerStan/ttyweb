package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// HookEvent identifies a lifecycle event for worktree hooks.
type HookEvent string

// Supported worktree lifecycle hook events.
const (
	HookEventPreCreate  HookEvent = "pre-create"
	HookEventPostCreate HookEvent = "post-create"
	HookEventPreMerge   HookEvent = "pre-merge"
	HookEventPostMerge  HookEvent = "post-merge"
	HookEventPreRemove  HookEvent = "pre-remove"
)

// String returns the event name as used for hook script filenames.
func (e HookEvent) String() string { return string(e) }

// HookContext provides context data passed to hook scripts via environment
// variables.
type HookContext struct {
	Event        HookEvent
	WorktreePath string
	Branch       string
}

// EnvVars returns the environment variables that expose hook context to the
// executed script.
func (ctx HookContext) EnvVars() []string {
	return []string{
		"TTYWEB_EVENT=" + string(ctx.Event),
		"TTYWEB_WORKTREE_PATH=" + ctx.WorktreePath,
		"TTYWEB_BRANCH=" + ctx.Branch,
	}
}

// RunHook executes the hook script for the given event if it exists.
// Hook scripts are located at <repoPath>/.ttyweb/hooks/<event-name>.
// If the hook script does not exist, RunHook returns nil — hooks are optional.
// If the script exits with a non-zero status, RunHook returns an error.
func RunHook(event HookEvent, repoPath string, ctx HookContext) error {
	hooksDir := filepath.Join(repoPath, ".ttyweb", "hooks")
	scriptPath := filepath.Join(hooksDir, event.String())

	// Hooks are optional — missing script is not an error.
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command(scriptPath) //nolint:gosec,noctx // hook script execution
	cmd.Env = append(os.Environ(), ctx.EnvVars()...)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook %q failed: %w: %s", event, err, string(output))
	}

	return nil
}
