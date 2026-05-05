---
title: "Frontend plan: auto-transition artifacts to blocked on open questions"
type: plan-frontend
status: draft
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/requirements/agent-questions-trigger-blocked-status-2.md
---

## Overview

The frontend already handles `blocked` status display well — status badges are colour-coded, the editor shows a "blocked on questions" banner, and the kanban card has an orange question-mark icon. The backend changes in [[agent-questions-trigger-blocked-status]] (backend plan) do not introduce new WebSocket event types; all views already listen to `artifact.indexed` and `feed.new`. This plan covers the small adjustments needed to ensure the UI behaves correctly when status changes happen server-side without a user-initiated save.

---

## Milestone 1 — Handle server-initiated status changes in the editor

**Description:** When the user is viewing (not editing) an artifact and the backend auto-blocks or auto-unblocks it, the editor view should reflect the change immediately. Currently, `ArtifactEditorView.vue` listens to `artifact.indexed` WebSocket events and reloads the artifact when not in edit mode (lines 180-186). This path already works, but the blocked banner visibility (`status === 'blocked' && hasOpenQuestions`, line 251) depends on the local `hasOpenQuestions` computed property which reads from the loaded artifact body. Verify this reactive chain is intact and the banner appears/disappears without a manual refresh.

**Files to change:**

- `web/src/views/project/ArtifactEditorView.vue` — verify that the `artifact.indexed` handler at lines 180-186 triggers a full artifact reload (not just metadata), ensuring `hasOpenQuestions` is recomputed from the fresh body. Add a brief integration note as a comment if the chain is non-obvious, but no logic change expected.

**Acceptance criteria:**

- [ ] When viewing an artifact (read mode), auto-block by the backend causes the blocked banner to appear within one WebSocket round-trip.
- [ ] When viewing a blocked artifact whose questions are removed externally, the banner disappears on the next `artifact.indexed` event.
- [ ] No flash of stale state — the banner is hidden/shown atomically with the status badge update.

---

## Milestone 2 — Toast notification for server-initiated auto-block

**Description:** The editor currently shows a toast when the user saves and the status changes to `blocked` (line 151-152). Extend this to also show a toast when an `artifact.indexed` event indicates the artifact the user is currently viewing was auto-blocked or auto-unblocked server-side. This gives the user clear feedback that the system acted.

**Files to change:**

- `web/src/views/project/ArtifactEditorView.vue` — in the `artifact.indexed` WebSocket handler, after reloading the artifact, compare the previous status with the new status. If the status changed:
  - To `blocked`: show info toast "Status changed to blocked — open questions detected."
  - From `blocked` to `draft`: show info toast "Blocked status cleared — open questions resolved."

**Acceptance criteria:**

- [ ] A toast appears when the viewed artifact is auto-blocked by the server.
- [ ] A toast appears when the viewed artifact is auto-unblocked by the server.
- [ ] No toast fires if the status did not change (idempotent re-index).
- [ ] No toast fires if the user is editing (not viewing) — edits take precedence.

---

## Milestone 3 — Feed entry rendering for system-initiated transitions

**Description:** The feed already renders `status_transition` events via `FeedEntry.vue`. The backend will emit events with `actor: "system"` for auto-transitions. Verify that the feed entry component renders these correctly — the actor name should display as "system" (or a friendlier label like "Auto") and the existing ArrowRightLeft icon applies.

**Files to change:**

- `web/src/components/feed/FeedEntry.vue` — if the actor field is `"system"`, display it as "System" (capitalised). No icon change needed; `status_transition` already maps to `ArrowRightLeft`.

**Acceptance criteria:**

- [ ] Feed entries for auto-block/unblock show actor "System" rather than a raw internal value.
- [ ] The event summary (e.g. "draft → blocked") is visible.
- [ ] The feed entry links to the affected artifact.

---

## Milestone 4 — Kanban card blocked-on-questions detection

**Description:** `KanbanCard.vue` currently detects "blocked on questions" by checking for a product-owner/agent assignee (lines 35-43). This heuristic works because the backend plan ensures `assignees` always includes `{role: product-owner, who: agent}` on auto-block. Verify this continues to work and the orange HelpCircle icon appears on kanban cards for auto-blocked artifacts.

**Files to change:**

- `web/src/components/artifact/KanbanCard.vue` — no logic change expected. Verify the existing assignee-based detection still matches after the backend moves auto-block into the indexer.

**Acceptance criteria:**

- [ ] Kanban cards for auto-blocked artifacts show the orange HelpCircle icon.
- [ ] Cards for manually-blocked artifacts (without the agent assignee) do not show the icon.
- [ ] The icon disappears when the artifact is auto-unblocked.

---

## Milestone 5 — Conflict handling when editing during auto-block

**Description:** If the user is editing an artifact and the backend auto-blocks it (because another process wrote `## Open Questions` to the file on disk), the `useExternalChange` composable should show the conflict banner. The user should not lose unsaved work. Verify this existing mechanism handles the auto-block rewrite correctly — the atomic rename on disk will trigger `file.changed`, which the composable catches.

**Files to change:**

- `web/src/composables/useExternalChange.ts` — no logic change expected. The existing save-grace-period check (SAVE_GRACE_MS = 3000ms) will correctly distinguish a user save from a server-side auto-block rewrite. Verify with manual testing.

**Acceptance criteria:**

- [ ] If the user is mid-edit and the backend auto-blocks the artifact, a conflict banner appears (not a silent overwrite).
- [ ] The user can choose to reload or keep their changes.
- [ ] If the user is not editing, the auto-refresh path applies as per milestone 1.
