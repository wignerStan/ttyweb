import { Trash2, X } from 'lucide-react'
import { type FormEvent, useEffect, useState } from 'react'
import { CommentThread } from './CommentThread'
import type { KanbanComment, KanbanStatus, KanbanTask } from './types'

interface TaskDialogProps {
  open: boolean
  task: KanbanTask | null
  defaultStatus?: KanbanStatus
  onSave: (task: TaskFormData) => Promise<KanbanTask | null>
  onCreate: (fields: Partial<TaskFormData>) => Promise<KanbanTask | null>
  onDelete: (id: string) => Promise<boolean>
  onClose: () => void
  fetchComments: (taskId: string) => Promise<KanbanComment[]>
  createComment: (taskId: string, content: string) => Promise<KanbanComment | null>
}

export interface TaskFormData {
  title: string
  description: string
  status: KanbanStatus
  priority: number
  tags: string[]
  due_date: string | null
}

const STATUS_OPTIONS: { value: KanbanStatus; label: string }[] = [
  { value: 'todo', label: 'Todo' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'done', label: 'Done' },
  { value: 'archived', label: 'Archived' },
]

const PRIORITY_OPTIONS = [
  { value: 1, label: '1 - Low' },
  { value: 2, label: '2 - Medium' },
  { value: 3, label: '3 - High' },
  { value: 4, label: '4 - Critical' },
  { value: 5, label: '5 - Urgent' },
]

function emptyForm(status: KanbanStatus): TaskFormData {
  return {
    title: '',
    description: '',
    status,
    priority: 2,
    tags: [],
    due_date: null,
  }
}

function formFromTask(task: KanbanTask): TaskFormData {
  return {
    title: task.title,
    description: task.description,
    status: task.status,
    priority: task.priority,
    tags: [...task.tags],
    due_date: task.due_date,
  }
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
  )
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (open) {
      setForm(task ? formFromTask(task) : emptyForm(defaultStatus))
      setErrors({})
    }
  }, [open, task, defaultStatus])

  function updateField<K extends keyof TaskFormData>(key: K, value: TaskFormData[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
    if (key === 'title' && String(value).trim()) {
      setErrors((prev) => {
        if (!prev.title) return prev
        const { title: _, ...rest } = prev
        return rest
      })
    }
  }

  function handleTagsInput(value: string) {
    const tags = value
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    updateField('tags', tags)
  }

  function validate(): boolean {
    const newErrors: Record<string, string> = {}
    if (!form.title.trim()) {
      newErrors.title = 'Title is required'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!validate()) return

    setSaving(true)
    try {
      if (task) {
        await onSave(form)
      } else {
        await onCreate(form)
      }
      onClose()
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!task) return
    setDeleting(true)
    try {
      const ok = await onDelete(task.id)
      if (ok) {
        onClose()
      }
    } finally {
      setDeleting(false)
    }
  }

  if (!open) return null

  const isEdit = task !== null
  const title = isEdit ? 'Edit Task' : 'New Task'
  const tagsString = form.tags.join(', ')

  return (
    <>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: overlay click-outside-to-close pattern */}
      <div
        data-testid="task-dialog-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="task-dialog-title"
        className="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-5 max-md:m-2.5"
        onClick={(e) => {
          if (e.target === e.currentTarget) onClose()
        }}
      >
        <div
          data-testid="task-dialog"
          className="flex max-h-[90vh] w-full max-w-[540px] flex-col overflow-hidden rounded-box border border-base-300 bg-base-100 text-base-content shadow-lg max-md:max-w-full"
        >
          <div className="flex flex-shrink-0 items-center justify-between border-b border-base-200 px-5 py-4">
            <h2 id="task-dialog-title" className="m-0 text-[15px] font-semibold text-base-content">
              {title}
            </h2>
            <button
              className="flex h-7 w-7 items-center justify-center rounded text-base-content/50 transition-colors hover:bg-base-200 hover:text-base-content"
              onClick={onClose}
              type="button"
              aria-label="Close dialog"
            >
              <X size={16} />
            </button>
          </div>

          <form
            id="task-dialog-form"
            className="flex flex-1 flex-col gap-4 overflow-y-auto px-5 py-5"
            onSubmit={handleSubmit}
          >
            <div className="flex flex-col gap-1.5">
              <label
                className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                htmlFor="task-title"
              >
                Title *
              </label>
              <input
                id="task-title"
                className="input input-bordered input-sm w-full font-mono text-xs"
                type="text"
                name="task-title"
                autoComplete="off"
                value={form.title}
                onChange={(e) => updateField('title', e.target.value)}
                placeholder="Task title"
                // biome-ignore lint/a11y/noAutofocus: dialog should focus title input
                autoFocus
              />
              {errors.title && <span className="text-xs text-error">{errors.title}</span>}
            </div>

            <div className="flex flex-col gap-1.5">
              <label
                className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                htmlFor="task-desc"
              >
                Description
              </label>
              <textarea
                id="task-desc"
                className="textarea textarea-bordered textarea-sm w-full font-mono text-xs min-h-[80px]"
                value={form.description}
                onChange={(e) => updateField('description', e.target.value)}
                placeholder="Describe the task..."
                rows={4}
              />
            </div>

            <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
              <div className="flex flex-col gap-1.5">
                <label
                  className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                  htmlFor="task-status"
                >
                  Status
                </label>
                <select
                  id="task-status"
                  className="select select-bordered select-sm w-full cursor-pointer font-mono text-xs"
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

              <div className="flex flex-col gap-1.5">
                <label
                  className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                  htmlFor="task-priority"
                >
                  Priority
                </label>
                <select
                  id="task-priority"
                  className="select select-bordered select-sm w-full cursor-pointer font-mono text-xs"
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

            <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
              <div className="flex flex-col gap-1.5">
                <label
                  className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                  htmlFor="task-tags"
                >
                  Tags
                </label>
                <input
                  id="task-tags"
                  className="input input-bordered input-sm w-full font-mono text-xs"
                  type="text"
                  value={tagsString}
                  onChange={(e) => handleTagsInput(e.target.value)}
                  placeholder="bug, feature, urgent"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <label
                  className="text-xs font-semibold uppercase tracking-wide text-base-content/50"
                  htmlFor="task-due"
                >
                  Due Date
                </label>
                <input
                  id="task-due"
                  className="input input-bordered input-sm w-full font-mono text-xs"
                  type="date"
                  value={form.due_date ?? ''}
                  onChange={(e) => updateField('due_date', e.target.value || null)}
                />
              </div>
            </div>

            {isEdit && (
              <>
                <div className="my-1 h-px bg-base-200" />
                <CommentThread
                  taskId={task.id}
                  fetchComments={fetchComments}
                  createComment={createComment}
                />
              </>
            )}
          </form>

          <div className="flex flex-shrink-0 items-center justify-end gap-2 border-t border-base-200 px-5 py-3">
            {isEdit && (
              <button
                className="btn btn-error btn-sm mr-auto gap-1.5 disabled:opacity-40"
                onClick={handleDelete}
                disabled={deleting}
                type="button"
              >
                <Trash2 size={13} />
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
            )}
            <button className="btn btn-ghost btn-sm gap-1.5" onClick={onClose} type="button">
              Cancel
            </button>
            <button
              className="btn btn-primary btn-sm gap-1.5 disabled:opacity-40"
              type="submit"
              form="task-dialog-form"
              disabled={saving || !form.title.trim()}
            >
              {saving ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Task'}
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

export type { TaskDialogProps }
export { TaskDialog }
