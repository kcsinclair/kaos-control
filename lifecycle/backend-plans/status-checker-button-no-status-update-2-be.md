---
title: "Fix Status Check API Response Shape"
type: plan-backend
status: draft
lineage: status-checker-button-no-status-update
parent: lifecycle/defects/status-checker-button-no-status-update.md
---

# Fix Status Check API Response Shape

This plan addresses the backend side of the status checker defect. The `statuscheck.Result.Children` field returns bare path strings, but the frontend contract expects objects with `path` and `status`. Additionally, the advance endpoint response shape uses `outcome`/`new_status` while the frontend expects `ok`/`advanced_to`.

## Milestone 1: Enrich Children Field in Status Check Results

**Description:** Change `statuscheck.Result.Children` from `[]string` to a structured type that includes both the path and the current status of each child artifact. This allows the frontend to display child statuses and perform WebSocket relevance matching.

**Files to change:**
- `internal/statuscheck/statuscheck.go`

**Changes:**
1. Add a `ChildInfo` struct with `Path string` and `Status string` JSON-tagged fields.
2. Change `Result.Children` from `[]string` to `[]ChildInfo`.
3. In `Check()`, populate both `Path` and `Status` when building the children list.

**Acceptance criteria:**
- `statuscheck.Result.Children` serialises as `[{"path": "...", "status": "..."}]`.
- `statuscheck_test.go` passes with updated assertions.
- `go build ./...` and `go vet ./...` pass.

## Milestone 2: Align Advance Endpoint Response with Frontend Contract

**Description:** The `handleStatusCheckAdvance` handler returns `outcome` (string) and `new_status`, but the frontend expects `ok` (boolean) and `advanced_to`. Add these fields to the response (keeping `outcome` and `reason` for backwards compatibility or replacing them outright).

**Files to change:**
- `internal/http/status_check.go`

**Changes:**
1. In the `advanceResult` struct, add an `Ok bool` field (`json:"ok"`) and rename or alias `NewStatus` to also emit `advanced_to`.
2. Set `Ok = true` when `Outcome == "advanced"`, `Ok = false` otherwise.
3. Emit `advanced_to` alongside or instead of `new_status`.

**Acceptance criteria:**
- `POST /api/p/{project}/status-check/advance` response includes `ok: true/false` and `advanced_to` for each result entry.
- Existing integration tests in `tests/integration/status_check_test.go` updated and passing.
- `go build ./...` and `go vet ./...` pass.

## Milestone 3: Add Diagnostic Logging to Status Check Flow

**Description:** The defect notes there is no visibility into why status propagation fails. Add structured `slog` logging at key decision points in the status check and advance handlers so operators can diagnose issues from server logs.

**Files to change:**
- `internal/http/status_check.go`
- `internal/statuscheck/statuscheck.go` (optional, for algorithm-level debug)

**Changes:**
1. Log at `slog.Debug` level when: a lineage is evaluated, when an artifact is determined stale, when an advance is skipped/blocked/errored.
2. Log at `slog.Info` level when an advance succeeds (includes path, from-status, to-status).

**Acceptance criteria:**
- Running with `LOG_LEVEL=debug` produces visible log lines during a status-check and advance flow.
- No logging at `Info` or higher during the GET (read-only) path — only at `Debug`.
- `go build ./...` and `go vet ./...` pass.

## Cross-references

- [[status-checker-button-no-status-update]] (frontend plan): Must update TypeScript types to match the new response shapes.
- [[status-checker-button-no-status-update]] (test plan): Integration tests validate the new JSON contract end-to-end.
