// Package worktree provides git worktree management operations.
// It uses go-git for repository-level introspection and worktree management.
package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/revlist"
)

// Info describes a single worktree in a repository.
type Info struct {
	Path        string `json:"path"`
	Branch      string `json:"branch"`
	IsMain      bool   `json:"isMain"`
	HeadCommit  string `json:"headCommit"`
	HeadMessage string `json:"headMessage,omitempty"`
}

// Status holds counts describing the state of a worktree.
type Status struct {
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

// FilterGitEnv removes git repository discovery variables from env.
// This prevents child git processes from discovering the parent repo.
func FilterGitEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
			"GIT_INDEX_FILE", "GIT_ALTERNATE_OBJECT_DIRECTORIES":
			// Skip variables that cause git to discover a parent repo.
		default:
			filtered = append(filtered, e)
		}
	}
	return filtered
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
func CreateWorktree(ctx context.Context, repoPath, branchName, baseBranch string, createBranch bool) (string, error) {
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

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	if createBranch {
		base := strings.TrimSpace(baseBranch)
		if base == "" {
			base = resolveDefaultBranch(ctx, absRepo)
		}
		if base == "" {
			base = "main"
		}
		baseRefName := plumbing.ReferenceName("refs/heads/" + base)
		baseRef, err := repo.Reference(baseRefName, false)
		if err != nil {
			return "", &OpError{Kind: KindNotFound, Path: absRepo, Detail: fmt.Sprintf("base branch %q not found", base), Cause: err}
		}
		branchRef := plumbing.ReferenceName("refs/heads/" + branchName)
		if _, err := repo.Reference(branchRef, false); err == nil {
			return "", &OpError{Kind: KindAlreadyExists, Path: absRepo, Detail: fmt.Sprintf("branch %q already exists", branchName)}
		}
		newRef := plumbing.NewHashReference(branchRef, baseRef.Hash())
		if err := repo.Storer.SetReference(newRef); err != nil {
			return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("create branch %q", branchName), Cause: err}
		}
	}

	worktreePath := filepath.Join(absRepo, ".worktrees", sanitizeBranchName(branchName))
	if err := addWorktree(absRepo, worktreePath, branchName); err != nil {
		if createBranch {
			branchRef := plumbing.ReferenceName("refs/heads/" + branchName)
			_ = repo.Storer.RemoveReference(branchRef)
		}
		return "", err
	}

	return worktreePath, nil
}

// ListWorktrees enumerates all worktrees attached to the repository.
func ListWorktrees(ctx context.Context, repoPath string) ([]Info, error) {
	_ = ctx // ctx reserved for future cancellation support
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, errEmptyPath
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}
	return listWorktreesManual(absRepo)
}

// RemoveWorktree removes a worktree. If force is true, uncommitted changes
// are ignored.
func RemoveWorktree(ctx context.Context, repoPath, worktreePath string, force bool) error {
	_ = ctx // ctx reserved for future cancellation support
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
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}

	// If worktree path doesn't exist, just prune metadata.
	if _, err := os.Stat(absWorktree); os.IsNotExist(err) {
		worktreeName := filepath.Base(absWorktree)
		_ = os.RemoveAll(filepath.Join(absRepo, ".git", "worktrees", worktreeName))
		return nil
	}

	return removeWorktreeManual(absRepo, absWorktree, force)
}

// GetWorktreeStatus returns the working-directory status of the worktree.
func GetWorktreeStatus(ctx context.Context, worktreePath string) (*Status, error) {
	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return nil, errors.New("worktree path is required")
	}

	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree path: %w", err)
	}

	return collectStatusGoGit(absPath)
}

// CommitWorktree stages all changes and commits in the worktree.
func CommitWorktree(ctx context.Context, worktreePath, message string) error {
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

	repo, err := defaultCache.Open(absPath)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absPath, Detail: "open repo", Cause: err}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absPath, Detail: "get worktree", Cause: err}
	}

	if err := wt.AddWithOptions(&goGit.AddOptions{All: true}); err != nil {
		return &OpError{Kind: KindGitFailed, Path: absPath, Detail: "stage all", Cause: err}
	}

	_, err = wt.Commit(message, &goGit.CommitOptions{})
	if err != nil {
		if errors.Is(err, goGit.ErrEmptyCommit) {
			return errors.New("nothing to commit: working tree clean")
		}
		return &OpError{Kind: KindGitFailed, Path: absPath, Detail: "commit", Cause: err}
	}

	return nil
}

// --- internal helpers ---

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

func resolveDefaultBranch(_ context.Context, repoPath string) string {
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		return ""
	}
	for _, name := range []string{"refs/heads/main", "refs/heads/master", "refs/heads/develop"} {
		if _, err := repo.Reference(plumbing.ReferenceName(name), false); err == nil {
			return strings.TrimPrefix(name, "refs/heads/")
		}
	}
	return ""
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

func headCommitMessage(_ context.Context, path string) string {
	repo, err := defaultCache.Open(path)
	if err != nil {
		return ""
	}
	head, err := repo.Head()
	if err != nil {
		return ""
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(commit.Message)
}

func collectStatusGoGit(path string) (*Status, error) {
	repo, err := defaultCache.Open(path)
	if err != nil {
		return nil, fmt.Errorf("collectStatusGoGit: open repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("collectStatusGoGit: get worktree: %w", err)
	}

	snap, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("collectStatusGoGit: get status: %w", err)
	}

	status := &Status{}
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
		}
		if fs.Staging != goGit.Unmodified && fs.Staging != goGit.Untracked {
			status.Staged++
		}
	}

	status.Ahead, status.Behind = computeAheadBehind(repo)
	return status, nil
}

// computeAheadBehind counts commits ahead/behind a tracking branch.
// Returns (0, 0) if no tracking branch or detached HEAD.
func computeAheadBehind(repo *goGit.Repository) (int, int) {
	head, err := repo.Head()
	if err != nil || !head.Name().IsBranch() {
		return 0, 0
	}

	upstreamName := plumbing.ReferenceName("refs/remotes/origin/" + head.Name().Short())
	upstream, err := repo.Reference(upstreamName, false)
	if err != nil {
		return 0, 0
	}

	ahead, err := revlist.Objects(repo.Storer, []plumbing.Hash{head.Hash()}, []plumbing.Hash{upstream.Hash()})
	if err != nil {
		return 0, 0
	}
	behind, err := revlist.Objects(repo.Storer, []plumbing.Hash{upstream.Hash()}, []plumbing.Hash{head.Hash()})
	if err != nil {
		return 0, 0
	}

	return len(ahead), len(behind)
}
