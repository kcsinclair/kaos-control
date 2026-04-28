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
    },
    // Force a single copy of pinia and vue so that the global `activePinia`
    // singleton is shared between store source files (resolved from web/src)
    // and test files (resolved from tests/web). Without deduplication, each
    // node_modules copy has its own `activePinia` variable, making
    // setActivePinia() in tests invisible to the store's defineStore().
    dedupe: ['pinia', 'vue', 'vue-router'],
  },
})
