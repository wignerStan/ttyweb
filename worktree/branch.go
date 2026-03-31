package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/revlist"
)

// BranchInfo describes a single branch in a repository.
type BranchInfo struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	IsRemote  bool   `json:"is_remote"`
	IsCurrent bool   `json:"is_current"`
	HeadHash  string `json:"head_hash"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
}

// ListBranches returns all local branches with their status information.
// Remote tracking branches are not included.
func ListBranches(ctx context.Context, repoPath string) ([]BranchInfo, error) {
	_ = ctx // reserved for future cancellation support

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return nil, &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	// Get the current HEAD to determine which branch is checked out.
	head, err := repo.Head()
	if err != nil {
		return nil, &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "get HEAD", Cause: err}
	}
	currentBranch := ""
	if head.Name().IsBranch() {
		currentBranch = head.Name().Short()
	}

	// Determine the default branch.
	defaultBranch := resolveDefaultBranch(ctx, absRepo)

	// Iterate all local references.
	refs, err := repo.References()
	if err != nil {
		return nil, &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "list references", Cause: err}
	}

	var branches []BranchInfo
	if err := refs.ForEach(func(ref *plumbing.Reference) error {
		// Only consider local branch refs.
		if !ref.Name().IsBranch() {
			return nil
		}
		name := ref.Name().Short()
		hash := ref.Hash()

		ahead, behind := computeBranchAheadBehind(repo, ref.Name())

		branches = append(branches, BranchInfo{
			Name:      name,
			IsDefault: name == defaultBranch,
			IsRemote:  false,
			IsCurrent: name == currentBranch,
			HeadHash:  hash.String(),
			Ahead:     ahead,
			Behind:    behind,
		})
		return nil
	}); err != nil {
		return nil, &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "iterate references", Cause: err}
	}

	return branches, nil
}

// CreateBranch creates a new branch from the current HEAD.
func CreateBranch(ctx context.Context, repoPath, name string) error {
	if err := ValidateBranchName(name); err != nil {
		return err
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	head, err := repo.Head()
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "get HEAD", Cause: err}
	}

	branchRef := plumbing.ReferenceName("refs/heads/" + name)
	if _, err := repo.Reference(branchRef, false); err == nil {
		return &OpError{Kind: KindAlreadyExists, Path: absRepo, Detail: fmt.Sprintf("branch %q already exists", name)}
	}

	newRef := plumbing.NewHashReference(branchRef, head.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("create branch %q", name), Cause: err}
	}

	return nil
}

// DeleteBranch deletes a local branch. The current branch and the default
// branch cannot be deleted.
func DeleteBranch(ctx context.Context, repoPath, name string) error {
	if err := ValidateBranchName(name); err != nil {
		return err
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	// Check that the branch exists.
	branchRef := plumbing.ReferenceName("refs/heads/" + name)
	if _, err := repo.Reference(branchRef, false); err != nil {
		return &OpError{Kind: KindNotFound, Path: absRepo, Detail: fmt.Sprintf("branch %q not found", name)}
	}

	// Cannot delete the current branch.
	head, err := repo.Head()
	if err == nil && head.Name().Short() == name {
		return &OpError{Kind: KindValidation, Path: absRepo, Detail: fmt.Sprintf("cannot delete the current branch %q", name)}
	}

	// Cannot delete the default branch.
	defaultBranch := resolveDefaultBranch(ctx, absRepo)
	if name == defaultBranch {
		return &OpError{Kind: KindValidation, Path: absRepo, Detail: fmt.Sprintf("cannot delete the default branch %q", name)}
	}

	if err := repo.Storer.RemoveReference(branchRef); err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("delete branch %q", name), Cause: err}
	}

	return nil
}

// MergeBranch merges source into target. For fast-forward merges, it uses
// go-git directly. For complex merges (non-fast-forward), it falls back to
// exec.Command for full merge support.
func MergeBranch(ctx context.Context, repoPath, source, target string) error {
	if err := ValidateBranchName(source); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if err := ValidateBranchName(target); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if source == target {
		return &OpError{Kind: KindValidation, Detail: fmt.Sprintf("cannot merge %q into itself", source)}
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	// Resolve source and target refs.
	sourceRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+source), false)
	if err != nil {
		return &OpError{Kind: KindNotFound, Path: absRepo, Detail: fmt.Sprintf("source branch %q not found", source)}
	}
	targetRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+target), false)
	if err != nil {
		return &OpError{Kind: KindNotFound, Path: absRepo, Detail: fmt.Sprintf("target branch %q not found", target)}
	}

	// Check if target is an ancestor of source (fast-forward possible).
	isAncestor, err := isAncestorOf(repo, targetRef.Hash(), sourceRef.Hash())
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "check ancestry", Cause: err}
	}

	if isAncestor {
		return fastForwardMerge(repo, absRepo, source, target, sourceRef)
	}

	// Fall back to git CLI for complex merges.
	return mergeViaCLI(ctx, absRepo, source, target)
}

// fastForwardMerge moves the target branch ref to the source hash and resets
// the worktree if the target branch is currently checked out.
func fastForwardMerge(repo *goGit.Repository, absRepo, source, target string, sourceRef *plumbing.Reference) error {
	newRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+target), sourceRef.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("fast-forward merge %q into %q", source, target), Cause: err}
	}

	// After fast-forward, update the worktree if target is currently checked out.
	wt, wtErr := repo.Worktree()
	if wtErr != nil || wt == nil {
		return nil
	}
	head, headErr := repo.Head()
	if headErr != nil || head == nil || head.Name().Short() != target {
		return nil
	}
	_ = wt.Reset(&goGit.ResetOptions{
		Mode:   goGit.HardReset,
		Commit: sourceRef.Hash(),
	})
	return nil
}

// computeBranchAheadBehind computes ahead/behind counts for a local branch
// against its remote tracking branch (origin/<name>).
func computeBranchAheadBehind(repo *goGit.Repository, branchRef plumbing.ReferenceName) (int, int) {
	remoteName := plumbing.ReferenceName("refs/remotes/origin/" + branchRef.Short())
	remote, err := repo.Reference(remoteName, false)
	if err != nil {
		return 0, 0
	}

	branch, err := repo.Reference(branchRef, false)
	if err != nil {
		return 0, 0
	}

	ahead, err := revlist.Objects(repo.Storer, []plumbing.Hash{branch.Hash()}, []plumbing.Hash{remote.Hash()})
	if err != nil {
		return 0, 0
	}
	behind, err := revlist.Objects(repo.Storer, []plumbing.Hash{remote.Hash()}, []plumbing.Hash{branch.Hash()})
	if err != nil {
		return 0, 0
	}

	return countCommits(repo, ahead), countCommits(repo, behind)
}

// isAncestorOf checks whether ancestorHash is an ancestor of descendantHash.
func isAncestorOf(repo *goGit.Repository, ancestorHash, descendantHash plumbing.Hash) (bool, error) {
	// If they're the same, it's trivially an ancestor.
	if ancestorHash == descendantHash {
		return true, nil
	}

	// Get all objects reachable from descendant that are NOT reachable from ancestor.
	// If the result is empty (ancestorHash is the only unreachable), it's an ancestor.
	objs, err := revlist.Objects(repo.Storer, []plumbing.Hash{descendantHash}, []plumbing.Hash{ancestorHash})
	if err != nil {
		return false, fmt.Errorf("revlist objects: %w", err)
	}

	// Filter to only commits.
	for _, h := range objs {
		if h == ancestorHash {
			continue
		}
		obj, err := repo.Storer.EncodedObject(plumbing.CommitObject, h)
		if err == nil && obj != nil {
			// Found a commit reachable from descendant but not from ancestor.
			return false, nil
		}
	}
	return true, nil
}

// mergeViaCLI uses git merge command for complex merges that cannot be
// fast-forwarded. This provides full merge resolution support.
func mergeViaCLI(ctx context.Context, repoPath, source, target string) error {
	// We need to check out target, merge source into it, then restore original HEAD.
	// First, save current HEAD.
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: "open repo for merge", Cause: err}
	}
	head, err := repo.Head()
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: "get HEAD for merge", Cause: err}
	}
	originalBranch := head.Name().Short()

	// If we're already on target, just merge.
	if originalBranch == target {
		return runGitMerge(ctx, repoPath, source)
	}

	// Otherwise, checkout target, merge, then restore original branch.
	if err := runGitCheckout(ctx, repoPath, target); err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: fmt.Sprintf("checkout %q for merge", target), Cause: err}
	}

	mergeErr := runGitMerge(ctx, repoPath, source)
	if mergeErr != nil {
		// Try to restore original branch on failure.
		_ = runGitCheckout(ctx, repoPath, originalBranch)
		return mergeErr
	}

	// Restore original branch.
	if err := runGitCheckout(ctx, repoPath, originalBranch); err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: fmt.Sprintf("restore branch %q after merge", originalBranch), Cause: err}
	}

	return nil
}

// runGitCheckout runs git checkout for a branch.
func runGitCheckout(ctx context.Context, dir, branch string) error {
	args := []string{"checkout", branch}
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // G204: branch name validated by ValidateBranchName
	cmd.Dir = dir
	cmd.Env = FilterGitEnv(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout %s: %w\n%s", branch, err, string(out))
	}
	return nil
}

// envOrDefault returns the value of the environment variable named by key,
// or the provided default if the variable is unset or empty.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// runGitMerge runs git merge with no-ff (to create a merge commit) for
// non-fast-forward merges.
func runGitMerge(ctx context.Context, dir, source string) error {
	args := []string{"merge", "--no-ff", source}
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // G204: source name validated by ValidateBranchName
	cmd.Dir = dir
	cmd.Env = append(FilterGitEnv(os.Environ()),
		"GIT_AUTHOR_NAME="+envOrDefault("GIT_AUTHOR_NAME", "ttyweb"),
		"GIT_AUTHOR_EMAIL="+envOrDefault("GIT_AUTHOR_EMAIL", "ttyweb@localhost"),
		"GIT_COMMITTER_NAME="+envOrDefault("GIT_COMMITTER_NAME", "ttyweb"),
		"GIT_COMMITTER_EMAIL="+envOrDefault("GIT_COMMITTER_EMAIL", "ttyweb@localhost"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Abort the merge to leave the repo in a clean state.
		abortCmd := exec.CommandContext(ctx, "git", "merge", "--abort")
		abortCmd.Dir = dir
		abortCmd.Env = FilterGitEnv(os.Environ())
		_ = abortCmd.Run()

		return &OpError{Kind: KindConflict, Detail: fmt.Sprintf("merge %q failed", source), Cause: fmt.Errorf("%w\n%s", err, string(out))}
	}
	return nil
}
