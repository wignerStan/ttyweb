import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MobileToolbox } from './MobileToolbox'

vi.mock('../shared/components/AiCommandTab', () => ({
  AiCommandTab: () => <div data-testid="ai-command-tab">AI Tab</div>,
}))

vi.mock('../shared/components/SnippetsTab', () => ({
  SnippetsTab: () => <div data-testid="snippets-tab">Snippets</div>,
}))

vi.mock('../shared/components/ConfigViewer', () => ({
  ConfigViewer: () => <div data-testid="config-viewer">Config</div>,
}))

vi.mock('../shared/components/FileUpload', () => ({
  FileUpload: () => <div data-testid="file-upload">Upload</div>,
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

  it('renders font size slider in default mode', () => {
    render(<MobileToolbox {...defaultProps} />)

    const slider = document.querySelector('input[type="range"]')
    expect(slider).toBeInTheDocument()
    expect(slider).toHaveValue('10')
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
  })

  it('sends key data on key press', () => {
    const onSend = vi.fn()

    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const escBtn = screen.getByText('esc').closest('button')!
    touchEnd(escBtn)

    expect(onSend).toHaveBeenCalledWith('\x1b')
  })

  it('sends arrow key data', () => {
    const onSend = vi.fn()

    render(<MobileToolbox {...defaultProps} onSend={onSend} />)

    const upBtn = screen.getByText('\u2191').closest('button')!
    touchEnd(upBtn)

    expect(onSend).toHaveBeenCalledWith('\x1b[A')
  })

  it('renders keyboard mode when keyboardMode is true', () => {
    render(<MobileToolbox {...defaultProps} keyboardMode onToggleKeyboard={vi.fn()} />)

    expect(screen.getByText('\u6536\u8D77\u952E\u76D8')).toBeInTheDocument()
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

  it('calls onFontSizeChange when slider value changes', () => {
    const onFontSizeChange = vi.fn()
    render(<MobileToolbox {...defaultProps} onFontSizeChange={onFontSizeChange} />)

    const slider = document.querySelector('input[type="range"]') as HTMLInputElement
    fireEvent.change(slider, { target: { value: '12' } })

    expect(onFontSizeChange).toHaveBeenCalledWith(12)
  })
})
