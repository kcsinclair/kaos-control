---
title: "Backend Plan: Test Artifact Status Lifecycle"
type: plan-backend
status: done
lineage: test-artifact-status-lifecycle
parent: requirements/test-artifact-status-lifecycle-2.md
---

# Backend Plan: Test Artifact Status Lifecycle

This plan implements the backend changes for the [[test-artifact-status-lifecycle]] feature — a cyclical `approved → in-qa → approved` status lifecycle for test artifacts, with defect traceability, concurrent-run guards, and crash recovery.

## Milestone 1: Add Type-Aware Workflow Transitions

### Description

Extend the workflow engine to support type-conditional transition rules. Currently, `CanTransition` checks only `(from, to, roles)` — it has no concept of artifact type. Two new transitions are needed:

- `approved → in-qa` for role `qa`, restricted to `type: test`
- `in-qa → approved` for role `system`, restricted to `type: test`

The existing `in-development → in-qa` and `in-qa → approved` (for `qa` role) rules must remain unaffected for non-test artifact types.

### Files to change

- `internal/workflow/workflow.go` — Add an optional `Types []string` field to the `rule` struct. Update `CanTransition` and `AllowedTargets` to accept an artifact type parameter and filter rules by type when `Types` is non-empty. Add the two new default rules with `Types: []string{"test"}`.
- `internal/config/config.go` — Add `Types []string` to `config.Transition` so projects can define type-scoped transitions in `config.yaml`.

### Acceptance criteria

- `CanTransition("approved", "in-qa", ["qa"], "test")` returns `true`.
- `CanTransition("approved", "in-qa", ["qa"], "requirement")` returns `false`.
- `CanTransition("in-qa", "approved", ["system"], "test")` returns `true`.
- `CanTransition("in-qa", "approved", ["system"], "requirement")` returns `false`.
- Existing `CanTransition("in-qa", "approved", ["qa"], "requirement")` still returns `true` (unchanged rule).
- Existing `CanTransition("in-development", "in-qa", [...], "requirement")` still returns `true`.
- `AllowedTargets` reflects type-scoped rules correctly.

## Milestone 2: Wire Type Into HTTP Transition Endpoint

### Description

The `handleTransitionArtifact` handler (`internal/http/transition.go`) currently calls `CanTransition(from, to, roles)` without passing artifact type. Update it to pass the artifact's type so the new type-conditional rules are enforced. Similarly update `handleAllowedTargets`.

### Files to change

- `internal/http/transition.go` — Pass `row.Type` to `CanTransition` and `AllowedTargets` calls. The artifact row is already fetched from the index before validation.

### Acceptance criteria

- `POST /api/p/:project/artifacts/tests/foo-6.md/transition` with `{ "to": "in-qa" }` from a user with `qa` role succeeds for a `type: test` artifact in `approved` status.
- The same request for a `type: requirement` artifact in `approved` status is rejected with 403.
- `GET /api/p/:project/artifacts/tests/foo-6.md/allowed-targets` returns `in-qa` for a test artifact in `approved` status when the user has the `qa` role.

## Milestone 3: Agent Runner Pre-Run and Post-Run Transitions for Test Artifacts

### Description

Update `Manager.StartRun` and `Manager.supervise` to implement the test lifecycle loop:

1. **Pre-run**: When the target artifact is `type: test` and the agent's `active_status` is `in-qa`, validate that the artifact is currently in `approved` status before transitioning to `in-qa`. Use the workflow engine's `CanTransition` (with type) rather than the raw `setArtifactStatus` bypass.

2. **Post-run on success**: When the agent run completes with exit code 0 and the target is `type: test`, automatically transition the artifact back to `approved` (using role `system`). This replaces the generic `DoneOnSuccess` behaviour for test artifacts.

3. **Post-run on failure**: When the agent run fails (non-zero exit), leave the test artifact in `in-qa` status for manual triage (per resolved question #1 in the requirement).

### Files to change

- `internal/agent/agent.go` — In `StartRun` (around line 415), before calling `setArtifactStatus`, look up the target artifact's type from the index. If `type == "test"`, call `CanTransition` to validate `approved → in-qa` for the agent's role. In `supervise` (around line 526), if the target is `type: test` and exit status is `"done"`, transition back to `approved` using role `system` instead of setting status to `"done"`.
- `internal/agent/agent.go` — Extract the artifact type lookup into a helper so both pre-run and post-run can use it.

### Acceptance criteria

- Starting a QA agent run against a `type: test` artifact in `approved` status transitions it to `in-qa` before the agent is invoked.
- Starting a QA agent run against a `type: test` artifact in `draft` status is rejected (transition not allowed).
- On successful QA run completion, the test artifact is automatically set back to `approved`.
- On failed QA run (non-zero exit), the test artifact remains in `in-qa`.
- The pre-run and post-run transitions are committed atomically (NFR-1).
- Non-test artifacts continue to use the existing `ActiveStatus`/`DoneOnSuccess` behaviour unchanged.

## Milestone 4: Concurrent Run Guard for Test Artifacts

### Description

If a test artifact is already in `in-qa` status, a second QA agent run against the same artifact must be rejected. The existing lineage lock mechanism already prevents concurrent runs on the same lineage, so this is largely covered. Add an explicit status check as a defence-in-depth measure: before acquiring the lineage lock, verify the target test artifact is not already in `in-qa`.

### Files to change

- `internal/agent/agent.go` — In `StartRun`, after fetching the artifact row from the index (around line 341), if the artifact is `type: test` and `status == "in-qa"`, return a descriptive error: `"test artifact is already in-qa; another QA run may be active"`.

### Acceptance criteria

- Attempting to start a QA run on a test artifact already in `in-qa` returns a clear error message.
- The lineage lock continues to provide the primary concurrency guard for all artifact types.
- Non-test artifacts are unaffected by this check.

## Milestone 5: Defect-to-Test Traceability

### Description

Ensure defect artifacts raised during a QA run against a test artifact include a `related_to` field referencing the test artifact path. The QA agent prompt template already specifies `parent` pointing to the failing feature, but `related_to` is a separate field for cross-referencing.

### Files to change

- `internal/agent/agent.go` — When building the `Run` struct for a QA agent targeting a test artifact, include the test artifact path as context available to the agent prompt. Add a `RelatedTestPath` field to `Run`.
- `lifecycle/config.yaml` — Update the QA agent prompt template to instruct the agent to include `related_to: [<test_artifact_path>]` in every defect it raises. The `{related_test}` placeholder will be populated from `Run.RelatedTestPath`.
- `internal/artifact/artifact.go` — Ensure `related_to` is a recognised frontmatter field (it may already be passthrough; verify and add to parsed fields if needed).

### Acceptance criteria

- Defects raised by the QA agent during a run against `tests/foo-6.md` include `related_to: [tests/foo-6.md]` in their frontmatter.
- The `related_to` field is preserved through parse/write round-trips.
- Defects raised against non-test artifacts are unaffected.

## Milestone 6: Crash Recovery — Orphaned `in-qa` Detection

### Description

If the kaos-control process crashes mid-QA-run, test artifacts will be left in `in-qa` with no active agent run. On startup, detect these orphans and reset them to `approved`.

### Files to change

- `internal/agent/agent.go` — Add a `RecoverOrphanedTests(idx *index.Index) error` function. On startup, query the index for all artifacts with `type = "test" AND status = "in-qa"`. For each, check whether an active agent run exists (via the `agent_runs` table with `status = "running"`). If no active run exists, patch the artifact status back to `approved`, commit, and log a warning.
- `cmd/kaos-control/main.go` — Call `RecoverOrphanedTests` after index initialisation but before the HTTP server starts accepting requests.

### Acceptance criteria

- On startup, any `type: test` artifact in `in-qa` with no corresponding running agent run is reset to `approved`.
- A warning is logged for each recovered artifact, including the artifact path.
- Artifacts that are legitimately in `in-qa` (active agent run exists) are not reset.
- The recovery commits are atomic and include a descriptive commit message.

## Milestone 7: Stale `in-qa` Warning (60-Minute Threshold)

### Description

Per resolved question #3 in the requirement, if a test artifact has been in `in-qa` for over 60 minutes, generate a UI warning. Implement this as a periodic check in the lock reaper or a similar background goroutine that broadcasts a WebSocket event.

### Files to change

- `internal/agent/agent.go` or `internal/lock/lock.go` — Add a periodic check (piggyback on the existing 60-second reaper tick) that queries `type = "test" AND status = "in-qa"` artifacts and compares their `mtime` to `now`. If `> 60 minutes`, broadcast a `test.stale` WebSocket event with the artifact path and duration.
- `internal/hub/hub.go` — Register the `test.stale` event type if needed (or use a generic event shape).

### Acceptance criteria

- When a test artifact has been in `in-qa` for over 60 minutes, a `test.stale` WebSocket event is broadcast.
- The event includes the artifact path and how long it has been in `in-qa`.
- The event is broadcast at most once per reaper cycle (no spam).
- The [[test-artifact-status-lifecycle]] frontend plan handles rendering this warning.
