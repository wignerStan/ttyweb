import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { TaskEventDetail, TaskRunDetail } from '../types'
import { TaskDetailModal } from './TaskDetailModal'

vi.mock('../hooks/useRunDetail', () => ({
  useRunDetail: () => ({ run: mockRun, events: mockEvents, loading: false, error: null }),
}))

const mockRun: TaskRunDetail = {
  id: 'run-abcdef12',
  task_id: 'task-12345678',
  state: 'running',
  trigger: null,
  attempt: 1,
  input_data: { intent: 'fix bug' },
  result: null,
  error: null,
  queued_at: null,
  started_at: '2026-01-01T10:00:00Z',
  ended_at: null,
  estimated_at: null,
}

const mockEvents: TaskEventDetail[] = [
  {
    id: 1,
    run_id: 'run-abcdef12',
    event_type: 'task_started',
    payload: null,
    created_at: '2026-01-01T10:00:00Z',
  },
]

describe('TaskDetailModal', () => {
  it('renders run detail info', () => {
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText(/Run: run-abc/)).toBeInTheDocument()
    expect(screen.getByText('running')).toBeInTheDocument()
    expect(screen.getByText('fix bug')).toBeInTheDocument()
  })

  it('renders event timeline', () => {
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('Events (1)')).toBeInTheDocument()
    expect(screen.getByText('task_started')).toBeInTheDocument()
  })

  it('close button calls onClose', async () => {
    const onClose = vi.fn()
    render(<TaskDetailModal runId="run-abcdef12" onClose={onClose} />)
    await userEvent.click(screen.getByTitle('Close'))
    expect(onClose).toHaveBeenCalled()
  })

  it('Escape key calls onClose', async () => {
    const onClose = vi.fn()
    render(<TaskDetailModal runId="run-abcdef12" onClose={onClose} />)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
