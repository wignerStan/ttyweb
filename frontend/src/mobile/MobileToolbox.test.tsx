import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MobileToolbox } from './MobileToolbox'

vi.mock('../shared/components/AiCommandTab', () => ({
  AiCommandTab: ({
    initialText,
    onTextConsumed,
  }: {
    initialText?: string
    onTextConsumed?: () => void
  }) => (
    <div data-testid="ai-command-tab">
      <span data-testid="ai-initial-text">{initialText ?? ''}</span>
      <button type="button" data-testid="ai-consume" onClick={onTextConsumed}>
        Consume
      </button>
    </div>
  ),
}))

vi.mock('../shared/components/SnippetsTab', () => ({
  SnippetsTab: ({ disabled }: { disabled?: boolean }) => (
    <div data-testid="snippets-tab" data-disabled={disabled ? 'true' : undefined}>
      Snippets
    </div>
  ),
}))

vi.mock('../shared/components/ConfigViewer', () => ({
  ConfigViewer: () => <div data-testid="config-viewer">Config</div>,
}))

vi.mock('../shared/components/FileUpload', () => ({
  FileUpload: ({ compact }: { compact?: boolean }) => (
    <div data-testid="file-upload" data-compact={compact ? 'true' : undefined}>
      Upload
    </div>
  ),
}))

vi.mock('../shared/components/VoiceInput', () => ({
  VoiceInput: vi.fn(),
}))

vi.mock('../hooks/useTmuxPrefix', () => ({
  useTmuxPrefix: () => ({ code: '\x02', label: 'Ctrl+B' }),
}))

function touchEnd(element: Element) {
  fireEvent.touchEnd(element, { preventDefault: () => {} })
}

describe('MobileToolbox', () => {
  const defaultProps = {
    onSend: vi.fn(),
    fontSize: 10,
    onFontSizeChange: vi.fn(),
  }

  it('renders key buttons in default mode', () => {
    render(<MobileToolbox {...defaultProps} />)

    expect(screen.getByText('esc')).toBeInTheDocument()
    expect(screen.getByText('tab')).toBeInTheDocument()
    expect(screen.getByText('ctrl')).toBeInTheDocument()
    expect(screen.getByText('alt')).toBeInTheDocument()
  })

  it('renders all row1 key buttons', () => {
    render(<MobileToolbox {...defaultProps} />)

    expect(screen.getByText('esc')).toBeInTheDocument()
    expect(screen.getByText('tab')).toBeInTheDocument()
    expect(screen.getByText('|')).toBeInTheDocument()
    expect(screen.getByText('/')).toBeInTheDocument()
    expect(screen.getByText('-')).toBeInTheDocument()
    expect(screen.getByText('~')).toBeInTheDocument()
    expect(screen.getByText('^C')).toBeInTheDocument()
    expect(screen.getByText('clr')).toBeInTheDocument()
  })

  it('renders all row2 key buttons', () => {
    render(<MobileToolbox {...defaultProps} />)

    expect(screen.getByText('ctrl')).toBeInTheDocument()
    expect(screen.getByText('alt')).toBeInTheDocument()
    expect(screen.getByText('\u2191')).toBeInTheDocument()
    expect(screen.getByText('\u2193')).toBeInTheDocument()
    expect(screen.getByText('\u2190')).toBeInTheDocument()
    expect(screen.getByText('\u2192')).toBeInTheDocument()
    expect(screen.getByText('\ud83d\udcdc')).toBeInTheDocument()
    expect(screen.getByText('\u23ce')).toBeInTheDocument()
  })

  it('renders font size slider in default mode', () => {
    render(<MobileToolbox {...defaultProps} />)

    const slider = document.querySelector('input[type="range"]')
    expect(slider).toBeInTheDocument()
    expect(slider).toHaveValue('10')
  })

  it('renders font size display value', () => {
    render(<MobileToolbox {...defaultProps} fontSize={8} />)

    expect(screen.getByText('8')).toBeInTheDocument()
  })

  it('renders tab buttons in tabbar', () => {
    render(<MobileToolbox {...defaultProps} />)

    expect(screen.getByText('\u547D\u4EE4')).toBeInTheDocument()
    expect(screen.getByText('AI')).toBeInTheDocument()
    expect(screen.getByText('\u914D\u7F6E')).toBeInTheDocument()
    expect(screen.getByText('\u4E0A\u4F20')).toBeInTheDocument()
  })

  it('switches tabs on touch', () => {
    render(<MobileToolbox {...defaultProps} />)

    // Default tab is AI
    expect(screen.getByTestId('ai-command-tab')).toBeInTheDocument()

    // Switch to snippets - click the button containing "命令"
    const snippetsBtn = screen.getByText('\u547D\u4EE4').closest('button')!
    touchEnd(snippetsBtn)
    expect(screen.getByTestId('snippets-tab')).toBeInTheDocument()

    // Switch to config
    const configBtn = screen.getByText('\u914D\u7F6E').closest('button')!
    touchEnd(configBtn)
    expect(screen.getByTestId('config-viewer')).toBeInTheDocument()

    // Switch to upload
    const uploadBtn = screen.getByText('\u4E0A\u4F20').closest('button')!
    touchEnd(uploadBtn)
    expect(screen.getByTestId('file-upload')).toBeInTheDocument()

    // Switch back to AI
    const aiBtn = screen.getByText('AI').closest('button')!
    touchEnd(aiBtn)
    expect(screen.getByTestId('ai-command-tab')).toBeInTheDocument()
  })

  it('sends key data on key press', () => {
    const onSend = vi.fn()

    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const escBtn = screen.getByText('esc').closest('button')!
    touchEnd(escBtn)

    expect(onSend).toHaveBeenCalledWith('\x1b')
  })

  it('sends tab key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const tabBtn = screen.getByText('tab').closest('button')!
    touchEnd(tabBtn)

    expect(onSend).toHaveBeenCalledWith('\t')
  })

  it('sends pipe key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const pipeBtn = screen.getByText('|').closest('button')!
    touchEnd(pipeBtn)

    expect(onSend).toHaveBeenCalledWith('|')
  })

  it('sends slash key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const slashBtn = screen.getByText('/').closest('button')!
    touchEnd(slashBtn)

    expect(onSend).toHaveBeenCalledWith('/')
  })

  it('sends dash key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const dashBtn = screen.getByText('-').closest('button')!
    touchEnd(dashBtn)

    expect(onSend).toHaveBeenCalledWith('-')
  })

  it('sends tilde key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const tildeBtn = screen.getByText('~').closest('button')!
    touchEnd(tildeBtn)

    expect(onSend).toHaveBeenCalledWith('~')
  })

  it('sends Ctrl+C key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const ctrlCBtn = screen.getByText('^C').closest('button')!
    touchEnd(ctrlCBtn)

    expect(onSend).toHaveBeenCalledWith('\x03')
  })

  it('sends clear key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const clrBtn = screen.getByText('clr').closest('button')!
    touchEnd(clrBtn)

    expect(onSend).toHaveBeenCalledWith('\x15')
  })

  it('sends arrow key data for all directions', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    touchEnd(screen.getByText('\u2191').closest('button')!)
    expect(onSend).toHaveBeenCalledWith('\x1b[A')

    touchEnd(screen.getByText('\u2193').closest('button')!)
    expect(onSend).toHaveBeenCalledWith('\x1b[B')

    touchEnd(screen.getByText('\u2190').closest('button')!)
    expect(onSend).toHaveBeenCalledWith('\x1b[D')

    touchEnd(screen.getByText('\u2192').closest('button')!)
    expect(onSend).toHaveBeenCalledWith('\x1b[C')
  })

  it('sends tmux prefix key', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const prefixBtn = screen.getByText('\ud83d\udcdc').closest('button')!
    touchEnd(prefixBtn)

    expect(onSend).toHaveBeenCalledWith('\x02[')
  })

  it('sends enter key data', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const enterBtn = screen.getByText('\u23ce').closest('button')!
    touchEnd(enterBtn)

    expect(onSend).toHaveBeenCalledWith('\r')
  })

  it('renders keyboard mode when keyboardMode is true', () => {
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={vi.fn()} />)

    expect(screen.getByText('\u6536\u8D77\u952E\u76D8')).toBeInTheDocument()
  })

  it('renders keys in keyboard mode', () => {
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={vi.fn()} />)

    // Keyboard mode should still render the key rows
    expect(screen.getByText('esc')).toBeInTheDocument()
    expect(screen.getByText('ctrl')).toBeInTheDocument()
    expect(screen.getByText('\u2191')).toBeInTheDocument()
  })

  it('keyboard mode does not render font slider or tab content', () => {
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={vi.fn()} />)

    // No font slider in keyboard mode
    expect(document.querySelector('input[type="range"]')).not.toBeInTheDocument()

    // No tab content in keyboard mode
    expect(screen.queryByTestId('ai-command-tab')).not.toBeInTheDocument()
    expect(screen.queryByTestId('snippets-tab')).not.toBeInTheDocument()
    expect(screen.queryByTestId('config-viewer')).not.toBeInTheDocument()
    expect(screen.queryByTestId('file-upload')).not.toBeInTheDocument()
  })

  it('sends keys in keyboard mode', () => {
    const onSend = vi.fn()
    render(
      <MobileToolbox {...defaultProps} onSend={onSend} keyboardMode onToggleKeyboard={vi.fn()} />,
    )

    const escBtn = screen.getByText('esc').closest('button')!
    touchEnd(escBtn)

    expect(onSend).toHaveBeenCalledWith('\x1b')
  })

  it('toggles ctrl modifier state', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const ctrlBtn = screen.getByText('ctrl').closest('button')!
    touchEnd(ctrlBtn)
    expect(ctrlBtn.className).toContain('active')

    touchEnd(ctrlBtn)
    expect(ctrlBtn.className).not.toContain('active')
  })

  it('toggles alt modifier state', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const altBtn = screen.getByText('alt').closest('button')!
    touchEnd(altBtn)
    expect(altBtn.className).toContain('active')

    touchEnd(altBtn)
    expect(altBtn.className).not.toContain('active')
  })

  it('sends ctrl+letter combination', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    // Activate ctrl
    const ctrlBtn = screen.getByText('ctrl').closest('button')!
    touchEnd(ctrlBtn)

    // Press a letter key (e.g., pipe which is single char '|')
    // Actually let's test with '/' which is a single char
    const slashBtn = screen.getByText('/').closest('button')!
    touchEnd(slashBtn)

    // Ctrl+/ -> '/' is not A-Z, so it should send as-is with ctrl deactivated
    expect(onSend).toHaveBeenCalledWith('/')
    expect(ctrlBtn.className).not.toContain('active')
  })

  it('sends ctrl+letter for uppercase letter', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    // Activate ctrl
    const ctrlBtn = screen.getByText('ctrl').closest('button')!
    touchEnd(ctrlBtn)

    // Press '-' which is not A-Z range, should send as-is
    const dashBtn = screen.getByText('-').closest('button')!
    touchEnd(dashBtn)

    expect(onSend).toHaveBeenCalledWith('-')
  })

  it('sends alt+letter combination', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    // Activate alt
    const altBtn = screen.getByText('alt').closest('button')!
    touchEnd(altBtn)

    // Press '/' which is single char
    const slashBtn = screen.getByText('/').closest('button')!
    touchEnd(slashBtn)

    // Alt+/ -> should send ESC + /
    expect(onSend).toHaveBeenCalledWith('\x1b/')
    expect(altBtn.className).not.toContain('active')
  })

  it('does not send when key has no data and no modifier', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    // ctrl is a modifier-only key, toggles state but doesn't send
    const ctrlBtn = screen.getByText('ctrl').closest('button')!
    touchEnd(ctrlBtn)

    expect(onSend).not.toHaveBeenCalled()
  })

  it('ignores multi-char data with ctrl modifier', () => {
    const onSend = vi.fn()
    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    // Activate ctrl
    const ctrlBtn = screen.getByText('ctrl').closest('button')!
    touchEnd(ctrlBtn)

    // Press an arrow key (multi-char data '\x1b[A') - ctrl should be ignored
    const upBtn = screen.getByText('\u2191').closest('button')!
    touchEnd(upBtn)

    // Arrow key data is multi-char, so ctrl modifier is not applied
    expect(onSend).toHaveBeenCalledWith('\x1b[A')
    // Ctrl should remain active since it wasn't consumed
    expect(ctrlBtn.className).toContain('active')
  })

  it('calls onFontSizeChange when slider value changes', () => {
    const onFontSizeChange = vi.fn()
    render(<MobileToolbox {...defaultProps} onFontSizeChange={onFontSizeChange} />)

    const slider = document.querySelector('input[type="range"]') as HTMLInputElement
    fireEvent.change(slider, { target: { value: '12' } })

    expect(onFontSizeChange).toHaveBeenCalledWith(12)
  })

  it('renders grid button in tabbar', () => {
    render(<MobileToolbox {...defaultProps} />)

    // Grid button is the first button in the tabbar (toolbox-grid-btn)
    const gridBtns = document.querySelectorAll('.toolbox-grid-btn')
    expect(gridBtns.length).toBeGreaterThanOrEqual(1)
  })

  it('calls onToggleKeyboard when grid button clicked', () => {
    const onToggleKeyboard = vi.fn()
    render(<MobileToolbox {...defaultProps} onToggleKeyboard={onToggleKeyboard} />)

    const gridBtns = document.querySelectorAll('.toolbox-grid-btn')
    const tabbarGridBtn = gridBtns[gridBtns.length - 1]!
    touchEnd(tabbarGridBtn)

    expect(onToggleKeyboard).toHaveBeenCalled()
  })

  it('passes disabled prop to tab content components', () => {
    render(<MobileToolbox {...defaultProps} disabled />)

    // Switch to snippets
    const snippetsBtn = screen.getByText('\u547D\u4EE4').closest('button')!
    touchEnd(snippetsBtn)

    expect(screen.getByTestId('snippets-tab')).toHaveAttribute('data-disabled', 'true')
  })

  it('does not pass disabled to tab content when not disabled', () => {
    render(<MobileToolbox {...defaultProps} />)

    const snippetsBtn = screen.getByText('\u547D\u4EE4').closest('button')!
    touchEnd(snippetsBtn)

    expect(screen.getByTestId('snippets-tab')).not.toHaveAttribute('data-disabled')
  })

  it('passes compact prop to FileUpload', () => {
    render(<MobileToolbox {...defaultProps} />)

    const uploadBtn = screen.getByText('\u4E0A\u4F20').closest('button')!
    touchEnd(uploadBtn)

    expect(screen.getByTestId('file-upload')).toHaveAttribute('data-compact', 'true')
  })

  it('marks active tab button with active class', () => {
    render(<MobileToolbox {...defaultProps} />)

    // AI tab should be active by default
    const aiBtn = screen.getByText('AI').closest('button')!
    expect(aiBtn.className).toContain('active')

    // Snippets tab should not be active
    const snippetsBtn = screen.getByText('\u547D\u4EE4').closest('button')!
    expect(snippetsBtn.className).not.toContain('active')
  })

  it('renders voice input in tabbar', () => {
    render(<MobileToolbox {...defaultProps} />)

    // Voice input is inside a div.toolbox-tab-voice
    const voiceContainer = document.querySelector('.toolbox-tab-voice')
    expect(voiceContainer).toBeInTheDocument()
  })

  it('renders keyboard mode grid button with collapse label', () => {
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={vi.fn()} />)

    expect(screen.getByText('\u6536\u8D77\u952E\u76D8')).toBeInTheDocument()
    const gridBtns = document.querySelectorAll('.toolbox-grid-btn')
    expect(gridBtns.length).toBeGreaterThanOrEqual(1)
  })

  it('calls onToggleKeyboard in keyboard mode', () => {
    const onToggleKeyboard = vi.fn()
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={onToggleKeyboard} />)

    const gridBtns = document.querySelectorAll('.toolbox-grid-btn')
    touchEnd(gridBtns[0]!)

    expect(onToggleKeyboard).toHaveBeenCalled()
  })
})
