---
title: "Test 01 uses wrong CSS selector `.stat-card` — SummaryCountCard renders `.summary-card`"
type: defect
status: approved
lineage: end-to-end-smoke-tests
parent: lifecycle/tests/end-to-end-smoke-tests-4-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# Test 01 uses wrong CSS selector `.stat-card`

## Reproduction Steps

1. Fix the `roles:` → `role:` YAML key bug (see `end-to-end-smoke-tests-6-defect.md`)
   so the test project loads correctly.
2. Run `cd tests/e2e && pnpm test flows/01-login.spec.ts`.
3. Observe the `loggedInPage lands on project dashboard with non-zero Lifecycle Total`
   test still fails.

## Expected Behaviour

The test locates the "Lifecycle Total" stat card on the dashboard and asserts it
has a non-zero value.

## Actual Behaviour

Playwright reports:

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.stat-card').filter({ hasText: 'Lifecycle Total' })
Expected: visible
Timeout: 10000ms
Error: element(s) not found
```

The element is never found because the CSS class does not match the rendered HTML.

## Root Cause

`web/src/components/dashboard/widgets/SummaryCountCard.vue` renders with class
`summary-card` (not `stat-card`):

```html
<!-- SummaryCountCard.vue template -->
<div
  class="summary-card"
  ...
>
  ...
  <span class="summary-card-label">{{ label }}</span>
</div>
```

`tests/e2e/flows/01-login.spec.ts` line 17 queries the wrong class:

```typescript
const lifecycleTotal = page.locator('.stat-card', { hasText: 'Lifecycle Total' })
//                                   ^^^^^^^^^^
//                                   should be '.summary-card'
```

The inner value class is also wrong: the test uses `.stat-value, .stat-number,
[class*="value"]` but the component renders `summary-card-value`.

## Logs / Output

```
1) flows/01-login.spec.ts:9:3 › Flow 01 — Login and project access ›
   loggedInPage lands on project dashboard with non-zero Lifecycle Total

Error: expect(locator).toBeVisible() failed

Locator: locator('.stat-card').filter({ hasText: 'Lifecycle Total' })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 10000ms
  - waiting for locator('.stat-card').filter({ hasText: 'Lifecycle Total' })

  16 |   const lifecycleTotal = page.locator('.stat-card', { hasText: 'Lifecycle Total' })
> 18 |   await expect(lifecycleTotal).toBeVisible({ timeout: 10_000 })
```

## Fix

In `tests/e2e/flows/01-login.spec.ts` update the locator to match the actual
component classes:

```typescript
// Before
const lifecycleTotal = page.locator('.stat-card', { hasText: 'Lifecycle Total' })
await expect(lifecycleTotal).toBeVisible({ timeout: 10_000 })
const valueLocator = lifecycleTotal.locator('.stat-value, .stat-number, [class*="value"]').first()

// After
const lifecycleTotal = page.locator('.summary-card', { hasText: 'Lifecycle Total' })
await expect(lifecycleTotal).toBeVisible({ timeout: 10_000 })
const valueLocator = lifecycleTotal.locator('.summary-card-value').first()
```
