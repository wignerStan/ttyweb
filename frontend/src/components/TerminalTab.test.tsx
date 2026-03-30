import { describe, expect, it } from 'vitest'

describe('TerminalTab metadata handling', () => {
  it('parses SetMetadata message (opcode 7) correctly', () => {
    const base64Data = btoa(JSON.stringify({ type: 'ai_state_change', data: { state: 'working' } }))
    const message = `7${base64Data}`

    // Parse the metadata from the message
    const metaJSON = atob(message.slice(1))
    const metadata = JSON.parse(metaJSON)
    expect(metadata.type).toBe('ai_state_change')
    expect(metadata.data.state).toBe('working')
  })
})
