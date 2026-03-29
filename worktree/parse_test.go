package worktree

import (
	"testing"
)

func TestParseWorktreeListEmpty(t *testing.T) {
	t.Parallel()
	result := parseWorktreeList("")
	if len(result) != 0 {
		t.Errorf("parseWorktreeList(\"\") returned %d items, want 0", len(result))
	}
}

func TestParseWorktreeListWhitespaceOnly(t *testing.T) {
	t.Parallel()
	result := parseWorktreeList("   \n  \n  \n")
	if len(result) != 0 {
		t.Errorf("parseWorktreeList(whitespace) returned %d items, want 0", len(result))
	}
}

func TestParseWorktreeListMalformed(t *testing.T) {
	t.Parallel()
	input := "no_equals_sign_here\njust_garbage\n"
	result := parseWorktreeList(input)
	if len(result) != 0 {
		t.Errorf("parseWorktreeList(malformed) returned %d items, want 0", len(result))
	}
}

func TestParseWorktreeListSingleEntry(t *testing.T) {
	t.Parallel()
	input := `worktree /home/user/project
branch refs/heads/main
HEAD abcdef1234567890
`
	result := parseWorktreeList(input)
	if len(result) != 1 {
		t.Fatalf("parseWorktreeList returned %d items, want 1", len(result))
	}

	wt := result[0]
	if wt.Path != "/home/user/project" {
		t.Errorf("Path = %q, want %q", wt.Path, "/home/user/project")
	}
	if wt.Branch != "main" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "main")
	}
	if wt.HeadCommit != "abcdef1" {
		t.Errorf("HeadCommit = %q, want %q", wt.HeadCommit, "abcdef1")
	}
	if wt.IsMain {
		t.Error("IsMain should be false (parseWorktreeList does not set it)")
	}
}

func TestParseWorktreeListMultipleEntries(t *testing.T) {
	t.Parallel()
	input := `worktree /home/user/project
branch refs/heads/main
HEAD abcdef1234567890

worktree /home/user/project/.worktrees/feature-x
branch refs/heads/feature-x
HEAD fedcba0987654321

worktree /home/user/project/.worktrees/bugfix
detached abcdef1234567890
HEAD 1122334455667788
`
	result := parseWorktreeList(input)
	if len(result) != 3 {
		t.Fatalf("parseWorktreeList returned %d items, want 3", len(result))
	}

	if result[0].Branch != "main" {
		t.Errorf("entry 0 Branch = %q, want %q", result[0].Branch, "main")
	}
	if result[1].Branch != "feature-x" {
		t.Errorf("entry 1 Branch = %q, want %q", result[1].Branch, "feature-x")
	}
	if result[2].Branch != "abcdef1234567890" {
		t.Errorf("entry 2 Branch (detached) = %q, want %q", result[2].Branch, "abcdef1234567890")
	}
}

func TestParseWorktreeListShortHEAD(t *testing.T) {
	t.Parallel()
	input := `worktree /tmp/repo
branch refs/heads/dev
HEAD abc
`
	result := parseWorktreeList(input)
	if len(result) != 1 {
		t.Fatalf("parseWorktreeList returned %d items, want 1", len(result))
	}
	if result[0].HeadCommit != "abc" {
		t.Errorf("HeadCommit = %q, want %q (short HEAD preserved)", result[0].HeadCommit, "abc")
	}
}

func TestParseWorktreeListEmptyHEAD(t *testing.T) {
	t.Parallel()
	input := `worktree /tmp/repo
branch refs/heads/main
HEAD
`
	result := parseWorktreeList(input)
	if len(result) != 1 {
		t.Fatalf("parseWorktreeList returned %d items, want 1", len(result))
	}
	if result[0].HeadCommit != "" {
		t.Errorf("HeadCommit = %q, want empty string", result[0].HeadCommit)
	}
}

func TestParseWorktreeListPathOnly(t *testing.T) {
	t.Parallel()
	// Entry with only path, no branch or HEAD -- should still be included.
	input := `worktree /tmp/repo
`
	result := parseWorktreeList(input)
	if len(result) != 1 {
		t.Fatalf("parseWorktreeList returned %d items, want 1", len(result))
	}
	if result[0].Path != "/tmp/repo" {
		t.Errorf("Path = %q, want %q", result[0].Path, "/tmp/repo")
	}
	if result[0].Branch != "" {
		t.Errorf("Branch = %q, want empty", result[0].Branch)
	}
}

func TestParsePorcelainStatusEmpty(t *testing.T) {
	t.Parallel()
	result := parsePorcelainStatus("")
	if result.Ahead != 0 || result.Behind != 0 || result.Modified != 0 ||
		result.Staged != 0 || result.Untracked != 0 || result.Conflicts != 0 {
		t.Errorf("parsePorcelainStatus(\"\") = %+v, want all zeroes", result)
	}
}

func TestParsePorcelainStatusWhitespace(t *testing.T) {
	t.Parallel()
	result := parsePorcelainStatus("   \n  \n  ")
	if result.Ahead != 0 || result.Behind != 0 || result.Modified != 0 {
		t.Errorf("parsePorcelainStatus(whitespace) = %+v, want all zeroes", result)
	}
}

func TestParsePorcelainStatusUntracked(t *testing.T) {
	t.Parallel()
	input := `? newfile.txt
? another_untracked.go
`
	result := parsePorcelainStatus(input)
	if result.Untracked != 2 {
		t.Errorf("Untracked = %d, want 2", result.Untracked)
	}
}

func TestParsePorcelainStatusStaged(t *testing.T) {
	t.Parallel()
	input := `1 M. N... 100644 100644 100644 abc1234_0 abc1234_1 modified.txt
1 A. N... 000000 100644 000000 0000000_0 def5678_0 newfile.go
`
	result := parsePorcelainStatus(input)
	if result.Staged != 2 {
		t.Errorf("Staged = %d, want 2", result.Staged)
	}
	if result.Modified != 0 {
		t.Errorf("Modified = %d, want 0 (only staged)", result.Modified)
	}
}

func TestParsePorcelainStatusModified(t *testing.T) {
	t.Parallel()
	input := `1 .M N... 100644 100644 100644 abc1234_0 abc1234_1 changed.go
`
	result := parsePorcelainStatus(input)
	if result.Modified != 1 {
		t.Errorf("Modified = %d, want 1", result.Modified)
	}
	if result.Staged != 0 {
		t.Errorf("Staged = %d, want 0", result.Staged)
	}
}

func TestParsePorcelainStatusBranchAB(t *testing.T) {
	t.Parallel()
	input := `# branch.ab +3 -2
`
	result := parsePorcelainStatus(input)
	if result.Ahead != 3 {
		t.Errorf("Ahead = %d, want 3", result.Ahead)
	}
	if result.Behind != 2 {
		t.Errorf("Behind = %d, want 2", result.Behind)
	}
}

func TestParsePorcelainStatusConflicts(t *testing.T) {
	t.Parallel()
	input := `uUU conflict.txt
1 UU N... 100644 100644 100644 abc1234_0 abc1234_1 merge.txt
`
	result := parsePorcelainStatus(input)
	if result.Conflicts != 2 {
		t.Errorf("Conflicts = %d, want 2", result.Conflicts)
	}
}

func TestParsePorcelainStatusMixed(t *testing.T) {
	t.Parallel()
	input := `# branch.ab +1 -0
? untracked.go
1 .M N... 100644 100644 100644 aaa1111_0 aaa1111_1 modified.go
1 M. N... 100644 100644 100644 bbb2222_0 bbb2222_1 staged.go
`
	result := parsePorcelainStatus(input)
	if result.Ahead != 1 {
		t.Errorf("Ahead = %d, want 1", result.Ahead)
	}
	if result.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", result.Untracked)
	}
	if result.Modified != 1 {
		t.Errorf("Modified = %d, want 1", result.Modified)
	}
	if result.Staged != 1 {
		t.Errorf("Staged = %d, want 1", result.Staged)
	}
}

func TestParsePorcelainStatusRenamed(t *testing.T) {
	t.Parallel()
	input := `2 R. N... 100644 100644 100644 abc1234_0 abc1234_1 old_name.txt new_name.txt
`
	result := parsePorcelainStatus(input)
	if result.Staged != 1 {
		t.Errorf("Staged = %d, want 1 (rename is staged)", result.Staged)
	}
}

func TestParsePorcelainStatusUnknownLineType(t *testing.T) {
	t.Parallel()
	// Lines with unknown prefix should be silently ignored.
	input := `x some garbage line
`
	result := parsePorcelainStatus(input)
	if result.Modified != 0 || result.Staged != 0 || result.Untracked != 0 {
		t.Errorf("unknown line type should be ignored, got %+v", result)
	}
}

func TestParseCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"+3", 3},
		{"-2", 2},
		{"+0", 0},
		{"+10", 10},
		{"abc", 0},
		{"", 0},
	}
	for _, tc := range tests {
		got := parseCount(tc.input)
		if got != tc.want {
			t.Errorf("parseCount(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestEqualPathSame(t *testing.T) {
	t.Parallel()
	if !EqualPath("/tmp/a", "/tmp/a") {
		t.Error("EqualPath should return true for identical paths")
	}
}

func TestEqualPathTrailingSlash(t *testing.T) {
	t.Parallel()
	if !EqualPath("/tmp/a/", "/tmp/a") {
		t.Error("EqualPath should handle trailing slashes")
	}
}

func TestEqualPathDifferent(t *testing.T) {
	t.Parallel()
	if EqualPath("/tmp/a", "/tmp/b") {
		t.Error("EqualPath should return false for different paths")
	}
}
