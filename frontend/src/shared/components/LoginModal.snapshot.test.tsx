import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { LoginModal } from './LoginModal'

vi.mock('../../utils/auth', () => ({
  login: (..._args: unknown[]) => vi.fn().mockResolvedValue({ success: true }),
}))

describe('LoginModal snapshot', () => {
  it('renders login form', () => {
    const { container } = render(<LoginModal onLogin={() => {}} />)
    expect(container).toMatchSnapshot()
  })
})
