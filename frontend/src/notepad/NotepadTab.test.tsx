import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { NotepadTab } from './NotepadTab'
import type { Note } from './useNotepad'

const note: Note = {
  id: 1,
  name: 'Test Note',
  content: 'Hello world',
  project_id: null,
  order_index: 0,
}

describe('NotepadTab', () => {
  it('renders tab header with note name', () => {
    renderWithProviders(<NotepadTab note={note} onUpdate={vi.fn()} />)
    expect(screen.getByText('Test Note')).toBeInTheDocument()
  })

  it('shows textarea for editing', () => {
    renderWithProviders(<NotepadTab note={note} onUpdate={vi.fn()} />)
    const textarea = screen.getByPlaceholderText('Start typing...')
    expect(textarea).toBeInTheDocument()
    expect(textarea).toHaveValue('Hello world')
  })

  it('save button triggers updateNote', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    // Click the title to enter edit mode
    await user.click(screen.getByText('Test Note'))
    // Change the title
    const input = screen.getByDisplayValue('Test Note')
    await user.clear(input)
    await user.type(input, 'Renamed')
    // Click save button
    await user.click(screen.getByLabelText('Save title'))
    expect(onUpdate).toHaveBeenCalledWith(1, { name: 'Renamed' })
  })

  it('does not save title when name is unchanged', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Test Note'))
    const input = screen.getByDisplayValue('Test Note')
    // Clear and re-type the same name
    await user.clear(input)
    await user.type(input, 'Test Note')
    await user.click(screen.getByLabelText('Save title'))
    expect(onUpdate).not.toHaveBeenCalled()
  })

  it('does not save title when name is empty after trim', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Test Note'))
    const input = screen.getByDisplayValue('Test Note')
    await user.clear(input)
    await user.type(input, '   ')
    await user.click(screen.getByLabelText('Save title'))
    expect(onUpdate).not.toHaveBeenCalled()
  })

  it('cancels title edit on Escape key', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Test Note'))
    const input = screen.getByDisplayValue('Test Note')
    await user.type(input, '{Escape}')
    // Title should revert and edit mode should close
    expect(screen.queryByDisplayValue('Test Note')).not.toBeInTheDocument()
    expect(screen.getByText('Test Note')).toBeInTheDocument()
  })

  it('saves title on Enter key', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    await user.click(screen.getByText('Test Note'))
    const input = screen.getByDisplayValue('Test Note')
    await user.clear(input)
    await user.type(input, 'New Title{Enter}')
    expect(onUpdate).toHaveBeenCalledWith(1, { name: 'New Title' })
  })

  it('calls onUpdate on content change with debounce', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    const textarea = screen.getByPlaceholderText('Start typing...')
    await user.type(textarea, ' more text')
    expect(onUpdate).toHaveBeenCalledWith(1, { content: 'Hello world more text' }, 500)
  })

  it('calls onUpdate on textarea blur', async () => {
    const onUpdate = vi.fn()
    renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const user = userEvent.setup()
    const textarea = screen.getByPlaceholderText('Start typing...')
    await user.type(textarea, ' updated')
    await user.tab()
    expect(onUpdate).toHaveBeenCalledWith(1, { content: 'Hello world updated' })
  })

  it('updates title draft when note.name prop changes', async () => {
    const onUpdate = vi.fn()
    const { rerender } = renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const newNote: Note = { ...note, name: 'Updated Name' }
    rerender(<NotepadTab note={newNote} onUpdate={onUpdate} />)
    expect(screen.getByText('Updated Name')).toBeInTheDocument()
  })

  it('updates content when note.content prop changes', async () => {
    const onUpdate = vi.fn()
    const { rerender } = renderWithProviders(<NotepadTab note={note} onUpdate={onUpdate} />)
    const newNote: Note = { ...note, content: 'New content' }
    rerender(<NotepadTab note={newNote} onUpdate={onUpdate} />)
    const textarea = screen.getByPlaceholderText('Start typing...')
    expect(textarea).toHaveValue('New content')
  })

  it('shows pencil icon on title display', () => {
    renderWithProviders(<NotepadTab note={note} onUpdate={vi.fn()} />)
    const editIcon = document.querySelector('.notepad-title-edit-icon')
    expect(editIcon).toBeInTheDocument()
  })

  it('title display has click to rename tooltip', () => {
    renderWithProviders(<NotepadTab note={note} onUpdate={vi.fn()} />)
    expect(screen.getByTitle('Click to rename')).toBeInTheDocument()
  })
})
