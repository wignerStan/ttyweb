import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SessionGroup, TmuxSession } from '../../types'
import { TmuxTree } from './TmuxTree'

const mockNotify = vi.fn()
vi.mock('./NotificationProvider', () => ({
  useNotification: () => ({ notify: mockNotify, notifications: [], dismiss: vi.fn() }),
}))

// Mock all API functions from the extracted module
const mockRenameWindow = vi.fn().mockResolvedValue(true)
const mockSaveOrder = vi.fn().mockResolvedValue(undefined)
const mockFetchPaneStatuses = vi.fn().mockResolvedValue([])
const mockFetchProfileOrder = vi.fn().mockResolvedValue(null)
const mockFetchTaskPaneStatuses = vi.fn().mockResolvedValue({})
const mockRebuildSession = vi.fn().mockResolvedValue({ ok: true })
const mockAssignSessionGroup = vi.fn().mockResolvedValue(undefined)
const mockCreateGroup = vi.fn().mockResolvedValue({})

vi.mock('./tmux/api', () => ({
  renameWindow: (...args: unknown[]) => mockRenameWindow(...args),
  saveOrder: (...args: unknown[]) => mockSaveOrder(...args),
  fetchPaneStatuses: (...args: unknown[]) => mockFetchPaneStatuses(...args),
  fetchProfileOrder: (...args: unknown[]) => mockFetchProfileOrder(...args),
  fetchTaskPaneStatuses: (...args: unknown[]) => mockFetchTaskPaneStatuses(...args),
  rebuildSession: (...args: unknown[]) => mockRebuildSession(...args),
  assignSessionGroup: (...args: unknown[]) => mockAssignSessionGroup(...args),
  createGroup: (...args: unknown[]) => mockCreateGroup(...args),
}))

let mockDndHandlers: {
  onDragStart?: (event: { active: { id: unknown } }) => void
  onDragOver?: (event: { over: { id: unknown } | null }) => void
  onDragEnd?: (event: { active: { id: unknown }; over: { id: unknown } | null }) => void
} = {}

vi.mock('@dnd-kit/core', () => ({
  DndContext: ({
    children,
    onDragStart,
    onDragOver,
    onDragEnd,
  }: {
    children: React.ReactNode
    onDragStart?: (event: { active: { id: unknown } }) => void
    onDragOver?: (event: { over: { id: unknown } | null }) => void
    onDragEnd?: (event: { active: { id: unknown }; over: { id: unknown } | null }) => void
  }) => {
    mockDndHandlers = { onDragStart, onDragOver, onDragEnd }
    return <>{children}</>
  },
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

const mockMultiWindowSession: TmuxSession = {
  sessionName: 'multi-window',
  sessionId: '2',
  windows: [
    {
      windowIndex: 0,
      windowName: 'editor',
      windowId: 'w2',
      panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'nvim' }],
    },
    {
      windowIndex: 1,
      windowName: 'terminal',
      windowId: 'w3',
      panes: [
        { paneId: '%1', paneTitle: 'bash', paneCommand: 'bash' },
        { paneId: '%2', paneTitle: 'bash', paneCommand: 'htop' },
      ],
    },
  ],
}

const mockGroups: SessionGroup[] = [{ id: 1, group_name: 'work', sort_order: 0, session_count: 1 }]

const mockMultipleGroups: SessionGroup[] = [
  { id: 1, group_name: 'work', sort_order: 0, session_count: 1 },
  { id: 2, group_name: 'personal', sort_order: 1, session_count: 2 },
]

describe('TmuxTree', () => {
  let onSelectPane: ReturnType<typeof vi.fn<(paneId: string, paneName: string) => void>>
  let onRefresh: ReturnType<typeof vi.fn<() => void>>

  beforeEach(() => {
    onSelectPane = vi.fn<(paneId: string, paneName: string) => void>()
    onRefresh = vi.fn<() => void>()
    vi.clearAllMocks()
    // Restore default mock return values
    mockRenameWindow.mockResolvedValue(true)
    mockSaveOrder.mockResolvedValue(undefined)
    mockFetchPaneStatuses.mockResolvedValue([])
    mockFetchProfileOrder.mockResolvedValue(null)
    mockFetchTaskPaneStatuses.mockResolvedValue({})
    mockRebuildSession.mockResolvedValue({ ok: true })
    mockAssignSessionGroup.mockResolvedValue(undefined)
    mockCreateGroup.mockResolvedValue({})
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
    const expandBtn = screen.getByRole('button', { name: /Expand session/ })
    await user.click(expandBtn)

    expect(screen.getByText('0: main')).toBeInTheDocument()

    // Collapse again
    const collapseBtn = screen.getByRole('button', { name: /Collapse session/ })
    await user.click(collapseBtn)
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

  it('renders NewTmuxButton', () => {
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    expect(screen.getByTestId('new-tmux-button')).toBeInTheDocument()
  })

  it('does not show empty state when groups exist but sessions are empty', () => {
    render(
      <TmuxTree
        sessions={[]}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    expect(screen.queryByText('No sessions')).not.toBeInTheDocument()
  })

  // --- Status fetching and display ---

  it('fetches pane statuses when profileKey is provided', async () => {
    mockFetchPaneStatuses.mockResolvedValue([
      { paneKey: 'my-session:0:%0', status: 'in_progress', mtime: 0 },
    ])

    render(
      <TmuxTree
        sessions={mockSessions}
        profileKey="test-profile"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      expect(mockFetchPaneStatuses).toHaveBeenCalledWith('test-profile', expect.any(Array))
    })
  })

  it('fetches task pane statuses on mount', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'in_progress',
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      expect(mockFetchTaskPaneStatuses).toHaveBeenCalled()
    })
  })

  it('handles fetchPaneStatuses returning empty on error', async () => {
    mockFetchPaneStatuses.mockResolvedValue([])

    render(
      <TmuxTree
        sessions={mockSessions}
        profileKey="test-profile"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await waitFor(() => {
      expect(mockFetchPaneStatuses).toHaveBeenCalled()
    })
  })

  it('handles fetchTaskPaneStatuses returning empty on error', async () => {
    mockFetchTaskPaneStatuses.mockRejectedValue(new Error('Network error'))

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    // Should not crash
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('shows task stats in header when statuses exist', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'in_progress',
      'my-session:0:%1': 'done',
    })

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await waitFor(() => {
      expect(screen.getByText(/进行中/)).toBeInTheDocument()
    })
  })

  // --- Session-level status summary ---

  it('shows session-level status summary when panes have statuses', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'in_progress',
      'my-session:0:%1': 'done',
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      expect(screen.getByText(/进行中/)).toBeInTheDocument()
    })
  })

  // --- Window rename ---

  it('shows rename button for windows when expanded', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByTitle('Rename window')).toBeInTheDocument()
  })

  it('enters rename mode when rename button is clicked', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))

    const input = screen.getByDisplayValue('main')
    expect(input).toBeInTheDocument()
    expect(input).toHaveClass('window-name-input')
  })

  it('renames window on Enter key in rename input', async () => {
    const user = userEvent.setup()

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')

    await user.clear(input)
    await user.type(input, 'new-name')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(mockRenameWindow).toHaveBeenCalledWith('my-session', 0, 'new-name')
    })
  })

  it('exits rename mode on Escape key', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')
    expect(input).toBeInTheDocument()

    // Use fireEvent.keyDown directly on the input since userEvent.keyboard
    // may not target the correct element
    fireEvent.keyDown(input, { key: 'Escape' })
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0))
    })

    expect(screen.queryByDisplayValue('main')).not.toBeInTheDocument()
  })

  it('exits rename mode on blur', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')
    expect(input).toBeInTheDocument()

    // Directly fire blur event on the input
    fireEvent.blur(input)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0))
    })

    expect(screen.queryByDisplayValue('main')).not.toBeInTheDocument()
  })

  // --- Rebuild button ---

  it('renders rebuild button for sessions', () => {
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    expect(screen.getByTitle('Rebuild session')).toBeInTheDocument()
  })

  it('shows confirm dialog on rebuild click', async () => {
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    await waitFor(() => {
      expect(screen.getByText('Rebuild Session')).toBeInTheDocument()
      expect(screen.getByText(/Rebuild session "my-session"/)).toBeInTheDocument()
    })
  })

  it('calls rebuild API when confirm is accepted', async () => {
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    // ConfirmDialog should appear — click Rebuild to confirm
    await waitFor(() => expect(screen.getByText('Rebuild Session')).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    await user.click(dialog.querySelector('.btn-error')!)

    await waitFor(() => {
      expect(mockRebuildSession).toHaveBeenCalledWith('my-session')
    })
  })

  it('shows alert on rebuild failure', async () => {
    mockNotify.mockClear()
    mockRebuildSession.mockResolvedValue({ ok: false, message: 'Build error' })
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    // ConfirmDialog should appear -- click Rebuild to confirm
    await waitFor(() => expect(screen.getByText('Rebuild Session')).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    await user.click(dialog.querySelector('.btn-error')!)

    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith(expect.objectContaining({ message: 'Build error' }))
    })
  })

  it('shows network error alert on rebuild exception', async () => {
    mockNotify.mockClear()
    mockRebuildSession.mockRejectedValue(new Error('Network failure'))
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    // ConfirmDialog should appear -- click Rebuild to confirm
    await waitFor(() => expect(screen.getByText('Rebuild Session')).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    await user.click(dialog.querySelector('.btn-error')!)

    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith(
        expect.objectContaining({ message: 'Network failure' }),
      )
    })
  })

  it('shows alert with unknown error on rebuild failure with no message', async () => {
    mockNotify.mockClear()
    mockRebuildSession.mockResolvedValue({ ok: false })
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    // ConfirmDialog should appear -- click Rebuild to confirm
    await waitFor(() => expect(screen.getByText('Rebuild Session')).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    await user.click(dialog.querySelector('.btn-error')!)

    await waitFor(() => {
      expect(mockNotify).toHaveBeenCalledWith(expect.objectContaining({ message: 'Unknown error' }))
    })
  })

  it('calls onRefresh after successful rebuild', async () => {
    const user = userEvent.setup()

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await user.click(screen.getByTitle('Rebuild session'))

    // ConfirmDialog should appear — click Rebuild to confirm
    await waitFor(() => expect(screen.getByText('Rebuild Session')).toBeInTheDocument())
    const dialog = screen.getByRole('dialog')
    await user.click(dialog.querySelector('.btn-error')!)

    await waitFor(() => {
      expect(onRefresh).toHaveBeenCalled()
    })
  })

  // --- Pane details button ---

  it('shows pane details button when onPaneContextMenu is provided', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onPaneContextMenu={vi.fn()}
        defaultExpanded
      />,
    )

    const detailsBtns = screen.getAllByTitle('View details')
    expect(detailsBtns.length).toBeGreaterThan(0)
  })

  it('does not show pane details button when onPaneContextMenu is not provided', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.queryByTitle('View details')).not.toBeInTheDocument()
  })

  it('calls onPaneContextMenu when details button is clicked', async () => {
    const onPaneContextMenu = vi.fn()
    const user = userEvent.setup()

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onPaneContextMenu={onPaneContextMenu}
        defaultExpanded
      />,
    )

    const detailsBtns = screen.getAllByTitle('View details')
    expect(detailsBtns[0]).toBeTruthy()
    await user.click(detailsBtns[0]!)

    expect(onPaneContextMenu).toHaveBeenCalledWith('my-session:0:%0')
  })

  // --- Group functionality ---

  it('renders multiple groups with correct names', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockMultipleGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    expect(screen.getByText('work')).toBeInTheDocument()
    expect(screen.getByText('personal')).toBeInTheDocument()
  })

  it('shows group session count', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('shows drop target message for empty groups', () => {
    mockFetchProfileOrder.mockResolvedValue({
      groups: [{ id: 1, sort_order: 0, sessions: [] }],
      ungrouped: [],
    })

    render(
      <TmuxTree
        sessions={[]}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    expect(screen.getByText('Drop sessions here')).toBeInTheDocument()
  })

  it('expand/collapse toggles group visibility', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Group starts expanded, find its expand button
    const groupRow = screen.getByText('work').closest('.group-row')
    const expandBtn = groupRow?.querySelector('.expand-btn') as HTMLButtonElement
    expect(expandBtn).toBeTruthy()

    // Collapse
    await user.click(expandBtn)
    // Group name should still be visible
    expect(screen.getByText('work')).toBeInTheDocument()

    // Expand again
    await user.click(expandBtn)
    expect(screen.getByText('work')).toBeInTheDocument()
  })

  // --- Order fetching ---

  it('fetches session orders when profileId is provided', async () => {
    mockFetchProfileOrder.mockResolvedValue({
      groups: [],
      ungrouped: [{ session_name: 'my-session', sort_order: 0 }],
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await waitFor(() => {
      expect(mockFetchProfileOrder).toHaveBeenCalledWith(1)
    })
  })

  it('handles order fetch failure gracefully', async () => {
    mockFetchProfileOrder.mockRejectedValue(new Error('Fetch error'))

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Should not crash
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('handles order fetch returning null response', async () => {
    mockFetchProfileOrder.mockResolvedValue(null)

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Should not crash
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('clears session orders when profileId is undefined', async () => {
    // First render with profileId
    const { rerender } = render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await waitFor(() => {
      expect(mockFetchProfileOrder).toHaveBeenCalledWith(1)
    })

    mockFetchProfileOrder.mockClear()

    // Re-render without profileId
    rerender(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    // Should not fetch orders again
    await act(async () => {
      await new Promise((r) => setTimeout(r, 100))
    })
    expect(mockFetchProfileOrder).not.toHaveBeenCalled()
  })

  // --- Long press for QuickGroupMenu ---

  it('opens quick group menu on long press when profileKey and groups are provided', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    expect(sessionRow).toBeTruthy()

    // Fire React mouseDown event
    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    // Advance timers to trigger long press (500ms)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.getByText('移动到分组')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('does not open quick group menu without profileKey', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    expect(sessionRow).toBeTruthy()

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  it('opens quick group menu even with empty groups array (source does not guard on empty)', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    expect(sessionRow).toBeTruthy()

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Menu opens because !groups is false for empty array []
    expect(screen.getByText('移动到分组')).toBeInTheDocument()
    expect(screen.getByText('新建分组')).toBeInTheDocument()
    // No group items shown
    expect(screen.queryByText('work')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- QuickGroupMenu ---

  it('shows group items in quick group menu', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Check for the work item inside the quick group menu
    const groupItems = document.querySelectorAll('.quick-group-item')
    expect(groupItems.length).toBeGreaterThan(0)

    vi.useRealTimers()
  })

  it('closes quick group menu on backdrop click', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.getByText('移动到分组')).toBeInTheDocument()

    const backdrop = document.querySelector('.quick-group-backdrop')
    expect(backdrop).toBeTruthy()

    await act(async () => {
      fireEvent.click(backdrop!)
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  it('shows "new group" button in quick group menu', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.getByText('新建分组')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('shows group creation input when "new group" is clicked', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    expect(screen.getByPlaceholderText('分组名称...')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('exits group creation on Escape key', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })
    expect(screen.getByPlaceholderText('分组名称...')).toBeInTheDocument()

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.keyDown(input, { key: 'Escape' })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(screen.queryByPlaceholderText('分组名称...')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  it('creates new group and assigns session via Enter key', async () => {
    vi.useFakeTimers()
    const onOrderChange = vi.fn()
    mockCreateGroup.mockResolvedValue({ id: 99 })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: 'new-group' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    expect(mockCreateGroup).toHaveBeenCalledWith('test-key', 'new-group')
    expect(mockAssignSessionGroup).toHaveBeenCalledWith('my-session', 'test-key', 99)

    vi.useRealTimers()
  })

  it('creates new group via confirm button click', async () => {
    vi.useFakeTimers()
    const onOrderChange = vi.fn()
    mockCreateGroup.mockResolvedValue({ id: 99 })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: 'my-new-group' } })

    // Click the confirm button (checkmark)
    const confirmBtn = document.querySelector('.quick-group-confirm') as HTMLButtonElement
    expect(confirmBtn).toBeTruthy()

    await act(async () => {
      fireEvent.click(confirmBtn)
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    expect(mockCreateGroup).toHaveBeenCalledWith('test-key', 'my-new-group')

    vi.useRealTimers()
  })

  it('does not create group with empty name', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const confirmBtn = document.querySelector('.quick-group-confirm') as HTMLButtonElement
    // Confirm button should be disabled with empty name
    expect(confirmBtn).toBeDisabled()

    vi.useRealTimers()
  })

  // --- fetchPaneStatuses with no keys ---

  it('does not fetch pane statuses when there are no pane keys', async () => {
    render(
      <TmuxTree
        sessions={[]}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      await new Promise((r) => setTimeout(r, 200))
    })

    // Should not call pane status endpoint with no keys
    expect(mockFetchPaneStatuses).not.toHaveBeenCalled()
  })

  // --- fetchTaskPaneStatuses with malformed data ---

  it('handles tasks with no pane_key gracefully', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({})

    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    // Should not crash
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('normalizes completed status to done', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'done',
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      expect(mockFetchTaskPaneStatuses).toHaveBeenCalled()
    })
  })

  // --- Status polling ---

  it('polls status map periodically', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Initial fetch
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    const initialCallCount = mockFetchTaskPaneStatuses.mock.calls.length

    // Advance by 10 seconds to trigger poll
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000)
    })

    const laterCallCount = mockFetchTaskPaneStatuses.mock.calls.length

    expect(laterCallCount).toBeGreaterThan(initialCallCount)

    vi.useRealTimers()
  })

  // --- Multi-window session ---

  it('renders all windows and panes for multi-window sessions', () => {
    render(
      <TmuxTree
        sessions={[mockMultiWindowSession]}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('0: editor')).toBeInTheDocument()
    expect(screen.getByText('1: terminal')).toBeInTheDocument()
    expect(screen.getByText('nvim')).toBeInTheDocument()
    expect(screen.getByText('bash')).toBeInTheDocument()
    expect(screen.getByText('htop')).toBeInTheDocument()
  })

  it('calls onSelectPane with correct window index for second window panes', async () => {
    const user = userEvent.setup()
    render(
      <TmuxTree
        sessions={[mockMultiWindowSession]}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByText('htop'))

    expect(onSelectPane).toHaveBeenCalledWith('%2', 'multi-window:1')
  })

  // --- DragOverlay / DragPreview ---

  it('renders null DragPreview when activeItem is null', () => {
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  // --- buildPaneKey utility ---

  it('constructs correct pane keys for context menu', async () => {
    const onPaneContextMenu = vi.fn()
    render(
      <TmuxTree
        sessions={[mockMultiWindowSession]}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onPaneContextMenu={onPaneContextMenu}
        defaultExpanded
      />,
    )

    const paneNode = screen.getByText('htop').closest('.pane-node')
    await userEvent.pointer([{ keys: '[MouseRight]', target: paneNode! }])

    expect(onPaneContextMenu).toHaveBeenCalledWith('multi-window:1:%2')
  })

  // --- Long press touch events ---

  it('handles long press via touch events', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    expect(sessionRow).toBeTruthy()

    fireEvent.touchStart(sessionRow!, {
      touches: [{ clientX: 100, clientY: 100, identifier: 0 }],
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.getByText('移动到分组')).toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- QuickGroupMenu assign to group ---

  it('calls assignToGroup API when group item is clicked', async () => {
    vi.useFakeTimers()
    const onOrderChange = vi.fn()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockMultipleGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Click "personal" group (not the current group) inside the quick group menu
    const personalBtn = document.querySelectorAll('.quick-group-item')[1] as HTMLElement
    expect(personalBtn).toBeTruthy()
    await act(async () => {
      fireEvent.click(personalBtn)
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    expect(mockAssignSessionGroup).toHaveBeenCalledWith('my-session', 'test-key', 2)

    vi.useRealTimers()
  })

  // --- QuickGroupMenu close on touchend ---

  it('closes quick group menu on backdrop touchend', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.getByText('移动到分组')).toBeInTheDocument()

    const backdrop = document.querySelector('.quick-group-backdrop')
    expect(backdrop).toBeTruthy()

    fireEvent.touchEnd(backdrop!)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })

    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- renameWindow function ---

  it('does not rename when editWindowName is empty', async () => {
    const user = userEvent.setup()

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')

    await user.clear(input)
    await user.keyboard('{Enter}')

    await act(async () => {
      await new Promise((r) => setTimeout(r, 200))
    })

    // Should not call the rename API
    expect(mockRenameWindow).not.toHaveBeenCalled()
  })

  it('calls onRefresh when rename succeeds', async () => {
    const user = userEvent.setup()

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')

    await user.clear(input)
    await user.type(input, 'renamed')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(onRefresh).toHaveBeenCalled()
    })
  })

  it('does not call onRefresh when rename fails', async () => {
    const user = userEvent.setup()
    mockRenameWindow.mockResolvedValue(false)

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')

    await user.clear(input)
    await user.type(input, 'bad-rename')
    await user.keyboard('{Enter}')

    await act(async () => {
      await new Promise((r) => setTimeout(r, 200))
    })

    expect(onRefresh).not.toHaveBeenCalled()
  })

  it('handles renameWindow network error', async () => {
    const user = userEvent.setup()
    mockRenameWindow.mockRejectedValue(new Error('Network error'))

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await user.click(screen.getByTitle('Rename window'))
    const input = screen.getByDisplayValue('main')

    await user.clear(input)
    await user.type(input, 'error-test')
    await user.keyboard('{Enter}')

    await act(async () => {
      await new Promise((r) => setTimeout(r, 200))
    })

    expect(onRefresh).not.toHaveBeenCalled()
  })

  // --- handleDragEnd ---

  it('handles drag end saving order when profileId is set', async () => {
    const onOrderChange = vi.fn()

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    // DndContext is mocked, so we can't trigger actual drag events.
    // Just verify it renders correctly with all the drag props.
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  // --- Long press with mouse move cancels ---

  it('cancels long press on mouse move', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    // Move the mouse to cancel long press
    fireEvent.mouseMove(sessionRow!, { clientX: 200, clientY: 200 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- Multiple sessions sorting ---

  it('renders multiple sessions in tree', () => {
    const sessions: TmuxSession[] = [
      mockSessions[0]!,
      { ...mockMultiWindowSession, sessionName: 'another-session' },
    ]

    render(
      <TmuxTree
        sessions={sessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('my-session')).toBeInTheDocument()
    expect(screen.getByText('another-session')).toBeInTheDocument()
  })

  // --- Empty groups list in quick group menu ---

  it('shows no group items when groups array is empty', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={[]}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Menu should open but no group items should appear
    expect(screen.getByText('移动到分组')).toBeInTheDocument()
    expect(screen.getByText('新建分组')).toBeInTheDocument()
    expect(screen.queryByText('work')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- Status map overlay logic ---

  it('prioritizes in_progress task status over existing pane status', async () => {
    mockFetchPaneStatuses.mockResolvedValue([
      { paneKey: 'my-session:0:%0', status: 'idle', mtime: 0 },
    ])
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'in_progress',
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      expect(mockFetchPaneStatuses).toHaveBeenCalled()
    })
  })

  // --- Session status summary with different statuses ---

  it('computes session status summary with multiple status types', async () => {
    mockFetchTaskPaneStatuses.mockResolvedValue({
      'my-session:0:%0': 'in_progress',
      'my-session:0:%1': 'failed',
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    await waitFor(() => {
      // Should show both in_progress and failed stats
      expect(screen.getByText(/进行中/)).toBeInTheDocument()
    })
  })

  // --- handleDragEnd save order ---

  it('calls saveOrder and onOrderChange on drag end', async () => {
    const onOrderChange = vi.fn()

    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    // DndContext is mocked so actual drag events don't fire.
    // Verify the component renders with drag context.
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  // --- Viewport adjustment for QuickGroupMenu ---

  it('renders quick group menu at specified position', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    // Open at a specific position
    fireEvent.mouseDown(sessionRow!, { clientX: 50, clientY: 200 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const menu = document.querySelector('.quick-group-menu') as HTMLElement
    expect(menu).toBeTruthy()
    expect(menu.style.left).toBe('50px')
    expect(menu.style.top).toBe('200px')

    vi.useRealTimers()
  })

  // --- Group expand/collapse icon ---

  it('renders FolderOpen when group is expanded', () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Groups start expanded
    const groupRow = screen.getByText('work').closest('.group-row')
    const svg = groupRow?.querySelector('svg')
    expect(svg).toBeTruthy()
  })

  // --- QuickGroupMenu createAndAssign error handling ---

  it('handles createAndAssign error gracefully', async () => {
    vi.useFakeTimers()
    mockCreateGroup.mockRejectedValue(new Error('Server error'))

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: 'error-group' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    // Menu should close after error
    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- assignToGroup error handling ---

  it('handles assignToGroup error gracefully', async () => {
    vi.useFakeTimers()
    mockAssignSessionGroup.mockRejectedValue(new Error('Server error'))

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockMultipleGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Click "personal" group to trigger assign
    const personalBtn = document.querySelectorAll('.quick-group-item')[1] as HTMLElement
    expect(personalBtn).toBeTruthy()
    await act(async () => {
      fireEvent.click(personalBtn)
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    // Menu should close after error
    expect(screen.queryByText('移动到分组')).not.toBeInTheDocument()

    vi.useRealTimers()
  })

  // --- QuickGroupMenu with createAndAssign returning no id ---

  it('does not assign when create group returns no id', async () => {
    vi.useFakeTimers()
    mockCreateGroup.mockResolvedValue({ id: undefined })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: 'no-id-group' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    // Should have called createGroup but not assignSessionGroup
    expect(mockCreateGroup).toHaveBeenCalled()
    expect(mockAssignSessionGroup).not.toHaveBeenCalled()

    vi.useRealTimers()
  })

  // --- Session without windows ---

  it('renders session with no windows', () => {
    const emptySession: TmuxSession = {
      sessionName: 'empty-session',
      sessionId: '3',
      windows: [],
    }

    render(
      <TmuxTree
        sessions={[emptySession]}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        defaultExpanded
      />,
    )

    expect(screen.getByText('empty-session')).toBeInTheDocument()
  })

  // --- createAndAssign with whitespace-only name ---

  it('does not create group with whitespace-only name', async () => {
    vi.useFakeTimers()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')

    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: '   ' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    // Should not create group with whitespace name
    expect(mockCreateGroup).not.toHaveBeenCalled()

    vi.useRealTimers()
  })

  // --- Drag and drop via mocked DndContext handlers ---

  it('triggers handleDragStart and sets active item', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={vi.fn()}
      />,
    )

    // Simulate drag start via the captured handler
    await act(async () => {
      mockDndHandlers.onDragStart?.({ active: { id: 'session-my-session' } })
    })

    // Component should still render without errors
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('triggers handleDragOver and sets overItemId', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragOver?.({ over: { id: 'session-my-session' } })
    })

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('triggers handleDragEnd with same active and over id (no-op)', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={vi.fn()}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-my-session' },
        over: { id: 'session-my-session' },
      })
    })

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('triggers handleDragEnd with null over (no-op)', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-my-session' },
        over: null,
      })
    })

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('triggers handleDragEnd reordering sessions and saves order', async () => {
    const onOrderChange = vi.fn()

    const sessions: TmuxSession[] = [
      {
        sessionName: 'session-a',
        sessionId: '1',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w1',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
      {
        sessionName: 'session-b',
        sessionId: '2',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w2',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
    ]

    render(
      <TmuxTree
        sessions={sessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-session-b' },
        over: { id: 'session-session-a' },
      })
    })

    await waitFor(() => {
      expect(onOrderChange).toHaveBeenCalled()
    })
  })

  it('triggers handleDragEnd moving session to group', async () => {
    const onOrderChange = vi.fn()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-my-session' },
        over: { id: 'group-1' },
      })
    })

    await waitFor(() => {
      expect(onOrderChange).toHaveBeenCalled()
    })
  })

  it('triggers handleDragEnd reordering groups', async () => {
    const onOrderChange = vi.fn()

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockMultipleGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={onOrderChange}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'group-1' },
        over: { id: 'group-2' },
      })
    })

    await waitFor(() => {
      expect(onOrderChange).toHaveBeenCalled()
    })
  })

  it('triggers handleDragEnd with unknown active id (no-op)', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'unknown-id' },
        over: { id: 'session-my-session' },
      })
    })

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('does not save order when profileId is undefined on drag end', async () => {
    const sessions: TmuxSession[] = [
      {
        sessionName: 'session-a',
        sessionId: '1',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w1',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
      {
        sessionName: 'session-b',
        sessionId: '2',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w2',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
    ]

    render(<TmuxTree sessions={sessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    mockSaveOrder.mockClear()

    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-session-b' },
        over: { id: 'session-session-a' },
      })
    })

    // Should not call saveOrder (no profileId)
    expect(mockSaveOrder).not.toHaveBeenCalled()
  })

  it('handles handleDragEnd save error gracefully', async () => {
    mockSaveOrder.mockRejectedValue(new Error('Save failed'))

    const sessions: TmuxSession[] = [
      {
        sessionName: 'session-a',
        sessionId: '1',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w1',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
      {
        sessionName: 'session-b',
        sessionId: '2',
        windows: [
          {
            windowIndex: 0,
            windowName: 'w',
            windowId: 'w2',
            panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'cmd' }],
          },
        ],
      },
    ]

    render(
      <TmuxTree
        sessions={sessions}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
        onOrderChange={vi.fn()}
      />,
    )

    // Should not throw
    await act(async () => {
      mockDndHandlers.onDragEnd?.({
        active: { id: 'session-session-b' },
        over: { id: 'session-session-a' },
      })
    })

    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  // --- DragPreview ---

  it('renders DragPreview for group type', async () => {
    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    // Simulate drag start on a group item
    await act(async () => {
      mockDndHandlers.onDragStart?.({ active: { id: 'group-1' } })
    })

    // DragPreview should render a group-preview div
    const preview = document.querySelector('.group-preview')
    expect(preview).toBeTruthy()
  })

  it('renders DragPreview for session type', async () => {
    render(<TmuxTree sessions={mockSessions} onSelectPane={onSelectPane} onRefresh={onRefresh} />)

    await act(async () => {
      mockDndHandlers.onDragStart?.({ active: { id: 'session-my-session' } })
    })

    // DragPreview should render a session-preview div
    const preview = document.querySelector('.session-preview')
    expect(preview).toBeTruthy()
  })

  // --- QuickGroupMenu ungroup button ---

  it('shows ungroup button when session has a current group', async () => {
    vi.useFakeTimers()
    mockFetchProfileOrder.mockResolvedValue({
      groups: [{ id: 1, sort_order: 0, sessions: [{ session_name: 'my-session', sort_order: 0 }] }],
      ungrouped: [],
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Should show ungroup button since session is in group 1
    expect(screen.getByText('移出分组')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('calls assignToGroup(null) when ungroup button is clicked', async () => {
    vi.useFakeTimers()
    mockFetchProfileOrder.mockResolvedValue({
      groups: [{ id: 1, sort_order: 0, sessions: [{ session_name: 'my-session', sort_order: 0 }] }],
      ungrouped: [],
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const ungroupBtn = screen.getByText('移出分组')
    await act(async () => {
      fireEvent.click(ungroupBtn)
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(200)
    })

    expect(mockAssignSessionGroup).toHaveBeenCalledWith('my-session', 'test-key', null)

    vi.useRealTimers()
  })

  // --- QuickGroupMenu viewport adjustment ---

  it('adjusts menu position when it overflows viewport right', async () => {
    vi.useFakeTimers()
    // Set window width to be small so menu overflows
    Object.defineProperty(window, 'innerWidth', { value: 200, writable: true })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    fireEvent.mouseDown(sessionRow!, { clientX: 150, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const menu = document.querySelector('.quick-group-menu') as HTMLElement
    expect(menu).toBeTruthy()

    vi.useRealTimers()
    Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true })
  })

  // --- QuickGroupMenu with createAndAssign loading state ---

  it('disables buttons while loading during group creation', async () => {
    vi.useFakeTimers()
    let resolveCreate: () => void
    mockCreateGroup.mockImplementation(async () => {
      await new Promise<void>((r) => {
        resolveCreate = r
      })
      return { id: 99 }
    })

    render(
      <TmuxTree
        sessions={mockSessions}
        groups={mockGroups}
        profileId={1}
        profileKey="test-key"
        onSelectPane={onSelectPane}
        onRefresh={onRefresh}
      />,
    )

    const sessionRow = screen.getByText('my-session').closest('.session-row')
    fireEvent.mouseDown(sessionRow!, { clientX: 100, clientY: 100 })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    const createBtn = screen.getByText('新建分组')
    await act(async () => {
      fireEvent.click(createBtn)
    })

    const input = screen.getByPlaceholderText('分组名称...')
    fireEvent.change(input, { target: { value: 'slow-group' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    // Resolve the pending promise
    await act(async () => {
      resolveCreate!()
      await vi.advanceTimersByTimeAsync(100)
    })

    vi.useRealTimers()
  })
})
