import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SessionGroup, TmuxSession } from '../../types'
import { TmuxTree } from './TmuxTree'

vi.mock('@dnd-kit/core', () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DragOverlay: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useDroppable: () => ({ setNodeRef: vi.fn(), isOver: false }),
  useSensor: (s: unknown) => s,
  useSensors: (...sensors: unknown[]) => sensors,
  closestCenter: vi.fn(),
  PointerSensor: vi.fn(),
  KeyboardSensor: vi.fn(),
}))

vi.mock('@dnd-kit/sortable', () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useSortable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    transition: null,
    isDragging: false,
  }),
  verticalListSortingStrategy: vi.fn(),
  sortableKeyboardCoordinates: vi.fn(),
}))

vi.mock('@dnd-kit/utilities', () => ({
  CSS: { Transform: { toString: () => '' } },
}))

vi.mock('./NewTmuxButton', () => ({
  NewTmuxButton: () => <div data-testid="new-tmux-button" />,
}))

const mockSessions: TmuxSession[] = [
  {
    sessionName: 'my-session',
    sessionId: '1',
    windows: [
      {
        windowIndex: 0,
        windowName: 'main',
        windowId: 'w1',
        panes: [
          { paneId: '%0', paneTitle: 'bash', paneCommand: 'vim' },
          { paneId: '%1', paneTitle: 'bash', paneCommand: 'node' },
        ],
      },
    ],
  },
]

const mockGroups: SessionGroup[] = [{ id: 1, group_name: 'work', sort_order: 0, session_count: 1 }]

describe('TmuxTree', () => {
  let onSelectPane: ReturnType<typeof vi.fn<(paneId: string, paneName: string) => void>>
  let onRefresh: ReturnType<typeof vi.fn<() => void>>

  beforeEach(() => {
    onSelectPane = vi.fn<(paneId: string, paneName: string) => void>()
    onRefresh = vi.fn<() => void>()
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders Sessions header and refresh button', () => {
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('renders session names from props', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('my-session')).toBeInTheDocument()
  })

  it('shows empty state when no sessions', () => {
    render(<TmuxTree sessions={[]} groups={[]} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    expect(screen.getByText('No sessions')).toBeInTheDocument()
  })

  it('shows windows when session is expanded', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('0: main')).toBeInTheDocument()
    expect(screen.getByText('vim')).toBeInTheDocument()
    expect(screen.getByText('node')).toBeInTheDocument()
  })

  it('expand/collapse toggles window visibility', async () => {
    const user = userEvent.setup()
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    // Initially collapsed - window not visible
    expect(screen.queryByText('0: main')).not.toBeInTheDocument()

    // Click expand button
    const expandBtn = screen.getByRole('button', { name: '' })
    await user.click(expandBtn)

    expect(screen.getByText('0: main')).toBeInTheDocument()

    // Collapse again
    await user.click(expandBtn)
    expect(screen.queryByText('0: main')).not.toBeInTheDocument()
  })

  it('calls onSelectPane when pane is clicked', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByText('vim'))

    expect(onSelectPane).toHaveBeenCalledWith('%0', 'my-session:0')
  })

  it('calls onPaneContextMenu on right-click', async () => {
    const onPaneContextMenu = vi.fn()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onPaneContextMenu={onPaneContextMenu}
        defaultExpanded
      />,
    )

    const paneNode = screen.getByText('vim').closest('.pane-node')
    expect(paneNode).toBeTruthy()

    await userEvent.pointer([{ keys: '[MouseRight]', target: paneNode! }])

    expect(onPaneContextMenu).toHaveBeenCalledWith('my-session:0:%0')
  })

  it('calls onPaneStatusClick when status badge is clicked', async () => {
    const onPaneStatusClick = vi.fn()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onPaneStatusClick={onPaneStatusClick}
        defaultExpanded
      />,
    )

    const statusBadge = screen
      .getByText('vim')
      .closest('.pane-node')
      ?.querySelector('.pane-status-clickable')
    expect(statusBadge).toBeTruthy()
    await userEvent.click(statusBadge!)

    expect(onPaneStatusClick).toHaveBeenCalledWith('my-session:0:%0')
  })

  it('renders grouped sessions under group nodes', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('work')).toBeInTheDocument()
  })

  it('calls onRefresh when refresh button is clicked', async () => {
    const user = userEvent.setup()
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    const refreshBtn = screen.getByTitle('Refresh')
    await user.click(refreshBtn)

    expect(onRefresh).toHaveBeenCalled()
  })
})
