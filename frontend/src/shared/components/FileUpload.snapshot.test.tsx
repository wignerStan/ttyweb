import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { FileUpload } from './FileUpload'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

Object.assign(navigator, {
  clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
})

describe('FileUpload snapshot', () => {
  it('renders upload zone', () => {
    const { container } = renderWithProviders(<FileUpload />)
    expect(container).toMatchSnapshot()
  })
})
