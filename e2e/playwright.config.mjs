import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: '.',
  outputDir: 'test-results',
  // Each spec signs in with its own seeded accounts (see services/api/internal/seed), so specs run in parallel.
  workers: 4,
  use: { baseURL: 'http://127.0.0.1:3000', channel: 'msedge', trace: 'retain-on-failure' },
  webServer: {
    command: 'corepack pnpm --filter @grupon/web start',
    url: 'http://127.0.0.1:3000/uz',
    reuseExistingServer: true,
    timeout: 120000,
  },
});
