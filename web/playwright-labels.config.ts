import { defineConfig, devices } from "@playwright/experimental-ct-react";
/** The browser uses the real production Next bridge and disposable restricted SQL API. */
export default defineConfig({
  testDir: "./tests/integration",
  testMatch: "labels.spec.tsx",
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  reporter: "list",
  outputDir: "test-results/labels",
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
