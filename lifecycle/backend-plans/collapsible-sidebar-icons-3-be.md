---
title: "Backend Plan — Collapsible Sidebar with Icon-Only Mode"
type: plan-backend
status: in-development
lineage: collapsible-sidebar-icons
parent: lifecycle/requirements/collapsible-sidebar-icons-2.md
---

# Backend Plan — Collapsible Sidebar with Icon-Only Mode

This feature is entirely frontend-driven. The sidebar collapse state is persisted in `localStorage` on the client, no new API endpoints or backend data model changes are required.

The backend plan therefore has **zero milestones** — it exists solely to satisfy the three-plan gate and to document that no backend work is needed.

## Rationale for No Backend Changes

1. **State persistence** — The requirement specifies `localStorage` (key `sidebar-collapsed`). No server-side user-preference storage is in scope.
2. **Navigation items** — The sidebar nav structure is defined statically in the `AppSidebar.vue` component. It is not served by an API.
3. **Icons** — Sourced from the existing `lucide-vue-next` package; no icon asset serving from the backend.
4. **Parse Errors badge** — Already served by `GET /p/:project/parse-errors`; no changes needed.
5. **Favicon for collapsed header** — Already served as a static asset from `web/dist/assets/`.

## Milestone 0 — Verification (no code changes)

**Description:** Confirm that no backend endpoint or embedded-asset change is required.

**Files to change:** None.

**Acceptance criteria:**
- [ ] `make build` succeeds without backend changes after [[collapsible-sidebar-icons]] frontend work is merged.
- [ ] `make test-unit` passes with no regressions.
- [ ] The existing `GET /p/:project/parse-errors` endpoint continues to return the badge count consumed by the sidebar in both collapsed and expanded states.
