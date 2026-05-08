---
title: "Collapsible Sidebar Icons — Defect Fix: nav-item count updated to 12"
type: test
status: draft
lineage: collapsible-sidebar-icons
parent: lifecycle/defects/collapsible-sidebar-icons-7-defect.md
---

# Collapsible Sidebar Icons — Defect Fix: nav-item count updated to 12

Fixes five failing tests in `tests/web/AppSidebar.test.ts` that asserted `.nav-item` /
`.nav-link` counts equal 6. `AppSidebar.vue` now renders 12 distinct nav items; the tests
have been updated to match.

All tests live in `tests/web/AppSidebar.test.ts` and run with Vitest + `@vue/test-utils` +
happy-dom.

Run the suite:
```sh
cd tests/web && pnpm install && pnpm exec vitest run AppSidebar.test.ts
```

Result: **59 / 59 pass** (was 54 / 59).

---

## Changes made

| File | Change |
|------|--------|
| `tests/web/AppSidebar.test.ts` | Updated `expectedLabels` in Milestone 2 from 6 to 12 items |
| `tests/web/AppSidebar.test.ts` | Updated Milestone 3 aria-label test to use `navLinks`-driven loop with 12-item label array |
| `tests/web/AppSidebar.test.ts` | Updated Milestone 7 nav-link count assertion from 6 → 12 |

---

## Scenarios fixed

### Milestone 2 — Icon Rendering

`expectedLabels` now contains all 12 current nav items:
`Dashboard, List, Board, Testing, Graph, Roadmap, Agents, Scheduler, Feed, Parse Errors, Config, Ollama`

| Scenario | Fix |
|----------|-----|
| SVG icons in expanded mode | `navItems.length` now asserted against 12-item `expectedLabels` |
| SVG icons in collapsed mode | Same |
| Nav labels hidden via CSS when collapsed | `.nav-label` count asserted against 12-item `expectedLabels` |
| All nav items rendered | Test title updated to "all twelve expected nav items are rendered" |

### Milestone 3 — Tooltip Behaviour

| Scenario | Fix |
|----------|-----|
| `aria-label` on nav link matches its label | Replaced hardcoded 6-item `expectedLabels` with 12-item `allExpectedLabels`; loop iterates over `navLinks.length` (not a fixed constant) to stay correct when items change |

### Milestone 7 — Layout Integrity

| Scenario | Fix |
|----------|-----|
| All nav links rendered for each view | Count assertion changed from `6` to `12`; error message updated accordingly |

---

## Unchanged passing tests (54)

Milestones 1, 4, 5, 6, and 8 were unaffected by the nav-item count change and continue to
pass without modification.
