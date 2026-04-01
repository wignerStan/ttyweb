import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockFetchHttpError, mockFetchSuccess } from '../../../../test-helpers'
import { useImperialStudies } from './useImperialStudies'

vi.mock('../../../../utils/auth', () => ({
  getAuthHeader: () => '',
  getAuthHeaders: () => ({}),
}))

const studies = [
  {
    id: '1',
    title: 'Study A',
    description: '',
    status: 'active',
    config: {},
    created_at: null,
    updated_at: null,
  },
]

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetchSuccess({ imperial_studies: studies }))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useImperialStudies', () => {
  it('fetches studies on mount from correct URL', async () => {
    const { result } = renderHook(() => useImperialStudies())
    expect(fetch).toHaveBeenCalledWith(
      '/api/butler/imperial_studies?status=active',
      expect.any(Object),
    )
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.studies).toEqual(studies)
  })

  it('sets error on fetch failure', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    const { result } = renderHook(() => useImperialStudies())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBeTruthy()
    expect(result.current.studies).toEqual([])
  })

  it('refetch re-fetches data', async () => {
    const { result } = renderHook(() => useImperialStudies())
    await waitFor(() => expect(result.current.loading).toBe(false))
    const updatedStudies = [
      {
        id: '2',
        title: 'Study B',
        description: '',
        status: 'active',
        config: {},
        created_at: null,
        updated_at: null,
      },
    ]
    vi.stubGlobal('fetch', mockFetchSuccess({ imperial_studies: updatedStudies }))
    await act(async () => {
      await result.current.refetch()
    })
    expect(result.current.studies).toEqual(updatedStudies)
  })
})
