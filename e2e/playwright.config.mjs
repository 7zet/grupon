import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: '.',
  outputDir: 'test-results',
  // The specs share seeded accounts, and each sign-in replaces the previous phone code, so run them one at a time.
  workers: 1,
  use: { baseURL: 'http://127.0.0.1:3000', channel: 'msedge', trace: 'retain-on-failure' },
  webServer: {
    command: 'pnpm --filter @grupon/web start',
    url: 'http://127.0.0.1:3000/uz',
    reuseExistingServer: true,
    timeout: 120000,
  },
});
