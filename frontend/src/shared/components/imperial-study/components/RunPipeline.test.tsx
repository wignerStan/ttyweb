import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { PipelineRun } from '../types'
import { RunPipeline } from './RunPipeline'

const baseRun: PipelineRun = {
  run_id: 'r1',
  task_id: 't1',
  intent: 'do thing',
  routing: { strategy: 'assistant', executor: 'opencode', delegated: false },
  stage: 'processing',
  status: 'running',
  events: [],
  startedAt: Date.now(),
}

const events = [
  {
    id: '1',
    study_id: 's1',
    worker_id: 'w1',
    event_type: 'task_started',
    summary: 'Started',
    detail: '',
    created_at: '2026-01-01T10:00:00Z',
  },
]

describe('RunPipeline', () => {
  it('renders null when no runs', () => {
    const { container } = render(<RunPipeline runs={[]} activeRun={null} onDismiss={vi.fn()} />)
    expect(container.innerHTML).toBe('')
  })

  it('renders pipeline stages', () => {
    render(<RunPipeline runs={[baseRun]} activeRun={baseRun} onDismiss={vi.fn()} />)
    expect(screen.getByText('Outflow')).toBeInTheDocument()
    expect(screen.getByText('Processing')).toBeInTheDocument()
    expect(screen.getByText('Return')).toBeInTheDocument()
  })

  it('renders intent', () => {
    render(<RunPipeline runs={[baseRun]} activeRun={baseRun} onDismiss={vi.fn()} />)
    expect(screen.getByText('do thing')).toBeInTheDocument()
  })

  it('shows strategy badge', () => {
    render(<RunPipeline runs={[baseRun]} activeRun={baseRun} onDismiss={vi.fn()} />)
    expect(screen.getByText('assistant')).toBeInTheDocument()
  })

  it('dismiss button calls onDismiss', async () => {
    const onDismiss = vi.fn()
    render(<RunPipeline runs={[baseRun]} activeRun={baseRun} onDismiss={onDismiss} />)
    await userEvent.click(screen.getByTitle('Dismiss'))
    expect(onDismiss).toHaveBeenCalledWith('r1')
  })

  it('renders events timeline when expanded', async () => {
    const runWithEvents: PipelineRun = { ...baseRun, events }
    render(<RunPipeline runs={[runWithEvents]} activeRun={runWithEvents} onDismiss={vi.fn()} />)
    await userEvent.click(screen.getByText('Timeline'))
    expect(screen.getByText('Started')).toBeInTheDocument()
  })

  it('renders result for terminal runs', () => {
    const doneRun: PipelineRun = {
      ...baseRun,
      stage: 'return',
      status: 'success',
      result: 'All done',
    }
    render(<RunPipeline runs={[doneRun]} activeRun={doneRun} onDismiss={vi.fn()} />)
    expect(screen.getByText('All done')).toBeInTheDocument()
  })
})
