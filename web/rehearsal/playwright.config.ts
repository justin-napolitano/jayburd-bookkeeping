import { defineConfig, devices } from 'playwright/test';
// Isolated synthetic browser proof uses the same static server; no provider request leaves the test.
export default defineConfig({
  testDir: '.', testMatch: 'rehearsal.spec.ts', reporter: 'list',
  use: { ...devices['Desktop Chrome'], baseURL: 'http://127.0.0.1:39091' },
  webServer: {command: 'npm run rehearsal:build && node ../scripts/identity-rehearsal-server.mjs', cwd: '..', url: 'http://127.0.0.1:39091', reuseExistingServer: false},
});
