import { vi } from 'vitest'

interface APIResponse<T> {
  success: boolean
  data?: T
  error?: string
}

function createResponse<T>(response: APIResponse<T>): Response {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve(response),
  } as Response
}

export function mockFetchSuccess<T>(data: T) {
  return vi.fn().mockResolvedValue(createResponse({ success: true, data }))
}

export function mockFetchError(message: string) {
  return vi.fn().mockResolvedValue(createResponse({ success: false, error: message }))
}

export function mockFetchHttpError(status: number) {
  return vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve({}),
  } as Response)
}
