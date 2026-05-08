---
title: "AppSidebar renders 12 nav items instead of 6 — items duplicated in DOM"
type: defect
status: draft
lineage: collapsible-sidebar-icons
parent: lifecycle/tests/collapsible-sidebar-icons-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# AppSidebar renders 12 nav items instead of 6 — items duplicated in DOM

Five tests in `tests/web/AppSidebar.test.ts` fail because the component renders 12 `.nav-item` / `.nav-link` elements when exactly 6 are expected.

## Reproduction Steps

1. `cd tests/web`
2. `pnpm exec vitest run AppSidebar.test.ts`
3. Observe failures in:
   - `Milestone 2 › renders an SVG icon for each nav item in expanded mode`
   - `Milestone 2 › renders an SVG icon for each nav item in collapsed mode`
   - `Milestone 2 › nav-label elements are hidden via CSS class when collapsed`
   - `Milestone 3 › aria-label on nav link matches the corresponding nav item label`
   - `Milestone 7 › all nav links are rendered for each view without errors`

## Expected Behaviour

The sidebar renders exactly 6 nav items corresponding to `['List', 'Board', 'Graph', 'Agents', 'Parse Errors', 'Config']`. `wrapper.findAll('.nav-item').length` returns 6.

## Actual Behaviour

`wrapper.findAll('.nav-item').length` returns 12. Each nav item appears twice in the rendered DOM, causing all count-based assertions to fail.

```
AssertionError: expected 12 to be 6 // Object.is equality
    207|     expect(navItems.length).toBe(expectedLabels.length)
```

The aria-label loop also fails because `navLinks[i]` indexes into doubled entries and the labels no longer align.

## Logs / Output

```
 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon for each nav item in expanded mode
AssertionError: expected 12 to be 6 // Object.is equality
    207|     expect(navItems.length).toBe(expectedLabels.length)

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > renders an SVG icon in collapsed mode
AssertionError: expected 12 to be 6 // Object.is equality
    216|     expect(navItems.length).toBe(expectedLabels.length)

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 2: icon rendering > nav-label elements are hidden via CSS class when collapsed
AssertionError: expected 12 to be 6 // Object.is equality
    238|     expect(labels.length).toBe(expectedLabels.length)

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 3: tooltip behaviour > aria-label on nav link matches the corresponding nav item label
AssertionError: expected 12 to be 6 // Object.is equality
    326|     expect(navLinks.length).toBe(expectedLabels.length)

 FAIL  AppSidebar.test.ts > AppSidebar — Milestone 7: layout integrity > all nav links are rendered for each view without errors
AssertionError: expected 6 nav links on /p/testproject/artifacts: expected 12 to be 6 // Object.is equality
    596|       expect(navLinks.length, `expected 6 nav links on ${path}`).toBe(6)
```
