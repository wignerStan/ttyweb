import { useState, useEffect, useCallback } from 'react';
import {
  GitBranch,
  X,
  Plus,
} from 'lucide-react';
import type { Project, CreateWorktreeParams } from './types';
import './worktree.css';

interface WorktreeDialogProps {
  open: boolean;
  project: Project | null;
  existingBranches: string[];
  onSubmit: (params: CreateWorktreeParams) => void;
  onClose: () => void;
}

const INITIAL_FORM = {
  branchName: '',
  createBranch: true,
  baseBranch: '',
};

export function WorktreeDialog({
  open,
  project,
  existingBranches,
  onSubmit,
  onClose,
}: WorktreeDialogProps) {
  const [form, setForm] = useState(INITIAL_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const resetForm = useCallback(() => {
    setForm(INITIAL_FORM);
    setError(null);
    setSubmitting(false);
  }, []);

  useEffect(() => {
    if (open) {
      resetForm();
    }
  }, [open, resetForm]);

  const updateField = useCallback(
    <K extends keyof typeof INITIAL_FORM>(key: K, value: (typeof INITIAL_FORM)[K]) => {
      setForm((prev) => ({ ...prev, [key]: value }));
      setError(null);
    },
    [],
  );

  const handleSubmit = async () => {
    if (!form.branchName.trim()) {
      setError('Branch name is required');
      return;
    }

    if (form.createBranch && !form.baseBranch.trim()) {
      setError('Base branch is required when creating a new branch');
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      await onSubmit({
        branchName: form.branchName.trim(),
        createBranch: form.createBranch,
        baseBranch: form.createBranch ? form.baseBranch.trim() : undefined,
      });
      onClose();
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : 'Failed to create worktree';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  if (!open) return null;

  return (
    <div className="wt-dialog-overlay" onClick={onClose}>
      <div className="wt-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="wt-dialog-header">
          <h3 className="wt-dialog-title">
            <GitBranch size={16} />
            New Worktree
          </h3>
          <button className="wt-dialog-close" onClick={onClose}>
            <X size={16} />
          </button>
        </div>

        <div className="wt-dialog-body">
          {error && <div className="wt-dialog-error">{error}</div>}

          <label className="wt-form-label">Branch Name</label>
          {!form.createBranch ? (
            <select
              className="wt-form-select"
              value={form.branchName}
              onChange={(e) => updateField('branchName', e.target.value)}
            >
              <option value="">Select branch...</option>
              {existingBranches.map((branch) => (
                <option key={branch} value={branch}>
                  {branch}
                </option>
              ))}
            </select>
          ) : (
            <input
              className="wt-form-input"
              type="text"
              value={form.branchName}
              onChange={(e) => updateField('branchName', e.target.value)}
              placeholder="feature/my-branch"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSubmit();
              }}
            />
          )}

          <label className="wt-form-toggle">
            <input
              type="checkbox"
              checked={form.createBranch}
              onChange={(e) => updateField('createBranch', e.target.checked)}
            />
            <span>Create new branch</span>
          </label>

          {form.createBranch && (
            <>
              <label className="wt-form-label">Base Branch</label>
              <input
                className="wt-form-input"
                type="text"
                value={form.baseBranch}
                onChange={(e) => updateField('baseBranch', e.target.value)}
                placeholder="main"
              />
            </>
          )}
        </div>

        <div className="wt-dialog-footer">
          <button className="wt-btn wt-btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            className="wt-btn wt-btn-primary"
            onClick={handleSubmit}
            disabled={submitting}
          >
            <Plus size={14} />
            {submitting ? 'Creating...' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  );
}
