---
title: CLI Auth User Management and Secured API — Frontend Plan
type: plan-frontend
status: draft
lineage: cli-auth-user-management
parent: lifecycle/requirements/cli-auth-user-management-2.md
release: KC-Release0
assignees:
  - role: frontend-developer
    who: agent
---

# CLI Auth User Management and Secured API — Frontend Plan

The backend plan [[cli-auth-user-management-3-be]] moves auth enforcement from per-route opt-in to global. The frontend must handle 401 responses gracefully: redirect to login, clear stale session state, and ensure the CSRF double-submit flow works end-to-end with the new global middleware. No web-based user-management UI is in scope (explicit non-goal in the requirement).

## Milestone 1: Global 401 Interceptor and Login Redirect

### Description

Add an Axios/fetch response interceptor (or equivalent) that catches any `401` response from the API and redirects the user to the login page. This ensures that when the global auth middleware rejects a request, the user sees a login prompt instead of a broken page or silent failure.

### Files to change

- **`web/src/api/client.ts`** (or wherever the shared HTTP client is configured)
  - Add a response interceptor: if `response.status === 401` and the current route is not already `/login`, redirect to `/login` via `router.push('/login')`.
  - Clear any cached user state in the auth Pinia store before redirecting.

- **`web/src/stores/auth.ts`** (or equivalent Pinia auth store)
  - Add a `clearSession()` action that resets user state to unauthenticated defaults.
  - Ensure the store exposes an `isAuthenticated` getter for route guards.

### Acceptance criteria

- [ ] An expired or missing session on any API call redirects to `/login`.
- [ ] The redirect does not trigger an infinite loop when already on `/login`.
- [ ] Cached user state is cleared before the redirect so the login page renders cleanly.

## Milestone 2: Login Page Handles Global Auth Context

### Description

Verify and adjust the existing login page/component so it works correctly with the new global middleware. The login page must be reachable without authentication (the backend exempts `POST /api/auth/login`), and after successful login the user should be sent to the page they originally requested.

### Files to change

- **`web/src/views/LoginView.vue`** (or equivalent)
  - After successful `POST /api/auth/login`, read a `redirect` query parameter (set by the 401 interceptor) and navigate there; default to `/` if absent.
  - Display a message if the user was redirected due to session expiry (e.g., "Your session has expired. Please log in again.").

- **`web/src/router/index.ts`**
  - Ensure the `/login` route does not have a navigation guard that itself requires auth.
  - Add a `beforeEach` guard: if the target route requires auth and the user is not authenticated, redirect to `/login?redirect=<originalPath>`.

### Acceptance criteria

- [ ] Navigating to a protected route when unauthenticated redirects to `/login?redirect=...`.
- [ ] After login, the user is redirected to the originally requested page.
- [ ] The login page is accessible without authentication.
- [ ] Session-expiry redirect shows an informational message.

## Milestone 3: CSRF Token Compatibility with Bearer Auth

### Description

The backend will skip CSRF enforcement for bearer-token-authenticated requests. The frontend uses session cookies, so it must continue to send the `X-CSRF-Token` header on mutating requests. Verify the existing CSRF double-submit implementation works with the updated middleware and that no regressions are introduced.

### Files to change

- **`web/src/api/client.ts`**
  - Confirm that the request interceptor reads the `kc_csrf` cookie and sets `X-CSRF-Token` on POST/PUT/DELETE requests.
  - If the CSRF cookie is missing (unauthenticated state), do not set the header — let the 401 interceptor handle the response.

### Acceptance criteria

- [ ] Mutating API calls from the SPA include `X-CSRF-Token` and succeed.
- [ ] If the CSRF cookie is missing, the request proceeds without the header (the 401 will handle it).
- [ ] No double-submit errors after the backend middleware changes.

## Milestone 4: Authenticated WebSocket Connections

### Description

Ensure the WebSocket connection to `/api/p/{project}/ws` works with the global auth middleware. The connection should include the session cookie automatically (browsers do this). If the WS connection is rejected with 401, the frontend should trigger the same login redirect flow.

### Files to change

- **`web/src/composables/useWebSocket.ts`** (or equivalent)
  - Add an `onerror` / `onclose` handler: if the close code indicates authentication failure (HTTP 401 during upgrade, or code 4401 if the backend uses a custom close code), trigger the auth store's `clearSession()` and redirect to login.
  - On reconnect attempts, check `isAuthenticated` before reconnecting to avoid a tight reconnect loop against a 401.

### Acceptance criteria

- [ ] WebSocket connections are established with the session cookie.
- [ ] A 401 during WS upgrade triggers login redirect.
- [ ] Reconnect logic does not loop against an unauthenticated state.

## Milestone 5: Build Verification

### Description

Ensure the frontend builds cleanly and the dev server works end-to-end with the backend auth changes.

### Files to change

- No new files — verification step only.
- Run `pnpm build` in `web/` to confirm no TypeScript errors.
- Run `pnpm dev` and test the login → dashboard flow manually in a browser.

### Acceptance criteria

- [ ] `make build-web` succeeds with zero errors and zero warnings.
- [ ] The login → protected-page flow works in the browser with the updated backend.
- [ ] Refreshing a protected page while logged in does not redirect to login.
- [ ] Refreshing a protected page after session expiry redirects to login.

## Cross-references

- [[cli-auth-user-management-3-be]] — Backend plan: global middleware, bearer tokens, session changes.
- [[cli-auth-user-management-5-test]] — Test plan: integration tests covering frontend auth flows.
