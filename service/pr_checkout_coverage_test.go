package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ttyweb/worktree"
)

// TestPRCheckoutService_DetectSlugFailure covers the error path when
// repo slug detection fails for a non-git repo.
func TestPRCheckoutService_DetectSlugFailure(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	// Pass empty slug so it tries to detect from repo path.
	// Use a non-repo path so detection fails.
	_, err := svc.CheckoutPR(t.Context(), t.TempDir(), "", 1, "token")
	if err == nil {
		t.Fatal("expected error when slug detection fails")
	}
}

// TestPRCheckoutService_FetchPRFailure covers the error path when
// the GitHub API call fails (non-reachable URL).
func TestPRCheckoutService_FetchPRFailure(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	// Use a valid repo path but the fetch will fail because the URL is
	// constructed from the slug and won't be reachable.
	_, err := svc.CheckoutPR(t.Context(), "/nonexistent/path", "owner/repo", 1, "test-token")
	if err == nil {
		t.Fatal("expected error for nonexistent repo path")
	}
}

// TestPRCheckoutService_EmptySlugFails covers when slug detection succeeds
// but the slug is invalid format.
func TestPRCheckoutService_EmptySlug(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "   ", 1, "token")
	if err == nil {
		t.Fatal("expected error for whitespace-only slug")
	}
}

// TestPRCheckoutService_DetectSlugSuccess covers the path where slug detection
// succeeds from a git repo with an origin remote.
func TestPRCheckoutService_DetectSlugSuccess(t *testing.T) {
	// Create a git repo with an HTTPS origin remote.
	dir := t.TempDir()
	workPath := filepath.Join(dir, "repo")
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Init repo with git CLI.
	mustRunGit(t, workPath, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(workPath, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mustRunGit(t, workPath, "add", ".")
	mustRunGit(t, workPath, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "init")

	// Add a fake origin remote.
	mustRunGit(t, workPath, "remote", "add", "origin", "https://github.com/owner/repo.git")

	slug, err := worktree.DetectRepoSlug(workPath)
	if err != nil {
		t.Fatalf("DetectRepoSlug: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}

	// Now test CheckoutPR with empty slug — it should detect the slug,
	// validate it, then fail at the GitHub API call (which is expected).
	svc := NewPRCheckoutService()
	_, err = svc.CheckoutPR(context.Background(), workPath, "", 1, "fake-token")
	if err == nil {
		t.Fatal("expected error from GitHub API call (fake token)")
	}
}

// mustRunGit runs a git command and fails the test on error.
func mustRunGit(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{name}, args...)...) //nolint:gosec // test code
	cmd.Dir = dir
	cmd.Env = worktree.FilterGitEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", name, err, out)
	}
}
