---
title: Integration Tests — Frontmatter Editor Dropdowns
type: test
status: draft
lineage: frontmatter-editor-dropdowns
parent: test-plans/frontmatter-editor-dropdowns-5-test.md
---

# Integration Tests — Frontmatter Editor Dropdowns

Integration test suite covering status and priority round-trips via the HTTP API,
vocabulary handling, edge cases (unset / unknown values), and regression checks
for combined updates.

## Test files

- `tests/integration/status_dropdown_test.go`
- `tests/integration/priority_dropdown_test.go`

## API changes

Per the test plan's resolved question (Option B), the `validPriorities` vocabulary
guard was removed from both `handleUpdateArtifact` (PUT) and `handlePatchPriority`
(PATCH) in `internal/http/write.go`. Priority now accepts any string value, matching
status behaviour. Two existing tests were updated to reflect this:

- `TestPutArtifactInvalidPriority` → renamed `TestPutArtifactUnknownPriority`, now
  asserts 200 and that the value is stored.
- `TestPriorityPatchInvalidValue` → renamed `TestPriorityPatchUnknownValue`, now
  asserts 200 and that the value is stored.

## Scenarios covered

### Status round-trip (`status_dropdown_test.go`)

| Test | What it checks |
|------|---------------|
| `TestStatusDropdownCreateDraft` | POST creates artifact with `status: draft`; GET returns `draft` |
| `TestStatusDropdownAllVocabValues` | PUT updates status to each of the 10 vocabulary values; each is persisted and returned by GET |
| `TestStatusDropdownUnknownValue` | PUT with `legacy-status` succeeds (no validation); value is stored and returned |
| `TestStatusDropdownCombinedUpdateNoRegression` | PUT changes status + priority in one request; title, type, lineage, and labels remain unchanged |

Run with: `go test ./tests/integration/ -run TestStatusDropdown -short`

### Priority round-trip (`priority_dropdown_test.go`)

| Test | What it checks |
|------|---------------|
| `TestPriorityDropdownCreateNormal` | POST creates artifact with `priority: normal`; GET returns `normal` |
| `TestPriorityDropdownUpdateToHigh` | PUT sets priority to `high`; persisted on disk and in API |
| `TestPriorityDropdownUnsetViaEmpty` | PUT with `priority: ""` removes the key from frontmatter on disk and returns empty |
| `TestPriorityDropdownUnknownValueAccepted` | PUT and PATCH with unknown values (`critical`, `extreme`) both return 200 and store the value |

Run with: `go test ./tests/integration/ -run TestPriorityDropdown -short`
