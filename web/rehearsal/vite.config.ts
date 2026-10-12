import { defineConfig } from 'vite';
import { fileURLToPath } from 'node:url';
// Build only this entry: no environment files, public-file copying, source maps or dev server.
export default defineConfig({
  root: fileURLToPath(new URL('.', import.meta.url)),
  envDir: false,
  publicDir: false,
  oxc: { jsx: { runtime: 'automatic' } },
  build: { outDir: '../../.local/identity-rehearsal', emptyOutDir: true, sourcemap: false },
});
