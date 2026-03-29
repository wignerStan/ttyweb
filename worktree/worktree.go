// Package worktree provides git worktree management operations.
// It wraps git CLI commands and go-git for repository-level introspection.
package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	goGit "github.com/go-git/go-git/v5"
)

// WorktreeInfo describes a single worktree in a repository.
type WorktreeInfo struct {
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	IsMain      bool   `json:"isMain"`
	HeadCommit  string `json:"headCommit"`
	HeadMessage string `json:"headMessage,omitempty"`
}

// WorktreeStatus holds counts describing the state of a worktree.
type WorktreeStatus struct {
	Ahead     int `json:"ahead"`
	Behind    int `json:"behind"`
	Modified  int `json:"modified"`
	Staged    int `json:"staged"`
	Untracked int `json:"untracked"`
	Conflicts int `json:"conflicts"`
}

var (
	// branchNameRe enforces safe branch names.
	branchNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_./-]{0,127}$`)

	errEmptyPath      = errors.New("path is required")
	errBranchRequired = errors.New("branch name is required")
	errInvalidBranch  = errors.New("invalid branch name")
)

// ValidateBranchName checks that a branch name is safe and well-formed.
func ValidateBranchName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errBranchRequired
	}
	if !branchNameRe.MatchString(trimmed) {
		return fmt.Errorf("%w: %q", errInvalidBranch, name)
	}
	return nil
}

// --- git command helpers ---

var (
	gitCommandEnv   = buildGitCommandEnv()
	testEnvOverride []string
	testEnvMu       sync.RWMutex
)

func buildGitCommandEnv() []string {
	env := os.Environ()
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_MERGE_AUTOEDIT=no",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
	)
	return env
}

// SetTestEnv allows tests to inject extra env vars for git commands.
func SetTestEnv(env []string) {
	testEnvMu.Lock()
	defer testEnvMu.Unlock()
	testEnvOverride = env
}

// newGitCmd creates a git exec.Cmd with safe env defaults.
func newGitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Env = append([]string(nil), gitCommandEnv...)

	testEnvMu.RLock()
	if len(testEnvOverride) > 0 {
		cmd.Env = append(cmd.Env, testEnvOverride...)
	}
	testEnvMu.RUnlock()

	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

// --- repository detection ---

// IsGitRepo returns true if path is a git repository.
func IsGitRepo(path string) bool {
	_, err := goGit.PlainOpen(path)
	return err == nil
}

// --- worktree operations ---

// CreateWorktree creates a new worktree for the repo at repoPath, checking out
// branchName. When createBranch is true, the branch is created from baseBranch
// first. Returns the absolute path of the new worktree.
func CreateWorktree(repoPath, branchName, baseBranch string, createBranch bool) (string, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "", errEmptyPath
	}
	if err := ValidateBranchName(branchName); err != nil {
		return "", err
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	// Create branch first if requested.
	if createBranch {
		base := strings.TrimSpace(baseBranch)
		if base == "" {
			base = resolveDefaultBranch(absRepo)
		}
		if base == "" {
			base = "main"
		}
		cmd := newGitCmd(absRepo, "branch", branchName, base)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("create branch failed: %s", strings.TrimSpace(string(out)))
		}
	}

	// Determine worktree directory.
	worktreePath := filepath.Join(absRepo, ".worktrees", sanitizeBranchName(branchName))
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", fmt.Errorf("create worktree parent dir: %w", err)
	}

	// Add the worktree.
	cmd := newGitCmd(absRepo, "worktree", "add", worktreePath, branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Clean up the branch if we just created it.
		if createBranch {
			_ = runGitCmd(absRepo, "branch", "-D", branchName)
		}
		return "", fmt.Errorf("add worktree failed: %s", strings.TrimSpace(string(out)))
	}

	return worktreePath, nil
}

// ListWorktrees enumerates all worktrees attached to the repository.
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}

	// Prune stale entries first.
	_ = runGitCmd(absRepo, "worktree", "prune")

	cmd := newGitCmd(absRepo, "worktree", "list", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list worktrees failed: %s", strings.TrimSpace(string(output)))
	}

	infos := parseWorktreeList(string(output))
	for i := range infos {
		if EqualPath(infos[i].Path, absRepo) {
			infos[i].IsMain = true
		}
		// Fetch head commit message.
		infos[i].HeadMessage = headCommitMessage(infos[i].Path)
	}

	return infos, nil
}

// RemoveWorktree removes a worktree. If force is true, --force is passed.
func RemoveWorktree(repoPath, worktreePath string, force bool) error {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return errEmptyPath
	}
	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return errors.New("worktree path is required")
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	cmd := newGitCmd(absRepo, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		errMsg := strings.TrimSpace(string(out))
		if strings.Contains(errMsg, "is not a working tree") || strings.Contains(errMsg, "not found") {
			_ = runGitCmd(absRepo, "worktree", "prune")
			return nil
		}
		return fmt.Errorf("remove worktree failed: %s", errMsg)
	}
	return nil
}

// GetWorktreeStatus returns the working-directory status of the worktree.
func GetWorktreeStatus(worktreePath string) (*WorktreeStatus, error) {
	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return nil, errors.New("worktree path is required")
	}

	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree path: %w", err)
	}

	// Try porcelain=v2 first for richer info.
	if status, err := collectStatusPorcelain(absPath); err == nil {
		return status, nil
	}
	// Fallback to go-git.
	return collectStatusGoGit(absPath)
}

// CommitWorktree stages all changes and commits in the worktree.
func CommitWorktree(worktreePath, message string) error {
	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return errors.New("worktree path is required")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("commit message is required")
	}

	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}

	if err := runGitCmd(absPath, "add", "--all"); err != nil {
		return fmt.Errorf("stage all: %w", err)
	}
	if err := runGitCmd(absPath, "commit", "-m", message); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			return errors.New("nothing to commit: working tree clean")
		}
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// --- internal helpers ---

func runGitCmd(dir string, args ...string) error {
	cmd := newGitCmd(dir, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func sanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "__",
		"\\", "__",
		":", "_",
		"*", "_",
		"?", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(strings.TrimSpace(branch))
}

func resolveDefaultBranch(repoPath string) string {
	for _, name := range []string{"main", "master", "develop"} {
		cmd := newGitCmd(repoPath, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
		if err := cmd.Run(); err == nil {
			return name
		}
	}
	return ""
}

func parseWorktreeList(output string) []WorktreeInfo {
	lines := strings.Split(output, "\n")
	result := make([]WorktreeInfo, 0)
	var current WorktreeInfo

	flush := func() {
		if current.Path != "" {
			result = append(result, current)
		}
		current = WorktreeInfo{}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			flush()
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], strings.TrimSpace(parts[1])

		switch key {
		case "worktree":
			current.Path = val
		case "branch":
			current.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "HEAD":
			if len(val) >= 7 {
				current.HeadCommit = val[:7]
			} else {
				current.HeadCommit = val
			}
		case "detached":
			current.Branch = val
		}
	}
	flush()
	return result
}

// EqualPath compares two file paths for equality, resolving symlinks on POSIX.
func EqualPath(a, b string) bool {
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	if evalA, err := filepath.EvalSymlinks(cleanA); err == nil {
		cleanA = evalA
	}
	if evalB, err := filepath.EvalSymlinks(cleanB); err == nil {
		cleanB = evalB
	}
	return cleanA == cleanB
}

func headCommitMessage(path string) string {
	cmd := newGitCmd(path, "log", "-1", "--pretty=format:%s")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func collectStatusPorcelain(path string) (*WorktreeStatus, error) {
	cmd := newGitCmd(path, "status", "--porcelain=2", "--branch")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parsePorcelainStatus(string(output)), nil
}

func parsePorcelainStatus(output string) *WorktreeStatus {
	if strings.TrimSpace(output) == "" {
		return &WorktreeStatus{}
	}

	status := &WorktreeStatus{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			parseStatusHeader(status, strings.TrimSpace(line[2:]))
			continue
		}
		switch line[0] {
		case '?':
			status.Untracked++
		case '1', '2':
			parseTrackedLine(status, line)
		case 'u':
			status.Conflicts++
		}
	}
	return status
}

func parseStatusHeader(status *WorktreeStatus, line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	switch fields[0] {
	case "branch.ab":
		if len(fields) >= 3 {
			status.Ahead = parseCount(fields[1])
			status.Behind = parseCount(fields[2])
		}
	}
}

func parseTrackedLine(status *WorktreeStatus, line string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}
	xy := fields[1]
	if len(xy) < 2 {
		return
	}
	x, y := rune(xy[0]), rune(xy[1])
	if x == 'U' || y == 'U' {
		status.Conflicts++
		return
	}
	if x != '.' {
		status.Staged++
	}
	if y != '.' {
		status.Modified++
	}
}

func parseCount(token string) int {
	token = strings.TrimLeft(strings.TrimSpace(token), "+-")
	n, err := strconv.Atoi(token)
	if err != nil {
		return 0
	}
	return n
}

func collectStatusGoGit(path string) (*WorktreeStatus, error) {
	repo, err := goGit.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	snap, err := worktree.Status()
	if err != nil {
		return nil, err
	}

	status := &WorktreeStatus{}
	for _, fs := range snap {
		if fs.Staging == goGit.Untracked || fs.Worktree == goGit.Untracked {
			status.Untracked++
			continue
		}
		if fs.Staging == goGit.UpdatedButUnmerged || fs.Worktree == goGit.UpdatedButUnmerged {
			status.Conflicts++
			continue
		}
		switch fs.Worktree {
		case goGit.Modified, goGit.Added, goGit.Deleted, goGit.Renamed:
			status.Modified++
		case goGit.Unmodified, goGit.Untracked, goGit.Copied, goGit.UpdatedButUnmerged:
			// handled above or no-op
		}
		if fs.Staging != goGit.Unmodified && fs.Staging != goGit.Untracked {
			status.Staged++
		}
	}
	return status, nil
}
