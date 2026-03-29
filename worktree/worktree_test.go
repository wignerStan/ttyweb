package worktree

import (
	"context"
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
	cmd := exec.CommandContext(context.Background(), "git", append([]string{name}, args...)...)
	cmd.Dir = dir
	cmd.Env = FilterGitEnv(os.Environ())
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

	worktrees, err := ListWorktrees(context.Background(), repo)
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

	wtPath, err := CreateWorktree(context.Background(), repo, "feature-test", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s: %v", wtPath, err)
	}

	// Verify it shows up in list.
	worktrees, err := ListWorktrees(context.Background(), repo)
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
	if err := RemoveWorktree(context.Background(), repo, wtPath, false); err != nil {
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

	wtPath, err := CreateWorktree(context.Background(), repo, "existing-branch", "main", false)
	if err != nil {
		t.Fatalf("CreateWorktree with existing branch failed: %v", err)
	}

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s: %v", wtPath, err)
	}

	// Clean up.
	_ = RemoveWorktree(context.Background(), repo, wtPath, false)
}

func TestGetWorktreeStatus(t *testing.T) {
	repo := initTestRepo(t)

	// Clean repo should have all zeroes.
	status, err := GetWorktreeStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus failed: %v", err)
	}
	if status.Modified != 0 || status.Staged != 0 || status.Untracked != 0 {
		t.Errorf("clean repo should have zero status, got: %+v", status)
	}

	// Make a modification.
	mustWriteFile(t, filepath.Join(repo, "changed.txt"), "new content\n")
	status, err = GetWorktreeStatus(context.Background(), repo)
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
	if err := CommitWorktree(context.Background(), repo, "add test file"); err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// Verify file is committed (status should be clean now).
	status, err := GetWorktreeStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus after commit failed: %v", err)
	}
	if status.Untracked != 0 || status.Modified != 0 {
		t.Errorf("expected clean status after commit, got: %+v", status)
	}
}

func TestCommitWorktreeClean(t *testing.T) {
	repo := initTestRepo(t)

	err := CommitWorktree(context.Background(), repo, "should fail")
	if err == nil {
		t.Fatal("expected error when committing clean tree")
	}
}

func TestCommitWorktreeEmptyMessage(t *testing.T) {
	repo := initTestRepo(t)

	err := CommitWorktree(context.Background(), repo, "")
	if err == nil {
		t.Fatal("expected error for empty commit message")
	}
}

func TestSyncWorktrees(t *testing.T) {
	repo := initTestRepo(t)

	worktrees, err := ListWorktrees(context.Background(), repo)
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
	err := RemoveWorktree(context.Background(), repo, "/nonexistent/path/nowhere", false)
	if err != nil {
		t.Fatalf("RemoveWorktree nonexistent should not error, got: %v", err)
	}
}

func TestCreateWorktreeInvalidBranch(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CreateWorktree(context.Background(), repo, "", "main", true)
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestListWorktreesInvalidPath(t *testing.T) {
	_, err := ListWorktrees(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	_, err = ListWorktrees(context.Background(), "/nonexistent")
	// May not error depending on go-git behavior, but shouldn't panic.
	_ = err
}

func TestGetWorktreeStatusInvalidPath(t *testing.T) {
	_, err := GetWorktreeStatus(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCreateWorktree_EmptyPath(t *testing.T) {
	_, err := CreateWorktree(context.Background(), "", "feature-1", "main", true)
	if err == nil {
		t.Fatal("expected error for empty repo path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestCreateWorktree_InvalidBranch(t *testing.T) {
	repo := initTestRepo(t)

	_, err := CreateWorktree(context.Background(), repo, "invalid branch!", "main", true)
	if err == nil {
		t.Fatal("expected error for branch with spaces and special chars")
	}
}

func TestCreateWorktree_WithCreateBranch(t *testing.T) {
	repo := initTestRepo(t)

	wtPath, err := CreateWorktree(context.Background(), repo, "new-feature", "main", true)
	if err != nil {
		t.Fatalf("CreateWorktree with createBranch failed: %v", err)
	}

	// Verify the worktree path exists.
	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path does not exist: %s: %v", wtPath, err)
	}

	// Verify the branch was created from main.
	worktrees, err := ListWorktrees(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	var found bool
	for _, wt := range worktrees {
		if wt.Branch == "new-feature" {
			found = true
		}
	}
	if !found {
		t.Error("created branch not found in worktree list")
	}

	// Clean up.
	_ = RemoveWorktree(context.Background(), repo, wtPath, false)
}

func TestCreateWorktree_ExistingBranch(t *testing.T) {
	repo := initTestRepo(t)

	// Pre-create a branch so createBranch=true should fail.
	mustRun(t, repo, "branch", "pre-existing")

	_, err := CreateWorktree(context.Background(), repo, "pre-existing", "main", true)
	if err == nil {
		t.Fatal("expected error when branch already exists and createBranch is true")
	}

	// But createBranch=false should work.
	wtPath, err := CreateWorktree(context.Background(), repo, "pre-existing", "main", false)
	if err != nil {
		t.Fatalf("CreateWorktree with existing branch and createBranch=false failed: %v", err)
	}
	_ = RemoveWorktree(context.Background(), repo, wtPath, false)
}

func TestRemoveWorktree_EmptyPath(t *testing.T) {
	err := RemoveWorktree(context.Background(), "", "/some/path", false)
	if err == nil {
		t.Fatal("expected error for empty repo path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestRemoveWorktree_EmptyWorktreePath(t *testing.T) {
	repo := initTestRepo(t)

	err := RemoveWorktree(context.Background(), repo, "", false)
	if err == nil {
		t.Fatal("expected error for empty worktree path")
	}
	if err.Error() != "worktree path is required" {
		t.Errorf("expected 'worktree path is required', got %q", err.Error())
	}
}

func TestRemoveWorktree_NotFound(t *testing.T) {
	repo := initTestRepo(t)

	// Removing a nonexistent worktree path should not error (prune handles it).
	err := RemoveWorktree(context.Background(), repo, filepath.Join(repo, "nonexistent-wt"), false)
	if err != nil {
		t.Fatalf("RemoveWorktree nonexistent should not error, got: %v", err)
	}
}

func TestGetWorktreeStatus_EmptyPath(t *testing.T) {
	_, err := GetWorktreeStatus(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "worktree path is required" {
		t.Errorf("expected 'worktree path is required', got %q", err.Error())
	}
}

func TestGetWorktreeStatus_WithChanges(t *testing.T) {
	repo := initTestRepo(t)

	// Create an untracked file.
	mustWriteFile(t, filepath.Join(repo, "newfile.txt"), "hello\n")
	status, err := GetWorktreeStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus failed: %v", err)
	}
	if status.Untracked < 1 {
		t.Errorf("expected at least 1 untracked file, got untracked=%d", status.Untracked)
	}
}

func TestGetWorktreeStatus_CleanRepo(t *testing.T) {
	repo := initTestRepo(t)

	status, err := GetWorktreeStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus failed: %v", err)
	}
	if status.Modified != 0 || status.Staged != 0 || status.Untracked != 0 || status.Conflicts != 0 {
		t.Errorf("clean repo should have zero status, got: %+v", status)
	}
}

func TestCommitWorktree_EmptyPath(t *testing.T) {
	err := CommitWorktree(context.Background(),"", "some message")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "worktree path is required" {
		t.Errorf("expected 'worktree path is required', got %q", err.Error())
	}
}

func TestCommitWorktree_EmptyMessage(t *testing.T) {
	dir := t.TempDir()
	err := CommitWorktree(context.Background(),dir, "")
	if err == nil {
		t.Fatal("expected error for empty message")
	}
	if err.Error() != "commit message is required" {
		t.Errorf("expected 'commit message is required', got %q", err.Error())
	}
}

func TestCommitWorktree_Success(t *testing.T) {
	repo := initTestRepo(t)

	// Create and stage a new file.
	mustWriteFile(t, filepath.Join(repo, "success.txt"), "data\n")
	mustRun(t, repo, "add", "success.txt")

	err := CommitWorktree(context.Background(),repo, "add success file")
	if err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// Verify the repo is clean.
	status, err := GetWorktreeStatus(context.Background(), repo)
	if err != nil {
		t.Fatalf("GetWorktreeStatus after commit failed: %v", err)
	}
	if status.Untracked != 0 || status.Modified != 0 || status.Staged != 0 {
		t.Errorf("expected clean status after commit, got: %+v", status)
	}
}

func TestCommitWorktree_NothingToCommit(t *testing.T) {
	repo := initTestRepo(t)

	err := CommitWorktree(context.Background(),repo, "should fail - clean tree")
	if err == nil {
		t.Fatal("expected error when committing clean tree")
	}
	if err.Error() != "nothing to commit: working tree clean" {
		t.Errorf("expected 'nothing to commit: working tree clean', got %q", err.Error())
	}
}

func TestFilterGitEnv(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"GIT_DIR=/some/repo/.git",
		"PATH=/usr/bin",
		"GIT_WORK_TREE=/some/repo",
		"GIT_COMMON_DIR=/some/repo/.git/objects",
		"GIT_OBJECT_DIRECTORY=/some/repo/.git/objects",
		"GIT_INDEX_FILE=/some/repo/.git/index",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES=/alt/objects",
		"GIT_TERMINAL_PROMPT=0",
	}
	filtered := FilterGitEnv(env)

	for _, e := range filtered {
		key, _, _ := strings.Cut(e, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
			"GIT_INDEX_FILE", "GIT_ALTERNATE_OBJECT_DIRECTORIES":
			t.Errorf("expected %s to be filtered out, got %q", key, e)
		}
	}

	expected := []string{"HOME=/home/user", "PATH=/usr/bin", "GIT_TERMINAL_PROMPT=0"}
	if len(filtered) != len(expected) {
		t.Errorf("expected %d entries, got %d: %v", len(expected), len(filtered), filtered)
	}
	for _, want := range expected {
		found := false
		for _, got := range filtered {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in filtered env", want)
		}
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
