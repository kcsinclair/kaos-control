---
title: Frontend plan — Release artefacts in markdown
type: plan-frontend
status: draft
lineage: release-artefacts
parent: requirements/release-artefacts-2.md
---

# Frontend plan — Release artefacts in markdown

UI changes flowing from `requirements/release-artefacts-2.md`. Bulk of
the work is on the backend ([[release-artefacts]]-3-be); the SPA must
surface the new `file_path`, react to the new `release.changed` WS
event, handle 409 collision/conflict responses, and add a rehydrate
control. Companion test plan: [[release-artefacts]]-5-test.

## Milestone F1 — Type & API client updates

**Description.** Extend the TypeScript types and API client to carry
`file_path`, `updated_at`, and the new `unscheduled` status. Round-trip
`updated_at` on PUT so the backend's conflict-detection (Resolved
Question 2) works.

**Files to change.**
- `web/src/api/releases.ts` — add `file_path: string` and
  `updated_at: string` to the `Release` interface; allow `'unscheduled'`
  in the `Status` union; ensure `updateRelease` includes `updated_at` in
  the request body (echoing whatever the store last received).
- `web/src/stores/releases.ts` — store `file_path` and `updated_at` on
  the cached release; expose a `getReleaseBySlug(slug)` selector for the
  WS handler in F3.
- `web/src/types/release.ts` (if separate from api file) — mirror the
  new fields.

**Acceptance criteria.**
- `pnpm -C web typecheck` passes.
- `Release` objects in the store contain `file_path` and `updated_at`
  for every release returned by `GET /releases`.
- `updateRelease` requests include the `updated_at` field; existing call
  sites compile without further changes.

## Milestone F2 — Release form: status, file_path display, collision handling

**Description.** Surface the new `unscheduled` status option, show the
`file_path` next to the release name as a read-only chip linking to the
artifact editor, and translate backend 409 responses into inline form
errors (slug collision and stale-update conflict produce distinct
messages).

**Files to change.**
- `web/src/components/releases/ReleaseFormModal.vue` — add
  `unscheduled` to the status `<select>`; show a read-only `file_path`
  field beneath the name input (only when editing an existing release);
  on submit, surface backend 409s: parse `error` body and display
  `"A release with this name already exists"` or
  `"This release was changed by another session — reload to continue"`
  beside the relevant field.
- `web/src/components/releases/ReleaseDetailModal.vue` — render
  `file_path` as a clickable link routing to
  `/projects/:project/artifacts/<file_path>` so users can jump to the
  source file.
- `web/src/views/project/__tests__/ReleaseFormModal.test.ts` (new or
  extended) — unit test using `@testing-library/vue` that mocks the API
  and asserts 409 maps to the slug-collision message.

**Acceptance criteria.**
- Selecting `Unscheduled` and submitting creates a release with no
  start/end date; the modal closes successfully.
- Creating a release whose name slugs to an existing one shows the
  inline collision error and keeps the modal open.
- The detail modal displays `file_path` as a hyperlink that navigates
  to the artifact editor for that file.
- Component unit tests pass under `pnpm -C web test`.

## Milestone F3 — `release.changed` WebSocket handling

**Description.** Subscribe to the new `release.changed` WS envelope and
update the releases store in place so the Roadmap and Gantt views
re-render without a manual refresh (DR-4 acceptance: "Manually editing
a release file on disk … triggers a WebSocket `release.changed` event
and the corresponding DB row is updated within one watcher debounce
window").

**Files to change.**
- `web/src/api/ws.ts` — extend the discriminated-union message type to
  include `{type: 'release.changed', project: string, release: Release}`
  and `{type: 'release.deleted', project: string, slug: string}`.
- `web/src/stores/releases.ts` — add `applyWsEvent(msg)` reducer:
  `release.changed` upserts by `id` (falling back to `slug` when `id`
  is unknown locally); `release.deleted` removes by `slug`. Increment a
  `lastWsSeq` counter the views can watch for re-render triggers.
- `web/src/composables/useReleasesSocket.ts` (new, or wire into the
  existing global WS composable) — registers the WS handler on view
  mount and tears it down on unmount.
- `web/src/components/releases/GanttChart.vue`,
  `web/src/components/releases/RoadmapGraphView.vue`,
  `web/src/components/releases/BacklogPanel.vue` — ensure they read from
  the store reactively (no local snapshots) so WS-driven changes
  propagate.

**Acceptance criteria.**
- Editing a release file on disk while the Roadmap page is open updates
  the visible release card / Gantt row within ~1 s without a page
  reload.
- Deleting a release file removes the card from the view.
- No duplicate cards appear when an API-originating change emits both
  the synchronous API response and any subsequent WS event (idempotent
  upsert).
- Existing WS handling for `artifact.indexed` remains unchanged.

## Milestone F4 — Rehydrate admin control

**Description.** Expose the new `POST /releases/rehydrate` endpoint
(Resolved Question 3) as a button in the Roadmap toolbar, gated to
admin/product-owner roles, that prompts for confirmation and reports
the result.

**Files to change.**
- `web/src/api/releases.ts` — add `rehydrateReleases(projectId):
  Promise<{inserted: number; skipped: number; errors: string[]}>`.
- `web/src/views/project/` — locate the Roadmap view (likely a new file
  `RoadmapView.vue` if not present, or the existing roadmap entry
  point) and add a "Rehydrate from disk" button next to the existing
  release-create button. Show a confirmation dialog, then call the API
  and surface a toast: `"Inserted N releases, skipped M"`.
- `web/src/composables/useCurrentUser.ts` (if it exists) — guard the
  button with `roles.includes('admin') || roles.includes('product-
  owner')`. If no role composable exists, hide the button behind a
  feature-flag query param `?rehydrate=1` and note the gap as a
  follow-up.

**Acceptance criteria.**
- An admin user sees and can click the "Rehydrate from disk" button;
  a non-admin does not see it.
- Clicking the button → confirming → calls `POST /releases/rehydrate`
  and shows a success toast with insert/skip counts. Errors are shown
  in a dismissible toast.
- After a successful rehydrate, the store refreshes via the existing
  `GET /releases` call so the view reflects the new state without a
  page reload.
