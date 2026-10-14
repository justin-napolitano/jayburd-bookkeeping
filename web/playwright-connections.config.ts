import { defineConfig } from "@playwright/experimental-ct-react";
import base from "./playwright-integration.config";
// Only this test bundle can reach the integration binary's synthetic acquisition trigger.
export default defineConfig({
  ...base,
  testMatch: "connections.spec.tsx",
  outputDir: "test-results/connections",
  timeout: 90000,
  use: {
    ...base.use,
    ctViteConfig: {
      server: {
        proxy: {
          "/v1": { target: "http://127.0.0.1:39082", changeOrigin: false },
          "/__proof": { target: "http://127.0.0.1:39081", changeOrigin: false },
        },
      },
    },
  },
});
