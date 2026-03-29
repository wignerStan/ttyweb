import '@testing-library/jest-dom'

// Polyfill browser APIs missing in jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
