---
title: Inline Status Transition Dropdown on Artefact View
type: requirement
status: planning
lineage: artefact-inline-status-change
created: "2026-05-06T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/artefact-inline-status-change.md
labels:
    - frontend
    - artefacts
    - usability
    - enhancement
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Inline Status Transition Dropdown on Artefact View

## Problem

When a user views an artefact, changing its status requires navigating away from the current view or invoking a separate modal/page action. This two-step flow adds friction to the most frequent editing operation (status transitions) and breaks the user's reading context.

## Goals / Non-goals

### Goals

- Replace the static status badge on the artefact detail view with an interactive dropdown that triggers transitions in-place.
- Only present statuses that are valid transitions from the current state for the current user's roles, enforcing the workflow state machine.
- Provide immediate visual feedback on success or failure without a full page reload.
- Maintain accessibility (keyboard navigation, screen-reader labels, focus management).

### Non-goals

- Batch status changes across multiple artefacts (out of scope).
- Editing other frontmatter fields inline (separate feature).
- Changing the backend transition API — the existing `POST /api/p/:project/artifacts/*/transition` and `GET .../allowed-targets` endpoints are sufficient.
- Supporting custom transition confirmation dialogs or required comments at this stage.

## Detailed Requirements

### Functional

1. **Interactive status badge** — On the artefact detail view, the status field renders as a clickable badge/chip. Clicking it opens a dropdown menu anchored to the badge.
2. **Dynamic option list** — On open, the dropdown fetches valid target statuses from `GET /api/p/:project/artifacts/{path}/allowed-targets`. While loading, a spinner or skeleton state is shown inside the dropdown.
3. **Empty state** — If no transitions are available (terminal status or insufficient role), the badge is rendered as non-interactive (no pointer cursor, no click handler) with a tooltip explaining why (e.g. "No transitions available").
4. **Transition execution** — Selecting a target status sends `POST /api/p/:project/artifacts/{path}/transition` with `{ "to": "<status>" }`. On success the badge updates to the new status with the appropriate colour. On failure an inline error toast or message appears near the badge.
5. **Optimistic UI** — The badge may update optimistically on click; if the request fails it reverts to the previous status and shows the error.
6. **Outside-click dismiss** — Clicking outside the open dropdown closes it without triggering a transition.
7. **WebSocket sync** — If an `artifact.indexed` event arrives for the currently viewed artefact while the dropdown is closed, the badge updates to reflect the new status (another user or agent may have transitioned it).

### Non-functional

1. **Performance** — The allowed-targets request must complete and render the dropdown within 300 ms on a local network under normal load.
2. **Accessibility** — The dropdown must be operable via keyboard (Enter/Space to open, arrow keys to navigate, Escape to close). ARIA roles: `listbox` for the menu, `option` for each item, `aria-expanded` on the trigger.
3. **Responsiveness** — The badge and dropdown must remain usable at viewport widths down to 360 px.
4. **Error resilience** — Network failures on the allowed-targets fetch or the transition POST must not leave the UI in an indeterminate state; always fall back to the last-known status.

## Acceptance Criteria

- [ ] Clicking the status badge on artefact detail view opens a dropdown listing only valid target statuses.
- [ ] Selecting a status from the dropdown transitions the artefact and updates the badge without page navigation.
- [ ] When no valid transitions exist, the badge is visually distinct (non-interactive) and shows a tooltip on hover.
- [ ] A failed transition displays an inline error message and reverts the badge to the previous status.
- [ ] The dropdown is fully keyboard-navigable (open, navigate, select, dismiss).
- [ ] The dropdown closes on outside click or Escape keypress.
- [ ] A real-time WebSocket update to the artefact's status is reflected in the badge when the dropdown is closed.
- [ ] The component passes an accessibility audit (axe-core, no critical or serious violations).
- [ ] Works correctly for users with multiple roles (union of allowed transitions is shown).
- [ ] Works on viewports as narrow as 360 px without overflow or clipping.

## Resolved Questions

- Should the dropdown show the role(s) required for each transition as a hint, or only list reachable statuses?

> If the user is the product-owner, all statuses should be displayed and they can override.  Which is the way the current Change Status works.

- Is a brief confirmation (e.g. "Are you sure?") needed for irreversible terminal transitions like `rejected` or `abandoned`, or is the single-click flow acceptable?

> No confirmation required.  Actions are not terminal, the file is still there.
