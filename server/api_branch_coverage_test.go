package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGolden_BranchMerge_MissingSource covers the missing source validation path.
func TestGolden_BranchMerge_MissingSource(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"target":"main"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchMerge_MissingTarget covers the missing target validation path.
func TestGolden_BranchMerge_MissingTarget(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"source":"feature"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchMerge_MethodNotAllowed covers non-POST on merge endpoint.
func TestGolden_BranchMerge_MethodNotAllowed(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/branches/merge?repo="+repoDir, nil,
	)
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchDelete_MethodNotAllowed covers non-DELETE on branch delete.
func TestGolden_BranchDelete_MethodNotAllowed(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/branches/some-branch?repo="+repoDir, nil,
	)
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchMerge_MergeFailure covers merge failure when branches don't fast-forward.
func TestGolden_BranchMerge_MergeFailure(t *testing.T) {
	repoDir := initTestGitRepo(t)

	// Create two divergent branches that won't merge cleanly.
	mustGit(t, repoDir, "branch", "diverge-a")
	mustGit(t, repoDir, "checkout", "diverge-a")
	mustGit(t, repoDir, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "--allow-empty", "-m", "commit on a")
	mustGit(t, repoDir, "checkout", "main")
	mustGit(t, repoDir, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "--allow-empty", "-m", "commit on main")

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"source":"diverge-a","target":"main"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchDetail_NoRepo covers missing repo parameter.
func TestGolden_BranchDetail_NoRepo(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/branches/some-branch", nil,
	)
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchMerge_Success covers the successful merge path.
func TestGolden_BranchMerge_Success(t *testing.T) {
	repoDir := initTestGitRepo(t)

	// Create a branch that is ahead of main.
	mustGit(t, repoDir, "branch", "to-merge")
	mustGit(t, repoDir, "checkout", "to-merge")
	mustGit(t, repoDir, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "--allow-empty", "-m", "merge me")
	mustGit(t, repoDir, "checkout", "main")

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"source":"to-merge","target":"main"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchDelete_Success covers the successful branch deletion path.
func TestGolden_BranchDelete_Success(t *testing.T) {
	repoDir := initTestGitRepo(t)

	// Create a branch to delete.
	mustGit(t, repoDir, "branch", "to-delete")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/branches/to-delete?repo="+repoDir, nil,
	)
	handleBranchDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchCreate_Success covers the successful branch creation path.
func TestGolden_BranchCreate_Success(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"name":"success-branch"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_BranchCreate_AlreadyExists covers creating a branch that already exists.
func TestGolden_BranchCreate_AlreadyExists(t *testing.T) {
	repoDir := initTestGitRepo(t)

	// Pre-create the branch.
	mustGit(t, repoDir, "branch", "existing-branch")

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"name":"existing-branch"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
