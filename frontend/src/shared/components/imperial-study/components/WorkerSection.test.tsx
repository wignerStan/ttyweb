import { render, screen } from '@testing-library/react'
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
})
