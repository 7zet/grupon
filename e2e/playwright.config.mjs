import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: '.',
  outputDir: 'test-results',
  // Each spec signs in with its own seeded accounts (see services/api/internal/seed), so specs run in parallel.
  workers: 4,
  // CI installs Playwright Chromium explicitly; do not depend on an image-specific Edge build.
  use: { baseURL: 'http://127.0.0.1:3000', browserName: 'chromium', trace: 'retain-on-failure' },
  webServer: {
    command: 'corepack pnpm --filter @dealo/web start',
    url: 'http://127.0.0.1:3000/uz',
    reuseExistingServer: true,
    timeout: 120000,
  },
});
