import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './flows',
  workers: 4,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  retries: 0,
  use: {
    headless: true,
    // baseURL is set per-test via the kctest fixture
  },
})
