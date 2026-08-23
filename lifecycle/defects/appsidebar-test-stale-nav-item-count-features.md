---
title: AppSidebar.test.ts hardcodes 16 nav items but sidebar now has 17 (missing Features entry)
type: defect
status: draft
lineage: appsidebar-test-stale-nav-item-count
parent: tests/web/AppSidebar.test.ts
labels:
    - defect
assignees:
    - role: test-developer
      who: agent
---

# AppSidebar.test.ts hardcodes 16 nav items but sidebar now has 17 (missing Features entry)

## Reproduction Steps

1. `cd tests/web && pnpm test` (or `pnpm exec vitest run AppSidebar.test.ts`)
2. Observe 5 failures, all `expected 17 to be 16`.

## Expected Behaviour

`tests/web/AppSidebar.test.ts`'s `EXPECTED_NAV_LABELS` array (line 51) matches the current nav item set rendered by `web/src/components/layout/AppSidebar.vue`.

## Actual Behaviour

Commit `5651c7de` ("feat(features): make 'feature' a first-class lifecycle type + Features view") added a `Features` nav entry to `AppSidebar.vue:124` (`{ label: 'Features', to: '/p/${p}/features', icon: Sparkles }`, between `Roadmap` and `Architecture`) but never updated `tests/web/AppSidebar.test.ts`. The component now renders 17 nav items; `EXPECTED_NAV_LABELS` still lists 16 and is missing `"Features"`.

This is the same recurring class of staleness previously fixed in `collapsible-sidebar-icons-7-defect.md` and `appsidebar-test-stale-nav-item-count-architecture.md` — that prior fix's suggestion to derive the expected count dynamically from the component's own nav-items source (instead of a hand-maintained literal array) was not carried through, so the same failure recurs whenever a nav item is added.

## Logs / Output

```
 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in expanded mode
AssertionError: expected 17 to be 16
 ❯ AppSidebar.test.ts:222:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in collapsed mode
AssertionError: expected 17 to be 16
 ❯ AppSidebar.test.ts:231:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > nav-label elements are hidden via CSS class when collapsed
AssertionError: expected 17 to be 16
 ❯ AppSidebar.test.ts:253:27

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 3: tooltip behaviour > aria-label on nav link matches the corresponding nav item label
AssertionError: expected 17 to be 16
 ❯ AppSidebar.test.ts:367:29

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 7: layout integrity > all nav links are rendered for each view without errors
AssertionError: expected 16 nav links on /p/testproject/artifacts: expected 17 to be 16
 ❯ AppSidebar.test.ts:637:94
```

## Fix guidance

Add `'Features'` to `EXPECTED_NAV_LABELS` (line 51) in the position matching `AppSidebar.vue`'s nav array (between `'Roadmap'` and `'Architecture'`). Consider deriving the expected nav set from the component itself (or a shared fixture) rather than a hand-maintained literal array, to stop this recurring every time a nav item is added.
