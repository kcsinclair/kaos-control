---
title: "Unscheduled release incorrectly included in roadmap timeline edges"
type: defect
status: approved
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Unscheduled release incorrectly included in roadmap timeline edges

## Reproduction Steps

1. Create two releases: one scheduled (with `start_date` and `end_date`) and one unscheduled (no dates).
2. Send `GET /api/p/testproject/releases/graph`.
3. Inspect the `edges` array in the JSON response for entries with `"kind": "timeline"`.

## Expected Behaviour

Only scheduled releases should be connected by timeline edges. An unscheduled release (no `start_date`/`end_date`) must appear as a disconnected node — it must not be the `source` or `target` of any edge with `kind: "timeline"`.

## Actual Behaviour

The roadmap graph endpoint produces a timeline edge that involves the unscheduled release node. In the failing test, the edge `"release:1"→"release:2"` was emitted even though one of the releases had no dates.

## Logs / Output

```
releases_unscheduled_test.go:178: unscheduled release should not participate in timeline edges; got edge "release:1"→"release:2"
--- FAIL: TestReleaseUnscheduled_RoadmapGraphDisconnected (0.11s)
```
