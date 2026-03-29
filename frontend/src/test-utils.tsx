import { render } from '@testing-library/react'
import type { ReactElement, ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'

interface RenderOptions {
  route?: string
  wrapper?: React.ComponentType<{ children: ReactNode }>
}

export function renderWithProviders(ui: ReactElement, options?: RenderOptions) {
  return render(<MemoryRouter initialEntries={[options?.route ?? '/']}>{ui}</MemoryRouter>, options)
}

export * from '@testing-library/react'
