import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    root: '.',
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    exclude: ['**/e2e/**', '**/node_modules/**'],
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
