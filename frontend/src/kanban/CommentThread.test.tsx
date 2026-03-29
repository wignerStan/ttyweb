import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { CommentThread } from './CommentThread'
import type { KanbanComment } from './types'

const comments: KanbanComment[] = [
  { id: 'c1', task_id: 't1', content: 'First comment', created_at: '2026-03-29T10:00:00Z' },
  { id: 'c2', task_id: 't1', content: 'Second comment', created_at: '2026-03-29T11:00:00Z' },
]

describe('CommentThread', () => {
  it('renders comments list', async () => {
    const fetchComments = vi.fn().mockResolvedValue(comments)
    renderWithProviders(
      <CommentThread taskId="t1" fetchComments={fetchComments} createComment={vi.fn()} />,
    )
    expect(await screen.findByText('First comment')).toBeInTheDocument()
    expect(screen.getByText('Second comment')).toBeInTheDocument()
  })

  it('shows empty state when no comments', async () => {
    const fetchComments = vi.fn().mockResolvedValue([])
    renderWithProviders(
      <CommentThread taskId="t1" fetchComments={fetchComments} createComment={vi.fn()} />,
    )
    expect(await screen.findByText('No comments yet')).toBeInTheDocument()
  })

  it('new comment input and submit', async () => {
    const newComment: KanbanComment = {
      id: 'c3',
      task_id: 't1',
      content: 'New reply',
      created_at: '2026-03-29T12:00:00Z',
    }
    const fetchComments = vi.fn().mockResolvedValue([])
    const createComment = vi.fn().mockResolvedValue(newComment)
    renderWithProviders(
      <CommentThread taskId="t1" fetchComments={fetchComments} createComment={createComment} />,
    )
    // Wait for initial load
    await screen.findByText('No comments yet')

    const user = userEvent.setup()
    const textarea = screen.getByPlaceholderText('Add a comment... (Cmd+Enter to submit)')
    await user.type(textarea, 'New reply')
    await user.click(screen.getByTitle('Send comment'))

    expect(createComment).toHaveBeenCalledWith('t1', 'New reply')
    expect(await screen.findByText('New reply')).toBeInTheDocument()
  })

  it('timestamps display', async () => {
    const fetchComments = vi.fn().mockResolvedValue(comments)
    renderWithProviders(
      <CommentThread taskId="t1" fetchComments={fetchComments} createComment={vi.fn()} />,
    )
    await screen.findByText('First comment')
    // Timestamps use toLocaleString which in jsdom may produce different output
    // Just verify the timestamp elements exist
    const timestamps = document.querySelectorAll('.comment-thread__timestamp')
    expect(timestamps).toHaveLength(2)
  })
})
