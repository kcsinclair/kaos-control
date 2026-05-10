---
title: "Dark assigned edge #475569 fails WCAG 3:1 contrast on dark canvas"
type: defect
status: draft
lineage: 3d-graph-edge-contrast
parent: lifecycle/tests/3d-graph-edge-contrast-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
created: "2026-05-10T00:00:00+10:00"
---

# Dark assigned edge #475569 fails WCAG 3:1 contrast on dark canvas

## Reproduction Steps

1. Check out the current `main` branch.
2. From the repository root, run the edge-contrast test suite:
   ```
   cd tests/web && npx vitest run 3d-graph-edge-contrast.graphConstants.test.ts
   ```
3. Observe two assertion failures in the **dark palette** group.

## Expected Behaviour

The `assigned` edge colour in the dark palette (`edgeColors.assigned` /
`assignedEdgeColor` in `web/src/components/map/graphConstants.ts`) must achieve
a contrast ratio of **≥ 3:1** against the dark canvas background (`#0f172a`),
satisfying the WCAG 2.1 graphical-object standard (SC 1.4.11).

The comment in `graphConstants.ts` claims `~3.2:1` for the current colour.

## Actual Behaviour

The current dark-palette `assigned` colour `#475569` achieves only **2.36:1**
contrast against dark canvas `#0f172a`:

```
AssertionError: dark edgeColors["assigned"] contrast 2.36:1 < 3:1 (fg=#475569 bg=#0f172a):
  expected 2.3559417615334315 to be greater than or equal to 3

AssertionError: assigned edge contrast 2.36:1 is below 3:1:
  expected 2.3559417615334315 to be greater than or equal to 3
```

Two tests fail:
- `dark palette — all edge kinds >= 3:1 on canvasBg > edgeColors["assigned"] achieves >= 3:1 contrast on canvasBg`
- `dark palette: assigned edge (#475569) achieves >= 3:1 on dark canvas (#0f172a)`

## Suggested Fix

Replace the dark-palette `assignedEdgeColor` / `edgeColors.assigned` value with
a lighter slate shade. The test artifact documents `#64748b` as a candidate
(achieves ~4.6:1 on `#0f172a`) and the corresponding comment in `graphConstants.ts`
must be corrected.

File to edit: `web/src/components/map/graphConstants.ts`

## Logs / Output

```
 RUN  v1.6.1 /Users/keith/Code/kaos-control/tests/web

 ❯ 3d-graph-edge-contrast.graphConstants.test.ts  (23 tests | 2 failed) 8ms
   ❯ … > dark palette — all edge kinds >= 3:1 on canvasBg > edgeColors["assigned"] achieves >= 3:1 contrast on canvasBg
     → dark edgeColors["assigned"] contrast 2.36:1 < 3:1 (fg=#475569 bg=#0f172a):
       expected 2.3559417615334315 to be greater than or equal to 3
   ❯ … > dark palette: assigned edge (#475569) achieves >= 3:1 on dark canvas (#0f172a)
     → assigned edge contrast 2.36:1 is below 3:1:
       expected 2.3559417615334315 to be greater than or equal to 3
 ✓ 3d-graph-edge-contrast.linkWidth.test.ts  (11 tests) 63ms

 Test Files  1 failed | 1 passed (2)
       Tests  2 failed | 32 passed (34)
    Duration  387ms
```
