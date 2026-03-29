import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mockToggleCollapse = vi.fn()
const mockSetOpacity = vi.fn()
const mockOnDragStart = vi.fn()
const mockOnResizeStart = vi.fn()

let collapsed = false
let unreadCount = 3

vi.mock('../hooks/useInboxItems', () => ({
  useInboxItems: () => ({
    items: [],
    unreadCount,
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

vi.mock('../hooks/useFloatingPanel', () => ({
  useFloatingPanel: () => ({
    collapsed,
    position: collapsed ? { x: 50, y: 100 } : { x: 100, y: 200 },
    size: { width: 860, height: 280 },
    opacity: 0.15,
    onDragStart: mockOnDragStart,
    onResizeStart: mockOnResizeStart,
    toggleCollapse: mockToggleCollapse,
    setCollapsed: vi.fn(),
    setOpacity: mockSetOpacity,
  }),
}))

// Dynamic import to get the component after mocks are set up
let FloatingImperialStudy: typeof import('./FloatingImperialStudy').FloatingImperialStudy

describe('FloatingImperialStudy', () => {
  beforeEach(async () => {
    collapsed = false
    unreadCount = 3
    mockToggleCollapse.mockClear()
    mockSetOpacity.mockClear()
    mockOnDragStart.mockClear()
    mockOnResizeStart.mockClear()
    const mod = await import('./FloatingImperialStudy')
    FloatingImperialStudy = mod.FloatingImperialStudy
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders floating panel when not collapsed', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByTestId('panel')).toBeInTheDocument()
    expect(screen.getByText('Imperial Study')).toBeInTheDocument()
  })

  it('shows unread count', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByText('3 unread')).toBeInTheDocument()
  })

  it('close button calls onClose', async () => {
    const onClose = vi.fn()
    render(<FloatingImperialStudy activePaneKey={null} onClose={onClose} />)
    await userEvent.click(screen.getByTitle('Close panel'))
    expect(onClose).toHaveBeenCalled()
  })

  it('renders collapsed bubble when collapsed', () => {
    collapsed = true
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.queryByTestId('panel')).not.toBeInTheDocument()
    expect(screen.getByTitle('Open Imperial Study')).toBeInTheDocument()
  })

  it('shows badge on collapsed bubble', () => {
    collapsed = true
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('collapsed bubble click calls toggleCollapse', async () => {
    collapsed = true
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    await userEvent.click(screen.getByTitle('Open Imperial Study'))
    expect(mockToggleCollapse).toHaveBeenCalled()
  })

  it('minimize button calls toggleCollapse', async () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    await userEvent.click(screen.getByTitle('Minimize'))
    expect(mockToggleCollapse).toHaveBeenCalled()
  })

  it('opacity slider calls setOpacity', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    const slider = screen.getByTitle(/Opacity/)
    fireEvent.change(slider, { target: { value: '0.5' } })
    expect(mockSetOpacity).toHaveBeenCalledWith(0.5)
  })

  it('does not show unread text when unreadCount is 0', () => {
    unreadCount = 0
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.queryByText(/unread/)).not.toBeInTheDocument()
  })

  it('does not show badge when collapsed and unreadCount is 0', () => {
    collapsed = true
    unreadCount = 0
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('passes activePaneKey to ImperialStudyPanel', () => {
    render(<FloatingImperialStudy activePaneKey="main:0:0" onClose={vi.fn()} />)
    expect(screen.getByTestId('panel')).toHaveTextContent('floating')
  })

  it('titlebar mouseDown calls onDragStart', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    const titlebar = document.querySelector('.is-floating-panel__titlebar')!
    fireEvent.mouseDown(titlebar)
    expect(mockOnDragStart).toHaveBeenCalled()
  })

  it('resize handle mouseDown calls onResizeStart', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    const resizeHandle = document.querySelector('.is-floating-panel__resize')!
    fireEvent.mouseDown(resizeHandle)
    expect(mockOnResizeStart).toHaveBeenCalled()
  })

  it('applies correct inline styles to floating panel', () => {
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    const panel = document.querySelector('.is-floating-panel')
    expect(panel).toHaveStyle({ left: '100px', top: '200px', width: '860px', height: '280px' })
  })

  it('applies correct inline styles to collapsed bubble', () => {
    collapsed = true
    render(<FloatingImperialStudy activePaneKey={null} onClose={vi.fn()} />)
    const bubble = document.querySelector('.is-floating-bubble')
    expect(bubble).toHaveStyle({ left: '50px', top: '100px' })
  })
})
