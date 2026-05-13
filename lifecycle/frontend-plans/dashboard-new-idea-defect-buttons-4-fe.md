---
title: "Frontend Plan — Dashboard New Idea & Defect Buttons"
type: plan-frontend
status: done
lineage: dashboard-new-idea-defect-buttons
parent: lifecycle/requirements/dashboard-new-idea-defect-buttons-2.md
created: "2026-05-13T00:00:00+10:00"
---

# Frontend Plan — Dashboard New Idea & Defect Buttons

## Summary

Add "New Idea" and "New Defect" quick-action buttons to the dashboard header and reorder the same buttons on the artifacts list page. Both sets of buttons open the existing `BrainDumpModal`. No new components, stores, or API calls are introduced.

---

## Milestone 1: Add Buttons and Modal to DashboardView

### Description
Modify `DashboardView.vue` to import `BrainDumpModal`, `useBrainDumpStore`, the `MessageSquarePlus` and `Bug` icons from `lucide-vue-next`, and `useRouter`. Add reactive state (`showBrainDump`, `brainDumpType`, `triggerButtonEl`) to manage modal visibility and focus return. Render two buttons inside the existing `.dashboard-header` section, right-aligned on the same row as the `<h2>`.

### Files to change
- `web/src/views/project/DashboardView.vue`

### Acceptance Criteria
- [ ] A "New Defect" button (`.btn-new-defect`, `Bug` icon at size 15) and a "New Idea" button (`.btn-new-idea`, `MessageSquarePlus` icon at size 15) appear in `.dashboard-header`, right-aligned.
- [ ] Button order left-to-right: "New Defect", "New Idea" (primary action rightmost, per FR-4).
- [ ] The `.dashboard-header` uses `display: flex; align-items: center;` with `margin-left: auto` on the button group to push buttons right.

---

## Milestone 2: Wire BrainDumpModal in DashboardView

### Description
Mount `<BrainDumpModal>` with `v-if="showBrainDump"` below the dashboard content, passing `:project` and `:artifact-type="brainDumpType"`. Implement two handlers following the pattern in `ArtifactListView.vue`:

- `onBrainDumpCreated(path: string)`: close modal, show success toast, navigate to `/p/${project}/artifacts/${path}`.
- `onBrainDumpClose()`: close modal, `nextTick` refocus the button that triggered the modal (tracked via `triggerButtonEl` ref).

Each button's `@click` sets `brainDumpType` to `'idea'` or `'defect'`, records its element ref in `triggerButtonEl`, resets the store, and sets `showBrainDump = true`.

### Files to change
- `web/src/views/project/DashboardView.vue`

### Acceptance Criteria
- [ ] Clicking "New Idea" opens `BrainDumpModal` in idea mode (`artifactType="idea"`).
- [ ] Clicking "New Defect" opens `BrainDumpModal` in defect mode (`artifactType="defect"`).
- [ ] Completing creation navigates to the new artifact's detail page.
- [ ] Dismissing the modal (Escape / cancel) returns focus to the triggering button.
- [ ] The store is reset before each open so no stale state leaks between uses.

---

## Milestone 3: Reorder Buttons on ArtifactListView

### Description
In `ArtifactListView.vue`, swap the DOM order of the "New Idea" and "New Defect" buttons in the `.list-header` so that "New Idea" appears first (leftmost) and "New Defect" appears second (rightmost of the pair). Move `margin-left: auto` from the "New Defect" button to the "New Idea" button (or to a wrapper) so the pair remains right-aligned. Retain all existing CSS classes, icon sizes, refs, and event handlers — only the source order changes.

### Files to change
- `web/src/views/project/ArtifactListView.vue`

### Acceptance Criteria
- [ ] "New Idea" (`.btn-new-idea`) is to the left of "New Defect" (`.btn-new-defect`) in the rendered button group.
- [ ] Both buttons retain their existing styling, icons, and click behaviour.
- [ ] The button group remains right-aligned in the header row.
- [ ] Focus-return after modal close still works correctly (update ref target if needed).

---

## Milestone 4: Theme and Accessibility Verification

### Description
Manually verify both light and dark themes. Confirm keyboard operability: Tab to each button, Enter/Space to activate, Escape to dismiss modal, focus returns to trigger.

### Files to change
_None (verification only)._

### Acceptance Criteria
- [ ] Buttons render correctly in both light and dark themes with no contrast or visibility issues.
- [ ] All buttons are reachable via Tab and activatable via Enter/Space.
- [ ] After modal close, focus is on the button that opened it (verifiable via `document.activeElement`).

---

## Cross-references

- [[dashboard-new-idea-defect-buttons]] (backend plan): confirms no backend changes are needed.
- [[dashboard-new-idea-defect-buttons]] (test plan): integration and visual tests for the new buttons and reordered layout.
