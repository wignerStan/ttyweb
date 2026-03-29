import { describe, expect, it } from 'vitest'
import { renderWithProviders, screen } from '../../test-utils'
import type { PaneStatus } from '../../types'
import { StatusBadge } from './StatusBadge'

const STATUSES: PaneStatus[] = ['idle', 'in_progress', 'done', 'failed', 'waiting']

describe('StatusBadge', () => {
  it('renders correct CSS class for each status', () => {
    for (const status of STATUSES) {
      const { container } = renderWithProviders(<StatusBadge status={status} />)
      expect(container.firstChild).toHaveClass(`status-badge--${status}`)
    }
  })

  it('renders correct icon class for each status', () => {
    for (const status of STATUSES) {
      const { container } = renderWithProviders(<StatusBadge status={status} />)
      const icon = container.querySelector('.status-icon')
      expect(icon).toBeInTheDocument()
      expect(icon).toHaveClass(`status-icon--${status === 'in_progress' ? 'spinning' : status}`)
    }
  })

  it('small size hides label text', () => {
    renderWithProviders(<StatusBadge status="idle" size="small" />)
    expect(screen.queryByText('Idle')).not.toBeInTheDocument()
  })

  it('medium size shows label text', () => {
    renderWithProviders(<StatusBadge status="idle" size="medium" />)
    expect(screen.getByText('Idle')).toBeInTheDocument()
  })

  it('medium size shows correct label for each status', () => {
    const labels: Record<PaneStatus, string> = {
      idle: 'Idle',
      in_progress: 'In Progress',
      done: 'Done',
      failed: 'Failed',
      waiting: 'Waiting',
    }
    for (const status of STATUSES) {
      const { unmount } = renderWithProviders(<StatusBadge status={status} size="medium" />)
      expect(screen.getByText(labels[status])).toBeInTheDocument()
      unmount()
    }
  })

  it('editable mode renders a select with all options', () => {
    const onChange = vi.fn()
    renderWithProviders(<StatusBadge status="idle" onChange={onChange} />)
    const select = screen.getByRole('combobox')
    expect(select).toBeInTheDocument()
    expect(select).toHaveValue('idle')
    const options = screen.getAllByRole('option')
    expect(options).toHaveLength(5)
    expect(options.map((o) => o.textContent)).toEqual([
      'Idle',
      'In Progress',
      'Done',
      'Failed',
      'Waiting',
    ])
  })

  it('non-editable mode does not render a select', () => {
    renderWithProviders(<StatusBadge status="idle" />)
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  })

  it('editable mode calls onChange when select changes', async () => {
    const user = await import('@testing-library/user-event')
    const userEvent = user.default.setup()
    const onChange = vi.fn()
    renderWithProviders(<StatusBadge status="idle" onChange={onChange} />)
    await userEvent.selectOptions(screen.getByRole('combobox'), 'done')
    expect(onChange).toHaveBeenCalledWith('done')
  })
})
