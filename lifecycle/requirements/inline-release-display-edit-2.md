---
title: Inline Release Display and Editing on Artifact View
type: requirement
status: approved
lineage: inline-release-display-edit
created: "2026-05-12T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/inline-release-display-edit.md
labels:
    - frontend
    - artefacts
    - enhancement
    - vue
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

## Problem

The artifact detail view's FrontmatterPanel currently displays metadata fields in a fixed order: Type, Status, Priority, Stage, Lineage, Parent, Labels, Assignees, …, Release, Sprint, Created, Modified. The release field is buried near the bottom, behind conditional fields, making it easy to overlook during triage. Worse, changing an artifact's release assignment requires entering full edit mode (Edit → FrontmatterEditor → Save), which is disproportionately heavy for a single-field change.

Status and Priority already have inline click-to-edit dropdowns (StatusDropdown, PriorityDropdown). Release lacks this affordance, creating an inconsistent experience and slowing release-planning workflows.

## Goals / Non-goals

### Goals

1. **Reorder metadata fields** so the displayed order in the read-mode FrontmatterPanel is: Status, Priority, Release, then the remaining fields (Type, Stage, Lineage, …).
2. **Display release inline** — always show the Release row in FrontmatterPanel (display "None" when unset rather than hiding the row).
3. **Single-click release editing** — clicking the release value opens a dropdown populated from the project's available releases (via the existing `GET /api/p/:project/releases` endpoint). Selecting a release persists the change immediately without entering full edit mode.
4. **Allow clearing** — the dropdown must include a "None" option to unassign the release.

### Non-goals

- Creating or deleting releases from the artifact view (use the dedicated Releases page).
- Bulk-assigning releases across multiple artifacts.
- Modifying the FrontmatterEditor (full edit mode) — it continues to work as-is.
- Changing the Kanban card layout or any view other than the artifact detail view.

## Detailed Requirements

### Functional

**FR-1: Field reorder in FrontmatterPanel**
Reorder the `<dl>` rows in `FrontmatterPanel.vue` so the first three rows are Status, Priority, Release (in that order). All other rows remain in their current relative order after these three.

**FR-2: Release row always visible**
The Release row must render unconditionally (remove the `v-if="artifact.frontmatter.release"` guard). When no release is assigned, display "None" in muted text.

**FR-3: ReleaseDropdown component**
Create a new `ReleaseDropdown.vue` component following the same interaction pattern as `PriorityDropdown.vue`:

- **Props**: `project: string`, `path: string`, `release: string | null`, `readonly?: boolean`.
- **Emit**: `changed(newRelease: string | null)`, `error(message: string)`.
- **Behaviour**:
  - Renders the current release name as a clickable badge/button. Shows "None" when null.
  - On click (unless `readonly`), fetches releases from `listReleases(project)` and opens a dropdown menu.
  - Selecting a release calls a backend endpoint to persist the change and emits `changed`.
  - Selecting "None" sets the release to null.
  - Keyboard accessible: arrow keys to navigate, Enter to select, Escape to close.
  - Closes on outside click.

**FR-4: Backend PATCH endpoint for release**
Add a `PATCH /api/p/:project/artifacts/*/release` endpoint (mirroring the existing priority PATCH) that accepts `{ "release": "<name>" | null }` and updates the artifact's frontmatter `release` field on disk. The endpoint must:

- Re-index the artifact after writing.
- Return the updated artifact or a success status.
- Validate that the release name corresponds to an existing release in the project (return 422 if not).
- Accept `null` to clear the release assignment.

**FR-5: Frontend API function**
Add a `patchRelease(project, path, release)` function in `web/src/api/artifacts.ts`, following the same pattern as `patchPriority`.

**FR-6: Integration into FrontmatterPanel**
Replace the static release `<dd>` text in `FrontmatterPanel.vue` with `<ReleaseDropdown>` when `project` and `targetPath` props are present. Wire the `changed` event to invalidate the artifacts store and emit a new `release-changed` event from FrontmatterPanel. Propagate this in `ArtifactEditorView.vue` to call `store.invalidate()`.

### Non-functional

**NFR-1: Consistency** — ReleaseDropdown must be visually and behaviourally consistent with StatusDropdown and PriorityDropdown (same animation timing, same keyboard nav, same dark-mode support).

**NFR-2: Performance** — Release list should be fetched once on dropdown open and cached for the component's lifetime (or until the dropdown is re-opened). Do not fetch on every render.

**NFR-3: Accessibility** — Dropdown must have appropriate ARIA attributes (`role="listbox"`, `aria-expanded`, `aria-activedescendant`).

**NFR-4: Optimistic update** — Show the selected release immediately on click; revert if the PATCH fails.

## Acceptance Criteria

- [ ] FrontmatterPanel read-mode field order is: Status, Priority, Release, Type, Stage, Lineage, …
- [ ] Release row is always visible (shows "None" when unset)
- [ ] Clicking the release badge opens a dropdown listing all project releases plus "None"
- [ ] Selecting a release from the dropdown persists the change via PATCH without entering edit mode
- [ ] Selecting "None" clears the release assignment
- [ ] Invalid release name returns 422 from the backend
- [ ] Dropdown supports keyboard navigation (arrows, Enter, Escape)
- [ ] Dropdown closes on outside click
- [ ] Dark mode renders correctly for the ReleaseDropdown
- [ ] `readonly` prop disables the dropdown (no click handler, muted styling)
- [ ] `pnpm exec vue-tsc --noEmit` passes with no new errors
- [ ] `go vet ./...` and `go build ./...` pass with no new errors
- [ ] Related artifacts: [[inline-release-display-edit]]

## Resolved Questions

- Should the release dropdown show release status (e.g., open/closed) or dates alongside the name, or just the release name? Currently the `listReleases` API returns full Release objects with `status`, `start_date`, and `end_date` — we could surface some of this context.

> Yes, just display the status for now in brackets.
