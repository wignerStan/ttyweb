import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { TaskDialog } from './TaskDialog'

vi.mock('./CommentThread', () => ({
  CommentThread: () => <div data-testid="comment-thread" />,
}))

const defaultProps = {
  open: true,
  task: null,
  onSave: vi.fn().mockResolvedValue(null),
  onCreate: vi.fn().mockResolvedValue(null),
  onDelete: vi.fn().mockResolvedValue(true),
  onClose: vi.fn(),
  fetchComments: vi.fn().mockResolvedValue([]),
  createComment: vi.fn().mockResolvedValue(null),
}

describe('TaskDialog snapshot', () => {
  it('renders new task dialog', () => {
    const { container } = renderWithProviders(<TaskDialog {...defaultProps} open={true} />)
    expect(container).toMatchSnapshot()
  })
})
