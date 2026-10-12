import { defineConfig, devices } from "playwright/test";
// Run against an explicitly started disposable demo; never a configurable live URL.
export default defineConfig({
  testDir: "./tests/demo",
  workers: 1,
  reporter: "list",
  use: { ...devices["Desktop Chrome"], baseURL: "http://localhost:39083" },
});
