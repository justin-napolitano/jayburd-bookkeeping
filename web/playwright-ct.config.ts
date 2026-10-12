import { defineConfig, devices } from "@playwright/experimental-ct-react";
// Synthetic component browser tests have their own bundle; no production auth bypass exists.
export default defineConfig({
  testDir: "./tests",
  testIgnore: ["**/integration/**", "**/demo/**"],
  fullyParallel: true,
  reporter: "list",
  use: { ...devices["Desktop Chrome"], ctPort: 3100 },
  projects: [{ name: "chromium" }],
});
