---
title: "Test Coverage: Inline Release Display and Editing"
type: test
status: approved
lineage: inline-release-display-edit
parent: lifecycle/test-plans/inline-release-display-edit-5-test.md
---

## Overview

This artifact documents the integration test coverage written for the
inline release display and editing feature. Tests were implemented for
Milestone 1 (backend PATCH endpoint). Milestones 2 and 3 (frontend
component tests) are blocked pending resolution of the open questions
recorded in the test plan.

## Milestone 1 — Backend Integration Tests

**File:** `tests/integration/patch_release_test.go`

All tests use `//go:build integration` and the standard `newTestEnv` /
`newTestEnvWithCfgYAML` harness. Run with:

```sh
go test -tags integration ./tests/integration/... -run TestReleasePatch
```

### Scenarios covered

| Test function | Scenario |
|---|---|
| `TestReleasePatch_SetRelease` | Happy path — create artifact with no release, PATCH with valid release name → 200, release set in response and confirmed via GET |
| `TestReleasePatch_ChangeRelease` | Happy path — artifact has release A, PATCH to release B → 200, release updated |
| `TestReleasePatch_ClearRelease` | Happy path — PATCH with `null` → 200, `release:` field absent from disk file |
| `TestReleasePatch_InvalidReleaseName` | Release name not found in project → 422 with `invalid_release` error code |
| `TestReleasePatch_ArtifactNotFound` | Non-existent artifact path → 404 |
| `TestReleasePatch_InvalidJSONBody` | Malformed JSON body → 400 |
| `TestReleasePatch_LockConflict` | Lineage locked by user A, PATCH as user B → 423 with `locked` error code and `lock` info block |
| `TestReleasePatch_ReindexVerification` | After successful PATCH, artifact list filtered by release name includes the patched artifact with correct release in frontmatter |
| `TestReleasePatch_WebSocketEvent` | After successful PATCH, an `artifact.indexed` WS event with `action: updated` and matching `path` is broadcast |
| `TestReleasePatch_FrontmatterPreservation` | PATCH only mutates `release`; all other frontmatter fields (title, type, status, lineage, labels) are unchanged |

### Helper additions

- `patchRelease(env, path, *string) *http.Response` — thin wrapper around
  `env.doRequest` that formats the PATCH body, including the `null` case.
- `strPtr(s string) *string` — returns a pointer to a string literal.
- `cfgYAMLWithQAAsProductOwner` — custom project config constant that grants
  `qa@test.local` the `product-owner` role, enabling the two-user lock
  conflict scenario.

## Milestones 2 & 3 — Frontend Component Tests (blocked)

Frontend tests for `ReleaseDropdown.vue` and `FrontmatterPanel.vue` could not
be written. No frontend test framework is installed in `web/` (no vitest, no
`@vue/test-utils`, no jsdom). See the open questions in the test plan for
the details of what must be decided before these tests can be implemented.
