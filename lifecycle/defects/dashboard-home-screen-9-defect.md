---
title: 'DashboardGrid chart section aria-label changed; tests expect "Charts" but component renders "Charts top"/"Charts bottom"'
type: defect
status: in-development
lineage: dashboard-home-screen
parent: lifecycle/tests/dashboard-home-screen-6-test.md
labels:
  - defect
assignees:
  - role: frontend-developer
    who: agent
---

# DashboardGrid chart section aria-label changed: tests expect "Charts" but component renders "Charts top"/"Charts bottom"

## Reproduction Steps

1. Run the frontend unit tests:
   ```sh
   cd tests/web && npx vitest run DashboardView.test.ts --config vitest.config.ts
   ```
2. Observe 5 failures — all assert on `section[aria-label="Charts"]`.

## Expected Behaviour

`DashboardGrid.vue` should render chart-slot widgets inside a `<section>` with `aria-label="Charts"`, as specified in Milestone 5 of the test artifact (`dashboard-home-screen-6-test.md`) and as the tests assert:

```ts
const section = wrapper.find('section[aria-label="Charts"]')
expect(section.exists()).toBe(true)
```

## Actual Behaviour

The component was refactored to split chart widgets into two sub-sections:

- `aria-label="Charts top"` — widgets with `order < 2`
- `aria-label="Charts bottom"` — widgets with `order >= 2`

Neither label matches `"Charts"`, so `wrapper.find('section[aria-label="Charts"]')` always returns an empty wrapper, causing every downstream assertion to fail with:

```
AssertionError: expected false to be true  // section.exists() === false
Error: Cannot call find on an empty DOMWrapper
Error: Cannot call html on an empty DOMWrapper
```

**Failing tests** (all in `tests/web/DashboardView.test.ts`):

| Test | Error |
|---|---|
| `DashboardGrid — slot rendering > renders a widget registered to the chart slot` | `section[aria-label="Charts"]` not found |
| `DashboardGrid — slot rendering > renders widgets in ascending order within the chart slot` | Cannot call find on empty wrapper |
| `DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC1b: stages-distribution widget renders in the Charts section` | `section[aria-label="Charts"]` not found |
| `DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC6: existing widgets … still render alongside stages-distribution` | Cannot call find on empty wrapper |
| `DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC6b: chart-slot widgets appear in correct order …` | Cannot call html on empty wrapper |

## Logs / Output

```
 FAIL  DashboardView.test.ts > DashboardGrid — slot rendering > renders a widget registered to the chart slot
AssertionError: expected false to be true // Object.is equality

 ❯ DashboardView.test.ts:126:30
    124| 
    125|     const section = wrapper.find('section[aria-label="Charts"]')
    126|     expect(section.exists()).toBe(true)
       |                              ^

 FAIL  DashboardView.test.ts > DashboardGrid — slot rendering > renders widgets in ascending order within the chart slot
Error: Cannot call find on an empty DOMWrapper.
 ❯ DashboardView.test.ts:182:26

 FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC1b
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:368:36

 FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC6
Error: Cannot call find on an empty DOMWrapper.
 ❯ DashboardView.test.ts:388:26

 FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: StagesDistributionWidget integration > TC6b
Error: Cannot call html on an empty DOMWrapper.
 ❯ DashboardView.test.ts:409:61

Test Files  1 failed (1)
Tests       5 failed | 47 passed (52)
```
