import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { WorkerContextMenu } from './WorkerContextMenu'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
  vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('WorkerContextMenu', () => {
  const props = { x: 100, y: 100, workerId: 'w1', paneTarget: 'butler/quant:%1', onClose: vi.fn() }

  it('renders all menu items', () => {
    render(<WorkerContextMenu {...props} />)
    expect(screen.getByText('Open Terminal')).toBeInTheDocument()
    expect(screen.getByText('Copy pane target')).toBeInTheDocument()
    expect(screen.getByText('Pause worker')).toBeInTheDocument()
    expect(screen.getByText('Kill worker')).toBeInTheDocument()
  })

  it('Open Terminal dispatches custom event', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
    render(<WorkerContextMenu {...props} />)
    await userEvent.click(screen.getByText('Open Terminal'))
    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'imperial:focus-pane' }),
    )
  })

  it('Copy pane target writes to clipboard', async () => {
    render(<WorkerContextMenu {...props} />)
    await userEvent.click(screen.getByText('Copy pane target'))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('butler/quant:%1')
  })

  it('Pause worker sends PUT request', async () => {
    render(<WorkerContextMenu {...props} />)
    await userEvent.click(screen.getByText('Pause worker'))
    expect(fetch).toHaveBeenCalledWith(
      '/api/butler/worker_sessions/w1',
      expect.objectContaining({ method: 'PUT' }),
    )
  })

  it('Escape calls onClose', () => {
    vi.useFakeTimers()
    const onClose = vi.fn()
    render(<WorkerContextMenu {...props} onClose={onClose} />)
    act(() => {
      vi.advanceTimersByTime(20)
    })
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalled()
    vi.useRealTimers()
  })

  it('click outside calls onClose', async () => {
    vi.useFakeTimers()
    const onClose = vi.fn()
    render(
      <div>
        <WorkerContextMenu {...props} onClose={onClose} />
        <button type="button" id="outside">
          Outside
        </button>
      </div>,
    )
    act(() => {
      vi.advanceTimersByTime(20)
    })
    // Directly dispatch a click event on the outside button
    document.getElementById('outside')?.click()
    expect(onClose).toHaveBeenCalled()
    vi.useRealTimers()
  })
})
