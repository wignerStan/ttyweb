import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

vi.mock('../hooks/useFloatingPanel', () => ({
  useFloatingPanel: () => ({
    collapsed: false,
    position: { x: 100, y: 200 },
    size: { width: 860, height: 280 },
    opacity: 0.15,
    onDragStart: vi.fn(),
    onResizeStart: vi.fn(),
    toggleCollapse: vi.fn(),
    setCollapsed: vi.fn(),
    setOpacity: vi.fn(),
  }),
}))

vi.mock('../hooks/useInboxItems', () => ({
  useInboxItems: () => ({
    items: [],
    unreadCount: 3,
    loading: false,
    error: null,
    refetch: vi.fn(),
  }),
}))

vi.mock('./ImperialStudyPanel', () => ({
  ImperialStudyPanel: ({ floating }: { floating?: boolean }) => (
    <div data-testid="panel">{floating ? 'floating' : 'sidebar'}</div>
  ),
}))

const mockUseFloatingPanel = vi.fn()

describe('FloatingImperialStudy', () => {
  beforeEach(() => {
    mockUseFloatingPanel.mockReturnValue({
      collapsed: false,
      position: { x: 100, y: 200 },
      size: { width: 860, height: 280 },
      opacity: 0.15,
      onDragStart: vi.fn(),
      onResizeStart: vi.fn(),
      toggleCollapse: vi.fn(),
      setCollapsed: vi.fn(),
      setOpacity: vi.fn(),
    })
    vi.doMock('../hooks/useFloatingPanel', () => ({
      useFloatingPanel: () => mockUseFloatingPanel(),
    }))
  })

  afterEach(() => {
    vi.doUnmock('../hooks/useFloatingPanel')
  })

  it('renders floating panel when not collapsed', async () => {
    const { FloatingImperialStudy: Comp } = await import('./FloatingImperialStudy')
    render(<Comp activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByTestId('panel')).toBeInTheDocument()
    expect(screen.getByText('Imperial Study')).toBeInTheDocument()
  })

  it('shows unread count', async () => {
    const { FloatingImperialStudy: Comp } = await import('./FloatingImperialStudy')
    render(<Comp activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByText('3 unread')).toBeInTheDocument()
  })

  it('close button calls onClose', async () => {
    const onClose = vi.fn()
    const { FloatingImperialStudy: Comp } = await import('./FloatingImperialStudy')
    render(<Comp activePaneKey={null} onClose={onClose} />)
    await userEvent.click(screen.getByTitle('Close panel'))
    expect(onClose).toHaveBeenCalled()
  })

  it('renders collapsed bubble when collapsed', async () => {
    mockUseFloatingPanel.mockReturnValue({
      collapsed: true,
      position: { x: 50, y: 100 },
      size: { width: 860, height: 280 },
      opacity: 0.15,
      onDragStart: vi.fn(),
      onResizeStart: vi.fn(),
      toggleCollapse: vi.fn(),
      setCollapsed: vi.fn(),
      setOpacity: vi.fn(),
    })
    const { FloatingImperialStudy: Comp } = await import('./FloatingImperialStudy')
    render(<Comp activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.queryByTestId('panel')).not.toBeInTheDocument()
    expect(screen.getByTitle('Open Imperial Study')).toBeInTheDocument()
  })

  it('shows badge on collapsed bubble', async () => {
    mockUseFloatingPanel.mockReturnValue({
      collapsed: true,
      position: { x: 50, y: 100 },
      size: { width: 860, height: 280 },
      opacity: 0.15,
      onDragStart: vi.fn(),
      onResizeStart: vi.fn(),
      toggleCollapse: vi.fn(),
      setCollapsed: vi.fn(),
      setOpacity: vi.fn(),
    })
    const { FloatingImperialStudy: Comp } = await import('./FloatingImperialStudy')
    render(<Comp activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByText('3')).toBeInTheDocument()
  })
})
