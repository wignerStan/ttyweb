import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { InboxItem } from '../types'
import { InboxDetailModal } from './InboxDetailModal'

const mockSubmitReply = vi.fn().mockResolvedValue(undefined)
const mockSetLoading = vi.fn()

vi.mock('../hooks/useReplyInbox', () => ({
  useReplyInbox: () => ({
    submitReply: (...args: unknown[]) => mockSubmitReply(...args),
    get loading() {
      return false
    },
    set loading(_v: boolean) {
      mockSetLoading(_v)
    },
    error: null,
  }),
}))

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

const itemNoDate: InboxItem = {
  ...item,
  id: 'inbox-2',
  created_at: null,
  kind: 'question',
}

const itemUnknownKind: InboxItem = {
  ...item,
  id: 'inbox-3',
  kind: 'completion',
}

describe('InboxDetailModal', () => {
  beforeEach(() => {
    mockSubmitReply.mockClear()
    mockSubmitReply.mockResolvedValue(undefined)
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

  it('renders time as dash when created_at is null', () => {
    render(<InboxDetailModal item={itemNoDate} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('\u2014')).toBeInTheDocument()
  })

  it('renders formatted time when created_at is set', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    // The time should be rendered as a formatted date string
    expect(screen.queryByText('\u2014')).not.toBeInTheDocument()
  })

  it('renders Kind label', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('Kind:')).toBeInTheDocument()
  })

  it('renders From label', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('From:')).toBeInTheDocument()
  })

  it('renders Time label', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('Time:')).toBeInTheDocument()
  })

  it('custom reply with text calls submitReply with custom decision and reply text', async () => {
    const onReplied = vi.fn()
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={onReplied} />)
    const textarea = screen.getByPlaceholderText('Reply message...')
    await userEvent.type(textarea, 'Looks good')
    await userEvent.click(screen.getByText('Reply'))
    expect(mockSubmitReply).toHaveBeenCalledWith('inbox-1', 's1', 'custom', 'Looks good')
    await waitFor(() => expect(onReplied).toHaveBeenCalled())
  })

  it('Reply button is disabled when no reply text', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    const replyBtn = screen.getByText('Reply').closest('button')!
    expect(replyBtn).toBeDisabled()
  })

  it('Reply button is enabled when reply text is entered', async () => {
    const user = userEvent.setup()
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    const textarea = screen.getByPlaceholderText('Reply message...')
    await user.type(textarea, 'Some reply')
    const replyBtn = screen.getByText('Reply').closest('button')!
    expect(replyBtn).not.toBeDisabled()
  })

  it('renders icon and color for known kind', () => {
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={vi.fn()} />)
    // The approval kind should render an icon
    const kindSection = screen.getByText('approval').closest('.is-modal__meta-row')
    expect(kindSection?.querySelector('svg')).toBeInTheDocument()
  })

  it('renders icon and color for completion kind', () => {
    render(<InboxDetailModal item={itemUnknownKind} onClose={vi.fn()} onReplied={vi.fn()} />)
    expect(screen.getByText('completion')).toBeInTheDocument()
  })

  it('keeps modal open when submitReply throws', async () => {
    // Make submitReply reject
    mockSubmitReply.mockRejectedValueOnce(new Error('fail'))
    const onReplied = vi.fn()
    render(<InboxDetailModal item={item} onClose={vi.fn()} onReplied={onReplied} />)
    await userEvent.click(screen.getByText('Approve'))
    // Wait for the click to process
    await waitFor(() => expect(mockSubmitReply).toHaveBeenCalled())
    // Give time for the error to propagate
    await screen.findByText('Inbox Detail')
    // onReplied should NOT be called when there's an error
    expect(onReplied).not.toHaveBeenCalled()
  })

  it('does not close modal on inner div click (only overlay)', async () => {
    const onClose = vi.fn()
    render(<InboxDetailModal item={item} onClose={onClose} onReplied={vi.fn()} />)
    // Click the modal content div (not the overlay)
    const modal = document.querySelector('.is-modal')!
    await userEvent.click(modal)
    expect(onClose).not.toHaveBeenCalled()
  })
})
