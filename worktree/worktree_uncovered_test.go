package worktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSetTestEnv(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	custom := []string{"GIT_AUTHOR_NAME=test-bot"}
	SetTestEnv(custom)
	defer SetTestEnv(nil)

	// Verify the env override is applied by creating a git command in a repo
	// and checking that the env is appended. We verify via newGitCmd indirectly.
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	env := cmd.Env
	found := false
	for _, e := range env {
		if e == "GIT_AUTHOR_NAME=test-bot" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected custom env var in git command")
	}
}

func TestSetTestEnv_Nil(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	SetTestEnv(nil)
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	// Should still have standard env vars.
	found := false
	for _, e := range cmd.Env {
		if e == "GIT_TERMINAL_PROMPT=0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected standard env var in git command")
	}
}

func TestSetTestEnv_Empty(t *testing.T) {
	// Not parallel: SetTestEnv modifies global state.
	SetTestEnv([]string{})
	defer SetTestEnv(nil)
	// Empty slice should not be appended (len check > 0).
	repo := initTestRepo(t)
	cmd := newGitCmd(repo, "rev-parse", "HEAD")
	// Standard env should still be present.
	found := false
	for _, e := range cmd.Env {
		if e == "GIT_TERMINAL_PROMPT=0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected standard env var in git command")
	}
}

func TestResolveDefaultBranch_Main(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	branch := resolveDefaultBranch(context.Background(), repo)
	if branch != "main" {
		t.Fatalf("expected 'main', got %q", branch)
	}
}

func TestResolveDefaultBranch_NoBranch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	branch := resolveDefaultBranch(context.Background(), dir)
	if branch != "" {
		t.Fatalf("expected empty string for non-repo, got %q", branch)
	}
}

func TestCollectStatusGoGit(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Modified != 0 || status.Staged != 0 || status.Untracked != 0 {
		t.Errorf("clean repo should have zero status, got: %+v", status)
	}
}

func TestCollectStatusGoGit_WithUntracked(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	mustWriteFile(t, filepath.Join(repo, "untracked.txt"), "new\n")
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Untracked < 1 {
		t.Errorf("expected untracked file, got untracked=%d", status.Untracked)
	}
}

func TestCollectStatusGoGit_WithModification(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("# modified\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	status, err := collectStatusGoGit(repo)
	if err != nil {
		t.Fatalf("collectStatusGoGit failed: %v", err)
	}
	if status.Modified < 1 {
		t.Errorf("expected modified file, got modified=%d", status.Modified)
	}
}

func TestCollectStatusGoGit_InvalidPath(t *testing.T) {
	t.Parallel()
	_, err := collectStatusGoGit("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestNewGitCmd_EmptyDir(t *testing.T) {
	t.Parallel()
	cmd := newGitCmd("", "status")
	if cmd.Dir != "" {
		t.Fatalf("expected empty dir, got %q", cmd.Dir)
	}
	if cmd.Path == "" {
		t.Fatal("expected non-empty command path")
	}
}

func TestParsePorcelainStatus_Empty(t *testing.T) {
	t.Parallel()
	status := parsePorcelainStatus("")
	if status.Ahead != 0 || status.Behind != 0 || status.Modified != 0 ||
		status.Staged != 0 || status.Untracked != 0 || status.Conflicts != 0 {
		t.Errorf("empty input should give zero status, got: %+v", status)
	}
}

func TestParsePorcelainStatus_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	status := parsePorcelainStatus("   \n  \n")
	if status.Ahead != 0 || status.Behind != 0 || status.Modified != 0 ||
		status.Staged != 0 || status.Untracked != 0 || status.Conflicts != 0 {
		t.Errorf("whitespace-only input should give zero status, got: %+v", status)
	}
}

func TestParsePorcelainStatus_BranchHeader(t *testing.T) {
	t.Parallel()
	output := "# branch.ab +3 -2\n"
	status := parsePorcelainStatus(output)
	if status.Ahead != 3 {
		t.Errorf("expected ahead=3, got %d", status.Ahead)
	}
	if status.Behind != 2 {
		t.Errorf("expected behind=2, got %d", status.Behind)
	}
}

func TestParsePorcelainStatus_Untracked(t *testing.T) {
	t.Parallel()
	output := "? newfile.txt\n"
	status := parsePorcelainStatus(output)
	if status.Untracked != 1 {
		t.Errorf("expected untracked=1, got %d", status.Untracked)
	}
}

func TestParsePorcelainStatus_MultipleUntracked(t *testing.T) {
	t.Parallel()
	output := "? file1.txt\n? file2.txt\n? file3.txt\n"
	status := parsePorcelainStatus(output)
	if status.Untracked != 3 {
		t.Errorf("expected untracked=3, got %d", status.Untracked)
	}
}

func TestParsePorcelainStatus_Staged(t *testing.T) {
	t.Parallel()
	// In porcelain v2, type 1 format is: "1 XY sub_module mH mI mW hH hI path"
	// The XY field is always exactly 2 chars. A single char like "M" is short and skipped.
	output := "1 M. N... 000000 100644 100644 file.txt\n"
	status := parsePorcelainStatus(output)
	if status.Staged != 1 {
		t.Errorf("expected staged=1, got %d", status.Staged)
	}
	if status.Modified != 0 {
		t.Errorf("expected modified=0 for staged-only change, got %d", status.Modified)
	}
}

func TestParsePorcelainStatus_Modified(t *testing.T) {
	t.Parallel()
	output := "1 .M N... 000000 100644 100644 file.txt\n"
	status := parsePorcelainStatus(output)
	if status.Modified != 1 {
		t.Errorf("expected modified=1, got %d", status.Modified)
	}
	if status.Staged != 0 {
		t.Errorf("expected staged=0 for modified-only change, got %d", status.Staged)
	}
}

func TestParsePorcelainStatus_StagedAndModified(t *testing.T) {
	t.Parallel()
	output := "1 MM file.txt\n"
	status := parsePorcelainStatus(output)
	if status.Staged != 1 {
		t.Errorf("expected staged=1, got %d", status.Staged)
	}
	if status.Modified != 1 {
		t.Errorf("expected modified=1, got %d", status.Modified)
	}
}

func TestParsePorcelainStatus_Conflict(t *testing.T) {
	t.Parallel()
	output := "u UU file.txt\n"
	status := parsePorcelainStatus(output)
	if status.Conflicts != 1 {
		t.Errorf("expected conflicts=1, got %d", status.Conflicts)
	}
}

func TestParsePorcelainStatus_RenamedEntry(t *testing.T) {
	t.Parallel()
	output := "2 R. N... 000000 100644 100644 oldname.txt newname.txt\n"
	status := parsePorcelainStatus(output)
	if status.Staged != 1 {
		t.Errorf("expected staged=1 for rename, got %d", status.Staged)
	}
}

func TestParsePorcelainStatus_ConflictInTrackedLine(t *testing.T) {
	t.Parallel()
	// Tracked line (type 1) with U in staging area.
	output := "1 UU file.txt\n"
	status := parsePorcelainStatus(output)
	if status.Conflicts != 1 {
		t.Errorf("expected conflicts=1, got %d", status.Conflicts)
	}
}

func TestParseTrackedLine_Short(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{}
	parseTrackedLine(status, "1")
	if status.Staged != 0 || status.Modified != 0 || status.Conflicts != 0 {
		t.Errorf("short line should not modify status, got: %+v", status)
	}
}

func TestParseTrackedLine_SingleXY(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{}
	// fields[1] = "M" (only one char, len < 2)
	parseTrackedLine(status, "1 M")
	if status.Staged != 0 || status.Modified != 0 {
		t.Errorf("single-char XY should not modify status, got: %+v", status)
	}
}

func TestParseTrackedLine_Unmodified(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{}
	// ".." means no change in either area.
	parseTrackedLine(status, "1 .. file.txt")
	if status.Staged != 0 || status.Modified != 0 {
		t.Errorf("unmodified file should not increment counters, got: %+v", status)
	}
}

func TestParseTrackedLine_WorktreeConflict(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{}
	parseTrackedLine(status, "1 .U file.txt")
	if status.Conflicts != 1 {
		t.Errorf("expected conflicts=1 for worktree conflict, got %d", status.Conflicts)
	}
}

func TestParseStatusHeader_Empty(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{Ahead: 5, Behind: 3}
	parseStatusHeader(status, "")
	if status.Ahead != 5 || status.Behind != 3 {
		t.Errorf("empty header should not modify status, got: %+v", status)
	}
}

func TestParseStatusHeader_UnknownKey(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{Ahead: 5, Behind: 3}
	parseStatusHeader(status, "unknown.key value")
	if status.Ahead != 5 || status.Behind != 3 {
		t.Errorf("unknown key should not modify status, got: %+v", status)
	}
}

func TestParseStatusHeader_BranchAb_InsufficientFields(t *testing.T) {
	t.Parallel()
	status := &WorktreeStatus{Ahead: 5, Behind: 3}
	// Only 2 fields: "branch.ab" and "+3", missing behind count.
	parseStatusHeader(status, "branch.ab +3")
	if status.Ahead != 5 || status.Behind != 3 {
		t.Errorf("insufficient fields should not modify status, got: %+v", status)
	}
}

func TestParseCount_Invalid(t *testing.T) {
	t.Parallel()
	got := parseCount("abc")
	if got != 0 {
		t.Errorf("expected 0 for invalid count, got %d", got)
	}
}

func TestParseCount_Empty(t *testing.T) {
	t.Parallel()
	got := parseCount("")
	if got != 0 {
		t.Errorf("expected 0 for empty count, got %d", got)
	}
}

func TestParseCount_Negative(t *testing.T) {
	t.Parallel()
	got := parseCount("-5")
	if got != 5 {
		t.Errorf("expected 5 for '-5' (stripped sign), got %d", got)
	}
}

func TestParseCount_Positive(t *testing.T) {
	t.Parallel()
	got := parseCount("+3")
	if got != 3 {
		t.Errorf("expected 3 for '+3', got %d", got)
	}
}

func TestParseCount_PlainNumber(t *testing.T) {
	t.Parallel()
	got := parseCount("42")
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestCollectStatusPorcelain_InvalidPath(t *testing.T) {
	t.Parallel()
	_, err := collectStatusPorcelain("/nonexistent/path/nowhere")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestHeadCommitMessage(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	msg := headCommitMessage(context.Background(),repo)
	if msg == "" {
		t.Error("expected non-empty commit message for repo with commits")
	}
}

func TestHeadCommitMessage_EmptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Init a bare repo with no commits.
	mustRun(t, dir, "init", "--bare")
	msg := headCommitMessage(context.Background(),dir)
	if msg != "" {
		t.Errorf("expected empty message for bare repo with no commits, got %q", msg)
	}
}

func TestRunGitCmd(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	err := runGitCmd(repo, "status")
	if err != nil {
		t.Fatalf("runGitCmd status failed: %v", err)
	}
}

func TestRunGitCmd_Failure(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	err := runGitCmd(repo, "nonexistent-subcommand")
	if err == nil {
		t.Fatal("expected error for invalid git subcommand")
	}
}
