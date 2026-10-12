import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";

// Separate review entrypoint: never imported by Next routes or the production image.
export default defineConfig({
  root: fileURLToPath(new URL(".", import.meta.url)),
  envDir: false,
  server: {
    host: "127.0.0.1",
    port: 39083,
    strictPort: true,
    proxy: { "/v1": { target: "http://127.0.0.1:39081", changeOrigin: false } },
    fs: { allow: [fileURLToPath(new URL("..", import.meta.url))] },
  },
});
