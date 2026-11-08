import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";

// Separate synthetic review bundle; no Next route, environment file, API proxy or external assets.
export default defineConfig({
  root: fileURLToPath(new URL(".", import.meta.url)),
  envDir: false,
  publicDir: false,
  base: "./",
  oxc: { jsx: { runtime: "automatic" } },
  server: { host: "127.0.0.1", port: 39093, strictPort: true },
  build: {
    outDir: "../../.local/frontend-review",
    emptyOutDir: true,
    sourcemap: false,
    cssCodeSplit: false,
  },
});
