import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ImperialStudyPanel } from './ImperialStudyPanel'

vi.mock('../hooks/useWorkerSessions', () => ({
  useWorkerSessions: () => ({ workers: [], loading: false, error: null, refetch: vi.fn() }),
}))

vi.mock('../hooks/useInboxItems', () => ({
  useInboxItems: () => ({
    items: [],
    unreadCount: 0,
    loading: false,
    error: null,
    refetch: vi.fn(),
  }),
}))

vi.mock('../hooks/useActivityEvents', () => ({
  useActivityEvents: () => ({ events: [], loading: false, error: null, refetch: vi.fn() }),
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
  CommandInput: () => <div data-testid="command-input">CommandInput</div>,
}))

vi.mock('./WorkerSection', () => ({
  WorkerSection: () => <div data-testid="workers-section">Workers</div>,
}))

vi.mock('./InboxSection', () => ({
  InboxSection: () => <div data-testid="inbox-section">Inbox</div>,
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
  InboxDetailModal: () => <div data-testid="inbox-modal">Modal</div>,
}))

vi.mock('./TaskDetailModal', () => ({
  TaskDetailModal: () => <div data-testid="task-modal">Modal</div>,
}))

describe('ImperialStudyPanel', () => {
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
})
