---
title: "Frontend Plan — Version Number Display"
type: plan-frontend
status: draft
lineage: version-number-display
parent: lifecycle/requirements/version-number-display-2.md
created: "2026-05-10"
---

# Frontend Plan — Version Number Display

This plan implements the SPA changes for [[version-number-display]]: fetching the version from the backend and rendering it as a persistent, subtle label in the application layout.

## Milestone 1 — API client function for version

### Description

Add a function to the API layer that calls `GET /api/version` and returns the version string. This endpoint is unauthenticated, so no session handling is required. The function should be simple — no caching beyond what the caller provides.

### Files to change

- `web/src/api/client.ts` (or a new `web/src/api/version.ts` if the team prefers per-resource modules) — add `fetchVersion(): Promise<string>` that calls `GET /api/version`, parses the JSON response, and returns the `version` field.

### Acceptance criteria

- [ ] `fetchVersion()` exists and returns the version string from `GET /api/version`.
- [ ] Handles non-200 responses gracefully (falls back to `"unknown"` or similar).
- [ ] No new npm packages introduced.

## Milestone 2 — Store or provide version at app level

### Description

Fetch the version once at app startup and make it available to layout components. The simplest approach: call `fetchVersion()` inside the existing router `beforeEach` guard (where `auth.fetchMe()` already runs) or in the `WorkspaceView.onMounted()` hook, and store the result in a reactive ref or a lightweight Pinia store.

A dedicated Pinia store (`web/src/stores/app.ts`) is the cleanest option — it can hold the version alongside any future app-level metadata.

### Files to change

- `web/src/stores/app.ts` (new) — minimal store with `version` ref and `fetchVersion` action.
- `web/src/views/project/WorkspaceView.vue` — call `appStore.fetchVersion()` in the existing `onMounted()` alongside `syncProject()`. Alternatively, trigger it from the router guard in `web/src/router/index.ts`.

### Acceptance criteria

- [ ] Version is fetched exactly once per app load, not on every route change.
- [ ] The version value is reactively available to any component via the store.
- [ ] A failed fetch does not break app initialisation — the label simply shows a fallback.
- [ ] No new npm packages introduced.

## Milestone 3 — Render version label in AppSidebar

### Description

Display `kaos-control <version>` in a fixed, always-visible location. The `AppSidebar` component is the natural home — place the label at the bottom of the sidebar, below navigation items. This keeps it visible on every view without competing with primary navigation or content.

### Files to change

- `web/src/components/layout/AppSidebar.vue`
  - Import the app store and read `version`.
  - Add a `<div>` or `<span>` at the bottom of the sidebar template rendering `kaos-control {{ version }}`.
  - Style: small font size (`0.75rem` / `text-xs`), muted colour (use the existing CSS custom property for secondary/muted text), no interaction affordance.

### Acceptance criteria

- [ ] The version label is visible on every page that has the sidebar.
- [ ] Format is exactly `kaos-control <version>` (e.g. `kaos-control 0.1.0`).
- [ ] The label does not obscure or shift navigation items.
- [ ] Font size is small and colour is muted relative to primary nav text.
- [ ] The label meets WCAG 2.1 AA contrast requirements against the sidebar background. Verify that the muted text colour has at least 4.5:1 contrast ratio (or 3:1 for large text) against the sidebar background in both light and dark themes.
- [ ] The displayed version matches the value returned by `GET /api/version` (cross-ref [[version-number-display]] test plan).
- [ ] No new npm packages introduced.
