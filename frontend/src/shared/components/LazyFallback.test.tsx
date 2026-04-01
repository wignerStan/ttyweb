import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LazyFallback } from './LazyFallback'

describe('LazyFallback', () => {
  it('matches snapshot', () => {
    const { container } = render(<LazyFallback />)
    expect(container).toMatchSnapshot()
  })
})
