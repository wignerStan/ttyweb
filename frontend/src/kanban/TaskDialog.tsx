import { useState, useEffect, type FormEvent } from 'react';
import { X, Trash2 } from 'lucide-react';
import type { KanbanTask, KanbanStatus, KanbanComment } from './types';
import { CommentThread } from './CommentThread';

interface TaskDialogProps {
  open: boolean;
  task: KanbanTask | null;
  defaultStatus?: KanbanStatus;
  onSave: (task: TaskFormData) => Promise<KanbanTask | null>;
  onCreate: (fields: Partial<TaskFormData>) => Promise<KanbanTask | null>;
  onDelete: (id: string) => Promise<boolean>;
  onClose: () => void;
  fetchComments: (taskId: string) => Promise<KanbanComment[]>;
  createComment: (taskId: string, content: string) => Promise<KanbanComment | null>;
}

export interface TaskFormData {
  title: string;
  description: string;
  status: KanbanStatus;
  priority: number;
  tags: string[];
  due_date: string | null;
}

const STATUS_OPTIONS: { value: KanbanStatus; label: string }[] = [
  { value: 'todo', label: 'Todo' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'done', label: 'Done' },
  { value: 'archived', label: 'Archived' },
];

const PRIORITY_OPTIONS = [
  { value: 1, label: '1 - Low' },
  { value: 2, label: '2 - Medium' },
  { value: 3, label: '3 - High' },
  { value: 4, label: '4 - Critical' },
  { value: 5, label: '5 - Urgent' },
];

function emptyForm(status: KanbanStatus): TaskFormData {
  return {
    title: '',
    description: '',
    status,
    priority: 2,
    tags: [],
    due_date: null,
  };
}

function formFromTask(task: KanbanTask): TaskFormData {
  return {
    title: task.title,
    description: task.description,
    status: task.status,
    priority: task.priority,
    tags: [...task.tags],
    due_date: task.due_date,
  };
}

function TaskDialog({
  open,
  task,
  defaultStatus = 'todo',
  onSave,
  onCreate,
  onDelete,
  onClose,
  fetchComments,
  createComment,
}: TaskDialogProps) {
  const [form, setForm] = useState<TaskFormData>(() =>
    task ? formFromTask(task) : emptyForm(defaultStatus),
  );
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (open) {
      setForm(task ? formFromTask(task) : emptyForm(defaultStatus));
      setErrors({});
    }
  }, [open, task, defaultStatus]);

  function updateField<K extends keyof TaskFormData>(key: K, value: TaskFormData[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
    if (key === 'title' && String(value).trim()) {
      setErrors((prev) => {
        if (!prev.title) return prev;
        const { title: _, ...rest } = prev;
        return rest;
      });
    }
  }

  function handleTagsInput(value: string) {
    const tags = value
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);
    updateField('tags', tags);
  }

  function validate(): boolean {
    const newErrors: Record<string, string> = {};
    if (!form.title.trim()) {
      newErrors['title'] = 'Title is required';
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!validate()) return;

    setSaving(true);
    try {
      if (task) {
        await onSave(form);
      } else {
        await onCreate(form);
      }
      onClose();
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!task) return;
    setDeleting(true);
    try {
      const ok = await onDelete(task.id);
      if (ok) {
        onClose();
      }
    } finally {
      setDeleting(false);
    }
  }

  if (!open) return null;

  const isEdit = task !== null;
  const title = isEdit ? 'Edit Task' : 'New Task';
  const tagsString = form.tags.join(', ');

  return (
    <div className="task-dialog-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="task-dialog">
        <div className="task-dialog__header">
          <h2 className="task-dialog__title">{title}</h2>
          <button className="task-dialog__close" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>

        <form id="task-dialog-form" className="task-dialog__body" onSubmit={handleSubmit}>
          <div className="task-dialog__field">
            <label className="task-dialog__label" htmlFor="task-title">
              Title *
            </label>
            <input
              id="task-title"
              className="task-dialog__input"
              type="text"
              value={form.title}
              onChange={(e) => updateField('title', e.target.value)}
              placeholder="Task title"
              autoFocus
            />
            {errors['title'] && (
              <span className="task-dialog__error">{errors['title']}</span>
            )}
          </div>

          <div className="task-dialog__field">
            <label className="task-dialog__label" htmlFor="task-desc">
              Description
            </label>
            <textarea
              id="task-desc"
              className="task-dialog__textarea"
              value={form.description}
              onChange={(e) => updateField('description', e.target.value)}
              placeholder="Describe the task..."
              rows={4}
            />
          </div>

          <div className="task-dialog__row">
            <div className="task-dialog__field">
              <label className="task-dialog__label" htmlFor="task-status">
                Status
              </label>
              <select
                id="task-status"
                className="task-dialog__select"
                value={form.status}
                onChange={(e) => updateField('status', e.target.value as KanbanStatus)}
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="task-dialog__field">
              <label className="task-dialog__label" htmlFor="task-priority">
                Priority
              </label>
              <select
                id="task-priority"
                className="task-dialog__select"
                value={form.priority}
                onChange={(e) => updateField('priority', Number(e.target.value))}
              >
                {PRIORITY_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="task-dialog__row">
            <div className="task-dialog__field">
              <label className="task-dialog__label" htmlFor="task-tags">
                Tags
              </label>
              <input
                id="task-tags"
                className="task-dialog__input"
                type="text"
                value={tagsString}
                onChange={(e) => handleTagsInput(e.target.value)}
                placeholder="bug, feature, urgent"
              />
            </div>

            <div className="task-dialog__field">
              <label className="task-dialog__label" htmlFor="task-due">
                Due Date
              </label>
              <input
                id="task-due"
                className="task-dialog__input"
                type="date"
                value={form.due_date ?? ''}
                onChange={(e) => updateField('due_date', e.target.value || null)}
              />
            </div>
          </div>

          {isEdit && (
            <>
              <div className="task-dialog__divider" />
              <CommentThread
                taskId={task.id}
                fetchComments={fetchComments}
                createComment={createComment}
              />
            </>
          )}
        </form>

        <div className="task-dialog__footer">
          {isEdit && (
            <button
              className="task-dialog__btn task-dialog__btn--danger"
              onClick={handleDelete}
              disabled={deleting}
              type="button"
              style={{ marginRight: 'auto' }}
            >
              <Trash2 size={13} />
              {deleting ? 'Deleting...' : 'Delete'}
            </button>
          )}
          <button
            className="task-dialog__btn task-dialog__btn--cancel"
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
          <button
            className="task-dialog__btn task-dialog__btn--primary"
            type="submit"
            form="task-dialog-form"
            disabled={saving || !form.title.trim()}
          >
            {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Task'}
          </button>
        </div>
      </div>
    </div>
  );
}

export { TaskDialog };
export type { TaskDialogProps };
