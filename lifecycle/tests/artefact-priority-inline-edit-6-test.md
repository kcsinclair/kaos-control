---
title: "Test Suite — Inline Priority Display and Editing"
type: test
status: draft
lineage: artefact-priority-inline-edit
parent: lifecycle/test-plans/artefact-priority-inline-edit-5-test.md
---

# Test Suite — Inline Priority Display and Editing

Backend integration tests for the inline priority editing feature: REST API behaviour, disk persistence, WebSocket broadcast, and lock-gated read-only enforcement.

## Test files

- `tests/integration/priority_patch_test.go` — core PATCH endpoint (Milestones 1, prior coverage)
- `tests/integration/priority_dropdown_test.go` — PUT-based priority round-trips (Milestone 2, prior coverage)
- `tests/integration/priority_roundtrip_test.go` — full round-trip, concurrent reads (Milestone 1/2, prior coverage)
- `tests/integration/priority_inline_edit_test.go` — gap-filling: idempotency, disk confirmation, WS action field, absent-priority default, external file write broadcast, lock rejection, lock-release recovery (Milestones 1, 2, 5, 6)

## Scenarios covered

### Milestone 1 — Backend API (PATCH /priority)

| Test | File | Scenario |
|------|------|----------|
| `TestPriorityPatchHappyPath` | `priority_patch_test.go` | PATCH `{"priority":"high"}` → 200, response body has updated priority |
| `TestPriorityPatchUnknownValue` | `priority_patch_test.go` | Unknown value (e.g. "critical") accepted, returned as-is |
| `TestPriorityPatchIdempotent` | `priority_inline_edit_test.go` | Same value re-patched → 200, no error |
| `TestPriorityPatchNonExistent` | `priority_patch_test.go` | Non-existent path → 404 |
| `TestPriorityPatchDiskConfirmation` | `priority_inline_edit_test.go` | Disk file contains `priority: high` after PATCH |
| `TestPriorityPatchWebSocketEvent` | `priority_patch_test.go` | PATCH emits `artifact.indexed` WS event with matching path |
| `TestPriorityPatchWebSocketActionField` | `priority_inline_edit_test.go` | `artifact.indexed` event contains `action: "updated"` |

### Milestone 2 — Priority field in API responses (backend-observable)

| Test | File | Scenario |
|------|------|----------|
| `TestPriorityDropdownCreateNormal` | `priority_dropdown_test.go` | Artifact created with `priority: normal` returns "normal" from GET |
| `TestPriorityDropdownUpdateToHigh` | `priority_dropdown_test.go` | PUT with priority "high" persists to disk and API |
| `TestPriorityDropdownUnsetViaEmpty` | `priority_dropdown_test.go` | PUT with `priority: ""` removes the field from disk |
| `TestPriorityDropdownUnknownValueAccepted` | `priority_dropdown_test.go` | Unknown priority stored and returned via PUT and PATCH |
| `TestPriorityAbsentOmittedFromResponse` | `priority_inline_edit_test.go` | No priority in frontmatter → API returns empty/absent priority (frontend maps to "normal") |

> **Note on Milestone 2 UI assertions**: Badge colours, dropdown option rendering, ordering of the "Priority" row relative to "Status", and unknown-value grey badge are frontend-only behaviours not testable at the backend API level. These require browser/E2E test coverage.

### Milestone 3 — Interaction and optimistic updates (backend-observable)

| Test | File | Scenario |
|------|------|----------|
| `TestPriorityPatchHappyPath` | `priority_patch_test.go` | Priority change persists to disk after PATCH (API side of optimistic update) |
| `TestPriorityPatchNonExistent` | `priority_patch_test.go` | PATCH fails 404 → API reflects failure (frontend reverts optimistic update) |

> **Note on Milestone 3 UI assertions**: Optimistic badge update before API response, no-API-call guard when re-selecting current value, and badge revert on API failure are frontend-only behaviours.

### Milestone 4 — Dismiss and keyboard navigation

> All acceptance criteria in Milestone 4 are frontend-only (dropdown dismiss, keyboard focus management, ARIA roles). No backend-observable equivalents exist.

### Milestone 5 — WebSocket sync

| Test | File | Scenario |
|------|------|----------|
| `TestPriorityPatchWebSocketEvent` | `priority_patch_test.go` | PATCH via API → connected WS client receives `artifact.indexed` event |
| `TestPriorityExternalFileWriteBroadcastsUpdate` | `priority_inline_edit_test.go` | Direct disk write with new priority → watcher broadcasts `artifact.indexed`, API returns updated priority |
| `TestPriorityExternalWriteClosedDropdownUpdatesAPI` | `priority_inline_edit_test.go` | External disk write re-indexes artifact; next GET returns updated priority (backend half of silent badge update) |

> **Note on Milestone 5 UI assertions**: "dropdown closes and badge reflects new value" when an external update arrives while the dropdown is open is a frontend-only behaviour.

### Milestone 6 — Read-only mode (lock-based)

| Test | File | Scenario |
|------|------|----------|
| `TestPriorityPatchLockedByOtherUser` | `priority_inline_edit_test.go` | PATCH while another user holds the lineage lock → 423 with `code: "locked"` and `lock` payload |
| `TestPriorityPatchWorksAfterLockRelease` | `priority_inline_edit_test.go` | After lock release, PATCH returns 200 and priority is updated |
| `TestPriorityGetReturnsLockHolder` | `priority_inline_edit_test.go` | GET artifact succeeds even when locked; GET /locks confirms lock holder |

> **Note on Milestone 6 UI assertions**: Non-interactive badge rendering (no `aria-haspopup`, no `tabindex="0"`) and re-enabling interactivity after lock release are frontend-only behaviours.

## Run commands

```sh
# All priority inline edit tests (new file only)
go test ./tests/... -tags integration -run "TestPriorityPatchIdempotent|TestPriorityPatchDiskConfirmation|TestPriorityPatchWebSocketActionField|TestPriorityAbsentOmittedFromResponse|TestPriorityExternalFileWriteBroadcastsUpdate|TestPriorityExternalWriteClosedDropdownUpdatesAPI|TestPriorityPatchLockedByOtherUser|TestPriorityPatchWorksAfterLockRelease|TestPriorityGetReturnsLockHolder"

# All priority-related tests (existing + new)
go test ./tests/... -tags integration -run "TestPriority"

# Run all integration tests
go test ./tests/... -tags integration
```
