---
title: "Tests — Version Number Display"
type: test
status: in-qa
lineage: version-number-display
parent: lifecycle/test-plans/version-number-display-5-test.md
created: "2026-05-10T00:00:00+10:00"
---

# Tests — Version Number Display

Integration tests for the version number display feature, covering the backend
API endpoints and the VERSION file format. All automated tests live in
`tests/integration/version_test.go`.

## Scenarios covered

### Milestone 1 — Backend endpoint tests

| Test function | Plan scenario |
|---|---|
| `TestVersion_Returns200WithJSON` | GET /api/version returns 200 with version JSON (status code, Content-Type header, non-empty `version` field) |
| `TestVersion_Unauthenticated` | GET /api/version succeeds without a session cookie (status 200, not 401) |
| `TestHealth_IncludesVersion` | GET /api/health response contains a non-empty `version` key |
| `TestVersion_ParityWithHealth` | Version value from /api/version equals the version value from /api/health |

All four tests use the standard `newTestEnv` harness (no login required) and
issue plain `http.Get` calls against the running test server.

### Milestone 2 — VERSION file format

| Test function | Plan scenario |
|---|---|
| `TestVersionFile_ExistsAndIsValidSemver` | VERSION file exists at repo root and its content matches `^[0-9]+\.[0-9]+\.[0-9]+$` (bare semver, no `v` prefix) |

This test uses `runtime.Caller` to locate the repo root relative to the test
source file, then reads and validates the VERSION file directly. It does not
start a server.

### Milestone 2 — Build and dev-fallback (manual)

The following scenarios from the test plan require a compiled binary and cannot
be exercised in a standard `go test` run:

- **Built binary reports correct version** — run `make build`, then hit
  `/api/version`; assert the returned version matches the VERSION file contents.
- **Dev build falls back to "dev"** — run `go run ./cmd/kaos-control` (no
  ldflags), hit `/api/version`; assert the returned version is `"dev"`.

These are validated by running `make test-version` (or manually following the
steps above) rather than by automated Go test code.

### Milestone 3 — Frontend display (manual protocol)

No browser automation framework is currently in use. The following cases are
covered by manual inspection:

1. Load the authenticated SPA and confirm an element containing
   `kaos-control <version>` is visible in the sidebar.
2. Compare the displayed version against the value returned by `GET /api/version`.
3. Verify the version label font size is smaller than primary navigation text
   and that its colour contrast meets WCAG 2.1 AA (≥ 4.5:1 for normal text).
4. At minimum viewport height, confirm the version label does not overlap or
   push navigation items off-screen.

## Test file

`tests/integration/version_test.go`
