---
title: "Frontend Plan: Inline Release Display and Editing"
type: plan-frontend
status: in-development
lineage: inline-release-display-edit
parent: lifecycle/requirements/inline-release-display-edit-2.md
---

## Overview

Create a `ReleaseDropdown.vue` component following the `PriorityDropdown.vue` interaction pattern, add a `patchRelease` API function, reorder the FrontmatterPanel fields so Release appears third (after Status and Priority), and wire everything together in `ArtifactEditorView.vue`.

## Milestone 1: API Function — `patchRelease`

**Description:** Add the frontend API function that calls the new backend PATCH endpoint.

**Files to change:**
- `web/src/api/artifacts.ts`: Add `patchRelease(project, path, release)` following the `patchPriority` pattern (lines 70-75).

**Implementation:**
```typescript
export function patchRelease(project: string, path: string, release: string | null) {
  return api.patch<{ artifact: ArtifactRow }>(
    `/p/${encodeURIComponent(project)}/artifacts/${path}/release`,
    { release },
  )
}
```

**Acceptance criteria:**
- [ ] Function exported from `artifacts.ts`.
- [ ] Sends `PATCH` with `{ release: string | null }` body.
- [ ] Returns typed `{ artifact: ArtifactRow }` response.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 2: ReleaseDropdown Component

**Description:** Create `web/src/components/artifact/ReleaseDropdown.vue` modelled on `PriorityDropdown.vue`.

**Files to change:**
- `web/src/components/artifact/ReleaseDropdown.vue` (new file).

**Props:**
- `project: string` (required)
- `path: string` (required)
- `release: string | null` (required)
- `readonly?: boolean` (default `false`)

**Emits:** `changed(newRelease: string | null)`, `error(message: string)`.

**Behaviour:**
1. **Display:** Render the current release name as a clickable badge (same styling as PriorityDropdown). Show "None" in muted text when `release` is null.
2. **Dropdown open:** On click (unless `readonly`), call `listReleases(project)` from `web/src/api/releases.ts`. Cache the result for the component's lifetime — only re-fetch if the dropdown is reopened after being closed. Display each release as `Name (status)` per the resolved question in the requirement.
3. **"None" option:** Render a "None" option at the top of the list to allow clearing the release.
4. **Selection:** On select, apply optimistic update (show selected value immediately), call `patchRelease(project, path, selectedRelease)`. On failure, revert to previous value and emit `error`.
5. **Keyboard navigation:** Arrow keys to navigate options, Enter/Space to select, Escape to close. Match the pattern in PriorityDropdown.
6. **Outside click:** Close dropdown on click outside (use the same `onClickOutside` approach as PriorityDropdown).
7. **ARIA:** `role="listbox"`, `aria-expanded`, `aria-activedescendant` on the active option.
8. **Readonly:** When `readonly` is true, render as a static badge with no click handler and muted styling.
9. **WebSocket sync:** Watch the `release` prop; if it changes externally (e.g. via WS) while the dropdown is closed, update `optimisticRelease`.

**Acceptance criteria:**
- [ ] Clicking the badge opens a dropdown with all project releases plus "None".
- [ ] Each release option shows `Name (status)` format.
- [ ] Selecting a release persists via PATCH and emits `changed`.
- [ ] Selecting "None" sends `null` and clears the assignment.
- [ ] Optimistic update shows immediately; reverts on error.
- [ ] Keyboard navigation works (arrows, Enter, Escape).
- [ ] Outside click closes the dropdown.
- [ ] ARIA attributes are present and correct.
- [ ] `readonly` prop disables interaction.
- [ ] Dark mode renders correctly.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 3: FrontmatterPanel Field Reorder and Integration

**Description:** Reorder the `<dl>` rows in `FrontmatterPanel.vue` and replace the static release text with `ReleaseDropdown`.

**Files to change:**
- `web/src/components/artifact/FrontmatterPanel.vue`

**Changes:**
1. **Reorder fields** so the displayed order is: Status, Priority, Release, Type, Stage, Lineage, then remaining fields in their current relative order.
2. **Remove conditional rendering** of the Release row — always show it (remove the `v-if="artifact.frontmatter.release"` guard).
3. **Replace static `<dd>` text** with `<ReleaseDropdown>` when `project` and `targetPath` props are available. Pass `project`, `path` (targetPath), `release` (from `artifact.frontmatter.release ?? null`), and `readonly`.
4. **Event wiring:** Listen to `changed` on ReleaseDropdown → emit a new `releaseChanged` event from FrontmatterPanel. Listen to `error` → emit existing `error` event.

**Acceptance criteria:**
- [ ] Field order in read mode is: Status, Priority, Release, Type, Stage, Lineage, …
- [ ] Release row is always visible — shows "None" when unset.
- [ ] ReleaseDropdown is rendered (not static text) when project context is present.
- [ ] `releaseChanged` event is emitted when release changes.

## Milestone 4: ArtifactEditorView Wiring

**Description:** Handle the new `releaseChanged` event in the parent view.

**Files to change:**
- `web/src/views/project/ArtifactEditorView.vue`

**Changes:**
- Listen to `@releaseChanged` on `<FrontmatterPanel>` and call `store.invalidate()` (same pattern as `priorityChanged`).

**Acceptance criteria:**
- [ ] Changing release via the dropdown triggers a store invalidation.
- [ ] Artifact detail view reflects the updated release without full page reload.

## Cross-links

- Depends on the [[inline-release-display-edit]] backend plan (PATCH endpoint must exist).
- The [[inline-release-display-edit]] test plan will validate end-to-end behaviour.
