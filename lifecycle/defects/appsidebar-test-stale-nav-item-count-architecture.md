---
title: AppSidebar.test.ts hardcodes 15 nav items but sidebar now has 16 (missing Architecture entry)
type: defect
status: in-development
lineage: collapsible-sidebar-icons
created: "2026-08-20T12:24:00+10:00"
parent: lifecycle/tests/collapsible-sidebar-icons-8-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: test-developer
      who: agent
rice_reach: 100
rice_impact: 0.25
rice_confidence: 25
rice_effort: 0.1
---

# AppSidebar.test.ts hardcodes 15 nav items but sidebar now has 16 (missing Architecture entry)

## Reproduction Steps

1. `cd tests/web && pnpm test` (or `pnpm exec vitest run AppSidebar.test.ts`)
2. Observe 5 failures, all `expected 16 to be 15`.

## Expected Behaviour

`tests/web/AppSidebar.test.ts`'s `expectedLabels` array (around line 210) matches the current nav item set rendered by `web/src/components/layout/AppSidebar.vue`.

## Actual Behaviour

Commit `54d8bfe3` ("feat(architecture-map): Milestone 2 — Architecture nav section & route") added an `Architecture` nav entry to `AppSidebar.vue:123` (`{ label: 'Architecture', to: '/p/${p}/architecture', icon: Boxes }`) but never updated `tests/web/AppSidebar.test.ts`. The component now renders 16 nav items; the test's `expectedLabels` still lists 15 and is missing `"Architecture"`, and the hardcoded count at line 630 is still `15`.

## Logs / Output

```
 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in expanded mode
AssertionError: expected 16 to be 15
 ❯ AppSidebar.test.ts:215:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in collapsed mode
AssertionError: expected 16 to be 15
 ❯ AppSidebar.test.ts:224:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > nav-label elements are hidden via CSS class when collapsed
AssertionError: expected 16 to be 15
 ❯ AppSidebar.test.ts:246:27

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 3: tooltip behaviour > aria-label on nav link matches the corresponding nav item label
AssertionError: expected 16 to be 15
 ❯ AppSidebar.test.ts:360:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 7: layout integrity > all nav links are rendered for each view without errors
AssertionError: expected 15 nav links on /p/testproject/artifacts: expected 16 to be 15
 ❯ AppSidebar.test.ts:630:67
```

## Fix guidance

Add `'Architecture'` to `expectedLabels` in the position matching `AppSidebar.vue`'s nav array (after `'Map'`), and update the hardcoded `15` at line 630 to `16`. Consider asserting the nav item count dynamically off the component's own nav-items source (or a shared fixture) instead of a hand-maintained literal array, to avoid this recurring every time a nav item is added — this is the same class of staleness previously fixed in `collapsible-sidebar-icons-7-defect.md`.
