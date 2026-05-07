---
title: Unscheduled releases collapse to shared terminus instead of forming individual chain
type: defect
status: done
lineage: roadmap-graph-release-connections
parent: lifecycle/tests/roadmap-graph-release-connections-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Unscheduled releases collapse to shared terminus instead of forming individual chain

## Reproduction Steps

1. Start the server against an empty project.
2. Create two or more releases **without** a `start_date` (status `planned`), e.g.:
   ```
   POST /api/p/testproject/releases  {"name": "zzz", "status": "planned"}
   POST /api/p/testproject/releases  {"name": "aaa", "status": "planned"}
   ```
3. `GET /api/p/testproject/releases/graph`
4. Inspect the `edges` array for `kind: "timeline"`.

## Expected Behaviour

Each unscheduled release should appear as its own node (`release:<id>`) and be connected in a single directed chain, sorted alphabetically by name. With two unscheduled releases "aaa" and "zzz" the timeline chain should be:

```
release:backlog → release:<aaa-id> → release:<zzz-id>
```

and two timeline edges should be emitted.

When mixed with scheduled releases the chain should be:

```
Backlog → S1 → S2 → UA → UZ
```

(N releases → N timeline edges total)

## Actual Behaviour

All unscheduled releases are fanned into a single synthetic `release:unscheduled` terminus node. Only **one** spine edge from the last scheduled release (or Backlog) leads to `release:unscheduled`, and each individual unscheduled release node also points to `release:unscheduled`. This produces the wrong chain shape and wrong edge count:

- `TestRoadmapGraph_SingleUnscheduled`: chain[1] `"release:unscheduled"` instead of `"release:1"`
- `TestRoadmapGraph_MultipleUnscheduledAlphabetical`: chain length 2 instead of 3 (`[release:backlog release:unscheduled]`)
- `TestRoadmapGraph_NoScheduledDirectToUnscheduled`: chain length 2 instead of 4
- `TestRoadmapGraph_MixedScheduledAndUnscheduled`: chain length 4 instead of 5
- `TestRoadmapGraph_UnscheduledEdgesNoLabel`: no edge `release:<sched>→release:<ua>` or `release:<ua>→release:<ub>`
- `TestRoadmapGraph_DeleteOnlyScheduledUpdatesChain`: post-delete chain[1] `"release:unscheduled"` instead of `"release:2"`
- `TestRoadmapGraph_TimelineEdgeCount`: 4 edges instead of 3 for 3 releases (2 scheduled + 1 unscheduled)

## Logs / Output

```
releases_graph_test.go:284: single unscheduled: chain[1]: want "release:1", got "release:unscheduled"
--- FAIL: TestRoadmapGraph_SingleUnscheduled (0.12s)

releases_graph_test.go:308: multiple unscheduled chain length: want 3, got 2: [release:backlog release:unscheduled]
--- FAIL: TestRoadmapGraph_MultipleUnscheduledAlphabetical (0.12s)

releases_graph_test.go:339: no-scheduled chain length: want 4, got 2: [release:backlog release:unscheduled]
--- FAIL: TestRoadmapGraph_NoScheduledDirectToUnscheduled (0.12s)

releases_graph_test.go:377: mixed chain length: want 5, got 4: [release:backlog release:2 release:1 release:unscheduled]
--- FAIL: TestRoadmapGraph_MixedScheduledAndUnscheduled (0.12s)

releases_graph_test.go:435: no timeline edge "release:1"→"release:2"
--- FAIL: TestRoadmapGraph_UnscheduledEdgesNoLabel (0.12s)

releases_graph_test.go:816: post-delete chain[1]: want "release:2", got "release:unscheduled"
--- FAIL: TestRoadmapGraph_DeleteOnlyScheduledUpdatesChain (0.12s)

releases_graph_test.go:950: timeline edge count: want 3, got 4
--- FAIL: TestRoadmapGraph_TimelineEdgeCount (0.12s)
```

**Root cause location**: `internal/http/releases.go` lines 482–514 — the `buildRoadmapGraph` function uses a shared `release:unscheduled` terminus with fan-in edges instead of chaining unscheduled releases individually in alphabetical order.

The fix should:
1. Remove the synthetic `release:unscheduled` terminus node and the fan-in edges.
2. Sort unscheduled releases alphabetically by name.
3. Append each unscheduled release to the chain after all scheduled releases (using `prevID` continuation), emitting one `timeline` edge per hop with an empty label.
