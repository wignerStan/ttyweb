import '@testing-library/jest-dom';

// Polyfill browser APIs missing in jsdom
global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
