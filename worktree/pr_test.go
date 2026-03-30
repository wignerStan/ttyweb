package worktree

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPRDetails_InvalidURL(t *testing.T) {
	t.Parallel()

	_, err := FetchPRDetails(context.Background(), "", "token")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFetchPRDetails_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "token")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestFetchPRDetails_Success(t *testing.T) {
	t.Parallel()

	resp := prResponse{
		Number: 42,
		State:  "open",
		Head: prHead{
			Label: "contributor:fix-bug",
			SHA:   "abc123def456",
			Ref:   "fix-bug",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers.
		if got := r.Header.Get("Accept"); got != "application/vnd.github.v3+json" {
			t.Errorf("Accept header = %q, want %q", got, "application/vnd.github.v3+json")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}

		body, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	details, err := FetchPRDetails(context.Background(), srv.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if details.Number != 42 {
		t.Errorf("Number = %d, want 42", details.Number)
	}
	if details.HeadSHA != "abc123def456" {
		t.Errorf("HeadSHA = %q, want %q", details.HeadSHA, "abc123def456")
	}
	if details.HeadBranch != "fix-bug" {
		t.Errorf("HeadBranch = %q, want %q", details.HeadBranch, "fix-bug")
	}
	if details.HeadLabel != "contributor:fix-bug" {
		t.Errorf("HeadLabel = %q, want %q", details.HeadLabel, "contributor:fix-bug")
	}
	if details.State != "open" {
		t.Errorf("State = %q, want %q", details.State, "open")
	}
}

func TestFetchPRDetails_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "bad-token")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestFetchPRDetails_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "token")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchPRDetails_NoToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no Authorization header is sent when token is empty.
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("expected no Authorization header, got %q", got)
		}

		resp := prResponse{
			Number: 1,
			State:  "open",
			Head: prHead{
				Label: "owner:main",
				SHA:   "sha123",
				Ref:   "main",
			},
		}
		body, _ := json.Marshal(resp)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	details, err := FetchPRDetails(context.Background(), srv.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Number != 1 {
		t.Errorf("Number = %d, want 1", details.Number)
	}
}

func TestParseRepoSlug_SSH(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"standard SSH", "git@github.com:owner/repo.git", "owner/repo", false},
		{"SSH without .git", "git@github.com:owner/repo", "owner/repo", false},
		{"SSH with nested org", "git@github.com:org/subgroup/repo.git", "org/subgroup/repo", false},
		{"SSH invalid no colon", "git@github.com/owner/repo.git", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRepoSlug(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseRepoSlug_HTTPS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"standard HTTPS", "https://github.com/owner/repo.git", "owner/repo", false},
		{"HTTPS without .git", "https://github.com/owner/repo", "owner/repo", false},
		{"HTTPS with path", "https://github.com/org/subgroup/repo.git", "org/subgroup/repo", false},
		{"HTTP scheme", "http://github.com/owner/repo.git", "owner/repo", false},
		{"HTTPS no path", "https://github.com", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRepoSlug(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseRepoSlug_Unrecognized(t *testing.T) {
	t.Parallel()

	_, err := parseRepoSlug("ftp://github.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error for unrecognized URL format")
	}
}

func TestParseRepoSlug_MalformedSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{"SSH missing repo", "git@github.com:owner/"},
		{"SSH missing owner", "git@github.com:/repo.git"},
		{"HTTPS missing repo", "https://github.com/owner/"},
		{"HTTPS missing owner", "https://github.com/repo.git"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseRepoSlug(tc.url)
			if err == nil {
				t.Fatal("expected error for malformed slug")
			}
		})
	}
}

func TestDetectRepoSlug_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := DetectRepoSlug("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestDetectRepoSlug_NotARepo(t *testing.T) {
	t.Parallel()

	_, err := DetectRepoSlug(t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

func TestDetectRepoSlug_NoOriginRemote(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)
	// Add and remove the origin remote to ensure it doesn't exist.
	mustRun(t, repoPath, "remote", "add", "origin", "https://github.com/dummy/dummy.git")
	mustRun(t, repoPath, "remote", "remove", "origin")

	_, err := DetectRepoSlug(repoPath)
	if err == nil {
		t.Fatal("expected error when no origin remote")
	}
}

func TestDetectRepoSlug_ValidSSHRemote(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)

	// Add a known SSH remote.
	mustRun(t, repoPath, "remote", "add", "origin", "git@github.com:testuser/myrepo.git")

	slug, err := DetectRepoSlug(repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "testuser/myrepo" {
		t.Errorf("got %q, want %q", slug, "testuser/myrepo")
	}
}

func TestDetectRepoSlug_ValidHTTPSRemote(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)

	// Add a known HTTPS remote.
	mustRun(t, repoPath, "remote", "add", "origin", "https://github.com/testuser/myrepo.git")

	slug, err := DetectRepoSlug(repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "testuser/myrepo" {
		t.Errorf("got %q, want %q", slug, "testuser/myrepo")
	}
}

func TestCheckoutPRBranch_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), "", "pr-1", "abc123")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCheckoutPRBranch_EmptyBranch(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), "/some/path", "", "abc123")
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestCheckoutPRBranch_EmptySHA(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), "/some/path", "pr-1", "")
	if err == nil {
		t.Fatal("expected error for empty SHA")
	}
}

func TestCheckoutPRBranch_InvalidBranchName(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), "/some/path", "invalid branch!", "abc123")
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestCheckoutPRBranch_NotARepo(t *testing.T) {
	t.Parallel()

	_, err := CheckoutPRBranch(context.Background(), t.TempDir(), "pr-1", "abc123")
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}
