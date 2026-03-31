# CLAUDE.md — worktree/

Git worktree management using native go-git (no `exec.Command` git CLI calls).

## File Layout

```
worktree.go        Core operations: Create, List, Remove, GetStatus, Commit
worktree_mgmt.go   Management helpers: status refresh, sync
errors.go          Sentinel errors (ErrRepoNotFound, ErrBranchExists, etc.)
repo_lock.go       RepoLock — per-repository re-entrant RWMutex
semaphore.go       OperationSemaphore — bounded concurrent operations
repo_cache.go      RepoCache — reuse open repository handles
branch.go        Branch CRUD (list, create, delete, merge) with go-git
hooks.go         Lifecycle hooks (pre/post create/merge/remove) via shell scripts
pr.go            GitHub PR checkout — fetch PR details, create worktree from SHA
worktree_diff.go Unified diff generation for worktree changes
```

## Key Patterns

**No exec.Command**: All git operations use `go-git/v5`. No shell escaping concerns, no PATH dependency.

**Concurrency Control**:
- `RepoLock`: Per-repo re-entrant `sync.RWMutex`. Prevents concurrent writes to the same repository. Supports nested locking by the same goroutine.
- `OperationSemaphore`: Bounded concurrency across all worktree operations. Prevents resource exhaustion.

**RepoCache**: Caches open `*git.Repository` instances by path. Avoids repeated object discovery overhead.

**Branch Validation**: `branchNameRe` (`^[a-zA-Z0-9][a-zA-Z0-9_./-]{0,127}$`) enforces safe branch names before any git operation.

**Environment Sanitization**: Filters `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE` from environment before go-git operations to prevent state leakage.

**Info/Status Types**: `Info` (path, branch, isMain, headCommit, headMessage) and `Status` (ahead, behind, modified, staged, untracked, conflicts) for API responses.

**Branch Operations**: `branch.go` provides ListBranches, CreateBranch, DeleteBranch, MergeBranch. Fast-forward merges use go-git natively; complex merges fall back to CLI (`git merge`) with environment sanitization.

**Worktree Hooks**: `hooks.go` supports optional shell scripts at `<repo>/.ttyweb/hooks/<event>` (pre-create, post-create, pre-merge, post-merge, pre-remove). Hooks receive context via environment variables.

**PR Checkout**: `pr.go` fetches PR details from GitHub API, detects repo slug from remote URLs, and creates local worktrees from PR SHAs via `CheckoutPRBranch`.

**Diff Generation**: `worktree_diff.go` produces unified diffs comparing HEAD to working tree/index, handling both staged and unstaged changes.

## Testing

```bash
go test ./worktree/... -v -count=1
go test ./worktree/... -run TestWorktree -v
go test ./worktree/... -run TestRepoLock -v
go test ./worktree/... -run TestSemaphore -v
go test ./worktree/... -run TestStress -v    # concurrency stress tests
```

Tests create temporary git repos. No external git binary required.
