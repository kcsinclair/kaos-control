---
title: "Fix Status Check Frontend Type Mismatches"
type: plan-frontend
status: draft
lineage: status-checker-button-no-status-update
parent: lifecycle/defects/status-checker-button-no-status-update.md
---

# Fix Status Check Frontend Type Mismatches

This plan addresses the frontend side of the status checker defect. The TypeScript API types do not match the actual backend response shape, causing silent failures in the advance flow and broken child-artifact rendering.

## Milestone 1: Align AdvanceResult Interface with Backend Response

**Description:** The `AdvanceResult` interface declares `ok: boolean` and `advanced_to?: string`, but the backend sends `outcome: string` and `new_status: string`. The frontend cannot determine whether an advance succeeded, so it never reflects the updated status.

**Files to change:**
- `web/src/api/statusCheck.ts`

**Changes:**
1. Update `AdvanceResult` to match the backend contract (after backend Milestone 2 lands, both `ok`/`advanced_to` and `outcome`/`new_status` will be present — use `ok` and `advanced_to` as the primary fields).
2. Remove the stale `StaleChild` interface definition that declares `status: string` if the backend is not yet sending it (coordinate with [[status-checker-button-no-status-update]] backend plan Milestone 1 which adds it).

**Acceptance criteria:**
- `AdvanceResult` fields match the actual backend JSON keys.
- `pnpm exec vue-tsc --noEmit` passes with no type errors related to statusCheck types.
- Calling "Fix all" or individual "Advance" buttons results in artifacts actually updating in the UI without a manual page refresh.

## Milestone 2: Fix Children Rendering in StatusCheckPanel

**Description:** The `StatusCheckPanel.vue` template iterates `artifact.children` expecting objects (`child.path`, `child.status`) but receives plain strings from the current backend (to be fixed in the backend plan). Once the backend returns `ChildInfo[]`, the template should work. However, defensive handling should ensure the panel doesn't silently break if the shape is unexpected.

**Files to change:**
- `web/src/components/artifact/StatusCheckPanel.vue`

**Changes:**
1. Add a runtime guard: if `artifact.children[0]` is a string (legacy shape), map it to `{ path: child, status: '?' }` so the UI degrades gracefully.
2. Confirm the WebSocket relevance check (`a.children.some(c => c.path === changedPath)`) works correctly when children are objects with a `path` field (it already does once types are correct).
3. Ensure the "Because:" section renders both path and status per child.

**Acceptance criteria:**
- No runtime errors when `children` is either `string[]` (old backend) or `ChildInfo[]` (new backend).
- Child paths and statuses display correctly in the status check panel.
- WebSocket-driven auto-refresh triggers correctly when a child artifact's status changes.

## Milestone 3: Surface Advance Errors to the User

**Description:** Currently, if an advance call returns `outcome: "error"`, no message is displayed to the user. The defect notes "No visible error is shown to the user."

**Files to change:**
- `web/src/components/artifact/StatusCheckPanel.vue`

**Changes:**
1. After `advanceStatuses` resolves, inspect each result. If any have `ok === false`, display the `error` (or `reason`) field in the existing `error` ref so it renders in the `.sc-error` div.
2. For partial failures (some advanced, some errored), show a summary like "2 advanced, 1 failed: <reason>".

**Acceptance criteria:**
- When an advance is blocked by permissions, the user sees a clear error message in the panel (not just a silent no-op).
- Successful advances still refresh the results list and clear the error.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Cross-references

- [[status-checker-button-no-status-update]] (backend plan): Milestones 1–2 fix the API response shapes that this plan's types depend on.
- [[status-checker-button-no-status-update]] (test plan): E2E tests validate the full user flow through the panel.
