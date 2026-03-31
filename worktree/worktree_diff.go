package worktree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetWorktreeDiff returns the unified diff of all changes (staged and unstaged)
// in the worktree at worktreePath, relative to the HEAD commit.
// Returns an empty string if there are no changes.
//
//nolint:gocyclo // single-pass diff generator with sequential error handling
func GetWorktreeDiff(ctx context.Context, repoPath, worktreePath string) (string, error) {
	_ = ctx // reserved for future cancellation support

	worktreePath = strings.TrimSpace(worktreePath)
	if worktreePath == "" {
		return "", errors.New("worktree path is required")
	}
	_ = repoPath // repoPath accepted for API consistency; resolved via worktreePath

	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return "", fmt.Errorf("resolve worktree path: %w", err)
	}

	repo, err := defaultCache.Open(absPath)
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "open repo", Cause: err}
	}

	headRef, err := repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return "", errors.New("no HEAD commit: repository has no commits")
		}
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "resolve HEAD", Cause: err}
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "read HEAD commit", Cause: err}
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "read HEAD tree", Cause: err}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "get worktree", Cause: err}
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "read index", Cause: err}
	}

	// Get worktree status to find all changed files.
	status, err := wt.Status()
	if err != nil {
		return "", &OpError{Kind: KindGitFailed, Path: absPath, Detail: "worktree status", Cause: err}
	}

	var buf bytes.Buffer

	// Sort file names for deterministic diff output.
	files := make([]string, 0, len(status))
	for file := range status {
		files = append(files, file)
	}
	sort.Strings(files)

	for _, file := range files {
		fs := status[file]
		// Determine the "old" content (from HEAD tree or index).
		var oldContent []byte
		var newContent []byte

		switch {
		case fs.Staging != goGit.Unmodified && fs.Staging != goGit.Untracked:
			// Staged change: old is HEAD, new is index.
			oldContent = readTreeFile(repo, headTree, file)
			newContent = readIndexFile(repo, idx, file)

		case fs.Worktree != goGit.Unmodified && fs.Worktree != goGit.Untracked:
			// Unstaged change: old is index, new is worktree.
			oldContent = readIndexFile(repo, idx, file)
			if fs.Worktree != goGit.Deleted {
				fullPath := filepath.Join(absPath, file)
				data, readErr := os.ReadFile(fullPath) //nolint:gosec // reason: path from git status
				if readErr == nil {
					newContent = data
				}
			}

		default:
			// Skip unmodified/untracked files.
			continue
		}

		writeFileDiff(&buf, file, oldContent, newContent)
	}

	return strings.TrimSpace(buf.String()), nil
}

// readTreeFile reads a file's content from a tree object.
func readTreeFile(repo *goGit.Repository, tree *object.Tree, file string) []byte {
	entry, err := tree.FindEntry(file)
	if err != nil {
		return nil
	}
	blob, err := repo.BlobObject(entry.Hash)
	if err != nil {
		return nil
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil
	}
	defer func() { _ = reader.Close() }()
	data, _ := io.ReadAll(reader)
	return data
}

// readIndexFile reads a file's content from the git index.
func readIndexFile(repo *goGit.Repository, idx *index.Index, file string) []byte {
	entry, err := idx.Entry(file)
	if err != nil {
		return nil
	}
	blob, err := repo.BlobObject(entry.Hash)
	if err != nil {
		return nil
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil
	}
	defer func() { _ = reader.Close() }()
	data, _ := io.ReadAll(reader)
	return data
}

// writeFileDiff writes a simple unified diff for a single file.
func writeFileDiff(buf *bytes.Buffer, file string, oldContent, newContent []byte) {
	oldLines := strings.Split(string(oldContent), "\n")
	newLines := strings.Split(string(newContent), "\n")

	fmt.Fprintf(buf, "diff --git a/%s b/%s\n", file, file)

	if len(oldContent) == 0 {
		buf.WriteString("new file mode 100644\n")
	} else if len(newContent) == 0 {
		buf.WriteString("deleted file mode 100644\n")
	}

	writeHunk(buf, oldLines, newLines)
}

// writeHunk writes a unified diff hunk comparing old and new lines.
func writeHunk(buf *bytes.Buffer, oldLines, newLines []string) {
	// Find common prefix.
	minLen := len(oldLines)
	if len(newLines) < minLen {
		minLen = len(newLines)
	}
	commonPrefix := 0
	for commonPrefix < minLen && oldLines[commonPrefix] == newLines[commonPrefix] {
		commonPrefix++
	}

	// Find common suffix.
	oldSuffix := len(oldLines)
	newSuffix := len(newLines)
	for oldSuffix > commonPrefix && newSuffix > commonPrefix && oldLines[oldSuffix-1] == newLines[newSuffix-1] {
		oldSuffix--
		newSuffix--
	}

	oldCount := oldSuffix - commonPrefix
	newCount := newSuffix - commonPrefix

	if oldCount == 0 && newCount == 0 {
		return
	}

	fmt.Fprintf(buf, "@@ -%d,%d +%d,%d @@\n", commonPrefix+1, oldCount, commonPrefix+1, newCount)

	for i := commonPrefix; i < oldSuffix; i++ {
		buf.WriteString("-" + oldLines[i] + "\n")
	}
	for i := commonPrefix; i < newSuffix; i++ {
		buf.WriteString("+" + newLines[i] + "\n")
	}
}
