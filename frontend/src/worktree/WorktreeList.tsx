import { useState, useCallback } from 'react';
import {
  RefreshCw,
  Trash2,
  GitCommitHorizontal,
  ArrowLeftRight,
  Plus,
  FolderGit2,
} from 'lucide-react';
import type { Project, Worktree } from './types';
import { WorktreeStatus } from './WorktreeStatus';
import { WorktreeDialog } from './WorktreeDialog';
import { GitBranchIcon } from './GitBranchIcon';
import './worktree.css';

interface WorktreeListProps {
  project: Project;
  worktrees: Worktree[];
  loading: boolean;
  onCreate: (params: { branch_name: string; create_branch?: boolean; base_branch?: string }) => void;
  onDelete: (worktreeId: string) => void;
  onRefreshStatus: (worktreeId: string) => void;
  onCommitAll: (worktreeId: string, message: string) => void;
  onSyncAll: () => void;
  onRefreshWorktrees: () => void;
}

export function WorktreeList({
  project,
  worktrees,
  loading,
  onCreate,
  onDelete,
  onRefreshStatus,
  onCommitAll,
  onSyncAll,
  onRefreshWorktrees,
}: WorktreeListProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [commitDialogId, setCommitDialogId] = useState<string | null>(null);
  const [commitMessage, setCommitMessage] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleCreate = useCallback(
    (params: {
      branch_name: string;
      create_branch?: boolean;
      base_branch?: string;
    }) => {
      onCreate(params);
      onRefreshWorktrees();
    },
    [onCreate, onRefreshWorktrees],
  );

  const handleDelete = useCallback(
    (worktree: Worktree) => {
      if (worktree.is_main) return;
      const confirmed = window.confirm(
        `Delete worktree "${worktree.branch_name}"? This cannot be undone.`,
      );
      if (!confirmed) return;

      setDeletingId(worktree.id);
      try {
        onDelete(worktree.id);
      } catch {
        // Error handled by hook
      } finally {
        setDeletingId(null);
      }
    },
    [onDelete],
  );

  const handleSubmitCommit = useCallback(() => {
    if (!commitDialogId || !commitMessage.trim()) return;
    try {
      onCommitAll(commitDialogId, commitMessage.trim());
      setCommitDialogId(null);
      setCommitMessage('');
    } catch {
      // Error handled by hook
    }
  }, [commitDialogId, commitMessage, onCommitAll]);

  const handleRefreshAll = useCallback(() => {
    try {
      onSyncAll();
      onRefreshWorktrees();
    } catch {
      // Error handled by hook
    }
  }, [onSyncAll, onRefreshWorktrees]);

  const existingBranches = worktrees.map((wt) => wt.branch_name);

  return (
    <div className="wt-list">
      <div className="wt-list-header">
        <div className="wt-list-header-left">
          <FolderGit2 size={14} />
          <span>{worktrees.length} worktree{worktrees.length !== 1 ? 's' : ''}</span>
        </div>
        <div className="wt-list-header-actions">
          <button
            className="wt-icon-btn"
            title="Sync all worktrees"
            disabled={loading}
            onClick={handleRefreshAll}
          >
            <ArrowLeftRight size={14} />
          </button>
          <button
            className="wt-icon-btn"
            title="Refresh"
            disabled={loading}
            onClick={onRefreshWorktrees}
          >
            <RefreshCw size={14} />
          </button>
          <button
            className="wt-btn wt-btn-primary wt-btn-sm"
            onClick={() => setDialogOpen(true)}
          >
            <Plus size={14} />
            New
          </button>
        </div>
      </div>

      {worktrees.length === 0 && !loading && (
        <div className="wt-empty">No worktrees found</div>
      )}

      <div className="wt-items">
        {worktrees.map((wt) => (
          <WorktreeCard
            key={wt.id}
            worktree={wt}
            deleting={deletingId === wt.id}
            onRefresh={() => onRefreshStatus(wt.id)}
            onDelete={() => handleDelete(wt)}
            onCommit={() => {
              setCommitDialogId(wt.id);
              setCommitMessage('');
            }}
          />
        ))}
      </div>

      <WorktreeDialog
        open={dialogOpen}
        project={project}
        existingBranches={existingBranches}
        onSubmit={handleCreate}
        onClose={() => setDialogOpen(false)}
      />

      {commitDialogId && (
        <div className="wt-dialog-overlay" onClick={() => setCommitDialogId(null)}>
          <div className="wt-dialog wt-dialog-sm" onClick={(e) => e.stopPropagation()}>
            <div className="wt-dialog-header">
              <h3 className="wt-dialog-title">
                <GitCommitHorizontal size={16} />
                Commit All
              </h3>
              <button
                className="wt-dialog-close"
                onClick={() => setCommitDialogId(null)}
              >
                <span className="wt-icon-x">&times;</span>
              </button>
            </div>
            <div className="wt-dialog-body">
              <textarea
                className="wt-form-textarea"
                value={commitMessage}
                onChange={(e) => setCommitMessage(e.target.value)}
                placeholder="Commit message..."
                autoFocus
                rows={4}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                    handleSubmitCommit();
                  }
                }}
              />
              <span className="wt-form-hint">
                Ctrl+Enter to submit
              </span>
            </div>
            <div className="wt-dialog-footer">
              <button
                className="wt-btn wt-btn-secondary"
                onClick={() => setCommitDialogId(null)}
              >
                Cancel
              </button>
              <button
                className="wt-btn wt-btn-primary"
                onClick={handleSubmitCommit}
                disabled={!commitMessage.trim() || loading}
              >
                Commit
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface WorktreeCardProps {
  worktree: Worktree;
  deleting: boolean;
  onRefresh: () => void;
  onDelete: () => void;
  onCommit: () => void;
}

function WorktreeCard({
  worktree: wt,
  deleting,
  onRefresh,
  onDelete,
  onCommit,
}: WorktreeCardProps) {
  const hasPendingChanges =
    (wt.status_modified ?? 0) > 0 ||
    (wt.status_staged ?? 0) > 0 ||
    (wt.status_untracked ?? 0) > 0;

  const commitLine =
    wt.head_commit && wt.head_commit_message
      ? `${wt.head_commit.slice(0, 8)} ${wt.head_commit_message.split('\n')[0]}`
      : wt.head_commit_message?.split('\n')[0] ?? null;

  return (
    <div className="wt-card">
      <div className="wt-card-header">
        <div className="wt-card-header-left">
          <GitBranchIcon size={13} />
          <span className="wt-card-branch">{wt.branch_name}</span>
          {wt.is_main && <span className="wt-badge wt-badge-main">main</span>}
        </div>
        <div className="wt-card-actions">
          <button
            className="wt-icon-btn"
            title="Refresh status"
            onClick={onRefresh}
          >
            <RefreshCw size={13} />
          </button>
          {hasPendingChanges && (
            <button
              className="wt-icon-btn"
              title="Commit all"
              onClick={onCommit}
            >
              <GitCommitHorizontal size={13} />
            </button>
          )}
          {!wt.is_main && (
            <button
              className="wt-icon-btn wt-icon-btn-danger"
              title="Delete worktree"
              onClick={onDelete}
              disabled={deleting}
            >
              <Trash2 size={13} />
            </button>
          )}
        </div>
      </div>

      <div className="wt-card-body">
        <WorktreeStatus
          ahead={wt.status_ahead}
          behind={wt.status_behind}
          modified={wt.status_modified}
          staged={wt.status_staged}
          untracked={wt.status_untracked}
          conflicts={wt.status_conflicts}
        />
        {commitLine && (
          <span className="wt-card-commit" title={wt.head_commit_message}>
            {commitLine}
          </span>
        )}
        <span className="wt-card-path" title={wt.path}>
          {wt.path}
        </span>
      </div>
    </div>
  );
}
