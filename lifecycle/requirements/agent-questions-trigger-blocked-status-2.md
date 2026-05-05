---
title: Auto-transition artifacts to blocked when open questions are detected
type: requirement
status: blocked
lineage: agent-questions-trigger-blocked-status
created: "2026-05-05T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/agent-questions-trigger-blocked-status.md
labels:
    - agent
    - workflow
    - artefacts
    - process
assignees:
    - role: product-owner
      who: agent
---

## Problem

When an agent writes open questions into an artifact (via a `## Open Questions` section), the artifact's status must be manually updated to `blocked`. This creates a window where downstream agents can pick up an artifact that is not fully resolved, leading to wasted agent runs and potentially incorrect outputs built on unresolved assumptions.

The building blocks already exist independently -- `artifact.HasOpenQuestions()` detects the questions section and `blocked` is a valid status with defined transitions -- but nothing couples them together automatically.

## Goals / Non-goals

### Goals

- Automatically transition an artifact's status to `blocked` when the indexer detects a `## Open Questions` section with content.
- Automatically clear the `blocked` status when the `## Open Questions` section is removed or emptied.
- Ensure the transition respects the existing workflow state machine (only transition if the current status permits it).
- Provide an auditable record of automatic transitions (feed events / WebSocket broadcasts).

### Non-goals

- Changing the `HasOpenQuestions` detection heuristic itself (the existing heading-based check is sufficient).
- Adding a `questions:` frontmatter key as an alternative trigger -- body-based detection via `## Open Questions` is the sole mechanism.
- Auto-assigning the `product-owner` role on block -- agents already set `assignees` in frontmatter when they write questions per their prompt templates.
- Blocking based on questions in linked/child artifacts (only the artifact's own body is inspected).

## Detailed Requirements

### Functional

1. **Auto-block on index.** When `index.IndexFile` (or `index.Upsert`) processes an artifact whose parsed body satisfies `artifact.HasOpenQuestions() == true` and whose current status is _not_ already `blocked`, the system must:
   - Validate that the transition `current-status -> blocked` is permitted by the workflow engine for an automated/system actor.
   - If permitted, update the artifact's `status` field to `blocked` **on disk** (rewrite the frontmatter) and re-index.
   - Emit a feed event (type `status_changed` or similar) and broadcast a WebSocket message so the UI reflects the change immediately.

2. **Auto-unblock on question removal.** When `index.IndexFile` processes an artifact whose status is `blocked` and whose body satisfies `HasOpenQuestions() == false`, the system must:
   - Transition the status back to `draft` (the defined unblock target in the workflow matrix).
   - Rewrite frontmatter on disk and re-index.
   - Emit a corresponding feed event and WebSocket broadcast.

3. **Transition authority.** The automatic transition must use a system-level actor (e.g. role `system` or an internal sentinel) that is permitted to execute `any -> blocked` and `blocked -> draft` transitions without requiring a human role. Alternatively, reuse the existing product-owner bypass if simpler, but document the choice.

4. **Idempotency.** If an artifact is already `blocked` and still has open questions, re-indexing must be a no-op (no duplicate events, no disk rewrite).

5. **Watcher integration.** No changes to the watcher itself are required -- it already calls `IndexFile` on every debounced file change. The new logic lives in the indexer/workflow layer.

6. **Startup scan.** The full scan at startup (`index.Scan`) must also apply the auto-block/unblock logic so that artifacts edited while the server was down are corrected on boot.

### Non-functional

1. **Performance.** `HasOpenQuestions` is a linear scan of the body string. This is acceptable for individual file events. The startup scan must not add measurable latency beyond the existing parse time.

2. **Atomicity.** The on-disk frontmatter rewrite and SQLite upsert should be as atomic as practical. Use a write-then-rename pattern for the file update to avoid partial writes.

3. **No circular triggers.** The on-disk rewrite will trigger a second fsnotify event. The indexer must detect that the content hash has not changed (or that the status already matches) and skip re-processing to avoid an infinite loop.

4. **Logging.** Each automatic transition must be logged at INFO level with the artifact path, old status, new status, and reason (`open_questions_detected` / `open_questions_resolved`).

## Acceptance Criteria

- [ ] An artifact whose body gains a `## Open Questions` section with content is automatically transitioned to `blocked` status on disk and in the index within one watcher cycle (~150 ms debounce + processing).
- [ ] An artifact whose `## Open Questions` section is removed or emptied is automatically transitioned from `blocked` back to `draft`.
- [ ] If the workflow matrix does not permit the transition from the current status to `blocked`, no change is made and a warning is logged.
- [ ] Re-indexing an already-`blocked` artifact with open questions produces no disk write and no duplicate events.
- [ ] The on-disk rewrite does not trigger an infinite re-index loop (content-hash or status guard prevents it).
- [ ] On server startup, artifacts with stale status (should be `blocked` but aren't, or are `blocked` but questions were removed offline) are corrected during the initial scan.
- [ ] Feed events and WebSocket broadcasts are emitted for every automatic status change.
- [ ] The feature does not break existing manual status transitions via the API.
- [ ] Unit tests cover: auto-block, auto-unblock, idempotent no-op, disallowed transition, and circular-trigger prevention.
- [ ] Integration test confirms end-to-end: write a file with questions -> observe `blocked` status in API response. See [[agent-questions-trigger-blocked-status]].

## Questions

1. Should the auto-unblock target status be `draft` unconditionally, or should the system remember and restore the pre-block status? The current workflow matrix only defines `blocked -> draft`, so restoring a prior status would require schema/matrix changes. Defaulting to `draft` is simpler and consistent with the existing rules.

> auto-unblock should transition to draft.  It is expected that the product-owner would edit, answer questions, and set the artefact to approved, then run the next step.

2. Should the auto-block logic also set or preserve the `assignees` field (e.g. ensure `product-owner` is assigned)? Currently agents set this themselves when writing questions, but if a human manually adds a `## Open Questions` section, no assignee would be set automatically.

> auto-block for questions should assign to product-owner.
