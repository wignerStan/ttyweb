package worktree

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// --- parseRepoSlug tests (0% -> 100%) ---

func TestParseRepoSlug_SSH(t *testing.T) {
	t.Parallel()
	slug, err := parseRepoSlug("git@github.com:owner/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestParseRepoSlug_SSH_NoGitSuffix(t *testing.T) {
	t.Parallel()
	slug, err := parseRepoSlug("git@github.com:owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestParseRepoSlug_HTTPS(t *testing.T) {
	t.Parallel()
	slug, err := parseRepoSlug("https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestParseRepoSlug_HTTP(t *testing.T) {
	t.Parallel()
	slug, err := parseRepoSlug("http://gitlab.com/owner/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestParseRepoSlug_HTTPS_NoGitSuffix(t *testing.T) {
	t.Parallel()
	slug, err := parseRepoSlug("https://github.com/owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestParseRepoSlug_SSH_Invalid_NoColon(t *testing.T) {
	t.Parallel()
	_, err := parseRepoSlug("git@github.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error for invalid SSH URL without colon")
	}
}

func TestParseRepoSlug_HTTPS_Invalid_NoSlash(t *testing.T) {
	t.Parallel()
	_, err := parseRepoSlug("https://github.com")
	if err == nil {
		t.Fatal("expected error for HTTPS URL without path")
	}
}

func TestParseRepoSlug_Unrecognized(t *testing.T) {
	t.Parallel()
	_, err := parseRepoSlug("ftp://github.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error for unrecognized URL scheme")
	}
}

func TestParseRepoSlug_EmptyOwner(t *testing.T) {
	t.Parallel()
	_, err := parseRepoSlug("https://github.com/.git")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
}

func TestParseRepoSlug_EmptyRepo(t *testing.T) {
	t.Parallel()
	_, err := parseRepoSlug("https://github.com/owner/.git")
	if err == nil {
		t.Fatal("expected error for empty repo name")
	}
}

// --- FetchPRDetails tests (0% -> 100%) ---

func TestFetchPRDetails_EmptyURL(t *testing.T) {
	t.Parallel()
	_, err := FetchPRDetails(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFetchPRDetails_WhitespaceURL(t *testing.T) {
	t.Parallel()
	_, err := FetchPRDetails(context.Background(), "   ", "")
	if err == nil {
		t.Fatal("expected error for whitespace URL")
	}
}

func TestFetchPRDetails_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("expected Accept header, got %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"number": 42,
			"state": "open",
			"head": {
				"label": "owner:feature-branch",
				"ref": "feature-branch",
				"sha": "abc123def456"
			},
			"html_url": "https://github.com/owner/repo/pull/42"
		}`))
	}))
	defer srv.Close()

	details, err := FetchPRDetails(context.Background(), srv.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Number != 42 {
		t.Errorf("expected number 42, got %d", details.Number)
	}
	if details.HeadBranch != "feature-branch" {
		t.Errorf("expected 'feature-branch', got %q", details.HeadBranch)
	}
	if details.HeadSHA != "abc123def456" {
		t.Errorf("expected 'abc123def456', got %q", details.HeadSHA)
	}
	if details.HeadLabel != "owner:feature-branch" {
		t.Errorf("expected 'owner:feature-branch', got %q", details.HeadLabel)
	}
	if details.State != "open" {
		t.Errorf("expected 'open', got %q", details.State)
	}
}

func TestFetchPRDetails_NoToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":1,"state":"closed","head":{"label":"o:b","ref":"b","sha":"sha1"}}`))
	}))
	defer srv.Close()

	details, err := FetchPRDetails(context.Background(), srv.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Number != 1 {
		t.Errorf("expected 1, got %d", details.Number)
	}
	if details.State != "closed" {
		t.Errorf("expected 'closed', got %q", details.State)
	}
}

func TestFetchPRDetails_WhitespaceToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header for whitespace token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":1,"state":"closed","head":{"label":"o:b","ref":"b","sha":"sha1"}}`))
	}))
	defer srv.Close()

	details, err := FetchPRDetails(context.Background(), srv.URL, "   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Number != 1 {
		t.Errorf("expected 1, got %d", details.Number)
	}
}

func TestFetchPRDetails_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "bad-token")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestFetchPRDetails_OtherHTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "not found"}`))
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestFetchPRDetails_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	_, err := FetchPRDetails(context.Background(), srv.URL, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestFetchPRDetails_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchPRDetails(ctx, "https://github.com/owner/repo/pulls/1", "")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// --- DetectRepoSlug tests (0% -> 100%) ---

func TestDetectRepoSlug_EmptyPath(t *testing.T) {
	t.Parallel()
	_, err := DetectRepoSlug("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestDetectRepoSlug_WhitespacePath(t *testing.T) {
	t.Parallel()
	_, err := DetectRepoSlug("   ")
	if err == nil {
		t.Fatal("expected error for whitespace path")
	}
}

func TestDetectRepoSlug_NotARepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := DetectRepoSlug(dir)
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
	var opErr *OpError
	if !errors.As(err, &opErr) {
		t.Errorf("expected OpError, got %T: %v", err, err)
	}
}

func TestDetectRepoSlug_NoOriginRemote(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	// Add origin first, then remove it to test the no-origin case.
	mustRun(t, repo, "remote", "add", "origin", "https://github.com/owner/repo.git")
	mustRun(t, repo, "remote", "remove", "origin")
	defaultCache.Clear()

	_, err := DetectRepoSlug(repo)
	if err == nil {
		t.Fatal("expected error when no origin remote")
	}
}

func TestDetectRepoSlug_OriginHTTPS(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	mustRun(t, repo, "remote", "add", "origin", "https://github.com/owner/repo.git")
	defaultCache.Clear()

	slug, err := DetectRepoSlug(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestDetectRepoSlug_OriginSSH(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	mustRun(t, repo, "remote", "add", "origin", "git@github.com:owner/repo.git")
	defaultCache.Clear()

	slug, err := DetectRepoSlug(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slug != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", slug)
	}
}

func TestDetectRepoSlug_OriginSSH_Invalid(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	mustRun(t, repo, "remote", "add", "origin", "git@github.com/owner/repo")
	defaultCache.Clear()

	_, err := DetectRepoSlug(repo)
	if err == nil {
		t.Fatal("expected error for SSH URL without colon")
	}
}

// --- EqualPath with symlinks ---

func TestEqualPath_ViaSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if !EqualPath(target, link) {
		t.Error("EqualPath should return true for symlink to same dir")
	}
}

func TestEqualPath_Nonexistent(t *testing.T) {
	t.Parallel()
	if !EqualPath("/tmp/nonexistent/a", "/tmp/nonexistent/a") {
		t.Error("EqualPath should return true for identical nonexistent paths")
	}
}

// --- RemoveWorktree nonexistent path ---

func TestRemoveWorktree_NonexistentPath(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	err := RemoveWorktree(context.Background(), repo, "/nonexistent/worktree/path", false)
	if err != nil {
		t.Errorf("expected no error for nonexistent worktree path, got: %v", err)
	}
}

func TestRemoveWorktree_EmptyRepoPath(t *testing.T) {
	t.Parallel()
	err := RemoveWorktree(context.Background(), "", "/some/path", false)
	if err == nil {
		t.Fatal("expected error for empty repo path")
	}
}

// --- CommitWorktree additional paths ---

func TestCommitWorktree_NoChanges(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	err := CommitWorktree(context.Background(), repo, "no changes")
	if err == nil {
		t.Fatal("expected error when no changes to commit")
	}
}

func TestCommitWorktree_WithChanges(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("# modified\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := CommitWorktree(context.Background(), repo, "modify readme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CreateWorktree additional paths ---

func TestCreateWorktree_InvalidBranchName_Gap(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	_, err := CreateWorktree(context.Background(), repo, "invalid branch!", "main", false)
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestCreateWorktree_NoCreateBranch(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	mustRun(t, repo, "branch", "existing-branch")
	defaultCache.Clear()

	wtPath, err := CreateWorktree(context.Background(), repo, "existing-branch", "main", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Errorf("worktree directory should exist at %s", wtPath)
	}
}

func TestCreateWorktree_DefaultBaseBranch(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	wtPath, err := CreateWorktree(context.Background(), repo, "auto-base-branch", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Errorf("worktree directory should exist at %s", wtPath)
	}
}

// --- CreateBranch error paths ---

func TestCreateBranch_NotARepo(t *testing.T) {
	t.Parallel()
	err := CreateBranch(context.Background(), t.TempDir(), "new-branch")
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

// --- DeleteBranch error paths ---

func TestDeleteBranch_NotARepo(t *testing.T) {
	t.Parallel()
	err := DeleteBranch(context.Background(), t.TempDir(), "branch")
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

func TestDeleteBranch_NonexistentBranch(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	defaultCache.Clear()

	err := DeleteBranch(context.Background(), repo, "nonexistent-branch")
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

// --- GetWorktreeStatus error paths ---

func TestGetWorktreeStatus_InvalidPath(t *testing.T) {
	t.Parallel()
	_, err := GetWorktreeStatus(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
