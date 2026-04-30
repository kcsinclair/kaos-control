---
title: "Test Plan — Move Running Agents Indicator to Menu Bar"
type: plan-test
status: done
lineage: agents-indicator-in-menu-bar
parent: lifecycle/requirements/agents-indicator-in-menu-bar-2.md
---

# Test Plan — Move Running Agents Indicator to Menu Bar

Integration tests verifying that the running-agents indicator appears in the application header, updates reactively, navigates correctly, and that the old floating pill has been fully removed. Tests interact with the Vue components at the DOM level using the existing Vitest + Vue Test Utils setup in `tests/web/`. See [[agents-indicator-in-menu-bar]] frontend plan for implementation details.

---

## Milestone 1: Test Helpers — Mock Agent Store with Active Runs

### Description

Create a reusable helper that provides a mock `useAgentsStore` with configurable `activeRuns` state, allowing tests to simulate zero, one, or many running agents without needing a live WebSocket connection or backend.

### Files to change
- `tests/web/helpers/mockAgentsStore.ts` — export a factory function that returns a Pinia store override with a settable `activeRuns` computed/ref. If a suitable helper already exists (e.g. in `tests/web/helpers/`), extend it rather than creating a duplicate.

### Acceptance criteria
- [ ] A helper exists that can provide an agents store with 0, 1, or N active runs.
- [ ] The helper is importable by all test files in subsequent milestones.
- [ ] Active runs can be mutated during a test to simulate agents starting and stopping.

---

## Milestone 2: Indicator Visibility Tests

### Description

Test that the indicator appears in the header when agents are running and is hidden when none are. Also verify it does not render when there is no project context (simulating login/project-picker pages).

### Files to change
- `tests/web/AppHeaderRunIndicator.test.ts` — new test file.

### Test cases
1. **Hidden when idle** — mount `AppHeader` with `activeRuns = []` and a valid project route; assert no indicator element is in the DOM.
2. **Visible when agents running** — mount with `activeRuns` containing 1 run; assert the indicator element is present.
3. **Hidden on non-project routes** — mount with `activeRuns` containing runs but no `project` route param; assert indicator is not rendered.

### Acceptance criteria
- [ ] All three test cases pass.
- [ ] Tests use the helper from Milestone 1 to control store state.

---

## Milestone 3: Count Display and Grammar Tests

### Description

Verify the indicator displays the correct count with proper singular/plural grammar and updates reactively as the count changes.

### Files to change
- `tests/web/AppHeaderRunIndicator.test.ts` — add test cases to the file from Milestone 2.

### Test cases
1. **Singular** — 1 active run → text contains "1 running agent" (no trailing "s").
2. **Plural** — 3 active runs → text contains "3 running agents".
3. **Reactive update** — start with 1 run, add a second run to the store, assert text updates to "2 running agents". Remove all runs, assert indicator disappears.

### Acceptance criteria
- [ ] Singular/plural grammar is correct for counts 1 and >1.
- [ ] Reactive updates are reflected in the DOM within the same test (using `nextTick` or `flushPromises`).

---

## Milestone 4: Click Navigation Test

### Description

Verify that clicking the indicator navigates to the Agents view for the current project.

### Files to change
- `tests/web/AppHeaderRunIndicator.test.ts` — add navigation test case.

### Test cases
1. **Click navigates** — mount with project `"my-project"` and 1 active run. Click the indicator. Assert that `router.push` was called with `/p/my-project/agents` (or the route resolved to the agents view).

### Acceptance criteria
- [ ] Clicking the indicator triggers navigation to `/p/:project/agents` with the correct project slug.

---

## Milestone 5: Accessibility Tests

### Description

Verify the indicator has a descriptive `aria-label` and that the pulsing dot respects `prefers-reduced-motion`.

### Files to change
- `tests/web/AppHeaderRunIndicator.test.ts` — add accessibility test cases.

### Test cases
1. **aria-label present** — with 2 active runs, assert the indicator element has `aria-label` containing "2 running agents".
2. **Reduced motion** — assert that the component's styles include a `prefers-reduced-motion` media query that disables the pulse animation (CSS inspection or snapshot).

### Acceptance criteria
- [ ] `aria-label` accurately reflects the current count and includes navigation hint.
- [ ] A `prefers-reduced-motion` rule exists in the component styles.

---

## Milestone 6: RunStatusChip Removal Verification

### Description

Verify that the old floating pill component has been completely removed and no references to it remain.

### Files to change
- `tests/web/AppHeaderRunIndicator.test.ts` — add removal verification test case.

### Test cases
1. **Component deleted** — assert that `web/src/components/agent/RunStatusChip.vue` does not exist on disk (or import fails).
2. **No Teleport for run status** — grep `WorkspaceView.vue` source for `RunStatusChip` or `Teleport` related to run status; assert no matches.

### Acceptance criteria
- [ ] `RunStatusChip.vue` does not exist.
- [ ] `WorkspaceView.vue` contains no import or template reference to `RunStatusChip`.
- [ ] No `<Teleport to="body">` for running-agents indicator exists anywhere in the codebase.
