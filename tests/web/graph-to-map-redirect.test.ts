// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Milestone 6 — Redirect test: /graph → /map
 *
 * Verifies that navigating to the legacy `/p/:project/graph` route is
 * transparently redirected to `/p/:project/map`, and that the resolved
 * route has name 'map'.
 *
 * Testing approach
 * ────────────────
 * A minimal Vue Router instance is created with memory history, mirroring
 * only the relevant routes from the app router:
 *   - /p/:project/map  (name: 'map')
 *   - /p/:project/graph  (redirect: { name: 'map' })
 *
 * No Vue components are mounted; this is a pure router behaviour test.
 */

import { describe, it, expect } from 'vitest'
import { createRouter, createMemoryHistory } from 'vue-router'

// ---------------------------------------------------------------------------
// Minimal router that mirrors the app's graph → map redirect
// ---------------------------------------------------------------------------

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/p/:project',
        component: { template: '<div><router-view/></div>' },
        children: [
          {
            path: 'map',
            name: 'map',
            component: { template: '<div/>' },
          },
          {
            path: 'graph',
            redirect: { name: 'map' },
          },
        ],
      },
    ],
  })
}

// ===========================================================================
// Redirect: /graph → /map
// ===========================================================================

describe('router — /graph redirects to /map', () => {
  it('navigating to /p/testproject/graph resolves to /p/testproject/map', async () => {
    const router = makeRouter()
    await router.push('/p/testproject/graph')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/p/testproject/map')
  })

  it('resolved route name is "map" after navigating to /graph', async () => {
    const router = makeRouter()
    await router.push('/p/testproject/graph')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('map')
  })

  it('route params are preserved through the redirect', async () => {
    const router = makeRouter()
    await router.push('/p/myproject/graph')
    await router.isReady()

    expect(router.currentRoute.value.params.project).toBe('myproject')
  })

  it('navigating directly to /p/testproject/map resolves to name "map"', async () => {
    const router = makeRouter()
    await router.push('/p/testproject/map')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('map')
    expect(router.currentRoute.value.path).toBe('/p/testproject/map')
  })

  it('the /graph route does not have its own name (it is a redirect-only route)', async () => {
    const router = makeRouter()
    // After redirect, the current route should be named 'map', not 'graph'
    await router.push('/p/testproject/graph')
    await router.isReady()

    expect(router.currentRoute.value.name).not.toBe('graph')
    expect(router.currentRoute.value.name).toBe('map')
  })
})
