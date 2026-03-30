package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// addWorktree creates a git worktree manually by constructing the directory
// structure, extracting files from the tree object, and building an index.
// go-git's HardReset/Checkout fail with EOF on linked worktrees, so we
// perform the checkout manually.
func addWorktree(repoPath, worktreePath, branchName string) error {
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: "open repo", Cause: err}
	}

	// Resolve branch to commit hash.
	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	ref, err := repo.Reference(refName, false)
	if err != nil {
		return &OpError{Kind: KindNotFound, Path: repoPath, Detail: fmt.Sprintf("branch %q not found", branchName), Cause: err}
	}

	// Get the commit tree.
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: "resolve commit", Cause: err}
	}
	tree, err := commit.Tree()
	if err != nil {
		return &OpError{Kind: KindGitFailed, Path: repoPath, Detail: "resolve tree", Cause: err}
	}

	// Create worktree directory.
	if err := os.MkdirAll(worktreePath, 0o750); err != nil {
		return fmt.Errorf("create worktree dir: %w", err)
	}

	worktreeName := filepath.Base(worktreePath)
	wtMetaDir := filepath.Join(repoPath, ".git", "worktrees", worktreeName)
	gitdir := wtMetaDir

	// Write .git file in worktree (points to metadata dir).
	gitFileContent := fmt.Sprintf("gitdir: %s\n", gitdir)
	if err := os.WriteFile(filepath.Join(worktreePath, ".git"), []byte(gitFileContent), 0o644); err != nil {
		os.RemoveAll(worktreePath)
		return fmt.Errorf("write .git file: %w", err)
	}

	// Create metadata directory.
	if err := os.MkdirAll(wtMetaDir, 0o750); err != nil {
		os.RemoveAll(worktreePath)
		return fmt.Errorf("create worktree metadata dir: %w", err)
	}

	// Write gitdir (points back to worktree).
	if err := os.WriteFile(filepath.Join(wtMetaDir, "gitdir"), []byte(worktreePath+"\n"), 0o644); err != nil {
		os.RemoveAll(worktreePath)
		return fmt.Errorf("write gitdir: %w", err)
	}

	// Write HEAD.
	if err := os.WriteFile(filepath.Join(wtMetaDir, "HEAD"), []byte("ref: refs/heads/"+branchName+"\n"), 0o644); err != nil {
		os.RemoveAll(worktreePath)
		return fmt.Errorf("write HEAD: %w", err)
	}

	// Write commondir (points to main .git).
	if err := os.WriteFile(filepath.Join(wtMetaDir, "commondir"), []byte(filepath.Join(repoPath, ".git")+"\n"), 0o644); err != nil {
		os.RemoveAll(worktreePath)
		return fmt.Errorf("write commondir: %w", err)
	}

	// Checkout files from the tree and build an index.
	if err := checkoutTree(repo, worktreePath, wtMetaDir, tree); err != nil {
		os.RemoveAll(worktreePath)
		return &OpError{Kind: KindGitFailed, Path: worktreePath, Detail: "checkout files", Cause: err}
	}

	return nil
}

// checkoutTree extracts all files from a tree object into worktreePath and
// writes a git index file into the worktree metadata directory.
func checkoutTree(repo *goGit.Repository, worktreePath, wtMetaDir string, tree *object.Tree) error {
	idx := &index.Index{Version: 2}

	err := tree.Files().ForEach(func(f *object.File) error {
		target := filepath.Join(worktreePath, f.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Name, err)
		}

		if f.Mode == filemode.Symlink {
			content, err := f.Contents()
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", f.Name, err)
			}
			if err := os.Symlink(content, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", f.Name, err)
			}
			return nil
		}

		from, err := f.Reader()
		if err != nil {
			return fmt.Errorf("read blob %s: %w", f.Name, err)
		}
		defer from.Close()

		to, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(f.Mode))
		if err != nil {
			return fmt.Errorf("create file %s: %w", f.Name, err)
		}
		defer to.Close()

		if _, err := io.Copy(to, from); err != nil {
			return fmt.Errorf("write file %s: %w", f.Name, err)
		}

		// Build index entry.
		fi, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("stat file %s: %w", f.Name, err)
		}
		idx.Entries = append(idx.Entries, &index.Entry{
			Name:        filepath.ToSlash(f.Name),
			Mode:        filemode.FileMode(os.FileMode(f.Mode)),
			Hash:        f.Hash,
			Size:        uint32(fi.Size()),
			CreatedAt:   fi.ModTime(),
			ModifiedAt:  fi.ModTime(),
			Dev:         0,
			Inode:       0,
			UID:         0,
			GID:         0,
		})
		return nil
	})
	if err != nil {
		return err
	}

	// Write index file.
	idxFile, err := os.Create(filepath.Join(wtMetaDir, "index"))
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer idxFile.Close()

	if err := index.NewEncoder(idxFile).Encode(idx); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	return nil
}

// listWorktreesManual enumerates all worktrees attached to the repository
// by reading .git/worktrees metadata directly.
func listWorktreesManual(repoPath string) ([]Info, error) {
	wtDir := filepath.Join(repoPath, ".git", "worktrees")

	entries, err := os.ReadDir(wtDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Info{mainWorktreeInfo(repoPath)}, nil
		}
		return nil, fmt.Errorf("read worktrees dir: %w", err)
	}

	infos := make([]Info, 1, len(entries)+1)
	infos[0] = mainWorktreeInfo(repoPath)

	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaDir := filepath.Join(wtDir, entry.Name())

		gitdirPath, err := os.ReadFile(filepath.Join(metaDir, "gitdir"))
		if err != nil {
			pruneStaleEntry(metaDir)
			continue
		}
		wtPath := strings.TrimSpace(string(gitdirPath))

		// Prune if worktree directory no longer exists.
		if _, err := os.Stat(wtPath); os.IsNotExist(err) {
			pruneStaleEntry(metaDir)
			continue
		}

		headContent, err := os.ReadFile(filepath.Join(metaDir, "HEAD"))
		if err != nil {
			continue
		}

		branch, headHash := resolveWorktreeHead(repo, string(headContent))

		commitMsg := ""
		if headHash != plumbing.ZeroHash {
			if commit, err := repo.CommitObject(headHash); err == nil {
				commitMsg = strings.TrimSpace(commit.Message)
			}
		}

		infos = append(infos, Info{
			Path:        wtPath,
			Branch:      branch,
			HeadCommit:  headHash.String()[:7],
			HeadMessage: commitMsg,
		})
	}

	return infos, nil
}

// removeWorktreeManual removes a worktree by deleting its directory and
// cleaning up metadata. If force is true, uncommitted changes are ignored.
func removeWorktreeManual(repoPath, worktreePath string, force bool) error {
	worktreeName := filepath.Base(worktreePath)
	wtMetaDir := filepath.Join(repoPath, ".git", "worktrees", worktreeName)

	// Check for uncommitted changes when not forcing.
	if !force {
		if repo, err := defaultCache.Open(worktreePath); err == nil {
			if wt, err := repo.Worktree(); err == nil {
				if status, err := wt.Status(); err == nil {
					for _, s := range status {
						if s.Staging != goGit.Unmodified || s.Worktree != goGit.Unmodified {
							return &OpError{Kind: KindConflict, Path: worktreePath, Detail: "worktree has uncommitted changes"}
						}
					}
				}
			}
		}
	}

	if err := os.RemoveAll(worktreePath); err != nil {
		return &OpError{Kind: KindGitFailed, Path: worktreePath, Detail: "remove worktree dir", Cause: err}
	}

	_ = os.RemoveAll(wtMetaDir)
	defaultCache.Remove(worktreePath)
	return nil
}

// mainWorktreeInfo builds an Info struct for the main (primary) worktree.
func mainWorktreeInfo(repoPath string) Info {
	repo, err := defaultCache.Open(repoPath)
	if err != nil {
		return Info{Path: repoPath, IsMain: true}
	}
	head, err := repo.Head()
	if err != nil {
		return Info{Path: repoPath, IsMain: true}
	}
	branch := ""
	if head.Name().IsBranch() {
		branch = head.Name().Short()
	}
	commitMsg := ""
	if commit, err := repo.CommitObject(head.Hash()); err == nil {
		commitMsg = strings.TrimSpace(commit.Message)
	}
	return Info{
		Path:        repoPath,
		Branch:      branch,
		IsMain:      true,
		HeadCommit:  head.Hash().String()[:7],
		HeadMessage: commitMsg,
	}
}

// resolveWorktreeHead parses a worktree HEAD file and returns the branch name
// and HEAD commit hash.
func resolveWorktreeHead(repo *goGit.Repository, headContent string) (string, plumbing.Hash) {
	headContent = strings.TrimSpace(headContent)
	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		branch := strings.TrimPrefix(headContent, "ref: refs/heads/")
		refName := plumbing.ReferenceName("refs/heads/" + branch)
		ref, err := repo.Reference(refName, false)
		if err != nil {
			return branch, plumbing.ZeroHash
		}
		return branch, ref.Hash()
	}
	hash := plumbing.NewHash(strings.TrimSpace(headContent))
	return "", hash
}

// pruneStaleEntry removes a worktree metadata directory that points to a
// nonexistent worktree.
func pruneStaleEntry(metaDir string) {
	_ = os.RemoveAll(metaDir)
}
