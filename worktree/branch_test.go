package worktree

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepoWithBranches creates a repo with multiple branches and commits
// for testing branch operations. Returns the repo path.
//
// Layout:
//
//	main  : init -> "second commit on main"
//	feature: init -> "feature commit 1" -> "feature commit 2"
//	hotfix: init -> "hotfix commit"
func initTestRepoWithBranches(t *testing.T) string {
	t.Helper()
	repo := initTestRepo(t)

	// Create a second commit on main.
	mustWriteFile(t, filepath.Join(repo, "main.txt"), "main content\n")
	mustRun(t, repo, "add", "main.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "second commit on main")

	// Create feature branch with 2 commits.
	mustRun(t, repo, "checkout", "-b", "feature")
	mustWriteFile(t, filepath.Join(repo, "feature.txt"), "feature content\n")
	mustRun(t, repo, "add", "feature.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "feature commit 1")
	mustWriteFile(t, filepath.Join(repo, "feature2.txt"), "feature2 content\n")
	mustRun(t, repo, "add", "feature2.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "feature commit 2")

	// Create hotfix branch from main.
	mustRun(t, repo, "checkout", "main")
	mustRun(t, repo, "checkout", "-b", "hotfix")
	mustWriteFile(t, filepath.Join(repo, "hotfix.txt"), "hotfix content\n")
	mustRun(t, repo, "add", "hotfix.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "hotfix commit")

	// Switch back to main.
	mustRun(t, repo, "checkout", "main")

	return repo
}

func TestListBranches(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}

	// Should have at least main, feature, and hotfix.
	if len(branches) < 3 {
		t.Fatalf("expected at least 3 branches, got %d: %v", len(branches), branchNames(branches))
	}

	mainBranch := findBranch(t, branches, "main")
	featureBranch := findBranch(t, branches, "feature")
	hotfixBranch := findBranch(t, branches, "hotfix")

	if !mainBranch.IsCurrent {
		t.Error("main should be the current branch")
	}
	if !mainBranch.IsDefault {
		t.Error("main should be marked as default")
	}
	if mainBranch.IsRemote {
		t.Error("main should not be marked as remote")
	}
	if mainBranch.HeadHash == "" {
		t.Error("main branch should have a head hash")
	}

	if featureBranch.IsCurrent {
		t.Error("feature should not be the current branch")
	}
	if featureBranch.IsDefault {
		t.Error("feature should not be marked as default")
	}
	if featureBranch.HeadHash == "" {
		t.Error("feature branch should have a head hash")
	}

	if hotfixBranch.IsCurrent {
		t.Error("hotfix should not be the current branch")
	}
}

func TestListBranches_CurrentBranch(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	// Switch to feature branch.
	mustRun(t, repo, "checkout", "feature")

	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}

	for _, b := range branches {
		if b.Name == "feature" && !b.IsCurrent {
			t.Error("feature should be the current branch after checkout")
		}
		if b.Name == "main" && b.IsCurrent {
			t.Error("main should not be the current branch after switching to feature")
		}
	}
}

func TestListBranches_AheadBehind(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	// Add a remote origin pointing to a bare clone.
	bareDir := t.TempDir()
	mustRun(t, bareDir, "init", "--bare")
	mustRun(t, repo, "remote", "add", "origin", bareDir)
	mustRun(t, repo, "push", "origin", "main")

	// Make a commit on main that's ahead of origin.
	mustWriteFile(t, filepath.Join(repo, "ahead.txt"), "ahead\n")
	mustRun(t, repo, "add", "ahead.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "ahead commit")

	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}

	var mainBranch *BranchInfo
	for i := range branches {
		if branches[i].Name == "main" {
			mainBranch = &branches[i]
			break
		}
	}

	if mainBranch == nil {
		t.Fatal("expected to find main branch")
	}
	if mainBranch.Ahead == 0 {
		t.Error("main should be ahead of origin/main by 1 commit")
	}
	if mainBranch.Behind != 0 {
		t.Errorf("main should be 0 behind origin/main, got %d", mainBranch.Behind)
	}
}

func TestListBranches_EmptyPath(t *testing.T) {
	_, err := ListBranches(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestListBranches_NonexistentPath(t *testing.T) {
	_, err := ListBranches(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestListBranches_SingleBranch(t *testing.T) {
	repo := initTestRepo(t)

	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}

	if len(branches) != 1 {
		t.Fatalf("expected exactly 1 branch, got %d: %v", len(branches), branchNames(branches))
	}
	if branches[0].Name != "main" {
		t.Errorf("expected branch name 'main', got %q", branches[0].Name)
	}
	if !branches[0].IsCurrent {
		t.Error("sole branch should be current")
	}
	if !branches[0].IsDefault {
		t.Error("main branch should be default")
	}
}

func TestCreateBranch(t *testing.T) {
	repo := initTestRepo(t)

	err := CreateBranch(context.Background(), repo, "new-branch")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Verify branch exists via git.
	out := mustOutput(t, repo, "branch", "--list", "new-branch")
	if !strings.Contains(out, "new-branch") {
		t.Errorf("branch 'new-branch' not found in git output: %s", out)
	}

	// Verify it appears in ListBranches.
	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	var found bool
	for _, b := range branches {
		if b.Name == "new-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("new-branch not found in ListBranches")
	}
}

func TestCreateBranch_InvalidName(t *testing.T) {
	repo := initTestRepo(t)

	err := CreateBranch(context.Background(), repo, "invalid branch!")
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
	if !strings.Contains(err.Error(), "invalid branch name") {
		t.Errorf("expected invalid branch name error, got: %v", err)
	}
}

func TestCreateBranch_EmptyName(t *testing.T) {
	repo := initTestRepo(t)

	err := CreateBranch(context.Background(), repo, "")
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}

func TestCreateBranch_AlreadyExists(t *testing.T) {
	repo := initTestRepo(t)

	err := CreateBranch(context.Background(), repo, "main")
	if err == nil {
		t.Fatal("expected error when creating branch that already exists")
	}
	if !errorsIsAlreadyExists(err) {
		t.Errorf("expected already_exists error, got: %v", err)
	}
}

func TestCreateBranch_EmptyPath(t *testing.T) {
	err := CreateBranch(context.Background(), "", "new-branch")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestCreateBranch_NonexistentPath(t *testing.T) {
	err := CreateBranch(context.Background(), "/nonexistent/path", "new-branch")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestDeleteBranch(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	err := DeleteBranch(context.Background(), repo, "hotfix")
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// Verify branch no longer exists.
	out := mustOutput(t, repo, "branch", "--list", "hotfix")
	if strings.Contains(out, "hotfix") {
		t.Errorf("branch 'hotfix' should have been deleted, but found in: %s", out)
	}

	// Verify it's gone from ListBranches.
	branches, err := ListBranches(context.Background(), repo)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	for _, b := range branches {
		if b.Name == "hotfix" {
			t.Error("hotfix should not appear in ListBranches after deletion")
		}
	}
}

func TestDeleteBranch_CurrentBranch(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	// Switch to hotfix, then try to delete it.
	mustRun(t, repo, "checkout", "hotfix")

	err := DeleteBranch(context.Background(), repo, "hotfix")
	if err == nil {
		t.Fatal("expected error when deleting current branch")
	}
}

func TestDeleteBranch_NotFound(t *testing.T) {
	repo := initTestRepo(t)

	err := DeleteBranch(context.Background(), repo, "nonexistent-branch")
	if err == nil {
		t.Fatal("expected error when deleting nonexistent branch")
	}
}

func TestDeleteBranch_DefaultBranch(t *testing.T) {
	repo := initTestRepo(t)

	err := DeleteBranch(context.Background(), repo, "main")
	if err == nil {
		t.Fatal("expected error when deleting default branch")
	}
}

func TestDeleteBranch_InvalidName(t *testing.T) {
	repo := initTestRepo(t)

	err := DeleteBranch(context.Background(), repo, "bad name!")
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestDeleteBranch_EmptyPath(t *testing.T) {
	err := DeleteBranch(context.Background(), "", "some-branch")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestMergeBranch_FastForward(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	// hotfix has 1 commit ahead of main. Merge hotfix into main (fast-forward).
	err := MergeBranch(context.Background(), repo, "hotfix", "main")
	if err != nil {
		t.Fatalf("MergeBranch fast-forward failed: %v", err)
	}

	// Verify main now has the hotfix commit.
	out := mustOutput(t, repo, "log", "--oneline", "main")
	if !strings.Contains(out, "hotfix commit") {
		t.Errorf("main should contain 'hotfix commit' after merge, got: %s", out)
	}
}

func TestMergeBranch_Conflict(t *testing.T) {
	repo := initTestRepo(t)

	// Create a shared file on main, commit.
	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "base version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "base commit")

	// Create a branch from this point.
	mustRun(t, repo, "checkout", "-b", "conflict-branch")

	// Diverge: main modifies the file.
	mustRun(t, repo, "checkout", "main")
	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "main version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "conflict on main")

	// Diverge: conflict-branch modifies the same file differently.
	mustRun(t, repo, "checkout", "conflict-branch")
	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "feature version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "conflict on feature")

	// Merge should fail due to conflict (both branches modified conflict.txt).
	err := MergeBranch(context.Background(), repo, "conflict-branch", "main")
	if err == nil {
		t.Fatal("expected error when merging conflicting branches")
	}
}

func TestMergeBranch_SameBranch(t *testing.T) {
	repo := initTestRepo(t)

	err := MergeBranch(context.Background(), repo, "main", "main")
	if err == nil {
		t.Fatal("expected error when merging a branch into itself")
	}
}

func TestMergeBranch_SourceNotFound(t *testing.T) {
	repo := initTestRepo(t)

	err := MergeBranch(context.Background(), repo, "nonexistent", "main")
	if err == nil {
		t.Fatal("expected error when source branch doesn't exist")
	}
}

func TestMergeBranch_TargetNotFound(t *testing.T) {
	repo := initTestRepo(t)

	err := MergeBranch(context.Background(), repo, "main", "nonexistent")
	if err == nil {
		t.Fatal("expected error when target branch doesn't exist")
	}
}

func TestMergeBranch_EmptyPath(t *testing.T) {
	err := MergeBranch(context.Background(), "", "main", "feature")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error() != "path is required" {
		t.Errorf("expected 'path is required', got %q", err.Error())
	}
}

func TestMergeBranch_InvalidBranchNames(t *testing.T) {
	repo := initTestRepo(t)

	err := MergeBranch(context.Background(), repo, "bad source!", "main")
	if err == nil {
		t.Fatal("expected error for invalid source branch name")
	}

	err = MergeBranch(context.Background(), repo, "main", "bad target!")
	if err == nil {
		t.Fatal("expected error for invalid target branch name")
	}
}

// --- helpers ---

// findBranch finds a branch by name or fails the test.
func findBranch(t *testing.T, branches []BranchInfo, name string) *BranchInfo {
	t.Helper()
	for i := range branches {
		if branches[i].Name == name {
			return &branches[i]
		}
	}
	t.Fatalf("expected to find branch %q in %v", name, branchNames(branches))
	return nil // unreachable
}

func branchNames(branches []BranchInfo) []string {
	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Name
	}
	return names
}

func mustOutput(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{name}, args...)...) //nolint:gosec // G204: test code, args are hardcoded literals
	cmd.Dir = dir
	cmd.Env = FilterGitEnv(cmd.Env)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func errorsIsAlreadyExists(err error) bool {
	var opErr *OpError
	if !isOpError(err, &opErr) {
		return false
	}
	return opErr.Kind == KindAlreadyExists
}

func isOpError(err error, target **OpError) bool {
	if err == nil {
		return false
	}
	if asOpErr, ok := err.(*OpError); ok {
		*target = asOpErr
		return true
	}
	// Check wrapped errors.
	if unwrapped := unwrapErrors(err); unwrapped != nil {
		for _, e := range unwrapped {
			if asOpErr, ok := e.(*OpError); ok {
				*target = asOpErr
				return true
			}
		}
	}
	return false
}

func unwrapErrors(err error) []error {
	var errs []error
	for err != nil {
		errs = append(errs, err)
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return errs
}
