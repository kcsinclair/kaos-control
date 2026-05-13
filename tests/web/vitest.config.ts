// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    // Set a real origin so window.location.origin is not 'null'. The API client
    // uses relative URLs ('/api/…') which fetch must resolve against an origin;
    // happy-dom defaults to 'about:blank' which causes ERR_INVALID_URL on every
    // un-mocked fetch leaking out of component onMounted/watch callbacks.
    //
    // The host here is a deliberately fake string — `http://test.local` — so
    // it's obviously not a real server. Un-mocked fetches that leak out of
    // tests will fail with `ECONNREFUSED test.local` (or DNS error), making
    // the leak source easy to spot in logs without colliding with any port
    // number used by the real application or its CI fixtures. Tracked under
    // lifecycle/defects/frontend-tests-leak-unmocked-fetches.md.
    environmentOptions: {
      happyDOM: { url: 'http://test.local' },
    },
    globals: true,
    // Run *.perf.test.ts / *.perf.spec.ts files in isolated forked processes
    // so they don't compete for CPU with component-mounting tests.  Without
    // this, wall-clock timings measured by performance.now() absorb OS
    // scheduler jitter introduced by the concurrent worker-thread pool and
    // cause intermittent threshold failures (see defect
    // sortable-table-columns-19-defect.md).
    poolMatchGlobs: [
      ['**/*.perf.test.ts', 'forks'],
      ['**/*.perf.spec.ts', 'forks'],
    ],
    poolOptions: {
      forks: {
        // Each perf file runs in its own dedicated fork — no sharing between
        // perf suites — so a slow sort in one file cannot inflate timings in
        // another.
        singleFork: false,
      },
    },
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
      // echarts lives in web/node_modules. Canonical aliases ensure vi.mock('echarts/core')
      // and the component's own import resolve to the same file, so mocks are intercepted.
      'echarts/core': fileURLToPath(new URL('../../web/node_modules/echarts/core.js', import.meta.url)),
      'echarts/charts': fileURLToPath(new URL('../../web/node_modules/echarts/charts.js', import.meta.url)),
      'echarts/components': fileURLToPath(new URL('../../web/node_modules/echarts/components.js', import.meta.url)),
      'echarts/renderers': fileURLToPath(new URL('../../web/node_modules/echarts/renderers.js', import.meta.url)),
    },
    // Force a single copy of pinia and vue so that the global `activePinia`
    // singleton is shared between store source files (resolved from web/src)
    // and test files (resolved from tests/web). Without deduplication, each
    // node_modules copy has its own `activePinia` variable, making
    // setActivePinia() in tests invisible to the store's defineStore().
    dedupe: ['pinia', 'vue', 'vue-router'],
  },
})
