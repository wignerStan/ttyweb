import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { ActivityEvent } from '../types'
import { ActivitySection } from './ActivitySection'

const events: ActivityEvent[] = [
  {
    id: '1',
    study_id: 's1',
    worker_id: 'w1',
    event_type: 'task_started',
    summary: 'Task started',
    detail: '',
    created_at: '2026-01-01T10:30:00Z',
  },
  {
    id: '2',
    study_id: 's1',
    worker_id: 'w2',
    event_type: 'task_completed',
    summary: 'Task done',
    detail: '',
    created_at: '2026-01-01T11:00:00Z',
  },
]

describe('ActivitySection', () => {
  it('renders Activity header', () => {
    render(<ActivitySection events={[]} />)
    expect(screen.getByText('Activity')).toBeInTheDocument()
  })

  it('body has collapsed class by default', () => {
    render(<ActivitySection events={events} />)
    const body = document.querySelector('.is-section__body')
    expect(body?.classList.contains('collapsed')).toBe(true)
  })

  it('shows empty state when expanded with no events', async () => {
    render(<ActivitySection events={[]} />)
    await userEvent.click(screen.getByText('Activity'))
    expect(screen.getByText('No recent activity')).toBeInTheDocument()
  })

  it('renders event list when expanded', async () => {
    render(<ActivitySection events={events} />)
    await userEvent.click(screen.getByText('Activity'))
    expect(screen.getByText('Task started')).toBeInTheDocument()
    expect(screen.getByText('Task done')).toBeInTheDocument()
  })

  it('calls onActivityClick when provided', async () => {
    const onClick = vi.fn()
    render(<ActivitySection events={events} onActivityClick={onClick} />)
    await userEvent.click(screen.getByText('Activity'))
    await userEvent.click(screen.getByText('Task started'))
    expect(onClick).toHaveBeenCalledWith(events[0])
  })
})
