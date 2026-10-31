import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: '.',
  outputDir: 'test-results',
  use: { baseURL: 'http://127.0.0.1:3000', channel: 'msedge', trace: 'retain-on-failure' },
  webServer: {
    command: 'pnpm --filter @grupon/web start',
    url: 'http://127.0.0.1:3000/uz',
    reuseExistingServer: true,
    timeout: 120000,
  },
});
