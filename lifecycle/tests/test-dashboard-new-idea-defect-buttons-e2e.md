---
title: "Test — Dashboard New Idea & Defect Buttons E2E"
type: test
status: done
lineage: dashboard-new-idea-defect-buttons
parent: lifecycle/test-plans/dashboard-new-idea-defect-buttons-5-test.md
created: "2026-05-13T00:00:00+10:00"
---

# Test — Dashboard New Idea & Defect Buttons E2E

## Summary

Vitest component tests verifying the dashboard quick-action buttons and the artifacts page button reordering introduced by the `dashboard-new-idea-defect-buttons` feature. All tests live in:

- `tests/web/dashboard-new-idea-defect-buttons.test.ts`

22 tests across 5 describe blocks; all pass against the happy-dom environment used by the project's Vitest suite.

---

## Test file

`tests/web/dashboard-new-idea-defect-buttons.test.ts`

---

## Scenarios covered

### Milestone 1 — Dashboard button presence and layout (5 tests)

- `M1-TC1` — `.btn-new-idea` is present inside `.dashboard-header`
- `M1-TC2` — `.btn-new-defect` is present inside `.dashboard-header`
- `M1-TC3` — `.btn-new-defect` precedes `.btn-new-idea` in DOM order (FR-4: Defect left, Idea right)
- `M1-TC4` — both buttons are children of `.header-actions` (right-aligned container)
- `M1-TC5` — `.header-actions` is inside `.dashboard-header` (structural proxy for right-alignment)

*Note: CSS `display:flex` / `margin-left:auto` layout cannot be asserted in happy-dom. Structural class presence is used as a proxy, consistent with existing test patterns in this repo.*

### Milestone 2 — Dashboard modal integration: idea flow (5 tests)

- `M2-TC1` — Clicking "New Idea" renders `BrainDumpModal` in the DOM
- `M2-TC2` — The modal's `artifactType` prop is `"idea"`
- `M2-TC3` — After the `created` event fires, `router.push` is called with the artifact path
- `M2-TC4` — A success toast (`ui.success('Artifact created!')`) is triggered
- `M2-TC5` — The modal is removed from the DOM after `created` fires

### Milestone 3 — Dashboard modal integration: defect flow (4 tests)

- `M3-TC1` — Clicking "New Defect" renders `BrainDumpModal` in the DOM
- `M3-TC2` — The modal's `artifactType` prop is `"defect"`
- `M3-TC3` — After the `created` event fires, `router.push` navigates to the defect's path
- `M3-TC4` — Only one modal is rendered at a time (switching from idea to defect replaces the modal)

### Milestone 4 — Modal dismiss and focus return (3 tests)

- `M4-TC1` — After opening via "New Idea" and emitting `close`, the modal is removed from the DOM
- `M4-TC2` — `document.activeElement` is the `.btn-new-idea` button after close
- `M4-TC3` — `document.activeElement` is the `.btn-new-defect` button after close

*Tests use `attachTo: document.body` so happy-dom tracks `document.activeElement` correctly.*

### Milestone 5 — ArtifactListView button reordering (5 tests)

- `M5-TC1` — `.btn-new-idea` precedes `.btn-new-defect` in DOM order on the artifacts list page
- `M5-TC2` — Clicking `.btn-new-idea` opens the modal in idea mode
- `M5-TC3` — Clicking `.btn-new-defect` opens the modal in defect mode
- `M5-TC4` — `.btn-check-status` is still present and unaffected
- `M5-TC5` — `.btn-check-status` precedes both new-idea and new-defect buttons in DOM order

---

## Requirement traceability

| Test ID | Requirement AC |
|---------|----------------|
| M1-TC1  | M1: `.btn-new-idea` present in `.dashboard-header` |
| M1-TC2  | M1: `.btn-new-defect` present in `.dashboard-header` |
| M1-TC3  | M1: "New Defect" appears before "New Idea" in DOM order |
| M1-TC4/5| M1: buttons right-aligned within header |
| M2-TC1  | M2: clicking "New Idea" causes `BrainDumpModal` to appear |
| M2-TC2  | M2: modal's `artifactType` is `"idea"` |
| M2-TC3  | M2: route changes to `/p/{project}/artifacts/{path}` after `created` |
| M2-TC4  | M2: success toast notification displayed |
| M2-TC5  | M2: modal removed after navigation |
| M3-TC1  | M3: clicking "New Defect" causes `BrainDumpModal` to appear |
| M3-TC2  | M3: modal's `artifactType` is `"defect"` |
| M3-TC3  | M3: route changes to defect's detail page after `created` |
| M4-TC1  | M4: Escape / close removes modal from DOM |
| M4-TC2  | M4: focus returns to "New Idea" button after dismiss |
| M4-TC3  | M4: focus returns to "New Defect" button after dismiss |
| M5-TC1  | M5: `.btn-new-idea` precedes `.btn-new-defect` on artifacts page |
| M5-TC2  | M5: clicking each button opens correct modal mode |
| M5-TC3  | M5: clicking each button opens correct modal mode |
| M5-TC4/5| M5: `btn-check-status` unaffected |

---

## Implementation notes

- `BrainDumpModal` is stubbed via `vi.mock` to avoid Teleport/async modal complexity. The stub exposes `data-artifact-type` and `data-testid="brain-dump-modal"` for assertions.
- `DashboardGrid` is stubbed to prevent widget dependency loading.
- `vue-router`, `@/stores/ui`, `@/stores/brainDump`, and all API modules are mocked.
- ArtifactListView tests mock `@/api/artifacts`, `@/api/releases`, `@/api/ws`, and `@/composables/useWebSocket` to prevent network I/O.
