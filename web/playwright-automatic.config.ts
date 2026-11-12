import { defineConfig, devices } from "@playwright/experimental-ct-react";

/** Production Next forwarding, actual Go background service and restricted disposable PostgreSQL. */
export default defineConfig({
  testDir: "./tests/integration",
  testMatch: ["automatic.spec.tsx"],
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  reporter: "list",
  outputDir: "test-results/automatic",
  use: {
    ...devices["Desktop Chrome"],
    ctPort: 39083,
    ctViteConfig: {
      server: {
        proxy: {
          "/v1": { target: "http://127.0.0.1:39082", changeOrigin: false },
          "/v2": { target: "http://127.0.0.1:39082", changeOrigin: false },
          "/v3": { target: "http://127.0.0.1:39082", changeOrigin: false },
        },
      },
    },
  },
  projects: [{ name: "chromium" }],
});
