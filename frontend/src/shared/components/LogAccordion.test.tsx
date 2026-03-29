import { describe, expect, it } from 'vitest'
import { renderWithProviders, screen } from '../../test-utils'
import { LogAccordion } from './LogAccordion'

describe('LogAccordion', () => {
  it('hides content by default', () => {
    renderWithProviders(
      <LogAccordion title="Logs" count={5}>
        <p>Hidden content</p>
      </LogAccordion>,
    )
    expect(screen.queryByText('Hidden content')).not.toBeInTheDocument()
  })

  it('clicking header toggles content open', async () => {
    const user = await import('@testing-library/user-event')
    const userEvent = user.default.setup()
    renderWithProviders(
      <LogAccordion title="Logs" count={3}>
        <p>Visible content</p>
      </LogAccordion>,
    )
    const button = screen.getByRole('button', { name: /logs/i })
    await userEvent.click(button)
    expect(screen.getByText('Visible content')).toBeInTheDocument()
  })

  it('clicking header again toggles content closed', async () => {
    const user = await import('@testing-library/user-event')
    const userEvent = user.default.setup()
    renderWithProviders(
      <LogAccordion title="Logs" count={1}>
        <p>Toggle content</p>
      </LogAccordion>,
    )
    const button = screen.getByRole('button', { name: /logs/i })
    await userEvent.click(button)
    expect(screen.getByText('Toggle content')).toBeInTheDocument()
    await userEvent.click(button)
    expect(screen.queryByText('Toggle content')).not.toBeInTheDocument()
  })

  it('defaultOpen={true} shows content on mount', () => {
    renderWithProviders(
      <LogAccordion title="Errors" count={2} defaultOpen={true}>
        <p>Auto-opened</p>
      </LogAccordion>,
    )
    expect(screen.getByText('Auto-opened')).toBeInTheDocument()
  })

  it('renders optional icon when provided', () => {
    renderWithProviders(
      <LogAccordion title="Alerts" count={4} icon={<span data-testid="custom-icon">!</span>}>
        <p>Content</p>
      </LogAccordion>,
    )
    expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
  })

  it('does not render icon slot when icon is omitted', () => {
    const { container } = renderWithProviders(
      <LogAccordion title="Logs" count={0}>
        <p>Content</p>
      </LogAccordion>,
    )
    expect(container.querySelector('[data-testid="custom-icon"]')).not.toBeInTheDocument()
  })
})
