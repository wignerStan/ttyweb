import { Check, Pencil } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Note } from './useNotepad'

interface NotepadTabProps {
  note: Note
  onUpdate: (id: number, updates: { name?: string; content?: string }, debounceMs?: number) => void
}

export function NotepadTab({ note, onUpdate }: NotepadTabProps) {
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState(note.name)
  const [content, setContent] = useState(note.content)
  const titleInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setTitleDraft(note.name)
  }, [note.name])

  useEffect(() => {
    setContent(note.content)
  }, [note.content])

  useEffect(() => {
    if (!editingTitle) return
    titleInputRef.current?.focus()
    titleInputRef.current?.select()
  }, [editingTitle])

  const handleTitleSave = useCallback(() => {
    const trimmed = titleDraft.trim()
    if (trimmed && trimmed !== note.name) {
      onUpdate(note.id, { name: trimmed })
    } else {
      setTitleDraft(note.name)
    }
    setEditingTitle(false)
  }, [titleDraft, note.name, note.id, onUpdate])

  const handleTitleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        handleTitleSave()
      } else if (e.key === 'Escape') {
        setTitleDraft(note.name)
        setEditingTitle(false)
      }
    },
    [handleTitleSave, note.name],
  )

  const handleContentChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newContent = e.target.value
      setContent(newContent)
      onUpdate(note.id, { content: newContent }, 500)
    },
    [note.id, onUpdate],
  )

  const handleContentBlur = useCallback(() => {
    onUpdate(note.id, { content })
  }, [note.id, content, onUpdate])

  return (
    <div className="notepad-tab">
      <div className="notepad-tab-header">
        {editingTitle ? (
          <div className="notepad-title-edit">
            <input
              ref={titleInputRef}
              className="notepad-title-input"
              value={titleDraft}
              onChange={(e) => setTitleDraft(e.target.value)}
              onBlur={handleTitleSave}
              onKeyDown={handleTitleKeyDown}
              maxLength={128}
            />
            <button
              type="button"
              className="notepad-title-save-btn"
              onClick={handleTitleSave}
              aria-label="Save title"
            >
              <Check size={14} />
            </button>
          </div>
        ) : (
          <button
            type="button"
            className="notepad-title-display"
            onClick={() => setEditingTitle(true)}
            title="Click to rename"
          >
            <span className="notepad-title-text">{note.name}</span>
            <Pencil size={12} className="notepad-title-edit-icon" />
          </button>
        )}
      </div>
      <textarea
        className="notepad-textarea"
        value={content}
        onChange={handleContentChange}
        onBlur={handleContentBlur}
        placeholder="Start typing…"
        spellCheck={false}
      />
    </div>
  )
}
