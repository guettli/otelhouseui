import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Vite builds the SPA into ./build/, which the explore Go package embeds.
//
// `base` is a PLACEHOLDER, not a real path. explore is an embeddable handler a
// host app can mount under any prefix (agentloop mounts it at /explore), so the
// build cannot know its own public URL. Every absolute URL Vite emits — asset
// <script>/<link> hrefs and the window.__EXPLORE_BASE__ bootstrap in
// index.html — therefore carries this token, and the Go handler substitutes the
// real mount base when it serves index.html. One committed build, any mount
// point, no build-time configuration. Keep in sync with httpapi.BasePlaceholder.
//
// In dev the placeholder would break Vite's own URLs, so the base stays '/' and
// api.ts falls back to '/' when it sees an unsubstituted token. /api and
// /healthz are proxied to the local Go service on :8080.
export default defineConfig(({ command }) => ({
  base: command === 'build' ? '/__EXPLORE_BASE__/' : '/',
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
}))
