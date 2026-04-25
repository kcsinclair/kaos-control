import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1400,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('three') || id.includes('3d-force-graph') || id.includes('ngraph') || id.includes('kapsule')) {
            return 'vendor-three'
          }
          if (id.includes('cytoscape')) {
            return 'vendor-cytoscape'
          }
          if (id.includes('@codemirror') || id.includes('codemirror')) {
            return 'vendor-codemirror'
          }
          if (id.includes('markdown-it')) {
            return 'vendor-markdown'
          }
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        ws: true,
      },
    },
  },
})
