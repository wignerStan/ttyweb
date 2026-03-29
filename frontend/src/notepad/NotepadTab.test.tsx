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
})
