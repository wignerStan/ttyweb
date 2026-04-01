import { Plus, X } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { NotepadTab } from './NotepadTab'
import { type Note, useNotepad } from './useNotepad'

interface NotepadPanelProps {
  projectId?: string | null
}

export function NotepadPanel({ projectId = null }: NotepadPanelProps) {
  const { notes, loading, error, createNote, updateNote, deleteNote } = useNotepad(projectId)
  const [activeNoteId, setActiveNoteId] = useState<number | null>(null)

  const sortedNotes = useMemo(
    () => [...notes].sort((a, b) => a.order_index - b.order_index),
    [notes],
  )

  const handleCreate = useCallback(async () => {
    const maxOrder = sortedNotes.reduce((max, n) => Math.max(max, n.order_index), -1)
    const newNote = await createNote('Untitled', '', maxOrder + 1)
    if (newNote) {
      setActiveNoteId(newNote.id)
    }
  }, [createNote, sortedNotes])

  const handleClose = useCallback(
    async (noteId: number) => {
      await deleteNote(noteId)
      if (activeNoteId === noteId) {
        const remaining = sortedNotes.filter((n) => n.id !== noteId)
        setActiveNoteId(remaining.length > 0 ? (remaining[remaining.length - 1]?.id ?? null) : null)
      }
    },
    [deleteNote, activeNoteId, sortedNotes],
  )

  const activeNote: Note | undefined = sortedNotes.find((n) => n.id === activeNoteId)

  if (loading) {
    return <div className="notepad-panel notepad-loading">Loading notes...</div>
  }

  return (
    <div className="notepad-panel">
      {error && <div className="notepad-error">{error}</div>}

      <div className="notepad-tab-bar">
        {sortedNotes.map((note) => (
          <div
            key={note.id}
            role="tab"
            tabIndex={0}
            className={`notepad-bar-tab ${note.id === activeNoteId ? 'notepad-bar-tab-active' : ''}`}
            onClick={() => setActiveNoteId(note.id)}
            aria-selected={note.id === activeNoteId}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                setActiveNoteId(note.id)
              }
            }}
          >
            <span className="notepad-bar-tab-label">{note.name}</span>
            <button
              type="button"
              className="notepad-bar-tab-close"
              onClick={(e) => {
                e.stopPropagation()
                void handleClose(note.id)
              }}
              aria-label={`Close ${note.name}`}
            >
              <X size={12} />
            </button>
          </div>
        ))}
        <button
          type="button"
          className="notepad-add-btn"
          onClick={() => void handleCreate()}
          aria-label="New note"
        >
          <Plus size={14} />
        </button>
      </div>

      <div className="notepad-content">
        {activeNote ? (
          <NotepadTab key={activeNote.id} note={activeNote} onUpdate={updateNote} />
        ) : (
          <div className="notepad-empty">
            {sortedNotes.length === 0
              ? 'No notes yet. Click + to create one.'
              : 'Select a note from the tabs above.'}
          </div>
        )}
      </div>
    </div>
  )
}
