---
title: "Frontend Plan — Inline Status Transition Dropdown"
type: plan-frontend
status: approved
lineage: artefact-inline-status-change
parent: lifecycle/requirements/artefact-inline-status-change-2.md
---

# Frontend Plan — Inline Status Transition Dropdown

Replace the static status badge in `FrontmatterPanel` with an interactive dropdown that fetches valid transitions from the backend and executes them in-place. Remove the separate "Change Status" button and `TransitionDialog` modal from the artefact editor topbar — all status transitions happen through the inline badge.

Cross-references: [[artefact-inline-status-change]] (backend plan for endpoint verification), [[artefact-inline-status-change]] (test plan for integration tests).

---

## Milestone 1 — Add `getAllowedTargets` API function

### Description

The backend `GET /api/p/:project/artifacts/{path}/allowed-targets` endpoint exists but the frontend has no client function for it. Add it to the artifacts API module.

### Files to change

- `web/src/api/artifacts.ts` — add `getAllowedTargets(project, path)` returning `Promise<{ targets: string[] }>`

### Acceptance criteria

- [ ] `getAllowedTargets` is exported and calls `api.get<{ targets: string[] }>(\`/p/\${project}/artifacts/\${path}/allowed-targets\`)`.
- [ ] TypeScript compiles with no errors (`pnpm exec vue-tsc --noEmit`).

---

## Milestone 2 — Create `StatusDropdown` component

### Description

Build a new `StatusDropdown.vue` component in `web/src/components/artifact/`. This is the core interactive element that replaces the static badge.

**Behaviour:**

1. Renders as a badge/chip showing the current status with the existing `data-status` colour scheme.
2. On click, fetches allowed targets from the API (showing a loading spinner inside the dropdown while waiting).
3. Displays the list of valid target statuses as a dropdown menu anchored below the badge.
4. If the targets list is empty (no valid transitions), the badge is rendered as non-interactive: no pointer cursor, no click handler, and a `title` tooltip reading "No transitions available".
5. Selecting a target status calls `transitionArtifact` and applies optimistic UI — the badge updates immediately to the new status. On failure, it reverts to the previous status and the parent is notified of the error.
6. Clicking outside or pressing Escape closes the dropdown without triggering a transition.

**Accessibility:**

- The badge trigger has `role="button"`, `aria-haspopup="listbox"`, and `aria-expanded`.
- The menu has `role="listbox"`.
- Each option has `role="option"`.
- Arrow keys navigate options, Enter/Space selects, Escape closes.
- Focus is trapped within the dropdown while open; focus returns to the trigger on close.

**Props:**
- `project: string`
- `path: string`
- `status: string`

**Emits:**
- `transitioned(newStatus: string)` — after a successful transition
- `error(message: string)` — on transition failure

### Files to change

- `web/src/components/artifact/StatusDropdown.vue` — new file

### Acceptance criteria

- [ ] Badge renders with correct colour via `data-status` attribute (reuse existing CSS scheme from `FrontmatterPanel`).
- [ ] Clicking the badge opens a dropdown listing only valid target statuses fetched from the API.
- [ ] A loading spinner/skeleton is shown inside the dropdown while the `allowed-targets` request is in flight.
- [ ] When `targets` is empty, the badge is non-interactive with `cursor: default`, no click handler, and `title="No transitions available"`.
- [ ] Selecting a status fires `transitionArtifact`, updates the badge optimistically, and emits `transitioned` on success.
- [ ] On failure, badge reverts to previous status and emits `error` with the error message.
- [ ] Outside click closes the dropdown.
- [ ] Escape keypress closes the dropdown.
- [ ] Full keyboard navigation: Enter/Space opens, Arrow Up/Down navigates, Enter/Space selects.
- [ ] ARIA attributes: `role="button"` + `aria-haspopup="listbox"` + `aria-expanded` on trigger; `role="listbox"` on menu; `role="option"` on items.
- [ ] Dropdown remains usable at 360px viewport width (no overflow or clipping).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3 — Integrate `StatusDropdown` into `FrontmatterPanel`

### Description

Replace the static `<span class="badge">` in `FrontmatterPanel.vue` with the new `StatusDropdown` component. Wire up the `transitioned` and `error` events.

The `FrontmatterPanel` currently does not receive `project` or `path` props suitable for API calls. Add these as new props (or derive from existing props).

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue` — replace badge markup, add `StatusDropdown` import, extend props with `project` and `artifactPath`
- `web/src/views/project/ArtifactEditorView.vue` — pass `project` and `artifactPath` as new props to `FrontmatterPanel`, wire `transitioned` event to update `artifact.value.status` and call `store.invalidate()`

### Acceptance criteria

- [ ] The status row in `FrontmatterPanel` renders `StatusDropdown` instead of a static `<span>`.
- [ ] `project` and `artifactPath` are passed to `FrontmatterPanel` and forwarded to `StatusDropdown`.
- [ ] A successful transition updates `artifact.value` in `ArtifactEditorView` without a page reload.
- [ ] A failed transition displays an error toast via `ui.error()`.
- [ ] The panel remains visually consistent with its current layout at all viewport widths.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 — Remove `TransitionDialog` and "Change Status" button

### Description

The inline dropdown fully replaces the modal-based transition flow. Remove the "Change Status" button from the topbar and the `TransitionDialog` component usage from `ArtifactEditorView`. Keep the `TransitionDialog.vue` file itself for now (it may be used elsewhere or in future for batch transitions) but remove its import and usage from the editor view.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue` — remove `showTransition` ref, the "Change Status" `<button>`, the `<TransitionDialog>` template block, the `onTransitioned` handler, and the `TransitionDialog` import

### Acceptance criteria

- [ ] The "Change Status" button no longer appears in the topbar.
- [ ] No `TransitionDialog` is rendered from `ArtifactEditorView`.
- [ ] Status transitions work exclusively through the inline `StatusDropdown` in the sidebar.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 — WebSocket live update for status badge

### Description

When an `artifact.indexed` WebSocket event arrives for the currently viewed artefact (and the dropdown is closed), the `StatusDropdown` badge must reflect the new status. The existing `useWebSocket` handler in `ArtifactEditorView` already re-fetches the artifact on `artifact.indexed` events, which updates `artifact.value.status`. Since `StatusDropdown` receives `status` as a prop, the badge will update reactively.

Verify this works end-to-end and add a guard: if the dropdown is currently open when a WS event arrives, do not close or disrupt it — let the user complete their action. On next open, the dropdown will re-fetch targets based on the (potentially changed) status.

### Files to change

- `web/src/components/artifact/StatusDropdown.vue` — watch the `status` prop for external changes; if the dropdown is open and the prop changes, close the dropdown and reset state (another user transitioned it)
- `web/src/views/project/ArtifactEditorView.vue` — verify existing `useWebSocket` handler propagates status changes to the `StatusDropdown` via the `FrontmatterPanel` → `StatusDropdown` prop chain

### Acceptance criteria

- [ ] A WebSocket `artifact.indexed` event with a status change updates the badge when the dropdown is closed.
- [ ] If the dropdown is open and a WS event changes the status, the dropdown closes and the badge shows the new status.
- [ ] No duplicate re-fetches occur (the existing auto-refresh grace period is respected).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 — Complete status colour coverage

### Description

The existing badge CSS in `FrontmatterPanel` is missing explicit colour rules for several statuses: `draft`, `in-development`, `in-qa`, `rejected`, `abandoned`. Add colour rules for all statuses to ensure the badge is always visually distinct. Move the badge colour CSS into the `StatusDropdown` component (or a shared CSS file) so it is self-contained.

### Files to change

- `web/src/components/artifact/StatusDropdown.vue` — add CSS rules for all status values

### Acceptance criteria

- [ ] Every status in the vocabulary (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `done`, `blocked`, `rejected`, `abandoned`) has a distinct background + text colour.
- [ ] Colours are legible in both light and dark mode (use CSS variables or `prefers-color-scheme` media queries).
- [ ] `pnpm build` passes.
