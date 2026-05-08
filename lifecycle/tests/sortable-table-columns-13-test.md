---
title: AgentsRunsView agent-name sort — corrected ascending/descending expectations
type: test
status: draft
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-12-defect.md
---

# AgentsRunsView agent-name sort — corrected ascending/descending expectations

Fixes the two failing assertions in the **"AgentsRunsView — Agent column sort"**
describe block that had transposed expected values for alphabetical sort order.

## Fix applied

In `tests/web/AgentsRunsView.sort.test.ts` the hardcoded expectations for the
Agent column ascending and descending sort tests were wrong. The fixture agents
sort alphabetically as `backend-developer` < `qa` < `requirements-analyst`
(`b < q < r`), but the tests had asserted a nonsensical order.

## Scenarios covered

| Test file | Describe | Scenario | Expected outcome |
|-----------|----------|----------|-----------------|
| `tests/web/AgentsRunsView.sort.test.ts` | AgentsRunsView — Agent column sort | clicking Agent header sorts runs alphabetically by agent name (ascending) | `names[0]` = `backend-developer`, `names[1]` = `qa`, `names[2]` = `requirements-analyst` |
| `tests/web/AgentsRunsView.sort.test.ts` | AgentsRunsView — Agent column sort | clicking Agent header again sorts descending | `names[0]` = `requirements-analyst`, `names[1]` = `qa`, `names[2]` = `backend-developer` |

## Files changed

| File | Change |
|------|--------|
| `tests/web/AgentsRunsView.sort.test.ts` | Corrected ascending expectations (lines ~145–147) and descending expectations (lines ~162–164) to match true alphabetical order |
