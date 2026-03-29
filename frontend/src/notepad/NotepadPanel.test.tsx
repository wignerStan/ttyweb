import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Note } from './useNotepad'

vi.mock('./useNotepad', () => ({
  useNotepad: () => ({
    notes: mockNotes,
    loading: mockLoading,
    error: mockError,
    fetchNotes: vi.fn(),
    createNote: mockCreateNote,
    updateNote: mockUpdateNote,
    deleteNote: mockDeleteNote,
    reorderNotes: vi.fn(),
  }),
}))

vi.mock('./NotepadTab', () => ({
  NotepadTab: ({
    note,
    onUpdate,
  }: {
    note: Note
    onUpdate: (id: number, u: { content?: string }) => void
  }) => (
    <div data-testid="notepad-tab">
      <span>{note.name}</span>
      <textarea
        data-testid="notepad-editor"
        defaultValue={note.content}
        onBlur={() => onUpdate(note.id, { content: 'changed' })}
      />
    </div>
  ),
}))

import { renderWithProviders } from '../test-utils'
import { NotepadPanel } from './NotepadPanel'

const notes: Note[] = [
  { id: 1, name: 'Note A', content: 'Content A', project_id: null, order_index: 0 },
  { id: 2, name: 'Note B', content: 'Content B', project_id: null, order_index: 1 },
]

let mockNotes: Note[] = notes
let mockLoading = false
let mockError: string | null = null
const mockCreateNote = vi
  .fn()
  .mockResolvedValue({ id: 3, name: 'Untitled', content: '', project_id: null, order_index: 2 })
const mockUpdateNote = vi.fn()
const mockDeleteNote = vi.fn().mockResolvedValue(undefined)

describe('NotepadPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockNotes = notes
    mockLoading = false
    mockError = null
  })

  it('renders note list with names', () => {
    renderWithProviders(<NotepadPanel />)
    expect(screen.getByText('Note A')).toBeInTheDocument()
    expect(screen.getByText('Note B')).toBeInTheDocument()
  })

  it('clicking note selects it and shows editor', async () => {
    renderWithProviders(<NotepadPanel />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Note A'))
    expect(screen.getByTestId('notepad-tab')).toBeInTheDocument()
  })

  it('create button calls createNote', async () => {
    renderWithProviders(<NotepadPanel />)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('New note'))
    expect(mockCreateNote).toHaveBeenCalled()
  })

  it('delete button calls deleteNote', async () => {
    renderWithProviders(<NotepadPanel />)
    // First select a note
    const user = userEvent.setup()
    await user.click(screen.getByText('Note A'))
    await user.click(screen.getByLabelText('Close Note A'))
    expect(mockDeleteNote).toHaveBeenCalledWith(1)
  })

  it('edit triggers updateNote', async () => {
    renderWithProviders(<NotepadPanel />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Note A'))
    const editor = screen.getByTestId('notepad-editor')
    await user.click(editor)
    await user.tab() // blur to trigger onUpdate
    expect(mockUpdateNote).toHaveBeenCalled()
  })

  it('empty state when no notes', () => {
    mockNotes = []
    renderWithProviders(<NotepadPanel />)
    expect(screen.getByText('No notes yet. Click + to create one.')).toBeInTheDocument()
  })
})
