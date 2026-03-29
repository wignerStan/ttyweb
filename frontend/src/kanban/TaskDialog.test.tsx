import { fireEvent, screen, waitFor } from '@testing-library/react'
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
    const formData = onSave.mock.calls[0]![0]!
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

  it('calls onCreate for new task on submit', async () => {
    const onCreate = vi.fn().mockResolvedValue(null)
    renderWithProviders(<TaskDialog {...defaultProps} onCreate={onCreate} />)
    const titleInput = screen.getByPlaceholderText('Task title')
    fireEvent.change(titleInput, { target: { value: 'New task title' } })
    const user = userEvent.setup()
    await user.click(screen.getByText('Create Task'))
    expect(onCreate).toHaveBeenCalledOnce()
  })

  it('shows title validation error when submitting empty title', async () => {
    const onCreate = vi.fn().mockResolvedValue(null)
    renderWithProviders(<TaskDialog {...defaultProps} onCreate={onCreate} />)
    const form = screen.getByPlaceholderText('Task title').closest('form')!
    fireEvent.submit(form)
    expect(screen.getByText('Title is required')).toBeInTheDocument()
    expect(onCreate).not.toHaveBeenCalled()
  })

  it('clears title error when user types a title', async () => {
    renderWithProviders(<TaskDialog {...defaultProps} onCreate={vi.fn()} />)
    const form = screen.getByPlaceholderText('Task title').closest('form')!
    fireEvent.submit(form)
    expect(screen.getByText('Title is required')).toBeInTheDocument()
    const titleInput = screen.getByPlaceholderText('Task title')
    fireEvent.change(titleInput, { target: { value: 'Now valid' } })
    expect(screen.queryByText('Title is required')).not.toBeInTheDocument()
  })

  it('delete button calls onDelete', async () => {
    const onDelete = vi.fn().mockResolvedValue(true)
    renderWithProviders(<TaskDialog {...defaultProps} task={baseTask} onDelete={onDelete} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Delete'))
    expect(onDelete).toHaveBeenCalledWith('1')
  })

  it('calls onClose after successful delete', async () => {
    const onClose = vi.fn()
    const onDelete = vi.fn().mockResolvedValue(true)
    renderWithProviders(
      <TaskDialog {...defaultProps} task={baseTask} onDelete={onDelete} onClose={onClose} />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByText('Delete'))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('does not close after failed delete', async () => {
    const onClose = vi.fn()
    const onDelete = vi.fn().mockResolvedValue(false)
    renderWithProviders(
      <TaskDialog {...defaultProps} task={baseTask} onDelete={onDelete} onClose={onClose} />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByText('Delete'))
    await waitFor(() => expect(onDelete).toHaveBeenCalled())
    expect(onClose).not.toHaveBeenCalled()
  })

  it('overlay click calls onClose', async () => {
    const onClose = vi.fn()
    renderWithProviders(<TaskDialog {...defaultProps} onClose={onClose} />)
    const overlay = document.querySelector('.task-dialog-overlay')!
    await userEvent.click(overlay)
    expect(onClose).toHaveBeenCalled()
  })

  it('does not close on inner modal click', async () => {
    const onClose = vi.fn()
    renderWithProviders(<TaskDialog {...defaultProps} onClose={onClose} />)
    const modal = document.querySelector('.task-dialog')!
    await userEvent.click(modal)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('renders CommentThread in edit mode', () => {
    renderWithProviders(<TaskDialog {...defaultProps} task={baseTask} />)
    expect(screen.getByTestId('comment-thread')).toBeInTheDocument()
  })

  it('does not render CommentThread in create mode', () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    expect(screen.queryByTestId('comment-thread')).not.toBeInTheDocument()
  })

  it('does not render Delete button in create mode', () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    expect(screen.queryByText('Delete')).not.toBeInTheDocument()
  })

  it('status and priority selects are rendered', () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    expect(screen.getByLabelText('Status')).toBeInTheDocument()
    expect(screen.getByLabelText('Priority')).toBeInTheDocument()
  })

  it('due date input is rendered', () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    expect(screen.getByLabelText('Due Date')).toBeInTheDocument()
  })

  it('prefills due date in edit mode', () => {
    renderWithProviders(<TaskDialog {...defaultProps} task={baseTask} />)
    expect(screen.getByDisplayValue('2026-06-01')).toBeInTheDocument()
  })

  it('Create Task button is disabled when title is empty', () => {
    renderWithProviders(<TaskDialog {...defaultProps} />)
    const btn = screen.getByText('Create Task').closest('button')!
    expect(btn).toBeDisabled()
  })

  it('prefills default status from defaultStatus prop', () => {
    renderWithProviders(<TaskDialog {...defaultProps} defaultStatus="in_progress" />)
    const statusSelect = screen.getByLabelText('Status') as HTMLSelectElement
    expect(statusSelect.value).toBe('in_progress')
  })
})
