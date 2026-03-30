package service

import (
	"testing"
)

func TestPRCheckoutService_InvalidRepoPath(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "", "owner/repo", 1, "token")
	if err == nil {
		t.Fatal("expected error for empty repo path")
	}
}

func TestPRCheckoutService_ZeroPRNumber(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "owner/repo", 0, "token")
	if err == nil {
		t.Fatal("expected error for zero PR number")
	}
}

func TestPRCheckoutService_NegativePRNumber(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "owner/repo", -1, "token")
	if err == nil {
		t.Fatal("expected error for negative PR number")
	}
}

func TestPRCheckoutService_EmptyToken(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "owner/repo", 1, "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestPRCheckoutService_InvalidSlug(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "no-slash", 1, "token")
	if err == nil {
		t.Fatal("expected error for invalid slug")
	}
}

func TestPRCheckoutService_SlugWithEmptyOwner(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "/repo", 1, "token")
	if err == nil {
		t.Fatal("expected error for slug with empty owner")
	}
}

func TestPRCheckoutService_SlugWithEmptyRepo(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "owner/", 1, "token")
	if err == nil {
		t.Fatal("expected error for slug with empty repo")
	}
}

func TestPRCheckoutService_WhitespaceRepoPath(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "   ", "owner/repo", 1, "token")
	if err == nil {
		t.Fatal("expected error for whitespace repo path")
	}
}

func TestPRCheckoutService_WhitespaceToken(t *testing.T) {
	t.Parallel()

	svc := NewPRCheckoutService()
	_, err := svc.CheckoutPR(t.Context(), "/some/path", "owner/repo", 1, "   ")
	if err == nil {
		t.Fatal("expected error for whitespace token")
	}
}

func TestValidateRepoSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid", "owner/repo", false},
		{"with org", "org/team/repo", false},
		{"no slash", "ownerrepo", true},
		{"empty owner", "/repo", true},
		{"empty repo", "owner/", true},
		{"empty", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateRepoSlug(tc.slug)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSanitizePRBranchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		number int
		branch string
		want   string
	}{
		{"simple", 42, "fix-bug", "pr-42-fix-bug"},
		{"with slashes", 42, "feature/my-branch", "pr-42-feature-my-branch"},
		{"with underscores", 42, "my_feature", "pr-42-my_feature"},
		{"with special chars", 42, "fix: bug #123", "pr-42-fix--bug--123"},
		{"unicode", 42, "fix-bug", "pr-42-fix-bug"},
		{"trailing special", 42, "fix!", "pr-42-fix"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizePRBranchName(tc.number, tc.branch)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
