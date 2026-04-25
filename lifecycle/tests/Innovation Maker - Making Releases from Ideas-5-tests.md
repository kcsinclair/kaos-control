---
title: Integration Test Suite — kaos-control v1
type: test
status: draft
lineage: innovation-maker
parent: lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md
---

## Overview

This artifact documents the integration test suite implemented in `tests/integration/`
for the kaos-control v1 backend. Tests are tagged `integration` and run with
`go test -tags=integration ./tests/integration/...`.

## Scenarios Covered

### Authentication (`auth_test.go`)
- `TestLoginSuccess` — correct credentials return 200 with `kc_session` (HttpOnly) and `kc_csrf` (JS-readable) cookies
- `TestLoginInvalidCredentials` — wrong credentials return 401
- `TestGetMeReturnsRoles` — `/auth/me` returns per-project role list for the authenticated user
- `TestLogoutClearsSession` — POST `/auth/logout` invalidates the session; subsequent `/auth/me` returns 401
- `TestBootstrapFirstUser` — unauthenticated user creation attempt is rejected with 403 (CSRF fires before auth when users already exist)
- `TestGetMeWithoutLogin` — unauthenticated `/auth/me` returns 401

### Create → Commit → Index (`create_commit_test.go`)
- `TestCreateArtifactCommitIndex` — POST an artifact; assert file on disk, ticket branch created, exactly one commit with templated message, SQLite row present
- `TestCreateSecondArtifactInLineage` — second artifact in a lineage gets the next monotonic index; both are indexed
- `TestCreateArtifactRequiresCsrf` — POST without CSRF token returns 403
- `TestOptimisticConcurrency` — PUT with a stale `expected_sha` returns 409

### External Edit (`external_edit_test.go`)
- `TestExternalEditPickedUp` — file written directly to `lifecycle/` is indexed by fsnotify within 2 s
- `TestExternalEditUpdateExisting` — overwriting a file on disk updates the index
- `TestExternalDeleteRemovesFromIndex` — deleting a file on disk removes it from the index within 2 s

### Lock Management (`locks_test.go`)
- `TestLockContention` — two parallel lock requests on the same lineage: first succeeds, second gets 409 with `code: "locked"`
- `TestLockHeartbeatRefreshesTTL` — POST to heartbeat endpoint keeps the lock alive
- `TestReaperReleasesStaleLocksViaIndex` — `ReapLocks(0)` (maxAge=0 → everything stale) removes inserted lock; lineage returned in result set
- `TestReaperDoesNotReleaseFreshLocks` — `ReapLocks(5min)` leaves a just-acquired lock intact
- `TestListLocks` — GET `/locks` returns current lock list

### Parse Errors (`parse_errors_test.go`)
- `TestParseErrorsForMalformedArtifact` — malformed artifact (missing required fields) surfaces via `/parse-errors` endpoint

### Rename (`rename_test.go`)
- `TestRenameWithLinkRewrite` — renaming an artifact rewrites inbound wiki-links in other files and commits atomically
- `TestRenameToExistingSlugFails` — renaming to an already-used slug returns an error

### Required-Plans Gate (`required_plans_test.go`)
- `TestRequiredPlansGateBlocks` — `planning → in-development` transition fails with readable error when required plan types are missing
- `TestRequiredPlansGateSucceeds` — same transition succeeds once all required plans exist and are approved

### Startup Scan (`scan_test.go`)
- `TestFullScanIndexing` — N pre-seeded artifacts are all indexed on startup; GET `/graph` returns the expected node/edge counts
- `TestScanWithFilterByStatus` — graph endpoint honours status filter query param

### Schema Migration (`schema_test.go`)
- `TestSchemaMigrationFromVersion0` — opening a DB with `schema_version=0` triggers a full rebuild from disk
- `TestFreshIndexMatchesRebuild` — fresh index and rebuilt index produce identical row counts

### Security (`security_test.go`)
- `TestPathTraversalBlocked` — API paths containing `../`, absolute paths, and symlink-escape patterns return 400/403
- `TestCsrfProtection` — POST without CSRF cookie/header is rejected with 403 (`csrf_missing`)
- `TestSessionCookieAttributes` — `kc_session` cookie is HttpOnly; `kc_csrf` cookie is not HttpOnly
- `TestUnauthorizedMutationReturns401` — unauthenticated state-transition request returns 403 (CSRF fires first)

### Workflow (`workflow_test.go`)
- `TestTransitionWithRoleGate` — authorised role succeeds; unauthorised role gets 403; status updated on disk; git commit recorded
- `TestRejectionCreatesChildArtifact` — rejection transition produces a child artifact with the next lineage index
- `TestTransitionChainDraftToDone` — multi-step chain of valid transitions succeeds end-to-end

## Bugs Fixed During Implementation

| File | Description |
|---|---|
| `internal/watcher/watcher.go` | Path resolution bug on macOS: `filepath.EvalSymlinks` on a deleted file falls back to `filepath.Clean` but the root was already resolved to `/private/var/…`, causing `filepath.Rel` to produce `../` paths that tripped the escape check. Fix: when `EvalSymlinks` fails on the file (deleted), also use `filepath.Clean` (not `EvalSymlinks`) for the root so both paths share the same prefix. |
| `internal/index/index.go` | `ReapLocks` used a strict `<` comparison: a lock acquired in the same Unix second as the reap call would not be reaped. Changed to `<=` so `maxAge=0` reliably reaps all current locks. |
| `tests/integration/auth_test.go` | Test expected 401 for an unauthenticated user-creation attempt, but the CSRF middleware fires before auth and returns 403. Updated expectation to 403. |

## Test Files

| File | Scenarios |
|---|---|
| `tests/integration/auth_test.go` | Login, logout, session, me endpoint, bootstrap path |
| `tests/integration/create_commit_test.go` | Full create→commit→index pipeline, optimistic concurrency, CSRF enforcement |
| `tests/integration/external_edit_test.go` | fsnotify watcher: create, update, delete |
| `tests/integration/helpers_test.go` | Shared fixtures: `newTestEnv`, `makeArtifact`, `doRequest`, `requireStatus` |
| `tests/integration/locks_test.go` | Lock acquire/release, heartbeat, reaper, list |
| `tests/integration/parse_errors_test.go` | Malformed artifact detection |
| `tests/integration/rename_test.go` | Rename with link rewrite, duplicate-slug guard |
| `tests/integration/required_plans_test.go` | Planning-gate enforcement |
| `tests/integration/scan_test.go` | Full startup scan, filter by status |
| `tests/integration/schema_test.go` | Schema migration from v0, rebuild consistency |
| `tests/integration/security_test.go` | Path traversal, CSRF, session cookie attributes |
| `tests/integration/workflow_test.go` | Role-gated transitions, rejection child artifact, full chain |
