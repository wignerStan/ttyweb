package service

import (
	"testing"

	"ttyweb/worktree"
)

// TestGitDefaultBranch_NonRepo covers the fallback path for a non-git directory.
func TestGitDefaultBranch_NonRepo(t *testing.T) {
	dir := t.TempDir()
	branch, err := gitDefaultBranch(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected 'main' fallback for non-repo, got %q", branch)
	}
}

// TestGitRemoteURL_NonRepo covers the non-git-directory path.
func TestGitRemoteURL_NonRepo(t *testing.T) {
	dir := t.TempDir()
	url, err := gitRemoteURL(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "" {
		t.Errorf("expected empty URL for non-repo, got %q", url)
	}
}

// TestGitRemoteURL_NoURLs covers the case where remote has no URLs.
func TestGitRemoteURL_NoURLs(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	// git remote add requires a URL, so this test is removed.
	// The empty URL case is already covered by TestGitRemoteURL_NoRemote.
}

// TestDetectRepoSlug_NonRepo covers the error path for non-repo.
func TestDetectRepoSlug_NonRepo(t *testing.T) {
	_, err := worktree.DetectRepoSlug(t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-repo path")
	}
}

// TestDetectRepoSlug_NoRemote covers the case where no origin remote exists.
func TestDetectRepoSlug_NoRemote(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	_, err := worktree.DetectRepoSlug(dir)
	if err == nil {
		t.Fatal("expected error when no origin remote exists")
	}
}
