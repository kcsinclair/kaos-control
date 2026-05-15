---
title: "Git Context Display — Integration and Component Tests"
type: test
status: approved
lineage: git-context-display
parent: lifecycle/test-plans/git-context-display-5-test.md
---

# Git Context Display — Integration and Component Tests

This artifact documents the test coverage for the `git-context-display` feature.
Tests span the backend REST endpoint, WebSocket event broadcasting, and the
`GitStatusBar` Vue component.

## Test files

| File | Type | Runner |
|------|------|--------|
| `tests/integration/git_status_api_test.go` | Go integration | `make test-unit` with `-tags integration` |
| `tests/integration/git_status_ws_test.go` | Go integration | `make test-unit` with `-tags integration` |
| `tests/web/GitStatusBar.test.ts` | Vitest component | `pnpm exec vitest run GitStatusBar.test.ts` |

---

## Milestone 1 — REST Endpoint (git_status_api_test.go)

### M1-TC1: Happy path — git-backed project

Asserts `GET /api/p/testproject/git/status` returns HTTP 200 with:
- `available: true`
- `branch` — non-empty string
- `dirty: false` — clean working tree after initial commit
- `head_sha` — exactly 7 lowercase hex characters
- `head_message` — non-empty, no newlines (first line only)
- `head_author` — non-empty
- `head_when` — valid ISO 8601 / RFC 3339 timestamp

**Status:** pass

### M1-TC2: Dirty working tree

Creates a temp git repo (via `newTestEnv`), writes an untracked file, then
asserts `dirty: true`.

**Status:** pass

### M1-TC3: Non-git project

Uses `newNonGitTestEnv` (a helper that skips `git.PlainInit`). Asserts the
response body is exactly `{"available":false}` — one field, no extras.

**Status:** pass

### M1-TC4: Performance — NFR1 ≤ 100 ms

Measures wall-clock time of a single `doRequest` call. The O(1) implementation
(no history walk) should complete well within the 100 ms budget.

**Status:** pass

---

## Milestone 2 — WebSocket Events (git_status_ws_test.go)

Uses `env.proj.Hub.Register(ch)` — the same hub-channel pattern used across the
existing WS integration tests. No real HTTP WebSocket connection is required.

### M2-TC1: Event after AddAndCommit

Registers a hub channel, creates an artifact via `POST /api/p/testproject/artifacts`
(which calls `AddAndCommit`), drains messages for up to 2 s, and asserts a
`git.status` event with all required fields (`branch`, `dirty`, `head_sha`,
`head_message`, `head_author`, `head_when`).

**Status:** pass

### M2-TC2: Event after branch checkout

Creates a feature branch with go-git, registers a hub channel, checks out the
branch (modifying `.git/HEAD`), and waits up to 500 ms for a `git.status` event
whose `branch` field equals the new branch name.

The watcher debounces `.git/HEAD` changes by 150 ms. On CI environments where
fsnotify does not deliver the event within the window, the test skips with a
manual-verification note rather than failing:

> *MANUAL VERIFICATION REQUIRED: after `git checkout -b feature-checkout-ws-test`
> the GitStatusBar in the sidebar should update to the new branch name within
> one second.*

**Status:** pass (or skip with manual note on restrictive fsnotify environments)

### M2-TC3: No event for non-git project

Uses `newNonGitTestEnv`, writes a lifecycle file to generate indexing events,
collects all hub messages for 400 ms, and fails if any have `type == "git.status"`.
The callback is only registered when `gitRepo != nil`, so non-git projects
produce no git.status events.

**Status:** pass

---

## Milestone 3 — Vue Component Tests (GitStatusBar.test.ts)

All tests run under Vitest + `@vue/test-utils` + happy-dom. The `gitStatus`
Pinia store is replaced with a reactive mock backed by a `ref` so tests can
control state without involving real API calls. `@/api/ws` is mocked to prevent
real WebSocket connections.

### M3-TC1: Renders branch, SHA, message, and icon

Mounts with `available: true, branch: "feature/login", headSha: "a1b2c3d",
headMessage: "fix login redirect"`. Asserts:
- `.git-branch-name` text = `"feature/login"`
- `.git-branch-row svg` exists (GitBranch icon)
- `.git-sha` text = `"a1b2c3d"` (7 chars; component runs `abbreviateSha`)
- `.git-commit-msg` text = `"fix login redirect"`
- Multi-line message: only the first line is rendered
- `dirty: false` shows `"clean"` in `.git-dirty-indicator`

**Status:** pass (5 cases)

### M3-TC2: Dirty indicator

Mounts with `dirty: true`. Asserts:
- `.git-dirty-indicator` text = `"modified"`
- `.git-dirty-indicator--dirty` class present
- `aria-label` = `"Working tree has uncommitted changes"` when dirty
- `aria-label` = `"Working tree is clean"` when clean

**Status:** pass (4 cases)

### M3-TC3: Hidden when unavailable

`available: false` (default). Asserts `.git-status-bar` does not exist in the
DOM. Also asserts the element appears after switching `available` to `true`.

**Status:** pass (2 cases)

### M3-TC4: Collapsed sidebar state

Mounts with `collapsed: true`. Asserts:
- `.git-icon-wrap` exists; `.git-branch-row` absent
- `.git-branch-name` absent (v-else branch not rendered)
- `.git-icon-wrap svg` present (icon only)
- Dirty-dot modifier classes applied correctly
- Switching to `collapsed: false` shows the expanded branch-row

**Status:** pass (6 cases)

### M3-TC5: WebSocket update reactivity

Mounts with initial state, then mutates `_git.value.branch / dirty / headSha`
directly and calls `await nextTick()`. Asserts rendered output updates without
remounting.

**Status:** pass (3 cases)

### M3-TC6: Accessibility

Asserts:
- Root `.git-status-bar` has `role="status"`
- Root has `aria-label="Git repository status"`
- Expanded dirty indicator has a non-empty `aria-label`
- Collapsed dirty dot has `aria-label="Working tree has uncommitted changes"` (dirty)
- Collapsed dirty dot has `aria-label="Working tree is clean"` (clean)

**Status:** pass (5 cases)
