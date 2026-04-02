import { act, fireEvent, render, renderHook, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NotificationProvider, useNotification, useNotificationList } from './NotificationProvider'

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

function TriggerNotify({ type }: { type: 'info' | 'ai_completion' | 'ai_approval' | 'error' }) {
  const { notify } = useNotification()
  return (
    <button
      type="button"
      onClick={() =>
        notify({
          type,
          title: `Test ${type}`,
          message: `Message for ${type}`,
          paneKey: 'session1:0:0',
        })
      }
    >
      Trigger
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

  it.each([
    'info',
    'ai_completion',
    'ai_approval',
    'error',
  ] as const)('renders %s notification type correctly', (type) => {
    const { container } = render(
      <NotificationProvider autoDismissMs={0}>
        <TriggerNotify type={type} />
      </NotificationProvider>,
    )

    fireEvent.click(screen.getByText('Trigger'))

    expect(container).toMatchSnapshot()
  })
})

function createWrapper() {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <NotificationProvider autoDismissMs={0}>{children}</NotificationProvider>
  }
}

describe('NotificationProvider (hook)', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('dismisses a notification', () => {
    const { result } = renderHook(
      () => ({ actions: useNotification(), list: useNotificationList() }),
      { wrapper: createWrapper() },
    )
    act(() => {
      result.current.actions.notify({ type: 'info', title: 'Test', message: 'Hello', paneKey: '' })
    })
    expect(result.current.list).toHaveLength(1)
    act(() => {
      result.current.actions.dismiss(result.current.list[0]!.id)
    })
    expect(result.current.list).toHaveLength(0)
  })

  it('handles multiple notifications', () => {
    const { result } = renderHook(
      () => ({ actions: useNotification(), list: useNotificationList() }),
      { wrapper: createWrapper() },
    )
    act(() => {
      result.current.actions.notify({ type: 'info', title: 'A', message: '', paneKey: '' })
      result.current.actions.notify({ type: 'ai_completion', title: 'B', message: '', paneKey: '' })
      result.current.actions.notify({ type: 'ai_approval', title: 'C', message: '', paneKey: '' })
    })
    expect(result.current.list).toHaveLength(3)
  })

  it('auto-dismisses after timeout', () => {
    vi.useFakeTimers()
    const { container } = render(
      <NotificationProvider autoDismissMs={3000}>
        <TestChild />
      </NotificationProvider>,
    )

    // Need to flush after fake timers are installed
    fireEvent.click(screen.getByText('Notify'))
    expect(screen.getByText('Task Done')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(container.querySelector('[aria-live]')?.children).toHaveLength(0)
  })

  it('throws when used outside provider', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    expect(() => renderHook(() => useNotification())).toThrow()
    spy.mockRestore()
  })

  it('renders aria-live and role attributes', () => {
    const { container } = render(
      <NotificationProvider>
        <div />
      </NotificationProvider>,
    )
    const liveRegion = container.querySelector('[aria-live="polite"]')
    expect(liveRegion).toBeInTheDocument()
    expect(liveRegion).toHaveAttribute('role', 'log')
  })
})
