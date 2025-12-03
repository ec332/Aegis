import type { PlaywrightTestConfig } from '@playwright/test';

const config: PlaywrightTestConfig = {
  testDir: './e2e',
  use: {
    baseURL: 'http://localhost:3000',
    headless: true,
  },
  webServer: {
    command: 'npm run start',
    cwd: './aegis-fe',
    port: 3000,
    timeout: 120000,
    reuseExistingServer: !process.env.CI,
  },
};

export default config;
