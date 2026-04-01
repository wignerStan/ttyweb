import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeAll, describe, expect, it, vi } from 'vitest'
import { ConfirmDialog } from './ConfirmDialog'

beforeAll(() => {
  if (typeof HTMLDialogElement !== 'undefined' && !HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
      this.setAttribute('open', '')
    })
    HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
      this.removeAttribute('open')
    })
  }
})

describe('ConfirmDialog', () => {
  it('renders with title and message', () => {
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="This cannot be undone."
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    )
    expect(screen.getByText('Delete?')).toBeInTheDocument()
    expect(screen.getByText('This cannot be undone.')).toBeInTheDocument()
  })

  it('calls onConfirm when confirm button clicked', async () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Sure?"
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: /confirm/i }))
    expect(onConfirm).toHaveBeenCalledOnce()
  })

  it('calls onCancel when cancel button clicked', async () => {
    const onCancel = vi.fn()
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Sure?"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onCancel).toHaveBeenCalledOnce()
  })

  it('renders nothing when open is false', () => {
    const { container } = render(
      <ConfirmDialog
        open={false}
        title="Delete?"
        message="Sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('uses destructive variant styling', () => {
    render(
      <ConfirmDialog
        open
        title="Delete?"
        message="Sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
        variant="destructive"
      />,
    )
    const confirmBtn = screen.getByRole('button', { name: /confirm/i })
    expect(confirmBtn.className).toContain('btn-error')
  })

  it('calls onCancel when dialog backdrop clicked', async () => {
    const onCancel = vi.fn()
    const { container } = render(
      <ConfirmDialog open title="Test" message="Test" onConfirm={vi.fn()} onCancel={onCancel} />,
    )
    const dialog = container.querySelector('dialog')!
    await userEvent.click(dialog)
    expect(onCancel).toHaveBeenCalledOnce()
  })
})
