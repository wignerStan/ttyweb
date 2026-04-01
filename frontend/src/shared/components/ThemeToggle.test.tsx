import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'
import { ThemeToggle } from './ThemeToggle'

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('renders a button', () => {
    render(<ThemeToggle />)

    const button = screen.getByRole('button')
    expect(button).toBeInTheDocument()
  })

  it('toggles theme on click', async () => {
    const user = userEvent.setup()
    render(<ThemeToggle />)

    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')

    await user.click(screen.getByRole('button'))

    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('shows sun icon in dark mode', () => {
    render(<ThemeToggle />)

    const button = screen.getByRole('button')
    const svg = button.querySelector('svg')
    expect(svg).toBeInTheDocument()
    // Sun icon has a circle (cx="8")
    const circles = svg?.querySelectorAll('circle')
    expect(circles?.length).toBe(1)
  })
})
