---
title: "Collapsible Sidebar Icons — Defect Fix: nav-item count updated to 16 (Architecture entry)"
type: test
status: draft
lineage: collapsible-sidebar-icons
parent: lifecycle/defects/appsidebar-test-stale-nav-item-count-architecture.md
created: "2026-08-20T14:35:00+10:00"
---

# Collapsible Sidebar Icons — Defect Fix: nav-item count updated to 16 (Architecture entry)

Fixes five failing tests in `tests/web/AppSidebar.test.ts` that asserted `.nav-item` /
`.nav-link` counts equal 15. `AppSidebar.vue` gained an `Architecture` nav entry
(commit `54d8bfe3`) and now renders 16 distinct nav items in the default (no
product-owner/devops role) test mount; the tests have been updated to match.

All tests live in `tests/web/AppSidebar.test.ts` and run with Vitest + `@vue/test-utils` +
happy-dom.

Run the suite:
```sh
cd tests/web && pnpm install && pnpm exec vitest run AppSidebar.test.ts
```

Result: **61 / 61 pass** (was 56 / 61).

---

## Changes made

| File | Change |
|------|--------|
| `tests/web/AppSidebar.test.ts` | Hoisted the nav-label list to a module-level `EXPECTED_NAV_LABELS` constant, shared by the Milestone 2, Milestone 3, and Milestone 7 assertions instead of three separately hand-maintained literal arrays |
| `tests/web/AppSidebar.test.ts` | Added `'Architecture'` to `EXPECTED_NAV_LABELS`, positioned after `'Map'`/`'Roadmap'` and before `'Testing'`, matching the Content section order in `AppSidebar.vue` |
| `tests/web/AppSidebar.test.ts` | Updated Milestone 7 nav-link count assertion from `15` → `EXPECTED_NAV_LABELS.length` (16), including the assertion failure message |

---

## Scenarios fixed

### Milestone 2 — Icon Rendering

`EXPECTED_NAV_LABELS` now contains all 16 current nav items in section order (Activity →
Content → Automation → System):
`Dashboard, Feed, Reports, List, Board, Map, Roadmap, Architecture, Testing, Documentation,
Agents, Queue, Scheduler, Config, Ollama, Parse Errors`

| Scenario | Fix |
|----------|-----|
| SVG icons in expanded mode | `navItems.length` now asserted against 16-item `EXPECTED_NAV_LABELS` |
| SVG icons in collapsed mode | Same |
| Nav labels hidden via CSS when collapsed | `.nav-label` count asserted against 16-item `EXPECTED_NAV_LABELS` |
| All nav items rendered | Unaffected by the count change (iterates the shared array; test title left as-is — already stale from a prior fix and out of scope for this defect) |

### Milestone 3 — Tooltip Behaviour

| Scenario | Fix |
|----------|-----|
| `aria-label` on nav link matches its label | Replaced the locally duplicated 15-item `allExpectedLabels` literal with the shared `EXPECTED_NAV_LABELS` constant (now 16 items); loop still iterates over `navLinks.length` to stay correct as items change |

### Milestone 7 — Layout Integrity

| Scenario | Fix |
|----------|-----|
| All nav links rendered for each view | Count assertion changed from hardcoded `15` to `EXPECTED_NAV_LABELS.length`; error message now reports the actual expected count instead of a stale literal |

---

## Unchanged passing tests (56)

Milestones 1, 4, 5, 6, and 8, plus the functional-section-grouping tests, were unaffected
by the nav-item count change and continue to pass without modification.

## Note on recurrence

This is the second time a nav-item addition has broken this test file (see
`collapsible-sidebar-icons-7-defect.md` / `-8-test.md` for the first). Hoisting the
expected-labels list to a single `EXPECTED_NAV_LABELS` constant reduces the blast radius
to one array, but the count and labels are still hand-maintained rather than derived from
`AppSidebar.vue`'s own `navSections` source — a future defect of the same class remains
possible if that tradeoff is revisited.
