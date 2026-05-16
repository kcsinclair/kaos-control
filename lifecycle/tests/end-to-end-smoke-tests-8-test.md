---
title: "Tests: Fix CSS selector in 01-login.spec.ts (summary-card)"
type: test
status: draft
lineage: end-to-end-smoke-tests
parent: lifecycle/defects/end-to-end-smoke-tests-7-defect.md
---

# Tests: Fix CSS selector in 01-login.spec.ts (summary-card)

Addresses defect `end-to-end-smoke-tests-7-defect.md`: the Playwright locator in
Flow 01 used the wrong CSS class names to find the "Lifecycle Total" stat card on
the dashboard.

## Change made

**`tests/e2e/flows/01-login.spec.ts`** — two selectors corrected:

| Before (wrong) | After (correct) |
|---|---|
| `.stat-card` | `.summary-card` |
| `.stat-value, .stat-number, [class*="value"]` | `.summary-card-value` |

These classes match the actual markup rendered by
`web/src/components/dashboard/widgets/SummaryCountCard.vue`.

## Scenarios covered

### Flow 01 — Login and project access (`flows/01-login.spec.ts`)

- Verifies unauthenticated navigation to `/p/testproject/dashboard` redirects to `/login`.
- Bootstraps admin user, drives SPA login form, and lands on the project dashboard.
- Locates the "Lifecycle Total" `.summary-card` element and asserts it is visible.
- Asserts the `.summary-card-value` inside that card is not `"0"` (fixture has 14 items).

## Test files

- `tests/e2e/flows/01-login.spec.ts` — the corrected spec
