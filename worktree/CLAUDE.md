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

## Testing

```bash
go test ./worktree/... -v -count=1
go test ./worktree/... -run TestWorktree -v
go test ./worktree/... -run TestRepoLock -v
go test ./worktree/... -run TestSemaphore -v
go test ./worktree/... -run TestStress -v    # concurrency stress tests
```

Tests create temporary git repos. No external git binary required.
