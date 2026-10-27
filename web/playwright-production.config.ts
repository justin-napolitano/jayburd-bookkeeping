import { defineConfig } from "@playwright/experimental-ct-react";
import base from "./playwright-connections.config";
// Isolated synthetic Production-profile proof; no external SDK, identity or provider.
export default defineConfig({
  ...base,
  testMatch: "production.spec.tsx",
  outputDir: "test-results/production-journey",
});
