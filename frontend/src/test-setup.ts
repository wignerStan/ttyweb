import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Polyfill HTMLDialogElement for jsdom (which lacks showModal/close)
if (typeof HTMLDialogElement !== 'undefined' && !HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
    this.setAttribute('open', '')
  })
  HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
    this.removeAttribute('open')
  })
}

// Polyfill browser APIs missing in jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

// Polyfill localStorage when bun provides a bare object without methods
if (typeof localStorage.getItem !== 'function') {
  const store: Record<string, string | null> = {}

  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem(key: string): string | null {
        return key in store ? (store[key] ?? null) : null
      },
      setItem(key: string, value: string): void {
        store[key] = String(value)
      },
      removeItem(key: string): void {
        delete store[key]
      },
      clear(): void {
        for (const key of Object.keys(store)) {
          delete store[key]
        }
      },
      get length(): number {
        return Object.keys(store).length
      },
      key(index: number): string | null {
        return Object.keys(store)[index] ?? null
      },
    },
    writable: true,
    configurable: true,
  })
}
