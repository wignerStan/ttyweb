package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a git repo with an initial commit, returns its path.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	workPath := filepath.Join(dir, "work")

	if err := os.MkdirAll(workPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a working repo and make an initial commit so we have a main branch.
	mustRun(t, workPath, "init")
	mustRun(t, workPath, "config", "user.name", "test")
	mustRun(t, workPath, "config", "user.email", "test@test.com")
	mustWriteFile(t, filepath.Join(workPath, "README.md"), "# test\n")
	mustRun(t, workPath, "add", ".")
	mustRun(t, workPath, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "init")

	return workPath
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	// Prepend "git" since all calls pass git subcommands.
	cmd := exec.Command("git", append([]string{name}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %s\n%s", name, strings.Join(args, " "), err, string(out))
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "main", false},
		{"valid with slash", "feature/my-thing", false},
		{"valid with dot", "release/v1.0", false},
		{"empty", "", true},
		{"spaces only", "   ", true},
		{"starts with dot", ".hidden", true},
		{"has spaces", "my branch", true},
		{"has special chars", "feat!@#", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBranchName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBranchName(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	repo := initTestRepo(t)

	if !IsGitRepo(repo) {
		t.Error("expected repo to be detected as git repo")
	}
	if IsGitRepo(dir) {
		t.Error("expected empty dir to not be a git repo")
	}
	if IsGitRepo("/nonexistent/path") {
		t.Error("expected nonexistent path to not be a git repo")
	}
}

func TestListWorktrees(t *testing.T) {
	repo := initTestRepo(t)

	worktrees, err := ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}

	if len(worktrees) == 0 {
		t.Fatal("expected at least one worktree (main)")
	}

	// The first worktree should be the main one.
	var foundMain bool
	for _, wt := range worktrees {
		if wt.IsMain {
			foundMain = true
			if wt.Branch != "main" && wt.Branch != "master" {
				t.Errorf("main worktree branch = %q, want main/master", wt.Branch)
			}
			if wt.HeadCommit == "" {
				t.Error("main worktree has no head commit")
			}
		}
	}
	if !foundMain {
		t.Error("expected to find main worktree")
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	repo := initTestRepo(t)

	wtPath, err := CreateWorktree(repo, "feature-test", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s: %v", wtPath, err)
	}

	// Verify it shows up in list.
	worktrees, err := ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees after create failed: %v", err)
	}

	var found bool
	for _, wt := range worktrees {
		if wt.Branch == "feature-test" {
			found = true
			if wt.IsMain {
				t.Error("feature worktree should not be main")
			}
		}
	}
	if !found {
		t.Error("created worktree not found in list")
	}

	// Remove it.
	if err := RemoveWorktree(repo, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree path should no longer exist after removal")
	}
}

func TestCreateWorktreeExistingBranch(t *testing.T) {
	repo := initTestRepo(t)

	// Create a branch first.
	mustRun(t, repo, "branch", "existing-branch")

	wtPath, err := CreateWorktree(repo, "existing-branch", "main", false)
	if err != nil {
		t.Fatalf("CreateWorktree with existing branch failed: %v", err)
	}

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s: %v", wtPath, err)
	}

	// Clean up.
	_ = RemoveWorktree(repo, wtPath, false)
}

func TestGetWorktreeStatus(t *testing.T) {
	repo := initTestRepo(t)

	// Clean repo should have all zeroes.
	status, err := GetWorktreeStatus(repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus failed: %v", err)
	}
	if status.Modified != 0 || status.Staged != 0 || status.Untracked != 0 {
		t.Errorf("clean repo should have zero status, got: %+v", status)
	}

	// Make a modification.
	mustWriteFile(t, filepath.Join(repo, "changed.txt"), "new content\n")
	status, err = GetWorktreeStatus(repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus after modification failed: %v", err)
	}
	if status.Untracked < 1 {
		t.Errorf("expected untracked file, got untracked=%d", status.Untracked)
	}
}

func TestCommitWorktree(t *testing.T) {
	repo := initTestRepo(t)

	// Write a new file and commit.
	mustWriteFile(t, filepath.Join(repo, "test.txt"), "hello\n")
	if err := CommitWorktree(repo, "add test file"); err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// Verify file is committed (status should be clean now).
	status, err := GetWorktreeStatus(repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus after commit failed: %v", err)
	}
	if status.Untracked != 0 || status.Modified != 0 {
		t.Errorf("expected clean status after commit, got: %+v", status)
	}
}

func TestCommitWorktreeClean(t *testing.T) {
	repo := initTestRepo(t)

	err := CommitWorktree(repo, "should fail")
	if err == nil {
		t.Fatal("expected error when committing clean tree")
	}
}

func TestCommitWorktreeEmptyMessage(t *testing.T) {
	repo := initTestRepo(t)

	err := CommitWorktree(repo, "")
	if err == nil {
		t.Fatal("expected error for empty commit message")
	}
}

func TestSyncWorktrees(t *testing.T) {
	repo := initTestRepo(t)

	worktrees, err := ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees (sync) failed: %v", err)
	}
	if len(worktrees) == 0 {
		t.Error("expected at least one worktree after sync")
	}
}

func TestRemoveNonexistentWorktree(t *testing.T) {
	repo := initTestRepo(t)

	// Removing a nonexistent path should not error (prune handles it).
	err := RemoveWorktree(repo, "/nonexistent/path/nowhere", false)
	if err != nil {
		t.Fatalf("RemoveWorktree nonexistent should not error, got: %v", err)
	}
}

func TestCreateWorktreeInvalidBranch(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CreateWorktree(repo, "", "main", true)
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestListWorktreesInvalidPath(t *testing.T) {
	_, err := ListWorktrees("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	_, err = ListWorktrees("/nonexistent")
	// May not error depending on go-git behavior, but shouldn't panic.
	_ = err
}

func TestGetWorktreeStatusInvalidPath(t *testing.T) {
	_, err := GetWorktreeStatus("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main", "main"},
		{"feature/test", "feature__test"},
		{"release/v1.0", "release__v1.0"},
		{"bug: fix stuff", "bug_ fix stuff"},
		{"wild*card", "wild_card"},
	}

	for _, tc := range tests {
		got := sanitizeBranchName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
