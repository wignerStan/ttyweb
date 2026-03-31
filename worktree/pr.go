package worktree

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// httpClient is a package-level HTTP client for GitHub API requests.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// PRDetails contains the information needed to check out a pull request branch.
type PRDetails struct {
	Number     int    `json:"number"`
	HeadSHA    string `json:"head_sha"`
	HeadBranch string `json:"head_branch"`
	HeadLabel  string `json:"head_label"`
	State      string `json:"state"`
}

// prResponse is the JSON structure returned by the GitHub PR API.
type prResponse struct {
	Number  int    `json:"number"`
	State   string `json:"state"`
	Head    prHead `json:"head"`
	HTMLURL string `json:"html_url"`
}

// prHead represents the head branch info in a GitHub PR response.
type prHead struct {
	Label string `json:"label"`
	SHA   string `json:"sha"`
	Ref   string `json:"ref"`
}

// FetchPRDetails fetches pull request details from a GitHub API URL.
// The apiURL should be the full URL to the PR, e.g.
// "https://api.github.com/repos/owner/repo/pulls/123".
func FetchPRDetails(ctx context.Context, apiURL, token string) (*PRDetails, error) {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return nil, fmt.Errorf("api URL is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch PR details: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized: check your GitHub token (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var pr prResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("parse PR response: %w", err)
	}

	return &PRDetails{
		Number:     pr.Number,
		HeadSHA:    pr.Head.SHA,
		HeadBranch: pr.Head.Ref,
		HeadLabel:  pr.Head.Label,
		State:      pr.State,
	}, nil
}

// DetectRepoSlug extracts the "owner/repo" slug from a local repository
// by reading the origin remote URL.
func DetectRepoSlug(repoPath string) (string, error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "", errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return "", &OpError{Kind: KindNotFound, Path: absRepo, Detail: "origin remote not found", Cause: err}
	}

	if len(remote.Config().URLs) == 0 {
		return "", &OpError{Kind: KindNotFound, Path: absRepo, Detail: "origin remote has no URLs"}
	}

	remoteURL := remote.Config().URLs[0]
	return parseRepoSlug(remoteURL)
}

// parseRepoSlug extracts "owner/repo" from a git remote URL.
// Handles SSH (git@host:owner/repo.git), HTTPS (https://host/owner/repo.git),
// and HTTP formats.
func parseRepoSlug(remoteURL string) (string, error) {
	// Strip .git suffix.
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	var slug string

	if strings.HasPrefix(remoteURL, "git@") {
		// SSH format: git@github.com:owner/repo
		// Remove the git@ prefix and extract after the colon.
		afterPrefix := strings.TrimPrefix(remoteURL, "git@")
		colonIdx := strings.Index(afterPrefix, ":")
		if colonIdx < 0 {
			return "", fmt.Errorf("invalid SSH remote URL: %s", remoteURL)
		}
		slug = afterPrefix[colonIdx+1:]
	} else if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		// HTTPS format: https://github.com/owner/repo
		// Remove scheme, then take the path after the host.
		withoutScheme := strings.TrimPrefix(strings.TrimPrefix(remoteURL, "https://"), "http://")
		slashIdx := strings.Index(withoutScheme, "/")
		if slashIdx < 0 {
			return "", fmt.Errorf("invalid HTTPS remote URL: %s", remoteURL)
		}
		slug = withoutScheme[slashIdx+1:]
	} else {
		return "", fmt.Errorf("unrecognized remote URL format: %s", remoteURL)
	}

	// slug should now be "owner/repo".
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot extract owner/repo from remote URL: %s", remoteURL)
	}

	return parts[0] + "/" + parts[1], nil
}

// CheckoutPRBranch fetches a specific SHA from origin and creates a worktree
// at the given branch name. Returns the absolute path of the new worktree.
//
//nolint:gocyclo // multi-step git checkout with fallback ref resolution
func CheckoutPRBranch(ctx context.Context, repoPath, branchName, sha string) (string, error) {
	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		return "", errors.New("branch name is required")
	}
	if err := ValidateBranchName(branchName); err != nil {
		return "", err
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "", fmt.Errorf("SHA is required")
	}

	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "", errEmptyPath
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve repo path: %w", err)
	}

	repo, err := defaultCache.Open(absRepo)
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: "open repo", Cause: err}
	}

	// Fetch the specific SHA from origin.
	refspec := config.RefSpec(sha + ":refs/remotes/pr/" + branchName)
	slog.Debug("fetching PR ref", "sha", sha, "refspec", refspec)

	fetchOpts := &goGit.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refspec},
	}

	err = repo.Fetch(fetchOpts)
	if err != nil && err != goGit.NoErrAlreadyUpToDate {
		// go-git returns an error even if the fetch succeeds but the ref
		// already exists. Check if we can resolve the ref anyway.
		remoteRef := plumbing.ReferenceName("refs/remotes/pr/" + branchName)
		if _, resolveErr := repo.Reference(remoteRef, false); resolveErr != nil {
			return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("fetch SHA %s", sha), Cause: err}
		}
	}

	// Create a local branch at the fetched SHA.
	remoteRef := plumbing.ReferenceName("refs/remotes/pr/" + branchName)
	remoteRefObj, err := repo.Reference(remoteRef, false)
	if err != nil {
		return "", &OpError{Kind: KindNotFound, Path: absRepo, Detail: fmt.Sprintf("pr ref %q not found", branchName), Cause: err}
	}

	branchRef := plumbing.ReferenceName("refs/heads/" + branchName)
	newRef := plumbing.NewHashReference(branchRef, remoteRefObj.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absRepo, Detail: fmt.Sprintf("create branch %q", branchName), Cause: err}
	}

	// Create the worktree.
	worktreePath := filepath.Join(absRepo, ".worktrees", sanitizeBranchName(branchName))
	if err := addWorktree(absRepo, worktreePath, branchName); err != nil {
		// Clean up the branch ref on failure.
		_ = repo.Storer.RemoveReference(branchRef)
		return "", err
	}

	return worktreePath, nil
}
