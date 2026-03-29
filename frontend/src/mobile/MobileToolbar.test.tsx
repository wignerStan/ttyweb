import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { MobileToolbar } from './MobileToolbar'

describe('MobileToolbar', () => {
  it('renders toolbar with key buttons', () => {
    render(<MobileToolbar onSendText={vi.fn()} />)

    expect(screen.getByText('Esc')).toBeInTheDocument()
    expect(screen.getByText('Tab')).toBeInTheDocument()
    expect(screen.getByText('Ctrl')).toBeInTheDocument()
    expect(screen.getByText('Paste')).toBeInTheDocument()
  })

  it('renders arrow buttons', () => {
    render(<MobileToolbar onSendText={vi.fn()} />)

    expect(screen.getByText('\u2190')).toBeInTheDocument()
    expect(screen.getByText('\u2191')).toBeInTheDocument()
    expect(screen.getByText('\u2193')).toBeInTheDocument()
    expect(screen.getByText('\u2192')).toBeInTheDocument()
  })

  it('sends escape key when Esc is clicked', async () => {
    const onSendText = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={onSendText} />)
    await user.click(screen.getByText('Esc'))

    expect(onSendText).toHaveBeenCalledWith('\x1b')
  })

  it('sends tab key when Tab is clicked', async () => {
    const onSendText = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={onSendText} />)
    await user.click(screen.getByText('Tab'))

    expect(onSendText).toHaveBeenCalledWith('\t')
  })

  it('toggles Ctrl active state on click', async () => {
    const onSendText = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={onSendText} />)

    const ctrlBtn = screen.getByText('Ctrl')
    await user.click(ctrlBtn)
    expect(ctrlBtn.className).toContain('active')

    await user.click(ctrlBtn)
    expect(ctrlBtn.className).not.toContain('active')
  })

  it('sends ctrl+letter when Ctrl is active and a key is pressed', async () => {
    const onSendText = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={onSendText} />)

    // Activate Ctrl
    await user.click(screen.getByText('Ctrl'))
    // Arrow keys should pass through (not a single letter)
    await user.click(screen.getByText('\u2191'))

    expect(onSendText).toHaveBeenCalledWith('\x1b[A')
  })

  it('calls onPaste when Paste button is clicked', async () => {
    const onPaste = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={vi.fn()} onPaste={onPaste} />)
    await user.click(screen.getByText('Paste'))

    expect(onPaste).toHaveBeenCalled()
  })

  it('sends arrow key directly', async () => {
    const onSendText = vi.fn()
    const user = userEvent.setup()

    render(<MobileToolbar onSendText={onSendText} />)
    await user.click(screen.getByText('\u2193'))

    expect(onSendText).toHaveBeenCalledWith('\x1b[B')
  })
})
