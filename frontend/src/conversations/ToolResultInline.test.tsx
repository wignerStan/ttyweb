import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { ToolResultInline } from './ToolResultInline'
import type { ToolResultBlock, ToolUseBlock } from './types'

const toolUse: ToolUseBlock = {
  id: 'tu1',
  name: 'Read',
  input: { file_path: '/home/user/src/app.ts' },
}

const toolResult: ToolResultBlock = {
  tool_use_id: 'tu1',
  output: 'Line 1: import React\nLine 2: export function App()',
}

describe('ToolResultInline', () => {
  it('renders tool name and status', () => {
    renderWithProviders(<ToolResultInline toolUse={toolUse} />)
    expect(screen.getByText('Read')).toBeInTheDocument()
  })

  it('expandable/collapsible content', async () => {
    renderWithProviders(<ToolResultInline toolUse={toolUse} toolResult={toolResult} />)
    const header = screen.getByText('Read').closest('.conv-tool-result-header')!
    // Initially collapsed - no output visible
    expect(screen.queryByText(/import React/)).not.toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(header)
    expect(screen.getByText(/import React/)).toBeInTheDocument()

    // Collapse again
    await user.click(header)
    expect(screen.queryByText(/import React/)).not.toBeInTheDocument()
  })

  it('truncated preview when collapsed', () => {
    const longOutput = 'A'.repeat(200)
    const tr: ToolResultBlock = { tool_use_id: 'tu1', output: longOutput }
    renderWithProviders(<ToolResultInline toolUse={toolUse} toolResult={tr} />)
    // When collapsed, output div should not be present
    const outputEl = document.querySelector('.conv-tool-result-output')
    expect(outputEl).not.toBeInTheDocument()
    // Chevron should be present
    expect(screen.getByText('\u25B6')).toBeInTheDocument()
  })
})
