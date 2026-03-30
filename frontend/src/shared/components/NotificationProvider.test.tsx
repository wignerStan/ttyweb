import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NotificationProvider, useNotification } from './NotificationProvider'

function TestChild() {
  const { notify } = useNotification()
  return (
    <button
      type="button"
      onClick={() =>
        notify({
          type: 'ai_completion',
          title: 'Task Done',
          message: 'Claude finished refactoring auth module',
          paneKey: 'session1:0:0',
        })
      }
    >
      Notify
    </button>
  )
}

describe('NotificationProvider', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders toast notification on notify call', async () => {
    render(
      <NotificationProvider>
        <TestChild />
      </NotificationProvider>,
    )

    fireEvent.click(screen.getByText('Notify'))

    await waitFor(() => {
      expect(screen.getByText('Task Done')).toBeInTheDocument()
    })
    expect(screen.getByText('Claude finished refactoring auth module')).toBeInTheDocument()
  })

  it('auto-dismisses notification after timeout', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    render(
      <NotificationProvider autoDismissMs={300}>
        <TestChild />
      </NotificationProvider>,
    )

    fireEvent.click(screen.getByText('Notify'))
    expect(screen.getByText('Task Done')).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.queryByText('Task Done')).not.toBeInTheDocument()
    })
  })
})
