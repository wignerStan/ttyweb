import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { NotepadPanel } from './NotepadPanel'
import type { Note } from './useNotepad'

const mockNotes: Note[] = [
  { id: 1, name: 'Meeting Notes', content: 'Discuss roadmap', project_id: null, order_index: 0 },
  { id: 2, name: 'Ideas', content: 'New feature ideas', project_id: null, order_index: 1 },
]

vi.mock('./useNotepad', () => ({
  useNotepad: () => ({
    notes: mockNotes,
    loading: false,
    error: null,
    createNote: vi.fn(),
    updateNote: vi.fn(),
    deleteNote: vi.fn(),
  }),
}))

describe('NotepadPanel snapshot', () => {
  it('renders panel with notes', () => {
    const { container } = render(<NotepadPanel />)
    expect(container).toMatchSnapshot()
  })
})
