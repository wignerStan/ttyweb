package worktree

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func compareGolden(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateGolden {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		_ = os.WriteFile(golden, got, 0o644)
	}
	want, err := os.ReadFile(golden) //nolint:gosec // reason: test code, golden file path is deterministic
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func mockSignature() object.Signature {
	return object.Signature{
		Name:  "test",
		Email: "test@test.com",
		When:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// initGoGitRepo creates a git repo with go-git (no CLI git needed) and an
// initial commit on main. Returns the repo path.
func initGoGitRepo(t *testing.T, initialFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	repo, err := goGit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("go-git init: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	for name, content := range initialFiles {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		_, err = wt.Add(name)
		if err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
	}

	if len(initialFiles) > 0 {
		sig := mockSignature()
		_, err = wt.Commit("initial", &goGit.CommitOptions{
			Author:    &sig,
			Committer: &sig,
		})
		if err != nil {
			t.Fatalf("commit: %v", err)
		}
	}

	return dir
}

// goGitStage stages a file in the go-git repo at dir.
func goGitStage(t *testing.T, dir, file string) {
	t.Helper()
	repo, err := goGit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	_, err = wt.Add(file)
	if err != nil {
		t.Fatalf("add %s: %v", file, err)
	}
}

// --- GetWorktreeDiff snapshot tests ---

func TestSnapshot_GetWorktreeDiff_EmptyRepo(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"file.txt": "hello\n",
	})

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

func TestSnapshot_GetWorktreeDiff_UnstagedModification(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"file.txt": "hello\n",
	})

	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("world\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

func TestSnapshot_GetWorktreeDiff_StagedChange(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"file.txt": "hello\n",
	})

	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("staged\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	goGitStage(t, repo, "file.txt")

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

func TestSnapshot_GetWorktreeDiff_NewUntrackedFile(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"file.txt": "hello\n",
	})

	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

func TestSnapshot_GetWorktreeDiff_DeletedFile(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"file.txt": "hello\n",
	})

	if err := os.Remove(filepath.Join(repo, "file.txt")); err != nil {
		t.Fatalf("remove: %v", err)
	}

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

func TestSnapshot_GetWorktreeDiff_MixedChanges(t *testing.T) {
	repo := initGoGitRepo(t, map[string]string{
		"staged.txt":  "original\n",
		"modified.txt": "original\n",
		"deleted.txt":  "to be deleted\n",
	})

	// Stage a change to staged.txt
	if err := os.WriteFile(filepath.Join(repo, "staged.txt"), []byte("staged change\n"), 0o644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	goGitStage(t, repo, "staged.txt")

	// Unstaged modification to modified.txt
	if err := os.WriteFile(filepath.Join(repo, "modified.txt"), []byte("unstaged change\n"), 0o644); err != nil {
		t.Fatalf("write modified.txt: %v", err)
	}

	// Delete deleted.txt from worktree
	if err := os.Remove(filepath.Join(repo, "deleted.txt")); err != nil {
		t.Fatalf("remove deleted.txt: %v", err)
	}

	diff, err := GetWorktreeDiff(context.Background(), repo, repo)
	if err != nil {
		t.Fatalf("GetWorktreeDiff: %v", err)
	}

	compareGolden(t, []byte(diff))
}

// --- writeHunk snapshot tests ---

func TestSnapshot_WriteHunk_AddOnly(t *testing.T) {
	var buf bytes.Buffer
	writeHunk(&buf, nil, []string{"line1", "line2"})
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteHunk_DeleteOnly(t *testing.T) {
	var buf bytes.Buffer
	writeHunk(&buf, []string{"old1", "old2"}, nil)
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteHunk_Mixed(t *testing.T) {
	var buf bytes.Buffer
	writeHunk(&buf, []string{"old1", "old2"}, []string{"new1", "new2"})
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteHunk_NoChanges(t *testing.T) {
	var buf bytes.Buffer
	writeHunk(&buf, []string{"same1", "same2"}, []string{"same1", "same2"})
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteHunk_CommonPrefixAndSuffix(t *testing.T) {
	var buf bytes.Buffer
	writeHunk(&buf,
		[]string{"prefix", "old-middle", "suffix"},
		[]string{"prefix", "new-middle", "suffix"},
	)
	compareGolden(t, buf.Bytes())
}

// --- writeFileDiff snapshot tests ---

func TestSnapshot_WriteFileDiff_NewFile(t *testing.T) {
	var buf bytes.Buffer
	writeFileDiff(&buf, "newfile.txt", nil, []byte("content\n"))
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteFileDiff_DeletedFile(t *testing.T) {
	var buf bytes.Buffer
	writeFileDiff(&buf, "oldfile.txt", []byte("gone\n"), nil)
	compareGolden(t, buf.Bytes())
}

func TestSnapshot_WriteFileDiff_ModifiedFile(t *testing.T) {
	var buf bytes.Buffer
	writeFileDiff(&buf, "changed.txt", []byte("old\n"), []byte("new\n"))
	compareGolden(t, buf.Bytes())
}
