package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

// headHashRe matches a 40-char hex commit hash in the JSON output.
var headHashRe = regexp.MustCompile(`"head_hash":"[0-9a-f]{40}"`)

// normalizeBranchList replaces non-deterministic fields (commit hashes) with
// placeholder values so the golden file comparison is stable across runs.
func normalizeBranchList(data []byte) []byte {
	return headHashRe.ReplaceAll(data, []byte(`"head_hash":"PLACEHOLDER_HASH"`))
}

// initTestGitRepo creates a temporary git repo with an initial commit on "main"
// using the git CLI, consistent with the worktree package's test helpers.
func initTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	mustGit(t, dir, "init", "-b", "main")
	mustGit(t, dir, "config", "user.name", "test")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	if err := exec.CommandContext( //nolint:gosec // reason: test code
		context.Background(), "sh", "-c", "echo init > "+filepath.Join(dir, "init.txt"),
	).Run(); err != nil {
		t.Fatalf("write init.txt: %v", err)
	}
	mustGit(t, dir, "add", "init.txt")
	mustGit(t, dir, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "init")

	return dir
}

func mustGit(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{name}, args...)...) //nolint:gosec // reason: test code
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", name, err, out)
	}
}

func TestGolden_BranchList(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/branches?repo="+repoDir, nil,
	)
	handleBranches(rec, req)

	compareGolden(t, normalizeBranchList(rec.Body.Bytes()))
}

func TestGolden_BranchList_NoRepo(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/branches", nil,
	)
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchCreate(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"name":"test-branch"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchCreate_InvalidName(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"name":""}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchCreate_BadJSON(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchDetail_NotFound(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/branches/nonexistent?repo="+repoDir, nil,
	)
	makeBranchDetailHandler("/api/branches")(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchMerge_BadJSON(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`not json`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	makeBranchDetailHandler("/api/branches")(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchMerge_MissingFields(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	body := bytes.NewReader([]byte(`{"source":"a"}`))
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/branches/merge?repo="+repoDir, body,
	)
	req.Header.Set("Content-Type", "application/json")
	makeBranchDetailHandler("/api/branches")(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchDelete_NotFound(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/branches/nonexistent?repo="+repoDir, nil,
	)
	makeBranchDetailHandler("/api/branches")(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchDelete_InvalidName(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/branches/!bad?repo="+repoDir, nil,
	)
	makeBranchDetailHandler("/api/branches")(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_BranchList_MethodNotAllowed(t *testing.T) {
	repoDir := initTestGitRepo(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPut,
		"/api/branches?repo="+repoDir, nil,
	)
	handleBranches(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
