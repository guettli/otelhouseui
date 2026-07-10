import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Vite builds the SPA into ./build/, which the Go binary embeds. During `pnpm
// run dev`, /api and /healthz are proxied to the local Go service on :8080.
export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'build',
    emptyOutDir: true,
    target: 'es2020',
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'node',
    globals: true,
  },
})
