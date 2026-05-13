---
title: "Git Context Display — Test Plan"
type: plan-test
status: done
lineage: git-context-display
parent: lifecycle/requirements/git-context-display-2.md
---

# Git Context Display — Test Plan

This plan covers both Go integration tests and Vue/Vitest component tests for the [[git-context-display]] feature. Tests are organised to validate the backend endpoint, WebSocket event broadcasting, and frontend rendering independently.

## Milestone 1: Backend Integration Tests — REST Endpoint

**Description:** Test the `GET /api/p/{project}/git/status` endpoint for both git-backed and non-git projects.

**Files to change:**
- `tests/integration/git_status_api_test.go` (new) — integration tests against the real HTTP server using the test helpers in `tests/helpers_test.go`.

**Test cases:**

1. **Happy path — git-backed project:** Register a project whose directory is a git repo (use `tests/fixtures/` or create a temp repo with `git init`). Assert:
   - HTTP 200 response.
   - `available` is `true`.
   - `branch` is a non-empty string (e.g. `main` or `master`).
   - `dirty` is `false` for a clean working tree.
   - `head_sha` is exactly 7 characters, hexadecimal.
   - `head_message` is a non-empty string with no newlines.
   - `head_author` is a non-empty string.
   - `head_when` is a valid ISO 8601 timestamp.

2. **Dirty working tree:** Create a temp git repo, make an initial commit, then modify a file without committing. Assert `dirty` is `true`.

3. **Non-git project:** Register a project whose directory is a plain folder (no `.git`). Assert:
   - HTTP 200 response.
   - Response body is exactly `{ "available": false }` (no other fields).

4. **Performance:** On the real kaos-control repository (which has substantial history), assert that the endpoint responds in under 100 ms (NFR1). Use `testing.B` or a simple wall-clock check.

**Acceptance criteria:**
- [ ] All four test cases pass.
- [ ] Tests use temporary directories and do not mutate the real repository.
- [ ] No new Go test dependencies beyond the standard library and existing test helpers.

## Milestone 2: Backend Integration Tests — WebSocket Events

**Description:** Test that `git.status` WebSocket events are broadcast after API-driven commits and after `.git/HEAD` changes.

**Files to change:**
- `tests/integration/git_status_ws_test.go` (new) — WebSocket integration tests.

**Test cases:**

1. **Event after AddAndCommit:** Connect a WebSocket client to `/api/p/{project}/ws`. Trigger a commit via the artifact create/update API (which calls `AddAndCommit` internally). Assert that a `git.status` event is received with the correct payload shape (fields: `branch`, `dirty`, `head_sha`, `head_message`, `head_author`, `head_when`).

2. **Event after branch checkout:** This test may be more complex since it requires simulating an external `git checkout`. If feasible within the test harness:
   - Create a temp repo with two branches.
   - Start the watcher.
   - Programmatically update `.git/HEAD` (or use `go-git` to checkout the other branch).
   - Assert a `git.status` event is received within 500 ms.
   - If external checkout simulation is too fragile, document this as a manual verification step.

3. **No event for non-git project:** Connect to a non-git project's WebSocket. Perform a file write. Assert that no `git.status` event is emitted.

**Acceptance criteria:**
- [ ] Post-commit WebSocket event test passes reliably.
- [ ] Branch-checkout event test passes or is documented as manual-only with justification.
- [ ] Tests clean up all temporary resources (repos, WebSocket connections).

## Milestone 3: Frontend Component Tests — GitStatusBar

**Description:** Vitest + Vue Test Utils tests for the `GitStatusBar` component rendering and reactivity.

**Files to change:**
- `tests/web/GitStatusBar.test.ts` (new) — component tests.

**Test cases:**

1. **Renders branch and commit info:** Mount `GitStatusBar` with a mocked Pinia store where `available: true`, `branch: "feature/login"`, `dirty: false`, `headSha: "a1b2c3d"`, `headMessage: "fix login redirect"`. Assert:
   - The branch name "feature/login" is rendered.
   - The `GitBranch` icon is present.
   - The SHA "a1b2c3d" is rendered.
   - The message "fix login redirect" is rendered.
   - No dirty indicator is shown (or it indicates "clean").

2. **Dirty indicator:** Same as above but with `dirty: true`. Assert that the dirty indicator (dot, text, or colour) is present and has the appropriate `aria-label` (e.g. "uncommitted changes").

3. **Hidden when unavailable:** Mount with `available: false`. Assert the component renders nothing (no DOM nodes or an empty wrapper).

4. **Collapsed sidebar state:** Mount with sidebar collapsed (mock `uiStore.sidebarCollapsed = true`). Assert that the branch label text is hidden and only the icon is visible.

5. **WebSocket update reactivity:** Mount the component, then programmatically update the store's `branch` ref. Assert the rendered branch name updates without remounting.

6. **Accessibility:** Assert the component has `role="status"` and an appropriate `aria-label`. Assert the dirty indicator has an `aria-label` describing the state.

**Acceptance criteria:**
- [ ] All six test cases pass under `vitest`.
- [ ] Tests mock the API and WebSocket — no real server needed.
- [ ] No new npm test dependencies beyond existing `vitest` + `@vue/test-utils` setup.

## Milestone 4: Test Artifact and CI Verification

**Description:** Create the lifecycle test artifact documenting coverage and verify all tests pass in CI.

**Files to change:**
- `lifecycle/tests/test-git-context-display-e2e.md` (new) — test artifact describing what the test code covers, following the existing pattern in `lifecycle/tests/`.

**Acceptance criteria:**
- [ ] Test artifact lists all test cases from milestones 1–3 with pass/fail status.
- [ ] `make test-unit` passes with the new backend tests included.
- [ ] `pnpm test` (or equivalent vitest runner) passes with the new frontend tests included.
- [ ] No regressions in existing test suites.
