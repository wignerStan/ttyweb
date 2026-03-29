import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { InboxItem } from '../types'
import { ImperialStudyPanel } from './ImperialStudyPanel'

const mockRefetch = vi.fn()

vi.mock('../hooks/useWorkerSessions', () => ({
  useWorkerSessions: () => ({ workers: [], loading: false, error: null, refetch: mockRefetch }),
}))

vi.mock('../hooks/useInboxItems', () => ({
  useInboxItems: () => ({
    items: [],
    unreadCount: 0,
    loading: false,
    error: null,
    refetch: mockRefetch,
  }),
}))

vi.mock('../hooks/useActivityEvents', () => ({
  useActivityEvents: () => ({ events: [], loading: false, error: null, refetch: mockRefetch }),
}))

vi.mock('../hooks/useAssistantPanes', () => ({
  useAssistantPanes: () => ({
    messages: [],
    streaming: false,
    sendMessage: vi.fn(),
    clearMessages: vi.fn(),
  }),
}))

vi.mock('../hooks/useRunPipeline', () => ({
  useRunPipeline: () => ({ runs: [], activeRun: null, dispatch: vi.fn(), dismiss: vi.fn() }),
}))

vi.mock('./CommandInput', () => ({
  CommandInput: ({
    onDispatched,
    onAssistantSend,
  }: {
    onDispatched?: (r: {
      run_id: string
      task_id: string
      routing?: unknown
      intent: string
    }) => void
    onAssistantSend?: (content: string, type: string) => void
    [key: string]: unknown
  }) => (
    <div data-testid="command-input">
      <button
        data-testid="cmd-dispatch"
        onClick={() => onDispatched?.({ run_id: 'r1', task_id: 't1', intent: 'test' })}
      >
        Dispatch
      </button>
      <button data-testid="cmd-assistant" onClick={() => onAssistantSend?.('hello', 'chat')}>
        Assistant
      </button>
    </div>
  ),
}))

vi.mock('./WorkerSection', () => ({
  WorkerSection: ({ onWorkerClick }: { onWorkerClick?: (runId: string) => void }) => (
    <div data-testid="workers-section">
      Workers
      <button data-testid="worker-click" onClick={() => onWorkerClick?.('r1')}>
        Click worker
      </button>
    </div>
  ),
}))

vi.mock('./InboxSection', () => ({
  InboxSection: ({ onItemClick }: { onItemClick?: (item: InboxItem) => void }) => {
    const item: InboxItem = {
      id: 'inbox-1',
      study_id: 's1',
      worker_id: 'w1',
      run_id: 'r1',
      kind: 'question',
      status: 'pending',
      title: 'Test',
      body: 'Body',
      metadata: {},
      created_at: null,
      updated_at: null,
    }
    return (
      <div data-testid="inbox-section">
        Inbox
        <button data-testid="inbox-click" onClick={() => onItemClick?.(item)}>
          Click inbox
        </button>
      </div>
    )
  },
}))

vi.mock('./ActivitySection', () => ({
  ActivitySection: () => <div data-testid="activity-section">Activity</div>,
}))

vi.mock('./RunPipeline', () => ({
  RunPipeline: () => <div data-testid="run-pipeline">Pipeline</div>,
}))

vi.mock('./AssistantChatPanel', () => ({
  AssistantChatPanel: () => <div data-testid="chat-panel">Chat</div>,
}))

vi.mock('./InboxDetailModal', () => ({
  InboxDetailModal: ({ onReplied, onClose }: { onReplied: () => void; onClose: () => void }) => (
    <div data-testid="inbox-modal">
      Modal
      <button data-testid="modal-replied" onClick={onReplied}>
        Replied
      </button>
      <button data-testid="modal-close" onClick={onClose}>
        Close
      </button>
    </div>
  ),
}))

vi.mock('./TaskDetailModal', () => ({
  TaskDetailModal: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="task-modal">
      Modal
      <button data-testid="task-modal-close" onClick={onClose}>
        Close
      </button>
    </div>
  ),
}))

describe('ImperialStudyPanel', () => {
  beforeEach(() => {
    mockRefetch.mockClear()
  })

  it('renders panel header when not floating', () => {
    render(<ImperialStudyPanel />)
    expect(screen.getByText('Imperial Study')).toBeInTheDocument()
    expect(screen.getByText(/0 workers/)).toBeInTheDocument()
    expect(screen.getByText(/0 inbox/)).toBeInTheDocument()
  })

  it('hides panel header when floating', () => {
    render(<ImperialStudyPanel floating />)
    expect(screen.queryByText(/0 workers/)).not.toBeInTheDocument()
  })

  it('renders command input', () => {
    render(<ImperialStudyPanel />)
    expect(screen.getByTestId('command-input')).toBeInTheDocument()
  })

  it('renders sections', () => {
    render(<ImperialStudyPanel />)
    expect(screen.getByTestId('workers-section')).toBeInTheDocument()
    expect(screen.getByTestId('inbox-section')).toBeInTheDocument()
    expect(screen.getByTestId('activity-section')).toBeInTheDocument()
  })

  it('has refresh button', () => {
    render(<ImperialStudyPanel />)
    expect(screen.getByTitle('Refresh')).toBeInTheDocument()
  })

  it('scroll area contains all sections', () => {
    render(<ImperialStudyPanel />)
    const scrollArea = document.querySelector('.is-scroll-area')
    expect(scrollArea).toContainElement(screen.getByTestId('workers-section'))
    expect(scrollArea).toContainElement(screen.getByTestId('inbox-section'))
    expect(scrollArea).toContainElement(screen.getByTestId('activity-section'))
  })

  it('applies is-floating class when floating prop is true', () => {
    render(<ImperialStudyPanel floating />)
    const panel = document.querySelector('.imperial-study')
    expect(panel?.className).toContain('is-floating')
  })

  it('does not apply is-floating class by default', () => {
    render(<ImperialStudyPanel />)
    const panel = document.querySelector('.imperial-study')
    expect(panel?.className).not.toContain('is-floating')
  })

  it('refresh button triggers refetch', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTitle('Refresh'))
    await waitFor(() => expect(mockRefetch).toHaveBeenCalled())
  })

  it('opens task detail modal when worker is clicked', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTestId('worker-click'))
    expect(screen.getByTestId('task-modal')).toBeInTheDocument()
  })

  it('closes task detail modal', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTestId('worker-click'))
    expect(screen.getByTestId('task-modal')).toBeInTheDocument()
    await user.click(screen.getByTestId('task-modal-close'))
    expect(screen.queryByTestId('task-modal')).not.toBeInTheDocument()
  })

  it('opens inbox detail modal when inbox item is clicked', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTestId('inbox-click'))
    expect(screen.getByTestId('inbox-modal')).toBeInTheDocument()
  })

  it('closes inbox detail modal on close', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTestId('inbox-click'))
    expect(screen.getByTestId('inbox-modal')).toBeInTheDocument()
    await user.click(screen.getByTestId('modal-close'))
    expect(screen.queryByTestId('inbox-modal')).not.toBeInTheDocument()
  })

  it('closes inbox detail modal on replied and refetches', async () => {
    const user = userEvent.setup()
    render(<ImperialStudyPanel />)
    await user.click(screen.getByTestId('inbox-click'))
    await user.click(screen.getByTestId('modal-replied'))
    expect(screen.queryByTestId('inbox-modal')).not.toBeInTheDocument()
    await waitFor(() => expect(mockRefetch).toHaveBeenCalled())
  })
})
