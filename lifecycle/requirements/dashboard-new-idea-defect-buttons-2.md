---
title: New Idea & New Defect Quick-Action Buttons on Dashboard and Button Reordering on Artifacts Page
type: requirement
status: planning
lineage: dashboard-new-idea-defect-buttons
created: "2026-05-13"
priority: normal
parent: lifecycle/ideas/dashboard-new-idea-defect-buttons.md
labels:
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

# New Idea & New Defect Quick-Action Buttons on Dashboard and Button Reordering on Artifacts Page

## Problem

The dashboard currently has no quick-action buttons. Users who want to capture an idea or log a defect must first navigate to the artifacts list page, locate the buttons there, and then begin the creation flow. This adds unnecessary friction to the two most common entry-point actions in the lifecycle workflow (creating ideas and logging defects). Additionally, on the artifacts page the "New Defect" button appears before "New Idea", which does not reflect the logical priority of ideas as the primary input to the lifecycle.

## Goals / Non-goals

### Goals

- Allow users to create ideas and defects directly from the dashboard without navigating away first.
- Reorder the artifact list page buttons so "New Idea" appears first (leftmost) and "New Defect" appears second.
- Reuse the existing `BrainDumpModal` component; no new modal or creation flow is required.

### Non-goals

- Changing the BrainDumpModal component itself (it is already reusable and type-aware).
- Adding quick-action buttons to other pages (sidebar, header, graph views, etc.).
- Changing the API endpoints for idea/defect generation or artifact creation.
- Adding new artifact types to the quick-action flow.

## Detailed Requirements

### Functional

**FR-1: Dashboard "New Idea" button**
Add a "New Idea" button to the dashboard header area (the `.dashboard-header` section of `DashboardView.vue`). The button must:
- Use the `MessageSquarePlus` icon from lucide-vue-next at size 15.
- Use the existing `btn-new-idea` CSS class (accent/primary style).
- On click, open the `BrainDumpModal` with `artifact-type="idea"`.

**FR-2: Dashboard "New Defect" button**
Add a "New Defect" button to the dashboard header area, positioned to the left of the "New Idea" button. The button must:
- Use the `Bug` icon from lucide-vue-next at size 15.
- Use the existing `btn-new-defect` CSS class (ghost style).
- On click, open the `BrainDumpModal` with `artifact-type="defect"`.

**FR-3: Dashboard BrainDumpModal integration**
- Mount `BrainDumpModal` in `DashboardView.vue` with `v-if` conditional rendering, following the same pattern as `ArtifactListView.vue`.
- On the `created` event, navigate the user to the newly created artifact's detail page (`/p/{project}/artifacts/{path}`).
- On the `close` event, dismiss the modal and return focus to the triggering button.

**FR-4: Dashboard button layout**
- The two buttons must be right-aligned within the dashboard header, on the same row as the "Dashboard" heading.
- Button order (left to right): "New Defect", "New Idea" — matching the visual hierarchy where the primary action ("New Idea") is rightmost.

**FR-5: Artifacts page button reordering**
In `ArtifactListView.vue`, swap the DOM order of the two buttons so that "New Idea" appears first (leftmost of the pair) and "New Defect" appears second (rightmost). The button must retain their existing CSS classes and styling so the accent button ("New Idea") visually leads.

### Non-functional

**NFR-1: No new components**
The implementation must reuse the existing `BrainDumpModal` component and `useBrainDumpStore` Pinia store. No new components, stores, or API endpoints are required.

**NFR-2: Consistent styling**
Dashboard buttons must be visually identical to their artifacts-page counterparts (same classes, icon sizes, and spacing conventions).

**NFR-3: Accessibility**
- Buttons must be focusable and operable via keyboard.
- After modal close, focus must return to the button that triggered the modal.

## Acceptance Criteria

- [ ] The dashboard page displays "New Idea" and "New Defect" buttons in the header row, right-aligned.
- [ ] Clicking "New Idea" on the dashboard opens the BrainDumpModal in idea mode.
- [ ] Clicking "New Defect" on the dashboard opens the BrainDumpModal in defect mode.
- [ ] Completing the creation flow from the dashboard navigates to the new artifact's detail page.
- [ ] Dismissing the modal (Escape or cancel) returns focus to the triggering button.
- [ ] On the artifacts list page, "New Idea" appears to the left of "New Defect" in the button group.
- [ ] Both dashboard buttons use the same icon, sizing, and CSS classes as the artifacts page buttons.
- [ ] No changes are made to `BrainDumpModal.vue`, `brainDump.ts`, or any backend API.
- [ ] The feature works in both light and dark themes.

## Resolved Questions

- Should the dashboard buttons be hidden or disabled when the user lacks permission to create artifacts (if role-based access is enforced in a future release)?

> Hidden
