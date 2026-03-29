import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders, screen } from '../../test-utils'
import type { Task } from '../../types'
import { TaskCard } from './TaskCard'

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 1,
    task_title: 'Build feature',
    task_status: 'in_progress',
    started_at: 1700000000,
    completed_at: 0,
    paneKey: 'pane-1',
    ...overrides,
  }
}

describe('TaskCard (shared)', () => {
  it('renders task title, #id, and status badge', () => {
    renderWithProviders(<TaskCard task={makeTask()} />)
    expect(screen.getByText('Build feature')).toBeInTheDocument()
    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText('\u25cf Active')).toBeInTheDocument()
  })

  it('shows "Done" badge for completed tasks', () => {
    renderWithProviders(<TaskCard task={makeTask({ task_status: 'completed' })} />)
    expect(screen.getByText('\u2713 Done')).toBeInTheDocument()
  })

  it('shows "Untitled Task" when title is empty', () => {
    renderWithProviders(<TaskCard task={makeTask({ task_title: '' })} />)
    expect(screen.getByText('Untitled Task')).toBeInTheDocument()
  })

  it('applies "current" class when isCurrent is true', () => {
    const { container } = renderWithProviders(<TaskCard task={makeTask()} isCurrent={true} />)
    expect(container.firstChild).toHaveClass('current')
  })

  it('does not apply "current" class when isCurrent is false', () => {
    const { container } = renderWithProviders(<TaskCard task={makeTask()} isCurrent={false} />)
    expect(container.firstChild).not.toHaveClass('current')
  })

  it('shows "Mark Done" button when isCurrent + in_progress + onComplete', () => {
    const onComplete = vi.fn()
    renderWithProviders(<TaskCard task={makeTask()} isCurrent={true} onComplete={onComplete} />)
    expect(screen.getByText('Mark Done')).toBeInTheDocument()
  })

  it('hides "Mark Done" button when task is completed', () => {
    const onComplete = vi.fn()
    renderWithProviders(
      <TaskCard
        task={makeTask({ task_status: 'completed' })}
        isCurrent={true}
        onComplete={onComplete}
      />,
    )
    expect(screen.queryByText('Mark Done')).not.toBeInTheDocument()
  })

  it('hides "Mark Done" button when isCurrent is false', () => {
    const onComplete = vi.fn()
    renderWithProviders(<TaskCard task={makeTask()} isCurrent={false} onComplete={onComplete} />)
    expect(screen.queryByText('Mark Done')).not.toBeInTheDocument()
  })

  it('sets role="button" and tabIndex={0} when onSelect is provided', () => {
    const onSelect = vi.fn()
    const { container } = renderWithProviders(<TaskCard task={makeTask()} onSelect={onSelect} />)
    const card = container.firstChild as HTMLElement
    expect(card).toHaveAttribute('role', 'button')
    expect(card).toHaveAttribute('tabindex', '0')
  })

  it('does not set role or tabIndex when onSelect is omitted', () => {
    const { container } = renderWithProviders(<TaskCard task={makeTask()} />)
    const card = container.firstChild as HTMLElement
    expect(card).not.toHaveAttribute('role')
    expect(card).not.toHaveAttribute('tabindex')
  })
})
