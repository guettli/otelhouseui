import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { viteSingleFile } from 'vite-plugin-singlefile'

// The report is served from the cluster as ONE self-contained file: caddy
// exposes a single `index.html` mounted from a ConfigMap, so every asset
// (JS, CSS, and the baked-in report-data.json) must be inlined. viteSingleFile
// inlines all script/style/asset references into index.html; combined with the
// data being `import`ed (not fetched) this yields a file that renders offline.
export default defineConfig({
  plugins: [svelte(), viteSingleFile()],
  build: {
    target: 'es2020',
    assetsInlineLimit: 100000000,
    cssCodeSplit: false,
    reportCompressedSize: false,
  },
})
