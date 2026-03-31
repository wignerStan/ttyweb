package worktree

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"testing"

	goGit "github.com/go-git/go-git/v5"
)

// hashRe matches 7-40 character hex commit hashes (git log --oneline uses 7-char).
var hashRe = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

// stripHashes replaces commit hashes with a placeholder for deterministic snapshots.
func stripHashes(s []byte) []byte {
	return hashRe.ReplaceAll(s, []byte("HASH_PLACEHOLDER"))
}

// TestSnapshot_IsAncestorOf_SameCommit verifies that a commit is its own ancestor
// (the trivial case handled by the hash-equality check).
func TestSnapshot_IsAncestorOf_SameCommit(t *testing.T) {
	repoDir := initTestRepo(t)

	repo, err := goGit.PlainOpen(repoDir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("HEAD: %v", err)
	}

	ok, err := isAncestorOf(repo, head.Hash(), head.Hash())
	if err != nil {
		t.Fatalf("isAncestorOf: %v", err)
	}
	if !ok {
		t.Error("a commit should be its own ancestor")
	}

	result := map[string]any{
		"hash":        head.Hash().String(),
		"is_ancestor": ok,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	compareGolden(t, stripHashes(append(data, '\n')))
}

// TestSnapshot_IsAncestorOf_DifferentCommits verifies that isAncestorOf returns
// false for any two different commits (even a direct parent-child), because
// the revlist.Objects implementation includes the descendant commit itself
// which is a commit object reachable from descendant but not from ancestor.
func TestSnapshot_IsAncestorOf_DifferentCommits(t *testing.T) {
	repoDir := initTestRepoWithBranches(t)

	repo, err := goGit.PlainOpen(repoDir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	mainRef, err := repo.Reference("refs/heads/main", false)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	featureRef, err := repo.Reference("refs/heads/feature", false)
	if err != nil {
		t.Fatalf("feature ref: %v", err)
	}

	ok, err := isAncestorOf(repo, mainRef.Hash(), featureRef.Hash())
	if err != nil {
		t.Fatalf("isAncestorOf: %v", err)
	}

	result := map[string]any{
		"ancestor_hash":   mainRef.Hash().String(),
		"descendant_hash": featureRef.Hash().String(),
		"is_ancestor":     ok,
		"description":     "main is parent of feature but isAncestorOf returns false due to revlist including descendant commit",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	compareGolden(t, stripHashes(append(data, '\n')))
}

// TestSnapshot_MergeViaCLI_FastForward verifies that a merge of hotfix into main
// produces the expected log structure (commits with messages, hash-stripped).
func TestSnapshot_MergeViaCLI_FastForward(t *testing.T) {
	repo := initTestRepoWithBranches(t)

	err := MergeBranch(context.Background(), repo, "hotfix", "main")
	if err != nil {
		t.Fatalf("MergeBranch failed: %v", err)
	}

	out := mustOutput(t, repo, "log", "--oneline", "main")
	compareGolden(t, stripHashes([]byte(out)))
}

// TestSnapshot_MergeViaCLI_Conflict verifies that a conflicting merge returns
// a structured error containing conflict details.
func TestSnapshot_MergeViaCLI_Conflict(t *testing.T) {
	repo := initTestRepo(t)

	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "base version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "base commit")

	mustRun(t, repo, "checkout", "-b", "conflict-branch")

	mustRun(t, repo, "checkout", "main")
	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "main version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "conflict on main")

	mustRun(t, repo, "checkout", "conflict-branch")
	mustWriteFile(t, filepath.Join(repo, "conflict.txt"), "feature version\n")
	mustRun(t, repo, "add", "conflict.txt")
	mustRun(t, repo, "-c", "user.name=test", "-c", "user.email=test@test.com", "commit", "-m", "conflict on feature")

	err := MergeBranch(context.Background(), repo, "conflict-branch", "main")
	if err == nil {
		t.Fatal("expected error when merging conflicting branches")
	}

	var buf bytes.Buffer
	buf.WriteString(err.Error())
	buf.WriteByte('\n')
	compareGolden(t, stripHashes(buf.Bytes()))
}
