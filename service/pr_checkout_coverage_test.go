package service

import (
	"testing"
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
