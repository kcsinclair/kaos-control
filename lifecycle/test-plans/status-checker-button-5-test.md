---
title: 'Test Plan: Lineage Status Checker'
type: plan-test
status: approved
lineage: status-checker-button
parent: lifecycle/requirements/status-checker-button-2.md
---

## Overview

Integration and unit tests covering the lineage status checker feature end-to-end: the staleness detection algorithm, the REST API endpoints, the batch advance logic, and the frontend interactions.

---

## Milestone 1: Unit Tests for Staleness Detection Algorithm

### Description

Test the core staleness logic in `internal/statuscheck` with various lineage topologies.

### Files to Change

- `internal/statuscheck/statuscheck_test.go` (new, alongside the package)

### Acceptance Criteria

- **Test: single-artifact lineage** — lineage with only one artifact reports no staleness.
- **Test: all children advanced** — parent at `draft`, all children at `planning` → parent is stale, suggested status = `planning`.
- **Test: mixed children** — parent at `draft`, one child at `planning`, another at `clarifying` → parent is NOT stale (not all children have advanced past `draft`; `clarifying` is only one step ahead — but all must be past parent's status, and `clarifying > draft` so actually stale to `clarifying`). Verify the minimum child status is chosen as the target.
- **Test: terminal children excluded** — parent at `draft`, one child `rejected`, one child at `planning` → terminal child excluded, remaining child at `planning` makes parent stale.
- **Test: all children terminal** — parent at `draft`, all children in terminal statuses → no staleness (no actively progressing children to compare against).
- **Test: furthest valid status** — parent at `draft`, all children at `in-development` → suggested status = `in-development` (skip intermediate statuses).
- **Test: parent already at correct status** — parent at `planning`, children at `planning` → no staleness.
- **Test: deeply nested lineage** — 4+ artifacts in a chain; verify staleness detected at multiple levels.

---

## Milestone 2: Integration Tests for REST API

### Description

Test the `GET /api/p/{project}/status-check` endpoint with real indexed artifacts.

### Files to Change

- `tests/status_check_test.go` (new)

### Acceptance Criteria

- **Test: single lineage check** — create 3 artifacts in one lineage with stale parent, call `GET /status-check?lineage=slug`, verify response matches expected stale artifacts.
- **Test: project-wide check** — create artifacts across 3 lineages (2 with staleness, 1 without), call `GET /status-check`, verify all stale artifacts returned.
- **Test: no staleness** — all artifacts up to date, response has empty `stale` array.
- **Test: can_advance permissions** — call endpoint as user with limited roles, verify `can_advance: false` and `blocked_reason` present for transitions requiring roles the user lacks.
- **Test: performance** — seed 1 000 artifacts, verify response time < 500 ms.

---

## Milestone 3: Integration Tests for Batch Advance

### Description

Test the `POST /api/p/{project}/status-check/advance` endpoint.

### Files to Change

- `tests/status_check_test.go` — additional test functions

### Acceptance Criteria

- **Test: advance single artifact** — post one path, verify artifact status updated on disk and in index.
- **Test: advance multiple artifacts sequentially** — post 3 paths where order matters (later artifacts depend on earlier transitions), verify all updated correctly.
- **Test: permission denied** — attempt advance on artifact requiring a role the user lacks, verify it's skipped with error in response.
- **Test: idempotency** — advance an artifact that is already at the correct status, verify no error and no disk write.
- **Test: WebSocket event** — connect WS client, advance an artifact, verify `artifact.indexed` event received.
- **Test: re-evaluation** — ensure the advance endpoint re-evaluates staleness at execution time rather than trusting the original suggestion (e.g. if another client fixed it first).

---

## Milestone 4: Frontend Component Tests

### Description

Test the UI components and interactions for the status checker panel.

### Files to Change

- `web/src/components/artifact/__tests__/StatusCheckPanel.spec.ts` (new)

### Acceptance Criteria

- **Test: renders stale artifacts** — mock API response with 2 stale artifacts, verify both rendered with correct current/suggested status badges.
- **Test: empty state** — mock empty response, verify "No stale statuses found" message displayed.
- **Test: advance button calls API** — click "Advance" on a stale artifact, verify `advanceStatuses` called with correct path.
- **Test: disabled advance** — stale artifact with `can_advance: false`, verify button disabled and tooltip shows `blocked_reason`.
- **Test: Fix all** — click "Fix all", verify all advanceable paths sent to the API.
- **Test: loading state** — verify spinner shown while API call in flight.
- **Test: panel refresh after advance** — after successful advance, verify `checkStatus` re-called to refresh results.

---

## Milestone 5: End-to-End Scenario Tests

### Description

Full integration tests exercising the feature from button click through to disk state change.

### Files to Change

- `tests/status_check_e2e_test.go` (new)

### Acceptance Criteria

- **Test: full flow** — create a stale lineage via API, call status-check, advance all, verify final disk state has correct statuses in frontmatter.
- **Test: concurrent clients** — two clients call advance on the same artifact simultaneously; verify only one transition occurs and neither errors.
- **Test: lineage with no children** — call check on single-artifact lineage, verify empty result.
- **Test: checker ignores terminal statuses** — lineage where parent is `rejected`, verify not included in stale results even if children have advanced.
