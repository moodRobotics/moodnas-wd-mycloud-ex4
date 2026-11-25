import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// `npm run dev` proxies the API to a moodd started with
//   moodd -simulate -insecure-http -listen :8080 -http "" -data ./work/data -ui ./ui/dist
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: { '/api': 'http://localhost:8080' },
  },
  build: {
    target: 'es2020',
    assetsInlineLimit: 0,
  },
});
