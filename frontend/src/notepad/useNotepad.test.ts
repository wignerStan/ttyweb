import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from 'vitest'
import type { Note } from './useNotepad'
import { useNotepad } from './useNotepad'

// --- Helpers ---

function makeNote(overrides: Partial<Note> = {}): Note {
  return {
    id: 1,
    name: 'Test Note',
    content: 'Hello world',
    project_id: null,
    order_index: 0,
    ...overrides,
  }
}

function mockFetchSuccess<T>(data: T): Mock {
  return vi.fn().mockResolvedValue({
    json: () => Promise.resolve({ success: true, data }),
  })
}

function mockFetchApiError(error: string): Mock {
  return vi.fn().mockResolvedValue({
    json: () => Promise.resolve({ success: false, error }),
  })
}

function mockFetchNetworkError(): Mock {
  return vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
}

// --- Tests ---

describe('useNotepad', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should fetch notes on mount', async () => {
    const notes = [makeNote({ id: 1 }), makeNote({ id: 2, name: 'Second' })]
    globalThis.fetch = mockFetchSuccess(notes)

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.notes).toHaveLength(2)
    expect(result.current.notes[0]!.name).toBe('Test Note')
  })

  it('should include project_id in fetch URL when provided', async () => {
    globalThis.fetch = mockFetchSuccess([])

    renderHook(() => useNotepad('proj-42'))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/notepad?project_id=proj-42', undefined)
    })
  })

  it('should set error on fetch failure', async () => {
    globalThis.fetch = mockFetchApiError('db error')

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('db error')
    expect(result.current.notes).toEqual([])
  })

  it('should set error on network failure', async () => {
    globalThis.fetch = mockFetchNetworkError()

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // TypeError is instanceof Error, so its message is used
    expect(result.current.error).toBe('Failed to fetch')
  })

  it('should create a note and append to state', async () => {
    globalThis.fetch = mockFetchSuccess([])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const newNote = makeNote({ id: 3, name: 'New Note' })
    globalThis.fetch = mockFetchSuccess(newNote)

    let created: Note | null = null
    await act(async () => {
      created = await result.current.createNote('New Note', 'content')
    })

    expect(created).toEqual(newNote)
    expect(result.current.notes).toHaveLength(1)
    expect(result.current.notes[0]!.name).toBe('New Note')
  })

  it('should send project_id when creating with a project', async () => {
    globalThis.fetch = mockFetchSuccess([])

    const { result } = renderHook(() => useNotepad('proj-1'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const newNote = makeNote({ id: 5, project_id: 'proj-1' })
    globalThis.fetch = mockFetchSuccess(newNote)

    await act(async () => {
      await result.current.createNote('Note')
    })

    const call = (globalThis.fetch as Mock).mock.calls[
      (globalThis.fetch as Mock).mock.calls.length - 1
    ]
    expect(call).toBeDefined()
    const body = JSON.parse(call![1]!.body as string)
    expect(body.project_id).toBe('proj-1')
  })

  it('should send order_index when creating with a position', async () => {
    globalThis.fetch = mockFetchSuccess([])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const newNote = makeNote({ id: 6, order_index: 3 })
    globalThis.fetch = mockFetchSuccess(newNote)

    await act(async () => {
      await result.current.createNote('Positioned', '', 3)
    })

    const call = (globalThis.fetch as Mock).mock.calls[
      (globalThis.fetch as Mock).mock.calls.length - 1
    ]
    expect(call).toBeDefined()
    const body = JSON.parse(call![1]!.body as string)
    expect(body.order_index).toBe(3)
  })

  it('should return null and set error on create failure', async () => {
    globalThis.fetch = mockFetchSuccess([])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchApiError('create failed')

    let created: Note | null = null
    await act(async () => {
      created = await result.current.createNote('Bad')
    })

    expect(created).toBeNull()
    expect(result.current.error).toBe('create failed')
  })

  it('should update a note in state immediately (no debounce)', async () => {
    const note = makeNote({ id: 1, content: 'old' })
    globalThis.fetch = mockFetchSuccess([note])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    globalThis.fetch = mockFetchSuccess(note)

    await act(async () => {
      result.current.updateNote(1, { content: 'new' })
    })

    expect(result.current.notes[0]!.content).toBe('new')
  })

  it('should debounce updates when debounceMs is provided', async () => {
    const note = makeNote({ id: 1, content: 'initial' })
    globalThis.fetch = mockFetchSuccess([note])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    globalThis.fetch = mockFetchSuccess(note)

    await act(() => {
      result.current.updateNote(1, { content: 'draft 1' }, 300)
    })

    // State should not have changed yet (debounced)
    expect(result.current.notes[0]!.content).toBe('initial')

    // Advance timers past debounce
    await act(async () => {
      await vi.advanceTimersByTimeAsync(500)
    })

    // After debounce fires, state should update
    expect(result.current.notes[0]!.content).toBe('draft 1')
  })

  it('should cancel previous debounced update when a new one arrives', async () => {
    const note = makeNote({ id: 1, content: 'initial' })
    globalThis.fetch = mockFetchSuccess([note])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    const updateMock = mockFetchSuccess(note)
    globalThis.fetch = updateMock

    await act(() => {
      result.current.updateNote(1, { content: 'first' }, 300)
    })

    await act(() => {
      result.current.updateNote(1, { content: 'second' }, 300)
    })

    // Advance timers past both debounce windows
    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })

    // Only the second update should have been sent (first was cancelled)
    expect(result.current.notes[0]!.content).toBe('second')
    // Initial fetch + 1 debounced update = 2 calls
    // The initial fetch was made with the first mock, so we only count the update mock calls
    expect(updateMock).toHaveBeenCalledTimes(1)
  })

  it('should delete a note and remove from state', async () => {
    const note = makeNote({ id: 1 })
    globalThis.fetch = mockFetchSuccess([note])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    globalThis.fetch = mockFetchSuccess(null)

    await act(async () => {
      await result.current.deleteNote(1)
    })

    expect(result.current.notes).toHaveLength(0)
  })

  it('should set error on delete failure', async () => {
    const note = makeNote({ id: 1 })
    globalThis.fetch = mockFetchSuccess([note])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    globalThis.fetch = mockFetchApiError('delete failed')

    await act(async () => {
      await result.current.deleteNote(1)
    })

    expect(result.current.error).toBe('delete failed')
    expect(result.current.notes).toHaveLength(1)
  })

  it('should reorder notes and update order_index in state', async () => {
    const note1 = makeNote({ id: 1, name: 'A', order_index: 0 })
    const note2 = makeNote({ id: 2, name: 'B', order_index: 1 })
    globalThis.fetch = mockFetchSuccess([note1, note2])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(2)
    })

    globalThis.fetch = mockFetchSuccess(null)

    await act(async () => {
      await result.current.reorderNotes([
        { id: 1, order_index: 1 },
        { id: 2, order_index: 0 },
      ])
    })

    // Notes should be reordered
    expect(result.current.notes[0]!.name).toBe('B')
    expect(result.current.notes[1]!.name).toBe('A')
    expect(result.current.notes[0]!.order_index).toBe(0)
    expect(result.current.notes[1]!.order_index).toBe(1)
  })

  it('should set error on reorder failure', async () => {
    const note1 = makeNote({ id: 1 })
    globalThis.fetch = mockFetchSuccess([note1])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchApiError('reorder failed')

    await act(async () => {
      await result.current.reorderNotes([{ id: 1, order_index: 5 }])
    })

    expect(result.current.error).toBe('reorder failed')
  })

  it('should manually refetch via fetchNotes', async () => {
    globalThis.fetch = mockFetchSuccess([makeNote({ id: 1 })])

    const { result } = renderHook(() => useNotepad())

    await waitFor(() => {
      expect(result.current.notes).toHaveLength(1)
    })

    globalThis.fetch = mockFetchSuccess([makeNote({ id: 1 }), makeNote({ id: 2 })])

    await act(async () => {
      await result.current.fetchNotes()
    })

    expect(result.current.notes).toHaveLength(2)
  })
})
