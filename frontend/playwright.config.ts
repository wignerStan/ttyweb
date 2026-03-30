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
  timeout: 30000,
  retries: 1,
  expect: {
    timeout: 5000,
  },
  workers: process.env.CI ? 2 : 8,
  forbidOnly: !!process.env.CI,
  webServer: webServers,
  use: {
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
    actionTimeout: 10000,
  },
  projects,
});
