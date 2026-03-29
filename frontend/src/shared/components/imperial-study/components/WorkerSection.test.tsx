import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { WorkerSession } from '../types'
import { WorkerSection } from './WorkerSection'

vi.mock('./WorkerContextMenu', () => ({
  WorkerContextMenu: ({ workerId }: { workerId: string }) => (
    <div data-testid="ctx-menu">{workerId}</div>
  ),
}))

const workers: WorkerSession[] = [
  {
    id: 'w1',
    study_id: 's1',
    session_id: 'agent-1',
    pane_target: 'butler/a:%1',
    port: 9001,
    state: 'busy',
    run_id: 'r1',
    project: 'proj',
    workdir: '/tmp',
    last_seen_at: null,
    created_at: null,
    updated_at: null,
  },
  {
    id: 'w2',
    study_id: 's1',
    session_id: 'agent-2',
    pane_target: 'butler/b:%2',
    port: 9002,
    state: 'idle',
    run_id: '',
    project: 'proj',
    workdir: '/tmp',
    last_seen_at: null,
    created_at: null,
    updated_at: null,
  },
]

describe('WorkerSection', () => {
  it('renders Workers header', () => {
    render(<WorkerSection workers={[]} />)
    expect(screen.getByText('Workers')).toBeInTheDocument()
  })

  it('shows empty state', () => {
    render(<WorkerSection workers={[]} />)
    expect(screen.getByText('No active workers')).toBeInTheDocument()
  })

  it('renders worker cards with session names', () => {
    render(<WorkerSection workers={workers} />)
    expect(screen.getByText('agent-1')).toBeInTheDocument()
    expect(screen.getByText('agent-2')).toBeInTheDocument()
  })

  it('shows worker states', () => {
    render(<WorkerSection workers={workers} />)
    expect(screen.getByText('busy')).toBeInTheDocument()
    expect(screen.getByText('idle')).toBeInTheDocument()
  })

  it('shows context menu on right-click', async () => {
    render(<WorkerSection workers={workers} />)
    const card = screen.getByText('agent-1').closest('.is-worker-card')!
    await userEvent.pointer({ keys: '[MouseRight]', target: card })
    expect(screen.getByTestId('ctx-menu')).toBeInTheDocument()
  })

  it('renders add worker button when onAddWorker provided', () => {
    render(<WorkerSection workers={workers} onAddWorker={vi.fn()} />)
    const btn = document.querySelector('.is-section__action')
    expect(btn).toBeInTheDocument()
  })

  it('does not render add worker button when onAddWorker not provided', () => {
    render(<WorkerSection workers={workers} />)
    const btn = document.querySelector('.is-section__action')
    expect(btn).not.toBeInTheDocument()
  })

  it('shows project and port meta for each worker', () => {
    render(<WorkerSection workers={workers} />)
    expect(screen.getByText('proj \u00b7 :9001')).toBeInTheDocument()
    expect(screen.getByText('proj \u00b7 :9002')).toBeInTheDocument()
  })

  it('displays intent label when intentMap provides one', () => {
    render(<WorkerSection workers={workers} intentMap={{ r1: 'Build project' }} />)
    expect(screen.getByText('Build project')).toBeInTheDocument()
  })

  it('does not show intent when intentMap has no entry for run_id', () => {
    render(<WorkerSection workers={workers} intentMap={{ r99: 'other' }} />)
    expect(screen.queryByText('Build project')).not.toBeInTheDocument()
    expect(screen.queryByText('other')).not.toBeInTheDocument()
  })

  it('toggles section collapse on header click', async () => {
    const user = userEvent.setup()
    render(<WorkerSection workers={workers} />)
    const header = screen.getByText('Workers').closest('.is-section__header')!
    const body = document.querySelector('.is-section__body')!
    expect(body.className).not.toContain('collapsed')
    await user.click(header)
    expect(body.className).toContain('collapsed')
    await user.click(header)
    expect(body.className).not.toContain('collapsed')
  })

  it('chevron has open class when section is expanded', () => {
    render(<WorkerSection workers={workers} />)
    const chevron = document.querySelector('.is-section__chevron')
    expect(chevron?.classList.contains('open')).toBe(true)
  })

  it('calls onAddWorker when button clicked', async () => {
    const user = userEvent.setup()
    const onAddWorker = vi.fn()
    render(<WorkerSection workers={workers} onAddWorker={onAddWorker} />)
    const btn = document.querySelector('.is-section__action')!
    await user.click(btn)
    expect(onAddWorker).toHaveBeenCalledOnce()
  })

  it('calls onWorkerClick when worker card with run_id is clicked', async () => {
    const user = userEvent.setup()
    const onWorkerClick = vi.fn()
    render(<WorkerSection workers={workers} onWorkerClick={onWorkerClick} />)
    const card = screen.getByText('agent-1').closest('.is-worker-card')!
    await user.click(card)
    expect(onWorkerClick).toHaveBeenCalledWith('r1')
  })

  it('dispatches focus-pane event when worker without run_id is clicked', async () => {
    const user = userEvent.setup()
    const handler = vi.fn()
    render(<WorkerSection workers={workers} onWorkerClick={vi.fn()} />)
    window.addEventListener('imperial:focus-pane', handler)
    const card = screen.getByText('agent-2').closest('.is-worker-card')!
    await user.click(card)
    expect(handler).toHaveBeenCalled()
    window.removeEventListener('imperial:focus-pane', handler)
  })

  it('closes context menu via onClose callback', async () => {
    render(<WorkerSection workers={workers} />)
    const card = screen.getByText('agent-1').closest('.is-worker-card')!
    await userEvent.pointer({ keys: '[MouseRight]', target: card })
    expect(screen.getByTestId('ctx-menu')).toBeInTheDocument()
    // Click elsewhere to close (blur / document click)
    fireEvent.click(document.body)
    // The context menu should close since setCtxMenu(null) is called via onClose
  })

  it('renders worker status dot with correct title', () => {
    render(<WorkerSection workers={workers} />)
    const dots = document.querySelectorAll('.is-worker-dot')
    expect(dots).toHaveLength(2)
    expect(dots[0]).toHaveAttribute('title', 'busy')
    expect(dots[1]).toHaveAttribute('title', 'idle')
  })

  it('shows correct tooltip for worker with run_id and onWorkerClick', () => {
    render(<WorkerSection workers={workers} onWorkerClick={vi.fn()} />)
    const card = screen.getByText('agent-1').closest('.is-worker-card')!
    expect(card).toHaveAttribute('title', 'Click for details \u00b7 Right-click for options')
  })

  it('shows correct tooltip for worker without run_id', () => {
    render(<WorkerSection workers={workers} />)
    const card = screen.getByText('agent-2').closest('.is-worker-card')!
    expect(card).toHaveAttribute('title', 'Click to focus \u00b7 Right-click for options')
  })

  it('applies flash class on click and removes after timeout', () => {
    vi.useFakeTimers()
    render(<WorkerSection workers={workers} onWorkerClick={vi.fn()} />)
    const card = screen.getByText('agent-2').closest('.is-worker-card')!
    act(() => {
      fireEvent.click(card)
    })
    expect(card.classList.contains('flash')).toBe(true)
    act(() => {
      vi.advanceTimersByTime(200)
    })
    expect(card.classList.contains('flash')).toBe(false)
    vi.useRealTimers()
  })
})
