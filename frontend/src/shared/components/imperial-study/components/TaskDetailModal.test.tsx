import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { TaskEventDetail, TaskRunDetail } from '../types'
import { TaskDetailModal } from './TaskDetailModal'

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

let mockRunDetail: {
  run: TaskRunDetail | null
  events: TaskEventDetail[]
  loading: boolean
  error: string | null
} = { run: mockRun, events: mockEvents, loading: false, error: null }

vi.mock('../hooks/useRunDetail', () => ({
  useRunDetail: () => mockRunDetail,
}))

describe('TaskDetailModal', () => {
  beforeEach(() => {
    mockRunDetail = { run: mockRun, events: mockEvents, loading: false, error: null }
  })

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

  it('shows loading state when loading and no run', () => {
    mockRunDetail = { run: null, events: [], loading: true, error: null }
    render(<TaskDetailModal runId="run-abc" onClose={vi.fn()} />)
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows error state when error and no run', () => {
    mockRunDetail = { run: null, events: [], loading: false, error: 'Failed to load' }
    render(<TaskDetailModal runId="run-abc" onClose={vi.fn()} />)
    expect(screen.getByText('Failed to load')).toBeInTheDocument()
  })

  it('shows "Run Detail" title when run is null', () => {
    mockRunDetail = { run: null, events: [], loading: false, error: null }
    render(<TaskDetailModal runId="run-abc" onClose={vi.fn()} />)
    expect(screen.getByText('Run Detail')).toBeInTheDocument()
  })

  it('renders thinking events with text payload', () => {
    mockRunDetail = {
      run: mockRun,
      events: [
        ...mockEvents,
        {
          id: 2,
          run_id: 'run-abcdef12',
          event_type: 'thinking',
          payload: { text: 'Let me analyze this...' },
          created_at: '2026-01-01T10:01:00Z',
        },
      ],
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText(/Thinking/)).toBeInTheDocument()
    expect(screen.getByText('Let me analyze this...')).toBeInTheDocument()
  })

  it('renders reasoning events in thinking section', () => {
    mockRunDetail = {
      run: mockRun,
      events: [
        {
          id: 3,
          run_id: 'run-abcdef12',
          event_type: 'reasoning',
          payload: { text: 'Step by step logic' },
          created_at: '2026-01-01T10:02:00Z',
        },
      ],
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('Step by step logic')).toBeInTheDocument()
  })

  it('renders result block when run has result', () => {
    mockRunDetail = {
      run: { ...mockRun, result: 'Task completed successfully' },
      events: mockEvents,
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('Result')).toBeInTheDocument()
    expect(screen.getByText('Task completed successfully')).toBeInTheDocument()
  })

  it('renders error block when run has error', () => {
    mockRunDetail = {
      run: { ...mockRun, error: 'Something went wrong' },
      events: mockEvents,
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('Error')).toBeInTheDocument()
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('renders ended_at when present', () => {
    mockRunDetail = {
      run: { ...mockRun, ended_at: '2026-01-01T11:00:00Z' },
      events: mockEvents,
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText(/Ended:/)).toBeInTheDocument()
  })

  it('renders em-dash for null intent in input_data', () => {
    mockRunDetail = {
      run: { ...mockRun, input_data: {} },
      events: mockEvents,
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    const intentSection = screen.getByText('Intent').parentElement!
    expect(intentSection.textContent).toContain('\u2014')
  })

  it('calls onClose when overlay is clicked', async () => {
    const onClose = vi.fn()
    render(<TaskDetailModal runId="run-abcdef12" onClose={onClose} />)
    const overlay = document.querySelector('.is-modal-overlay') as HTMLElement
    expect(overlay).toBeInTheDocument()
    await userEvent.click(overlay)
    expect(onClose).toHaveBeenCalled()
  })

  it('extracts string payload text', () => {
    mockRunDetail = {
      run: mockRun,
      events: [
        {
          id: 4,
          run_id: 'run-abcdef12',
          event_type: 'thinking',
          payload: 'plain text payload' as unknown as Record<string, unknown>,
          created_at: '2026-01-01T10:03:00Z',
        },
      ],
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('plain text payload')).toBeInTheDocument()
  })

  it('stringifies object payload without text field', () => {
    mockRunDetail = {
      run: mockRun,
      events: [
        {
          id: 5,
          run_id: 'run-abcdef12',
          event_type: 'thinking',
          payload: { data: 'complex', count: 42 },
          created_at: '2026-01-01T10:04:00Z',
        },
      ],
      loading: false,
      error: null,
    }
    render(<TaskDetailModal runId="run-abcdef12" onClose={vi.fn()} />)
    expect(screen.getByText('{"data":"complex","count":42}')).toBeInTheDocument()
  })
})
