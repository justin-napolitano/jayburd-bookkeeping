import { defineConfig, devices } from "@playwright/experimental-ct-react";
/** Test proxy forwards to production Next API bridge; it never invents API responses. */
export default defineConfig({
  testDir: "./tests/integration",
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  reporter: "list",
  outputDir: "test-results/integration",
  use: {
    ...devices["Desktop Chrome"],
    ctPort: 39083,
    ctViteConfig: {
      server: {
        proxy: {
          "/v1": { target: "http://127.0.0.1:39082", changeOrigin: false },
        },
      },
    },
  },
  projects: [{ name: "chromium" }],
});
