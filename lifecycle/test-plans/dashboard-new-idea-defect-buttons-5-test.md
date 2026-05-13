---
title: "Test Plan — Dashboard New Idea & Defect Buttons"
type: plan-test
status: done
lineage: dashboard-new-idea-defect-buttons
parent: lifecycle/requirements/dashboard-new-idea-defect-buttons-2.md
created: "2026-05-13T00:00:00+10:00"
---

# Test Plan — Dashboard New Idea & Defect Buttons

## Summary

Verify the dashboard quick-action buttons and the artifacts page button reordering through end-to-end browser tests. Tests use the existing Vitest + browser-mode infrastructure in `tests/web/`. No backend test changes are needed since no API surface changed.

---

## Milestone 1: Dashboard Button Presence and Layout

### Description
Write an e2e test that navigates to the dashboard and asserts the two quick-action buttons exist with correct attributes.

### Files to change
- `tests/web/dashboard-new-idea-defect-buttons.test.ts` (new file)

### Acceptance Criteria
- [ ] Test asserts a `.btn-new-idea` button is present in `.dashboard-header` with a `MessageSquarePlus` icon (SVG or `lucide` class).
- [ ] Test asserts a `.btn-new-defect` button is present in `.dashboard-header` with a `Bug` icon.
- [ ] Test asserts the "New Defect" button appears before the "New Idea" button in DOM order (left-to-right per FR-4).
- [ ] Test asserts both buttons are right-aligned within the header (e.g., parent has `display: flex` and buttons are after auto-margin).

---

## Milestone 2: Dashboard Modal Integration — Idea Flow

### Description
Test the full happy-path for creating an idea from the dashboard: click "New Idea", verify the BrainDumpModal opens in idea mode, complete the flow (or mock the API response), and assert navigation to the new artifact's detail page.

### Files to change
- `tests/web/dashboard-new-idea-defect-buttons.test.ts`

### Acceptance Criteria
- [ ] Clicking "New Idea" causes `BrainDumpModal` to appear (assert modal element visible in DOM).
- [ ] The modal's `artifactType` is `"idea"`.
- [ ] After the `created` event fires, the route changes to `/p/{project}/artifacts/{path}`.
- [ ] A success toast notification is displayed.

---

## Milestone 3: Dashboard Modal Integration — Defect Flow

### Description
Same as Milestone 2 but for the "New Defect" button, verifying the modal opens in defect mode.

### Files to change
- `tests/web/dashboard-new-idea-defect-buttons.test.ts`

### Acceptance Criteria
- [ ] Clicking "New Defect" causes `BrainDumpModal` to appear.
- [ ] The modal's `artifactType` is `"defect"`.
- [ ] After the `created` event fires, the route changes to the new defect's detail page.

---

## Milestone 4: Modal Dismiss and Focus Return

### Description
Test that dismissing the modal (via Escape key or cancel action) hides the modal and returns focus to the button that triggered it.

### Files to change
- `tests/web/dashboard-new-idea-defect-buttons.test.ts`

### Acceptance Criteria
- [ ] After opening via "New Idea" and pressing Escape, the modal is removed from the DOM.
- [ ] `document.activeElement` is the "New Idea" button after dismiss.
- [ ] After opening via "New Defect" and pressing Escape, focus returns to the "New Defect" button.

---

## Milestone 5: Artifacts Page Button Reordering

### Description
Test that the artifacts list page now renders "New Idea" before "New Defect" in DOM order.

### Files to change
- `tests/web/dashboard-new-idea-defect-buttons.test.ts`

### Acceptance Criteria
- [ ] On the artifacts list page, the `.btn-new-idea` button precedes `.btn-new-defect` in DOM order.
- [ ] Both buttons still function correctly (clicking each opens the correct modal mode).
- [ ] Existing `btn-check-status` button (if present) is unaffected.

---

## Milestone 6: Test Artifact

### Description
Create the corresponding lifecycle test artifact documenting what the test file covers.

### Files to change
- `lifecycle/tests/test-dashboard-new-idea-defect-buttons-e2e.md` (new file)

### Acceptance Criteria
- [ ] Artifact has correct frontmatter (`type: test`, `lineage: dashboard-new-idea-defect-buttons`).
- [ ] Body describes the scenarios covered and maps them back to the requirement's acceptance criteria.

---

## Cross-references

- [[dashboard-new-idea-defect-buttons]] (frontend plan): defines the implementation this test plan validates.
- [[dashboard-new-idea-defect-buttons]] (backend plan): confirms no backend test changes required.
