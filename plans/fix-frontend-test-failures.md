# Fix 4 frontend test failures (AppSidebar badges × 3, QueueView URL cleanup × 1)

## Context

`make lint` is green (after the recent `nonInitEvent` removal) but `pnpm test`
in `tests/web/` fails with 4 / 1358 tests red. Two distinct root causes,
neither related to a regression in functionality the user sees — they're
test-vs-component drift and one aspirational test waiting on its
implementation.

```
Tests  4 failed | 1354 passed (1358)
```

Failing tests:

1. `AppSidebar.test.ts` (3 failures, all in "Milestone 4: badge preservation")
   - badge is visible in expanded mode when parse errors exist
   - badge-dot is visible in collapsed mode when parse errors exist
   - badge-dot aria-label includes the error count
2. `QueueView.projectNav.test.ts` (1 failure)
   - M5-3: mount with ?project=nonexistent — URL is cleaned up (query param removed)

## Cause 1 — AppSidebar badge tests

The test file mocks `@/api/client` with a default `vi.fn().mockResolvedValue({ errors: [] })`,
then each failing test queues a single override via `mockResolvedValueOnce({ errors: [{...}] })`
intended to satisfy the parse-errors `api.get` call.

When the test was written (`71055e80b`, 2026-04-29), AppSidebar made one
`api.get` call on mount: `fetchParseErrors`. Two later changes added more:

- `d0810f3e` (2026-05-06) added `testingStore.fetchApprovedCount(p)` to onMounted,
  which routes through `artifactsApi.listArtifacts` → `api.get('/p/.../artifacts?…')`.
- `02c81d4f` (also 2026-05-06) wired `<GitStatusBar>` into AppSidebar.
  `GitStatusBar.onMounted` → `gitStatusStore.fetch(project)` → `api.get('/p/.../git/status')`.

Child `onMounted` fires before parent in Vue, so **the first `api.get` call
during mount is now `git/status` — not `parse-errors`**. The
`mockResolvedValueOnce` is consumed by the git/status call. `fetchParseErrors`
gets the default `{errors: []}`, `parseErrorCount` is 0, and the badge stays
hidden.

The simplest robust fix: stub `GitStatusBar` in the test's existing
module-mock block so it doesn't make an api.get call. This also stops bleed
from any future api.get additions inside GitStatusBar. With GitStatusBar
stubbed, parse-errors is the first api.get call (the second is the testing
store's artifacts fetch, which falls back to the default `{errors: []}` and
its no-op catch). Badge renders correctly.

## Cause 2 — QueueView URL cleanup

The test's own comment is unambiguous:

```ts
// NOTE: This assertion documents the expected behaviour per the test plan.
// The current QueueView implementation does NOT perform this cleanup —
// if this test fails it indicates a gap in the implementation.
```

`QueueView.vue:23-34` reads `?project=<name>` and applies it as the filter
only when the name matches a known project. If it doesn't match, the param
silently lingers in the URL. The fix is a one-line `else if` branch that
calls `router.replace` to strip the param so refresh/share state stays clean.

## Approach

Two surgical edits. Neither is invasive, neither changes user-visible
behaviour except in the edge case the QueueView test documents.

### Edit 1 — `tests/web/AppSidebar.test.ts`

Add to the existing module-mocks block near the top of the file (lines
~47–66), alongside the existing `vi.mock('@/api/client', …)`,
`vi.mock('@/api/ws', …)`, `vi.mock('@/stores/project', …)`:

```ts
vi.mock('@/components/layout/GitStatusBar.vue', () => ({
  default: { name: 'GitStatusBar', template: '<div data-testid="git-status-bar-stub" />' },
}))
```

Replaces the real `GitStatusBar` component with a no-op stub, so its
`onMounted` git-status fetch never runs. The three failing tests then
pass — the parse-errors `api.get` call is once again the first to consume
`mockResolvedValueOnce`. The fourth (negative) test in the same describe
block continues to pass because it relies only on the default mock.

### Edit 2 — `web/src/views/QueueView.vue`

Extend the `onMounted` block (lines 23–34):

```ts
onMounted(() => {
  void queueStore.fetch()
  void projectStore.fetchProjects().then(() => {
    const qp = route.query.project
    const name = typeof qp === 'string' ? qp : null
    if (name && projectStore.projects.some((p) => p.name === name)) {
      activeProject.value = name
    } else if (name) {
      // Unknown project name in the URL — strip the query param so the URL
      // reflects the actual ("All Projects") filter state. Use router.replace
      // so we don't push a stray history entry.
      const next = { ...route.query }
      delete next.project
      void router.replace({ query: next })
    }
  })
})
```

The `else if (name)` branch covers exactly the `?project=nonexistent` case;
the no-query-param case (`name === null`) is untouched.

## Files to modify

| File | Change |
|---|---|
| [tests/web/AppSidebar.test.ts](tests/web/AppSidebar.test.ts) (near line 47) | Add `vi.mock('@/components/layout/GitStatusBar.vue', …)` alongside the other module mocks. |
| [web/src/views/QueueView.vue](web/src/views/QueueView.vue#L23-L34) | Add the `else if (name)` cleanup branch inside `onMounted`'s `projectStore.fetchProjects().then(…)`. |

## Verification

1. `cd tests/web && pnpm test` — expect 1358/1358 pass, no failures.
2. `cd web && pnpm run build` — expect clean (vue-tsc happy).
3. Manual smoke for QueueView cleanup:
   - With server running, hit `http://localhost:8042/queue?project=does-not-exist`.
   - After page load, URL bar should read `/queue` (no query string) and
     the sidebar should show "All Projects" as the active filter.
   - Hit `http://localhost:8042/queue?project=kaos-control` (or a real project).
   - URL unchanged; sidebar shows kaos-control selected.

## Notes / non-goals

- I'm not addressing the **root cause** of "AppSidebar test was fragile
  because it relied on call order through a shared `vi.fn().mockResolvedValueOnce()`
  queue". A more thorough fix would replace the `mockResolvedValueOnce`
  pattern in those three tests with a URL-aware `mockImplementation`. That's
  worth doing as a separate hygiene pass; the targeted fix above unblocks CI
  without rewriting the test ergonomics.
- The QueueView fix matches what the test author explicitly stated they
  expected. No design discussion needed.

## Context

Server logs show repeated `GET /api/auth/me 401` entries when an unauthenticated user opens (or refreshes) any guarded SPA route — for example a deep-link to `/p/<project>/artifacts/board`. The 401s arrive in pairs (or worse) per navigation and the user sees the request grow stale. This appeared after commit `b2921c1` (global auth middleware) made every `/api/*` endpoint require auth, exposing a latent client-side bug.

**Root cause** — a recursive interaction between three pieces:

1. Router guard in [web/src/router/index.ts:115-126](web/src/router/index.ts#L115-L126): on the first navigation, calls `auth.fetchMe()` while `auth.initialized === false`.
2. Auth store in [web/src/stores/auth.ts:15-23](web/src/stores/auth.ts#L15-L23): catches a 401 from `fetchMe()` and sets `initialized = true` **only inside the `finally` block**.
3. API client interceptor in [web/src/api/client.ts:62-74](web/src/api/client.ts#L62-L74): on any 401 (including `/auth/me`), `await router.push('/login', …)` **before** rethrowing.

Because the interceptor's `await router.push('/login')` runs *inside* the original `fetchMe()` call's try-block, the new `/login` `beforeEach` fires while `initialized` is still `false`. That guard calls `fetchMe()` a second time, which 401s again. The inner push to `/login` is then de-duplicated by Vue Router, but the second `/auth/me` request has already been made. Net effect per failed page load: **two `/api/auth/me` 401s** (sometimes more if other parallel page-data requests also redirect-on-401).

The interceptor's redirect for `/auth/me` is also semantically redundant: `fetchMe()` is the *probe* the auth store and router use to decide if the user is logged in. Returning 401 to that probe is the expected "not logged in" signal — the router guard already converts that into a redirect to `/login` via the `requiresAuth` meta flag.

## Approach

One-line guard in [web/src/api/client.ts](web/src/api/client.ts): skip the interceptor's `router.push('/login')` when the failing request is `/auth/me`. Let `auth.fetchMe()` complete its `try/catch/finally` cycle so `initialized` stabilises, and let the router `beforeEach` issue the redirect via the standard `requiresAuth && !isAuthenticated` path.

No changes needed to the auth store, the router guard, or the backend.

## Files to modify

| File | Change |
|---|---|
| [web/src/api/client.ts](web/src/api/client.ts) (lines 61–74) | Wrap the 401 → push-to-login block in `if (path !== '/auth/me')`. Still throw the `ApiError` so callers see the failure. |

## Code change

```ts
// web/src/api/client.ts
if (!res.ok) {
  // /auth/me is the session probe used by the auth store; the router's
  // beforeEach guard turns its 401 into a redirect via requiresAuth.
  // Redirecting from here triggers a recursive beforeEach that re-fetches
  // /auth/me before initialized=true is set, producing duplicate 401s.
  if (res.status === 401 && path !== '/auth/me') {
    const [{ default: router }, { useAuthStore }] = await Promise.all([
      import('@/router'),
      import('@/stores/auth'),
    ])
    if (router.currentRoute.value.path !== '/login') {
      useAuthStore().clearSession()
      const currentPath = router.currentRoute.value.fullPath
      await router.push({ path: '/login', query: { redirect: currentPath, expired: '1' } })
    }
  }
  const err: ApiErrorBody = data?.error ?? { code: 'unknown', message: res.statusText }
  throw new ApiError(err.code, err.message, res.status)
}
```

## Verification

1. **Cold deep-link, no session.** Clear cookies. `make all && make run`. Open `http://localhost:8042/p/kaos-control/artifacts/board`. Expect:
   - Server log: exactly one `GET /api/auth/me 401`.
   - Browser lands on `/login?redirect=/p/kaos-control/artifacts/board&expired=1`.

2. **Login round-trip.** Log in. Verify exactly one `GET /api/auth/me 200` after login (from `auth.login → fetchMe`). Navigate between Map / Board / Roadmap — `/auth/me` should not be called again (because `initialized=true`).

3. **Mid-session expiry.** While logged in, expire the session (delete the `kc_session` cookie via DevTools, or `curl -X POST /api/auth/logout` with the cookie). Click any nav link that fetches data (e.g. Board → `/api/p/.../artifacts`). Expect:
   - One 401 on the page-data endpoint.
   - Redirect to `/login?expired=1`.
   - **No** trailing `/auth/me` 401 storm.

4. **Frontend tests.** `cd web && pnpm test` — no regressions in router/auth-store specs.

5. **Lint.** `make lint` clean.
