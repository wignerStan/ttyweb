package service

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"ttyweb/worktree"
)

// PRCheckoutService orchestrates PR checkout by combining GitHub API calls
// with local git worktree operations.
type PRCheckoutService struct{}

// NewPRCheckoutService creates a new PRCheckoutService.
func NewPRCheckoutService() *PRCheckoutService {
	return &PRCheckoutService{}
}

// CheckoutPR fetches PR details from GitHub, validates the PR state,
// and creates a local worktree for the PR branch.
// Returns the absolute path of the created worktree.
func (s *PRCheckoutService) CheckoutPR(ctx context.Context, repoPath, repoSlug string, prNumber int, githubToken string) (string, error) {
	_ = s
	if err := validateCheckoutInputs(repoPath, prNumber, githubToken); err != nil {
		return "", err
	}

	if repoSlug = strings.TrimSpace(repoSlug); repoSlug == "" {
		detected, err := worktree.DetectRepoSlug(repoPath)
		if err != nil {
			return "", fmt.Errorf("detect repo slug: %w", err)
		}
		repoSlug = detected
	}

	if err := validateRepoSlug(repoSlug); err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repoSlug, prNumber)

	prDetails, err := worktree.FetchPRDetails(ctx, apiURL, githubToken)
	if err != nil {
		return "", fmt.Errorf("fetch PR #%d: %w", prNumber, err)
	}

	if prDetails.State != "open" {
		return "", fmt.Errorf("PR #%d is not open (state: %s)", prDetails.Number, prDetails.State)
	}

	branchName := sanitizePRBranchName(prDetails.Number, prDetails.HeadBranch)

	worktreePath, err := worktree.CheckoutPRBranch(ctx, repoPath, branchName, prDetails.HeadSHA)
	if err != nil {
		return "", fmt.Errorf("checkout PR branch: %w", err)
	}

	return worktreePath, nil
}

// validateCheckoutInputs validates the required inputs for PR checkout.
func validateCheckoutInputs(repoPath string, prNumber int, githubToken string) error {
	if strings.TrimSpace(repoPath) == "" {
		return fmt.Errorf("repo path is required")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be positive")
	}
	if strings.TrimSpace(githubToken) == "" {
		return fmt.Errorf("GitHub token is required")
	}
	return nil
}

// validateRepoSlug checks that a repo slug has the form "owner/repo".
func validateRepoSlug(slug string) error {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repo slug: %q (expected owner/repo)", slug)
	}
	return nil
}

// sanitizePRBranchName creates a safe branch name for a PR checkout.
// Format: pr-{number}-{sanitized-branch}
func sanitizePRBranchName(number int, branch string) string {
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, branch)

	result := fmt.Sprintf("pr-%d-%s", number, sanitized)

	// Trim trailing dashes.
	result = strings.TrimRight(result, "-")

	return result
}
