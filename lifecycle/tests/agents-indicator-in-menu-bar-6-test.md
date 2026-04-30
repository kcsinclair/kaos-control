---
title: "Test Suite — Move Running Agents Indicator to Menu Bar"
type: test
status: draft
lineage: agents-indicator-in-menu-bar
parent: lifecycle/test-plans/agents-indicator-in-menu-bar-5-test.md
---

# Test Suite — Move Running Agents Indicator to Menu Bar

Integration tests verifying that the running-agents indicator appears in the
application header, updates reactively, navigates correctly, and that the old
floating pill has been fully removed.

## Test files

- `tests/web/helpers/mockAgentsStore.ts` — shared helper
- `tests/web/AppHeaderRunIndicator.test.ts` — 27 test cases across 6 milestones

## Scenarios covered

### Milestone 1 — Mock Store Helper (5 tests)
- Helper starts with zero active runs.
- Helper can be configured to 1 or N active runs.
- Runs can be mutated mid-test to simulate agents starting and stopping.
- `makeRunningRun` factory produces a valid `AgentRunRow` with `status: "running"`.

### Milestone 2 — Indicator Visibility (4 tests)
- Indicator is **absent** when `activeRuns` is empty on a project route.
- Indicator is **present** when 1 or more agents are running on a project route.
- Indicator is **absent** on a non-project route (e.g. `/login`) even when agents
  are running — `route.params.project` is undefined so the `v-if` guard hides it.

### Milestone 3 — Count Display and Grammar (6 tests)
- Displays `"1 running agent"` (no trailing "s") for a single run.
- Displays `"3 running agents"` (with "s") for multiple runs.
- Indicator disappears reactively when all runs are cleared (`nextTick`).
- Text updates reactively when a second run is added.
- Indicator reappears after being cleared and a new run is added.

### Milestone 4 — Click Navigation (3 tests)
- Indicator renders as an `<a>` with `href="/p/:project/agents"`.
- Clicking the indicator navigates to `/p/:project/agents` using the router.
- Correct project slug is used when the route param differs from `"my-project"`.

### Milestone 5 — Accessibility (4 tests)
- `aria-label` is present and contains the count and "running agent(s)".
- `aria-label` is pluralised correctly (`"2 running agents"`).
- `aria-label` contains a navigation hint (e.g. "view" or "click").
- Component source includes a `prefers-reduced-motion` media query that sets
  `animation: none` on `.run-dot`, verified by reading the `.vue` file directly
  (happy-dom does not inject scoped styles into `getComputedStyle`).

### Milestone 6 — RunStatusChip Removal (4 tests)
- `web/src/components/agent/RunStatusChip.vue` does not exist on disk.
- `web/src/views/project/WorkspaceView.vue` does not reference `RunStatusChip`.
- `WorkspaceView.vue` has no `<Teleport>` for a running-agents indicator.
- No file under `web/src` imports `RunStatusChip` (full recursive walk of
  `.vue` and `.ts` files).

## Testing approach

The agents store is intercepted via `vi.mock('@/stores/agents', ...)` returning a
function that reads from a module-level `_runsRef` reactive ref. Each test resets
`_runsRef` in `beforeEach`. This makes `activeRuns` genuinely reactive inside the
mounted `AppHeader` without any real API or WebSocket calls.

CSS inspection for `prefers-reduced-motion` reads the raw component source with
`fs.readFileSync` rather than `getComputedStyle`, which is not populated by
happy-dom for scoped styles.
