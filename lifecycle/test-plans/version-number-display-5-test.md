---
title: "Test Plan — Version Number Display"
type: plan-test
status: draft
lineage: version-number-display
parent: lifecycle/requirements/version-number-display-2.md
created: "2026-05-10"
---

# Test Plan — Version Number Display

This plan defines the test strategy for [[version-number-display]], covering the backend endpoint, build integration, and frontend display. Tests are integration-level, exercised against a running instance.

## Milestone 1 — Backend endpoint tests

### Description

Verify the `GET /api/version` endpoint behaviour: correct response shape, status code, content type, and that it requires no authentication. Also verify `/api/health` continues to include the version.

### Files to change

- `tests/version_test.go` (new) — integration tests against a running server.

### Test cases

1. **GET /api/version returns 200 with version JSON**
   - Send `GET /api/version` with no auth headers.
   - Assert status 200.
   - Assert `Content-Type` is `application/json`.
   - Assert body parses to `{"version": "<non-empty string>"}`.

2. **GET /api/version is unauthenticated**
   - Send the request without a session cookie.
   - Assert status 200 (not 401).

3. **GET /api/health includes version**
   - Send `GET /api/health`.
   - Assert body contains `"version"` key.
   - Assert its value is a non-empty string.

4. **Version parity between endpoints**
   - Fetch both `/api/version` and `/api/health`.
   - Assert the version values are identical.

### Acceptance criteria

- [ ] All four test cases pass against a running server.
- [ ] Tests do not require authentication setup.
- [ ] No new Go test dependencies introduced.

## Milestone 2 — Build integration tests

### Description

Verify that the `VERSION` file is correctly read by the build process and that the dev fallback works.

### Test approach

These are validated manually or via a Makefile target rather than Go test code, since they depend on build-time linker flags.

### Test cases

1. **VERSION file exists and is valid semver**
   - Assert `VERSION` file exists at repo root.
   - Assert contents match `^[0-9]+\.[0-9]+\.[0-9]+$` (no `v` prefix, no extra whitespace).

2. **Built binary reports correct version**
   - Run `make build`.
   - Execute `./dist/kaos-control --version` or start the binary and hit `/api/version`.
   - Assert returned version matches contents of `VERSION` file.

3. **Dev build falls back to "dev"**
   - Run `go run ./cmd/kaos-control` (no ldflags).
   - Hit `/api/version`.
   - Assert version is `"dev"`.

### Files to change

- `tests/version_test.go` — add a test that reads the `VERSION` file and validates its format.
- Optionally, a `make test-version` target in `Makefile` for the build-binary check.

### Acceptance criteria

- [ ] VERSION file format is validated programmatically.
- [ ] Build-produced binary version matches VERSION file contents.
- [ ] Unlinked binary defaults to `"dev"`.

## Milestone 3 — Frontend display tests

### Description

Verify the version label is rendered correctly in the UI. These are best covered by an end-to-end check or a manual test protocol, given the project does not currently use a browser test framework.

### Test cases

1. **Version label is visible**
   - Load the app in a browser (authenticated).
   - Assert an element containing `kaos-control` followed by a version string is visible in the sidebar.

2. **Version matches API**
   - Read the displayed version from the sidebar.
   - Call `GET /api/version`.
   - Assert both values match.

3. **Visual subtlety**
   - Assert the version label font size is smaller than primary navigation text.
   - Assert the label colour contrast meets WCAG 2.1 AA (≥ 4.5:1 for normal text).

4. **Label does not obscure navigation**
   - With the sidebar at minimum viewport height, confirm the version label does not overlap or push navigation items off-screen.

### Files to change

- `lifecycle/tests/version-number-display-fe-tests.md` (new, optional) — manual test protocol artifact if no browser automation is added.
- If browser tests exist or are introduced, add test cases to the relevant test file.

### Acceptance criteria

- [ ] A documented test protocol or automated test covers all four cases.
- [ ] Version display is confirmed matching `/api/version` response.
- [ ] WCAG contrast verified (manual inspection or automated tool).
