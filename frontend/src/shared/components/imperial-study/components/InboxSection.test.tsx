import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { InboxItem } from '../types'
import { InboxSection } from './InboxSection'

const items: InboxItem[] = [
  {
    id: '1',
    study_id: 's1',
    worker_id: 'w1',
    run_id: 'r1',
    kind: 'question',
    status: 'pending',
    title: 'Need approval',
    body: 'Please review',
    metadata: {},
    created_at: null,
    updated_at: null,
  },
  {
    id: '2',
    study_id: 's1',
    worker_id: 'w2',
    run_id: 'r2',
    kind: 'report',
    status: 'read',
    title: 'Report ready',
    body: '',
    metadata: {},
    created_at: null,
    updated_at: null,
  },
]

describe('InboxSection', () => {
  it('renders Inbox header with count', () => {
    render(<InboxSection items={items} onItemClick={vi.fn()} />)
    expect(screen.getByText('Inbox (2)')).toBeInTheDocument()
  })

  it('shows empty state', () => {
    render(<InboxSection items={[]} onItemClick={vi.fn()} />)
    expect(screen.getByText('Inbox empty')).toBeInTheDocument()
  })

  it('renders inbox cards with titles', () => {
    render(<InboxSection items={items} onItemClick={vi.fn()} />)
    expect(screen.getByText('Need approval')).toBeInTheDocument()
    expect(screen.getByText('Report ready')).toBeInTheDocument()
  })

  it('click calls onItemClick', async () => {
    const onClick = vi.fn()
    render(<InboxSection items={items} onItemClick={onClick} />)
    await userEvent.click(screen.getByText('Need approval'))
    expect(onClick).toHaveBeenCalledWith(items[0])
  })

  it('shows worker_id on cards', () => {
    render(<InboxSection items={items} onItemClick={vi.fn()} />)
    expect(screen.getByText('w1')).toBeInTheDocument()
    expect(screen.getByText('w2')).toBeInTheDocument()
  })
})
