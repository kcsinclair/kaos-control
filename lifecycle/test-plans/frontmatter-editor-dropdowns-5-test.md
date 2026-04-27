---
title: 'Test Plan: Frontmatter Editor Dropdowns'
type: plan-test
status: in-development
lineage: frontmatter-editor-dropdowns
parent: requirements/frontmatter-editor-dropdowns-2.md
---

# Test Plan: Frontmatter Editor Dropdowns

Integration tests verifying that status and priority values set via the frontmatter editor (and persisted through the API) round-trip correctly, respect the vocabulary constraints, and handle edge cases.

These tests exercise the HTTP API layer. The UI behaviour (dropdown rendering, focus styling) is covered by the [[frontmatter-editor-dropdowns]] frontend plan's acceptance criteria and manual verification.

## Milestone 1: Status Round-Trip Tests

### Description

Write integration tests that create an artifact via the API, update its status to each value in the vocabulary, and verify the persisted value matches.

### Files to change

- `tests/integration/status_dropdown_test.go` (new file)

### Acceptance criteria

- [ ] A test creates an artifact with `status: draft` and confirms it is returned as `draft` by `GET /artifacts/:path`.
- [ ] A test updates the artifact's status to each of the 10 vocabulary values (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked`) via `PUT /artifacts/:path` and confirms each is persisted and returned correctly.
- [ ] A test confirms that an unknown/legacy status value (e.g. `legacy-status`) can be written and read back without error (the backend does not validate status values).
- [ ] All tests pass with `go test ./tests/integration/ -run TestStatusDropdown -short`.

## Milestone 2: Priority Round-Trip Tests

### Description

Write integration tests that set priority via the API and verify persistence and omission behaviour.

### Files to change

- `tests/integration/priority_dropdown_test.go` (new file)

### Acceptance criteria

- [ ] A test creates an artifact with `priority: normal` and confirms it is returned as `normal`.
- [ ] A test updates priority to `high` and confirms it persists.
- [ ] A test sets priority to empty/omitted (simulating "— none —" selection) and confirms the `priority` key is absent or empty in the returned frontmatter.
- [ ] A test confirms that an unknown priority value (e.g. `critical`) can be written and read back without error.
- [ ] All tests pass with `go test ./tests/integration/ -run TestPriorityDropdown -short`.

## Milestone 3: Combined Update — No Regression

### Description

Write a test that updates both status and priority in a single `PUT` request and verifies that other frontmatter fields (title, type, lineage, labels) are not affected.

### Files to change

- `tests/integration/status_dropdown_test.go` or `tests/integration/priority_dropdown_test.go` (add test case to whichever is more appropriate)

### Acceptance criteria

- [ ] A test creates an artifact with known values for title, type, lineage, labels, status, and priority.
- [ ] A `PUT` that changes status and priority leaves title, type, lineage, and labels unchanged.
- [ ] The test reads the artifact back and asserts all fields match expected values.
- [ ] All existing integration tests in `tests/integration/` continue to pass (`go test ./tests/integration/ -short`).

## Milestone 4: Test Artifact Documentation

### Description

Create a companion `test` artifact in `lifecycle/tests/` documenting what the test suite covers, per the test-developer agent convention.

### Files to change

- `lifecycle/tests/frontmatter-editor-dropdowns-6-test.md` (new file, next lineage index)

### Acceptance criteria

- [ ] The artifact has frontmatter: `type: test`, `status: draft`, `lineage: frontmatter-editor-dropdowns`, `parent: test-plans/frontmatter-editor-dropdowns-5-test.md`.
- [ ] The body summarises the scenarios covered and references the specific test files in `tests/integration/`.

## Cross-references

- Backend plan [[frontmatter-editor-dropdowns]] confirms the API already supports both fields — tests verify this assumption.
- Frontend plan [[frontmatter-editor-dropdowns]] defines the UI behaviour; these integration tests cover the data layer beneath it.
- Note: existing `tests/integration/priority_roundtrip_test.go` and `tests/integration/priority_patch_test.go` may already cover some priority scenarios. The test developer should review those files and avoid duplication, extending them if appropriate rather than creating entirely new files.
