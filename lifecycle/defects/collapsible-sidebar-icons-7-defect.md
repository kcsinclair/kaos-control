---
title: "AppSidebar tests hardcode 6 nav items but sidebar now has 12 — tests need updating"
type: defect
status: done
lineage: collapsible-sidebar-icons
parent: lifecycle/tests/collapsible-sidebar-icons-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# AppSidebar tests hardcode 6 nav items but sidebar now has 12 — tests need updating

Five tests in `tests/web/AppSidebar.test.ts` fail because they assert `.nav-item` / `.nav-link` count equals exactly 6, but `AppSidebar.vue` now renders 12 distinct nav items.

## Reproduction Steps

1. `cd tests/web && pnpm install`
2. `pnpm exec vitest run AppSidebar.test.ts`
3. Observe 5 failures, all with `expected 12 to be 6`:
   - `Milestone 2 › renders an SVG icon for each nav item in expanded mode`
   - `Milestone 2 › renders an SVG icon for each nav item in collapsed mode`
   - `Milestone 2 › nav-label elements are hidden via CSS class when collapsed`
   - `Milestone 3 › aria-label on nav link matches the corresponding nav item label`
   - `Milestone 7 › all nav links are rendered for each view without errors`

## Expected Behaviour

The tests reflect the current sidebar nav item set. When `wrapper.findAll('.nav-item')` is called, the returned count matches the number of items actually rendered, and the assertions targeting specific labels (List, Board, Graph, Agents, Parse Errors, Config) pass without relying on an exact total-count check.

## Actual Behaviour

The tests were written when the sidebar had 6 items (`['List', 'Board', 'Graph', 'Agents', 'Parse Errors', 'Config']`). The component has since been extended to 12 distinct items:

```
Dashboard, List, Board, Testing, Graph, Roadmap,
Agents, Scheduler, Feed, Parse Errors, Config, Ollama
```

All five failing tests call `.findAll('.nav-item').length` or `.findAll('.nav-link').length` and assert `toBe(6)` / `toBe(expectedLabels.length)` where `expectedLabels` is still the old 6-item array. Every assertion therefore gets `12` and fails.

The `aria-label` loop test also fails because `navLinks[i]` indexes into the 12-item set and labels no longer align with the first 6 entries.

The underlying feature (collapse/expand, icon rendering, tooltips, badges, persistence, hover-overlay, animation) is unaffected — 54 of 59 tests pass.

## Logs / Output

```
 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in expanded mode
AssertionError: expected 12 to be 6 // Object.is equality
 ❯ AppSidebar.test.ts:207:29
    205|     const { wrapper } = await mountSidebar({ collapsed: false })
    206|     const navItems = wrapper.findAll('.nav-item')
    207|     expect(navItems.length).toBe(expectedLabels.length)

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in collapsed mode
AssertionError: expected 12 to be 6 // Object.is equality
 ❯ AppSidebar.test.ts:216:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > nav-label elements are hidden via CSS class when collapsed
AssertionError: expected 12 to be 6 // Object.is equality
 ❯ AppSidebar.test.ts:238:27

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 3: tooltip behaviour > aria-label on nav link matches the corresponding nav item label
AssertionError: expected 12 to be 6 // Object.is equality
 ❯ AppSidebar.test.ts:326:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 7: layout integrity > all nav links are rendered for each view without errors
AssertionError: expected 6 nav links on /p/testproject/artifacts: expected 12 to be 6 // Object.is equality
 ❯ AppSidebar.test.ts:596:66

 Test Files  1 failed (1)
       Tests  5 failed | 54 passed (59)
```

## Fix guidance

- Update `expectedLabels` (and any hardcoded `6` counts) in the five failing tests to reflect all 12 current nav items, or restructure the count assertions to use `navItems.length` dynamically.
- The Milestone 3 aria-label loop test should iterate over `navLinks` rather than `expectedLabels.length` to avoid index misalignment when new items are added later.
