import { defineConfig } from '@playwright/test';

const projectRoot = new URL('..', import.meta.url).pathname;

const BACKENDS = [
  { name: 'local', port: 18899 },
  { name: 'tmux', port: 18900 },
  { name: 'zellij', port: 18901 },
] as const;

const webServers = BACKENDS.map((backend) => ({
  command: `${projectRoot}ttyweb -backend ${backend.name} -port ${backend.port} -w`,
  port: backend.port,
  reuseExistingServer: !process.env.CI,
  timeout: 15000,
}));

const projects = BACKENDS.map((backend) => ({
  name: backend.name,
  use: {
    browserName: 'chromium',
    baseURL: `http://localhost:${backend.port}`,
  },
}));

export default defineConfig({
  reporter: process.env.CI
    ? [['html', { open: 'never' }], ['junit', { outputFile: 'test-results/results.xml' }]]
    : [['html', { open: 'never' }]],
  testDir: './e2e',
  testMatch: '**/*.spec.ts',
  fullyParallel: true,
  timeout: 30000,
  retries: process.env.CI ? 2 : 0,
  expect: {
    timeout: 5000,
  },
  workers: process.env.CI ? 2 : undefined,
  forbidOnly: !!process.env.CI,
  webServer: webServers,
  use: {
    screenshot: 'only-on-failure',
    video: process.env.CI ? 'retain-on-failure' : 'off',
    trace: 'on-first-retry',
    actionTimeout: 10000,
    navigationTimeout: 15_000,
  },
  projects,
});
