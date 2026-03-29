import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { FileUpload } from './FileUpload'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const uploadResult = {
  filename: 'test.txt',
  originalname: 'test.txt',
  size: 1024,
  mimetype: 'text/plain',
  url: '/uploads/test.txt',
  path: '/tmp/test.txt',
}

describe('FileUpload', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(uploadResult),
      }),
    )
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders upload zone', () => {
    renderWithProviders(<FileUpload />)
    expect(screen.getByText('Drop files or click to browse')).toBeInTheDocument()
  })

  it('has hidden file input', () => {
    renderWithProviders(<FileUpload />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    expect(input).toBeInTheDocument()
    expect(input).toHaveStyle({ display: 'none' })
  })

  it('shows drag state on drag enter/leave', async () => {
    renderWithProviders(<FileUpload />)
    const zone = screen.getByText('Drop files or click to browse').closest('.file-upload-zone')!
    fireEvent.dragEnter(zone, { dataTransfer: { types: ['Files'] } })
    expect(zone.className).toContain('dragging')
    fireEvent.dragLeave(zone, { dataTransfer: { types: ['Files'] } })
    expect(zone.className).not.toContain('dragging')
  })

  it('shows upload result after success', async () => {
    renderWithProviders(<FileUpload />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' })
    await userEvent.upload(input, file)
    await waitFor(() => expect(screen.getByText('test.txt')).toBeInTheDocument())
  })

  it('copies path to clipboard', async () => {
    renderWithProviders(<FileUpload />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' })
    await userEvent.upload(input, file)
    await waitFor(() => expect(screen.getByText('test.txt')).toBeInTheDocument())
    const pathBtns = screen.getAllByText('路径')
    expect(pathBtns[0]).toBeTruthy()
    await userEvent.click(pathBtns[0]!)
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('/tmp/test.txt'))
  })

  it('sends path via onSend', async () => {
    const onSend = vi.fn()
    renderWithProviders(<FileUpload onSend={onSend} />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' })
    await userEvent.upload(input, file)
    await waitFor(() => expect(screen.getByText('test.txt')).toBeInTheDocument())
    const sendBtn = screen.getByText('终端')
    await userEvent.click(sendBtn)
    await waitFor(() => expect(onSend).toHaveBeenCalledWith('/tmp/test.txt'))
  })

  it('shows error state', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ message: 'Upload failed' }),
      }),
    )
    renderWithProviders(<FileUpload />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' })
    await userEvent.upload(input, file)
    await waitFor(() => expect(screen.getByText('Upload failed')).toBeInTheDocument())
  })

  it('handles multiple file upload', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            files: [
              { ...uploadResult, originalname: 'a.txt', url: '/u/a.txt' },
              { ...uploadResult, originalname: 'b.txt', url: '/u/b.txt' },
            ],
          }),
      }),
    )
    renderWithProviders(<FileUpload />)
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const files = [new File(['a'], 'a.txt'), new File(['b'], 'b.txt')]
    await userEvent.upload(input, files)
    await waitFor(() => expect(screen.getByText('a.txt')).toBeInTheDocument())
    expect(screen.getByText('b.txt')).toBeInTheDocument()
  })
})
