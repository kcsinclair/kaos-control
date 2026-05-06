---
title: Test Artifact Status Lifecycle
type: requirement
status: planning
lineage: test-artifact-status-lifecycle
parent: ideas/test-artifact-status-lifecycle.md
labels:
    - test
    - testing
    - qa
    - workflow
    - artefacts
assignees:
    - role: product-owner
      who: agent
---

# Test Artifact Status Lifecycle

## Problem

Test artifacts (`type: test`) currently follow the same generic workflow state machine as all other artifact types. There is no mechanism for the QA agent to signal that a test is actively running, nor does the system automatically reset a test back to `approved` after a QA run completes. This means:

1. After a QA agent finishes executing an approved test, the test artifact is left in whatever status the agent or supervisor last set — there is no defined "return to ready" transition.
2. There is no `running` (or equivalent) status to indicate a test is currently being executed, so users and agents cannot distinguish between "approved and idle" and "approved and actively running."
3. Defects raised during a QA run are not formally linked back to the test artifact that produced them, making it difficult to trace which test execution surfaced which issues.

Without a well-defined test lifecycle loop, tests can end up in ambiguous states, re-runs require manual status resets, and the defect-to-test traceability is implicit rather than enforced.

## Goals / Non-goals

### Goals

- Define a cyclical status lifecycle for test artifacts: **approved → in-qa → approved**, where `in-qa` represents active execution by the QA agent.
- Ensure the agent runner automatically transitions a test artifact back to `approved` after a QA run completes (regardless of whether defects were raised).
- Ensure defect artifacts raised during a QA run include a `parent` or `related_to` link back to the test artifact that was being executed.
- Make the test lifecycle visible in the UI — users should see when a test is running vs idle.

### Non-goals

- Tracking individual test-case-level pass/fail results within the artifact (out of scope; defect artifacts serve this purpose).
- Adding a `failed` terminal status for tests — the test itself is never "failed"; defects are the mechanism for tracking failures.
- Changing the lifecycle of non-test artifact types.
- Implementing test scheduling or automatic re-run triggers (future work).

## Detailed Requirements

### Functional

**FR-1: `in-qa` status for test artifacts**
When the QA agent begins executing a test artifact, the agent runner MUST transition the test artifact's status from `approved` to `in-qa` before invoking the agent. This transition uses the existing workflow engine and requires the `qa` role.

**FR-2: Automatic post-run transition back to `approved`**
When a QA agent run completes (exit code 0 or non-zero), the agent runner MUST transition the test artifact's status back to `approved`. This applies regardless of whether defects were raised. The transition is performed by the system/supervisor, not the agent itself.

**FR-3: New workflow transition `approved → in-qa` for test artifacts**
Add a workflow rule permitting `approved → in-qa` for the `qa` role. This transition is only valid for artifacts with `type: test`. The existing `in-development → in-qa` rule for other artifact types is unaffected.

**FR-4: New workflow transition `in-qa → approved` for test artifacts**
Add a workflow rule permitting `in-qa → approved` for the `system` role (agent runner / supervisor). This transition is only valid for artifacts with `type: test`. The existing `in-qa → approved` rule for the `qa` role on other artifact types remains unchanged.

**FR-5: Defect-to-test traceability**
Each defect artifact raised during a QA run against a test artifact MUST include a `related_to` entry referencing the test artifact path (e.g., `related_to: [tests/login-6.md]`). The QA agent prompt template already specifies `parent` pointing to the failing test or feature; this requirement ensures the test artifact itself is always referenced.

**FR-6: Guard against concurrent runs**
If a test artifact is already in `in-qa` status, a second QA agent run against the same artifact MUST be rejected by the agent runner with a clear error message. The existing lineage lock mechanism should be leveraged for this.

### Non-functional

**NFR-1: Atomicity**
The pre-run (`approved → in-qa`) and post-run (`in-qa → approved`) transitions MUST each be committed atomically — if the transition write fails, the agent run must not proceed (pre-run) or the failure must be logged and the artifact left in `in-qa` for manual recovery (post-run).

**NFR-2: Crash recovery**
If the agent runner or kaos-control process crashes mid-run, the test artifact will be left in `in-qa`. On startup, the lock reaper (or a similar mechanism) SHOULD detect orphaned `in-qa` test artifacts with no active agent run and reset them to `approved`, logging a warning.

**NFR-3: No schema migration required**
All changes should use existing frontmatter fields and the existing SQLite index schema. No new database columns are needed — `status` and `type` are already indexed.

## Acceptance Criteria

- [ ] A test artifact in `approved` status can be transitioned to `in-qa` by the QA agent runner before execution begins
- [ ] A test artifact in `in-qa` status is automatically transitioned back to `approved` when the QA agent run completes
- [ ] The `approved → in-qa` transition is rejected if the actor does not hold the `qa` role
- [ ] The `in-qa → approved` transition for test artifacts is performed by the system/supervisor role, not the agent
- [ ] Attempting to start a QA run on a test artifact already in `in-qa` status returns an error
- [ ] Defect artifacts raised during a QA run against a test artifact include a `related_to` link to the test artifact
- [ ] The Kanban board and artifact detail view correctly display `in-qa` status for test artifacts during a run
- [ ] If the agent runner crashes mid-run, orphaned `in-qa` test artifacts are detected and reset on next startup
- [ ] Existing non-test artifact workflows (e.g., `in-qa → approved` by QA role for requirements) are unaffected
- [ ] The [[test-artifact-status-lifecycle]] lineage test artifacts cycle through `approved → in-qa → approved` correctly in integration tests

## Resolved Questions

1. Should the `in-qa` → `approved` post-run transition happen even if the agent run fails (non-zero exit)? The idea says "once defect-raising is complete" — but a crashed agent may not have finished raising defects. Should a failed run leave the test in `in-qa` for manual triage, or always reset to `approved`?

> A failed run should remain in in-qa.

2. Should there be a configurable option (in `lifecycle/config.yaml`) to control the post-run reset behaviour per project, or is the always-reset-to-approved behaviour sufficient for v1?

> always reset to approved.

3. The idea mentions the cycle ensures tests are "never left in a terminal or ambiguous state." Should we add a UI warning or notification when a test has been in `in-qa` for longer than the agent timeout, as an early indicator of a stuck run?

> If something has been in-qa for over 60 minutes, generate a UI warning.
