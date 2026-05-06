---
title: "Frontend Plan: Test Artifact Status Lifecycle"
type: plan-frontend
status: in-development
lineage: test-artifact-status-lifecycle
parent: requirements/test-artifact-status-lifecycle-2.md
---

# Frontend Plan: Test Artifact Status Lifecycle

This plan implements the frontend changes for the [[test-artifact-status-lifecycle]] feature — making the test artifact `approved → in-qa → approved` lifecycle visible in the UI, fixing a colour gap in the graph modal, and surfacing stale-run warnings.

Most of the frontend already supports `in-qa` as a status. The Kanban board routes it via project config, `StatusDropdown` and `FrontmatterPanel` have `in-qa` CSS, `FrontmatterEditor` includes it in `STATUS_VOCAB`, and `tokens.css` defines badge tokens. The work here is focused on closing gaps and adding the stale-run warning.

## Milestone 1: Fix `in-qa` Colours in ArtifactModal

### Description

The `ArtifactModal.vue` graph quick-view has hardcoded `STATUS_COLORS` and `STATUS_TEXT` maps (lines 118–129) that are missing `in-qa`. This causes test artifacts in `in-qa` to render with a generic grey badge in the graph modal. Add the missing entry.

### Files to change

- `web/src/components/artifact/ArtifactModal.vue` — Add `'in-qa'` to the `STATUS_COLORS` map (use `#ede9fe`) and `STATUS_TEXT` map (use `#6d28d9`) to match `StatusDropdown.vue` styling.

### Acceptance criteria

- Opening a graph node for a test artifact in `in-qa` status shows a purple/violet badge, not grey.
- Light and dark mode both render correctly (the inline colours work in both themes since they match the existing purple palette).
- No other status badges are affected.

## Milestone 2: Status-Change Toast for `in-qa` Transitions

### Description

The `ArtifactEditorView.vue` WebSocket handler (lines 204–219) shows a toast when an artifact transitions to `blocked`. Add a similar notification when an artifact transitions to or from `in-qa`, so users editing or viewing a test artifact see real-time feedback that a QA run has started or completed.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue` — In the `artifact.indexed` handler, add branches for:
  - `newStatus === 'in-qa'` → toast: "QA run started — artifact is now in-qa"
  - `prevStatus === 'in-qa' && newStatus === 'approved'` → toast: "QA run completed — artifact returned to approved"

### Acceptance criteria

- When viewing a test artifact and a QA agent starts a run, an info toast appears indicating the QA run has started.
- When the QA run completes and the artifact returns to `approved`, a success toast appears.
- Toasts do not appear if the user is in edit mode (`editing.value === true`), consistent with existing behaviour.
- No toasts fire for non-`in-qa` transitions (existing behaviour preserved).

## Milestone 3: Stale `in-qa` Warning Banner

### Description

The [[test-artifact-status-lifecycle]] backend plan (Milestone 7) broadcasts a `test.stale` WebSocket event when a test artifact has been in `in-qa` for over 60 minutes. The frontend should surface this as a warning to the user.

Display the warning in two places:
1. **ArtifactEditorView**: If the currently viewed artifact is the stale one, show a warning banner below the toolbar.
2. **KanbanBoardView**: Show a subtle warning indicator (icon or border) on the Kanban card for stale test artifacts.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue` — Add a `useWebSocket` listener for `test.stale`. When the event's artifact path matches the currently viewed artifact, set a reactive `isStale` flag and render a warning banner: "This test has been in-qa for over 60 minutes — the QA run may be stuck."
- `web/src/composables/useKanbanBoard.ts` — Track a `Set<string>` of stale artifact paths, updated via a `test.stale` WS listener. Expose it so `KanbanCard` can check membership.
- `web/src/components/artifact/KanbanCard.vue` — Accept an `isStale` prop. When true, render a small warning icon (lucide `AlertTriangle`) and apply an amber border to the card.

### Acceptance criteria

- A test artifact stuck in `in-qa` for over 60 minutes shows a warning banner in the editor view.
- The same artifact's Kanban card displays a warning icon and amber border.
- The warning clears when the artifact transitions out of `in-qa` (the `artifact.indexed` event resets the stale flag).
- No warnings appear for artifacts not in `in-qa` or under 60 minutes.

## Milestone 4: Verify Kanban Config Includes `in-qa`

### Description

The Kanban board is config-driven — artifacts appear in columns based on the `statuses` array in `lifecycle/config.yaml`. The current config already includes `in-qa` under the "In-Progress" column. Verify this is correct and that test artifacts in `in-qa` appear in the expected column without code changes.

This is a verification milestone, not a code change.

### Files to change

- None (config verification only). If `in-qa` were missing from the Kanban config, the fix would be in `lifecycle/config.yaml` (backend/ops concern, not frontend code).

### Acceptance criteria

- A test artifact transitioned to `in-qa` appears in the "In-Progress" Kanban column.
- Filtering by `in-qa` status in the Kanban filter bar shows only test artifacts currently in QA.
- The "Uncategorised" column does not contain `in-qa` artifacts (they are routed to "In-Progress").
