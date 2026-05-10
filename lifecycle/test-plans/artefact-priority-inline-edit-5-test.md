---
title: "Test Plan — Inline Priority Display and Editing"
type: plan-test
status: done
lineage: artefact-priority-inline-edit
parent: lifecycle/requirements/artefact-priority-inline-edit-2.md
---

# Test Plan — Inline Priority Display and Editing

## Overview

Integration tests verifying the inline priority editing feature end-to-end: API behaviour, UI rendering, optimistic updates, error handling, keyboard accessibility, WebSocket sync, and read-only mode. Tests target the `PriorityDropdown` component integrated within `FrontmatterPanel` on the artifact detail view, and the existing `PATCH /api/p/:project/artifacts/:path/priority` backend endpoint.

Tests are written in the project's integration test directory (`tests/`) following existing patterns established by [[artefact-priority-inline-edit]] status inline-edit tests.

---

## Milestone 1 — Backend API tests for PATCH priority

### Description

Verify the PATCH priority endpoint accepts valid and edge-case inputs, returns correct responses, and triggers WebSocket events.

### Files to change

- `tests/` — new test file for priority PATCH endpoint (e.g. `tests/patch-priority.test.ts` or extend existing artifact API test file)

### Acceptance criteria

- [ ] Test: PATCH with `{"priority":"high"}` returns `200` and the response body contains the updated priority value.
- [ ] Test: PATCH with an unknown value (e.g. `{"priority":"critical"}`) returns `200` — backend accepts any string.
- [ ] Test: PATCH with the same priority value already set returns `200` (idempotent, no error).
- [ ] Test: PATCH on a non-existent artifact path returns `404`.
- [ ] Test: After a successful PATCH, reading the artifact file from disk confirms the YAML frontmatter contains the updated `priority` field.
- [ ] Test: After PATCH, a connected WebSocket client receives an `artifact.indexed` event with `"action":"updated"`.

---

## Milestone 2 — UI rendering tests

### Description

Verify the `PriorityDropdown` renders correctly in the `FrontmatterPanel` with proper badge colours and handles missing/unknown priority values.

### Files to change

- `tests/` — new or extended UI test file (e.g. `tests/priority-dropdown-ui.test.ts`)

### Acceptance criteria

- [ ] Test: Artifact detail view displays a "Priority" row in the metadata sidebar.
- [ ] Test: The "Priority" row appears immediately after the "Status" row.
- [ ] Test: Priority badge displays the correct colour for each standard value (`high` = red, `medium` = orange, `normal` = green, `low` = blue).
- [ ] Test: An artifact with no priority set displays `"normal"` with the green badge.
- [ ] Test: An artifact with an unknown priority value (e.g. `"critical"`) displays a neutral/grey badge with the raw string as label.
- [ ] Test: The dropdown lists exactly four options: `high`, `medium`, `normal`, `low`, each with a colour indicator.

---

## Milestone 3 — Interaction and optimistic update tests

### Description

Verify selection behaviour, optimistic updates, error handling, and the no-change guard.

### Files to change

- `tests/` — extend priority UI test file

### Acceptance criteria

- [ ] Test: Clicking the priority badge opens the dropdown; clicking an option updates the badge immediately (before API response).
- [ ] Test: After selecting a new priority, the artifact file on disk reflects the change.
- [ ] Test: Re-selecting the current priority value does not trigger an API call (no-change guard).
- [ ] Test: When the API call fails (e.g. simulated server error), the badge reverts to the previous value and an error indicator is shown.

---

## Milestone 4 — Dismiss and keyboard navigation tests

### Description

Verify dropdown dismiss behaviour and full keyboard accessibility.

### Files to change

- `tests/` — extend priority UI test file

### Acceptance criteria

- [ ] Test: Dropdown closes when an option is selected.
- [ ] Test: Dropdown closes on click outside.
- [ ] Test: Dropdown closes on `Escape` key press.
- [ ] Test: `ArrowDown` key moves focus to the next option; `ArrowUp` moves to the previous.
- [ ] Test: `Enter` key selects the focused option.
- [ ] Test: `Space` key selects the focused option.
- [ ] Test: After `Escape`, focus returns to the trigger button.
- [ ] Test: Dropdown has `role="listbox"`; options have `role="option"` and `aria-selected`.

---

## Milestone 5 — WebSocket sync tests

### Description

Verify that priority changes from external sources (another session, filesystem edit, agent run) update the displayed badge in real time.

### Files to change

- `tests/` — extend priority UI test file or create WebSocket-specific test

### Acceptance criteria

- [ ] Test: With the artifact detail view open, modifying the priority via a direct file write (or second API call) causes the badge to update without a page refresh.
- [ ] Test: If the dropdown is closed when an external update arrives, the badge updates silently.
- [ ] Test: If the dropdown is open when an external update arrives, the dropdown closes and the badge reflects the new value.

---

## Milestone 6 — Read-only mode tests

### Description

Verify that the priority badge is non-interactive when the artifact is locked or the user lacks write access.

### Files to change

- `tests/` — extend priority UI test file

### Acceptance criteria

- [ ] Test: When the artifact is locked by another user, the priority badge is visible but clicking it does not open the dropdown.
- [ ] Test: The non-interactive badge does not have `aria-haspopup` or `tabindex="0"`.
- [ ] Test: When the lock is released (simulated), the dropdown becomes interactive again on the next data refresh.
