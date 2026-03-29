import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { InboxItem } from '../types'
import { InboxDetailModal } from './InboxDetailModal'

vi.mock('../hooks/useReplyInbox', () => ({
  useReplyInbox: () => ({ submitReply: mockSubmitReply, loading: false, error: null }),
}))

const mockSubmitReply = vi.fn().mockResolvedValue(undefined)

const item: InboxItem = {
  id: 'inbox-1',
  study_id: 's1',
  worker_id: 'w1',
  run_id: 'r1',
  kind: 'approval',
  status: 'pending',
  title: 'Need approval',
  body: 'Please review this change',
  metadata: {},
  created_at: '2026-01-01T10:00:00Z',
  updated_at: null,
}

describe('InboxDetailModal', () => {
  beforeEach(() => {
    mockSubmitReply.mockClear()
  })

  it('renders item details', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('Inbox Detail')).toBeInTheDocument()
    expect(screen.getByText('approval')).toBeInTheDocument()
    expect(screen.getByText('w1')).toBeInTheDocument()
    expect(screen.getByText('Please review this change')).toBeInTheDocument()
  })

  it('renders reply textarea', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByPlaceholderText('Reply message...')).toBeInTheDocument()
  })

  it('close button calls onClose', async () => {
    const onClose = vi.fn()
    render(<InboxDetailModal item={item} onClose={onClose} onReplied={vi.fn()} />)
    await userEvent.click(screen.getByTitle('Close'))
    expect(onClose).toHaveBeenCalled()
  })

  it('approve calls submitReply with correct decision', async () => {
    const onReplied = vi.fn()
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={onReplied} />)
    await userEvent.click(screen.getByText('Approve'))
    expect(mockSubmitReply).toHaveBeenCalledWith('inbox-1', 's1', 'approved', '')
    await waitFor(() => expect(onReplied).toHaveBeenCalled())
  })

  it('reject calls submitReply with rejected decision', async () => {
    const onReplied = vi.fn()
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={onReplied} />)
    await userEvent.click(screen.getByText('Reject'))
    expect(mockSubmitReply).toHaveBeenCalledWith('inbox-1', 's1', 'rejected', '')
  })

  it('overlay click calls onClose', async () => {
    const onClose = vi.fn()
    render(<InboxDetailModal item={item} onClose={onClose} onReplied={vi.fn()} />)
    const overlay = document.querySelector('.is-modal-overlay')!
    await userEvent.click(overlay)
    expect(onClose).toHaveBeenCalled()
  })
})
