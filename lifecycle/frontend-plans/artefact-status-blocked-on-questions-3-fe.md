---
title: "Frontend: Show blocked-on-questions state and auto-assign feedback"
type: plan-frontend
status: done
lineage: artefact-status-blocked-on-questions
parent: lifecycle/defects/artefact-status-blocked-on-questions.md
---

# Frontend: Show blocked-on-questions state and auto-assign feedback

When the backend auto-transitions an artifact to `blocked` due to open questions (see [[artefact-status-blocked-on-questions]]), the frontend must reflect this change accurately in the editor, detail view, and kanban board — giving the user clear feedback that the save triggered a status change and that the artifact is now awaiting product-owner input.

## Milestone 1 — Handle status override in save response

### Description

After the editor saves an artifact, the backend may return a different status than the one the user submitted (because the backend auto-blocked it). The frontend must detect this and update its local state accordingly, showing a toast notification explaining what happened.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue` — in the `save()` function (~line 112-122), after successful API response, compare the returned `artifact.status` against the submitted `editFrontmatter.status`. If they differ and the returned status is `"blocked"`, show an informational toast.

### Acceptance criteria

- When the backend returns `status: "blocked"` but the user submitted a different status, an info-level toast appears: "Status changed to blocked — open questions require product-owner review."
- The editor's frontmatter state updates to reflect the actual returned status and assignees.
- When the backend returns the same status as submitted, no extra toast appears (existing success toast is sufficient).

## Milestone 2 — Blocked-on-questions indicator in artifact detail view

### Description

In the read-only artifact detail view, when the artifact is `blocked` and the body contains an `## Open Questions` section, display a prominent banner below the frontmatter summary directing attention to the questions.

### Files to change

- `web/src/views/project/ArtifactDetailView.vue` — add a conditional alert/banner component.

### Acceptance criteria

- A warning-styled banner appears when `artifact.status === "blocked"` and the body contains `## Open Questions`.
- The banner text reads: "This artifact is blocked pending answers to open questions below."
- The banner does not appear for non-blocked artifacts or blocked artifacts without open questions.
- The banner links/scrolls to the `## Open Questions` heading in the rendered body (anchor link).

## Milestone 3 — Kanban card blocked-on-questions badge

### Description

On the kanban board, cards in the "Blocked" column that are blocked due to open questions should display a small visual indicator (e.g., a question-mark icon from lucide-vue-next) to distinguish them from other blocked reasons.

### Files to change

- `web/src/components/kanban/KanbanCard.vue` — add conditional icon rendering based on the `blocked_reason` field from the WebSocket event or the artifact's body content.

### Acceptance criteria

- Kanban cards for artifacts with `status: "blocked"` that have an `## Open Questions` section in their body show a `HelpCircle` (or equivalent) lucide icon.
- Cards for non-blocked artifacts or blocked artifacts without open questions do not show the icon.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
