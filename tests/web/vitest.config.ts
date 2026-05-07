import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    globals: true,
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('../../web/src', import.meta.url)),
      // The graph libraries live in web/node_modules, not tests/web/node_modules.
      // Providing explicit aliases gives Vite a canonical resolved path that both
      // vi.mock() registrations and the component's dynamic imports agree on.
      // Without these aliases, vi.mock('cytoscape') resolves from tests/web (not
      // found) while the component resolves from web/node_modules (found but
      // unintercepted), causing mocks to be silently skipped.
      'cytoscape': fileURLToPath(new URL('../../web/node_modules/cytoscape/dist/cytoscape.esm.mjs', import.meta.url)),
      'cytoscape-fcose': fileURLToPath(new URL('../../web/node_modules/cytoscape-fcose/cytoscape-fcose.js', import.meta.url)),
      'cytoscape-dagre': fileURLToPath(new URL('../../web/node_modules/cytoscape-dagre/cytoscape-dagre.js', import.meta.url)),
      '3d-force-graph': fileURLToPath(new URL('../../web/node_modules/3d-force-graph/dist/3d-force-graph.mjs', import.meta.url)),
      'three': fileURLToPath(new URL('../../web/node_modules/three/build/three.module.js', import.meta.url)),
    },
    // Force a single copy of pinia and vue so that the global `activePinia`
    // singleton is shared between store source files (resolved from web/src)
    // and test files (resolved from tests/web). Without deduplication, each
    // node_modules copy has its own `activePinia` variable, making
    // setActivePinia() in tests invisible to the store's defineStore().
    dedupe: ['pinia', 'vue', 'vue-router'],
  },
})
