package worktree

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
)

// sanitizeTempPath replaces non-deterministic temp directory paths in error messages.
var tempPathRe = regexp.MustCompile(`/tmp/[^\s/:]+`)

func sanitizeTempPath(s string) string {
	return tempPathRe.ReplaceAllString(s, "/tmp/TESTDIR")
}

// TestSnapshot_CheckoutPRBranch_NotARepo verifies that checking out a PR
// against a non-repository path returns a structured OpError.
func TestSnapshot_CheckoutPRBranch_NotARepo(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), t.TempDir(), "pr-999", "deadbeef1234567890abcdef1234567890abcdef")
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}

	result := map[string]any{
		"error": sanitizeTempPath(err.Error()),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	compareGolden(t, append(data, '\n'))
}

// TestSnapshot_CheckoutPRBranch_InvalidBranchName verifies that an invalid
// branch name returns a validation error.
func TestSnapshot_CheckoutPRBranch_InvalidBranchName(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), "/some/path", "invalid branch!", "abc123")
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}

	result := map[string]any{
		"error": err.Error(),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	compareGolden(t, append(data, '\n'))
}

// TestSnapshot_CheckoutPRBranch_EmptyFields verifies that empty required fields
// return appropriate errors.
func TestSnapshot_CheckoutPRBranch_EmptyFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		repoPath   string
		branchName string
		sha        string
	}{
		{"empty path", "", "pr-1", "abc123"},
		{"empty branch", "/some/path", "", "abc123"},
		{"empty sha", "/some/path", "pr-1", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := CheckoutPRBranch(context.Background(), tc.repoPath, tc.branchName, tc.sha)
			if err == nil {
				t.Fatal("expected error")
			}

			result := map[string]any{
				"error": err.Error(),
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			compareGolden(t, append(data, '\n'))
		})
	}
}

// TestSnapshot_CheckoutPRBranch_FetchFailure verifies that a valid repo without
// a remote origin returns an error when trying to fetch a PR SHA.
func TestSnapshot_CheckoutPRBranch_FetchFailure(t *testing.T) {
	repoPath := initTestRepo(t)

	_, err := CheckoutPRBranch(context.Background(), repoPath, "pr-snapshot-test", "deadbeef1234567890abcdef1234567890abcdef")
	if err == nil {
		t.Fatal("expected error when repo has no remote origin")
	}

	result := map[string]any{
		"error": sanitizeTempPath(err.Error()),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	compareGolden(t, append(data, '\n'))
}
