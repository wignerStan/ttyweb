import { fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { TaskDialog } from './TaskDialog'
import type { KanbanTask } from './types'

vi.mock('./CommentThread', () => ({
  CommentThread: () => <div data-testid="comment-thread" />,
}))

const baseTask: KanbanTask = {
  id: '1',
  title: 'Existing task',
  description: 'A description',
  status: 'in_progress',
  priority: 3,
  tags: ['bug'],
  due_date: '2026-06-01',
  order_index: 0,
  created_at: '',
  updated_at: '',
}

const defaultProps = {
  open: true,
  task: null as KanbanTask | null,
  onSave: vi.fn().mockResolvedValue(null),
  onCreate: vi.fn().mockResolvedValue(null),
  onDelete: vi.fn().mockResolvedValue(true),
  onClose: vi.fn(),
  fetchComments: vi.fn().mockResolvedValue([]),
  createComment: vi.fn().mockResolvedValue(null),
}

describe('TaskDialog', () => {
  it('renders modal when open={true}', () => {
    renderWithProviders(<TaskDialog {...defaultProps} open={true} />)
    expect(screen.getByText('New Task')).toBeInTheDocument()
    expect(screen.getByText('Create Task')).toBeInTheDocument()
  })

  it('does not render when open={false}', () => {
    renderWithProviders(<TaskDialog {...defaultProps} open={false} />)
    expect(screen.queryByText('New Task')).not.toBeInTheDocument()
  })

  it('prefills fields when task prop provided (edit mode)', () => {
    renderWithProviders(<TaskDialog {...defaultProps} task={baseTask} />)
    expect(screen.getByText('Edit Task')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Existing task')).toBeInTheDocument()
    expect(screen.getByDisplayValue('A description')).toBeInTheDocument()
    expect(screen.getByDisplayValue('bug')).toBeInTheDocument()
  })

  it('calls onSave with form data on submit', async () => {
    const onSave = vi.fn().mockResolvedValue(null)
    const task = { ...baseTask, title: 'Updated title' }
    renderWithProviders(<TaskDialog {...defaultProps} task={task} onSave={onSave} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Save Changes'))
    expect(onSave).toHaveBeenCalledOnce()
    const formData = onSave.mock.calls[0][0]
    expect(formData.title).toBe('Updated title')
  })

  it('calls onClose when cancel clicked', async () => {
    const onClose = vi.fn()
    renderWithProviders(<TaskDialog {...defaultProps} onClose={onClose} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Cancel'))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('tag input add and remove', async () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    const tagsInput = screen.getByPlaceholderText('bug, feature, urgent')
    // Use fireEvent.change to set value directly (avoids per-char onChange)
    fireEvent.change(tagsInput, { target: { value: 'security, api' } })
    expect(tagsInput).toHaveValue('security, api')
    // Clear the field to remove tags
    fireEvent.change(tagsInput, { target: { value: '' } })
    expect(tagsInput).toHaveValue('')
  })
})
