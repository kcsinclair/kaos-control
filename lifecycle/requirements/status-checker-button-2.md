---
title: Lineage Status Checker
type: requirement
status: approved
lineage: status-checker-button
parent: lifecycle/ideas/status-checker-button.md
assignees:
    - role: product-owner
      who: agent
---

## Problem

Artifact statuses in a lineage can drift out of sync with actual progress. A parent idea may still read `clarifying` while its child requirement has moved to `planning` and all three plans are `done`. Today, the only way to detect and fix this is manual inspection of each artifact in the lineage — tedious for small lineages, impractical for large projects with dozens of active lineages.

## Goals / Non-goals

### Goals

- Let a user inspect a single lineage (or all lineages in a project) and instantly see which artifact statuses are stale relative to the progress of their descendants.
- Offer one-click batch advancement of stale artifacts to the correct status, subject to existing workflow transition rules and role permissions.
- Surface results in both the artifact detail panel and the graph view.

### Non-goals

- Changing the workflow state machine or adding new statuses.
- Automatically running the checker on a schedule or in the background (manual trigger only for v1).
- Resolving conflicts where multiple valid target statuses exist — present the options and let the user choose.

## Detailed Requirements

### Functional

1. **Staleness detection algorithm.** Given a lineage, walk all artifacts from leaves to root. An artifact is *stale* if every one of its direct children has advanced past the parent's current status according to the transition order: `draft → clarifying → planning → in-development → in-qa → approved → done`. Terminal statuses (`rejected`, `abandoned`, `blocked`) are excluded from the comparison — a stale check only considers actively progressing children.

2. **Single-lineage check.** A "Check status" button on the artifact detail panel triggers the checker for the selected artifact's lineage. The button must be visible on every artifact type.

3. **Project-wide check.** A "Check all statuses" action accessible from the project toolbar (or graph view controls) runs the checker across every lineage in the project and aggregates results.

4. **Results summary panel.** Display a list of stale artifacts grouped by lineage, showing:
   - Artifact title, current status, and suggested target status.
   - The child artifact(s) whose progress makes the parent stale.
   - A per-artifact "Advance" action and a "Fix all" batch action.

5. **Advance action.** When the user clicks "Advance" (single) or "Fix all" (batch):
   - The system must validate each proposed transition against the workflow engine (`Engine.CanTransition`) using the current user's roles.
   - Transitions that the user is not authorised to perform must be shown but disabled, with a tooltip explaining the required role.
   - Permitted transitions are applied via the existing `PUT /artifacts/{path}` endpoint, updating frontmatter status on disk and re-indexing.

6. **WebSocket notification.** Each status change triggered by the checker must broadcast the standard `artifact.indexed` event so other connected clients see the update in real time.

7. **Backend API endpoint.** Expose `GET /api/status-check?lineage={slug}` (single lineage) and `GET /api/status-check` (all lineages). Response schema:

   ```json
   {
     "stale": [
       {
         "path": "lifecycle/ideas/foo.md",
         "lineage": "foo",
         "current_status": "clarifying",
         "suggested_status": "planning",
         "reason": "All children have advanced past clarifying",
         "children": [
           {"path": "lifecycle/requirements/foo-2.md", "status": "planning"}
         ],
         "can_advance": true
       }
     ]
   }
   ```

   When `can_advance` is `false`, include a `blocked_reason` field (e.g. `"requires role: approver"`).

### Non-functional

1. **Performance.** The project-wide check must complete within 500 ms for projects with up to 1 000 artifacts. Use the SQLite index, not disk reads.
2. **Idempotency.** Running the checker twice with no intervening changes must produce identical results and no side effects.
3. **Atomicity.** "Fix all" must apply changes sequentially (not in parallel) so each transition sees the updated state of previously fixed artifacts in the same batch.

## Acceptance Criteria

- [ ] A "Check status" button is present on the artifact detail panel for every artifact type.
- [ ] Clicking "Check status" on an artifact in a stale lineage displays the results summary panel listing all stale artifacts in that lineage.
- [ ] Clicking "Check status" on a lineage with no staleness shows a "No stale statuses found" message.
- [ ] A "Check all statuses" action is available from the project toolbar or graph view controls.
- [ ] The project-wide check returns stale artifacts across all lineages.
- [ ] The "Advance" button transitions a single stale artifact to its suggested status and the UI updates in real time.
- [ ] The "Fix all" button transitions all advanceable stale artifacts in the results, processing them sequentially.
- [ ] Transitions that violate workflow rules or role permissions are shown as disabled with an explanatory tooltip.
- [ ] `GET /api/status-check?lineage={slug}` returns the correct stale artifacts for a given lineage.
- [ ] `GET /api/status-check` returns stale artifacts across all lineages.
- [ ] The checker correctly ignores artifacts in terminal statuses (`rejected`, `abandoned`, `blocked`).
- [ ] The checker handles lineages with no children (single-artifact lineage) and returns no staleness.
- [ ] WebSocket `artifact.indexed` events fire for each status change made by the checker.
- [ ] Project-wide check completes within 500 ms for 1 000 indexed artifacts.

## Open Questions

1. Should the checker consider the `parent:` frontmatter field to build the lineage tree, or should it rely solely on `lineage:` slug grouping combined with stage ordering? The former respects explicit parentage; the latter is simpler but may miss non-linear lineage structures.

> Lets go with lineage for now.

2. When a stale artifact could advance through multiple statuses in one step (e.g. `draft` → `planning` skipping `clarifying`), should the checker suggest the furthest valid status or advance one step at a time?

> furthest valid status

3. Should the results panel persist (e.g. as a sidebar or modal) or appear as a transient notification/toast?

> The results panel should persist.
