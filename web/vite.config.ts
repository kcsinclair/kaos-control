import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: false,
  },
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
          if (id.includes('node_modules/echarts') || id.includes('node_modules/zrender')) {
            return 'vendor-echarts'
          }
        },
      },
      // Silence the "dynamic import will not move module into another chunk"
      // warning for @/router and @/stores/auth. These two modules are
      // intentionally dynamic-imported from api/client.ts and api/ws.ts to
      // break a load-time circular dependency:
      //   router → stores/auth → api/auth → api/client → router
      // They're also statically imported by many components (header, sidebar,
      // login form, etc.) so Rollup correctly observes that the dynamic
      // imports don't move them to a separate chunk. That's fine — the goal
      // here is cycle-breaking, not chunk-splitting.
      onwarn(warning, defaultHandler) {
        const msg = warning.message ?? ''
        if (
          msg.includes('dynamic import will not move module into another chunk') &&
          (msg.includes('stores/auth.ts') || msg.includes('router/index.ts'))
        ) {
          return
        }
        defaultHandler(warning)
      },
    },
  },
  server: {
    proxy: {
      // `pnpm run dev` proxies /api/* (and WebSocket upgrades on /api/p/*/ws)
      // to a locally running kaos-control backend. The default backend port
      // is :8042 (see defaultApp() in internal/config and the README quick
      // start). Override here only if you're running the backend on a
      // non-default port for dev.
      '/api': {
        target: 'http://localhost:8042',
        ws: true,
      },
    },
  },
})
