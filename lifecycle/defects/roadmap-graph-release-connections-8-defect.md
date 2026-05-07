---
title: 7-day gap between releases labelled "7 days" instead of "1 week"
type: defect
status: in-development
lineage: roadmap-graph-release-connections
parent: lifecycle/tests/roadmap-graph-release-connections-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# 7-day gap between releases labelled "7 days" instead of "1 week"

## Reproduction Steps

1. Create two scheduled releases exactly 7 days apart:
   ```
   POST /api/p/testproject/releases  {"name": "v1", "status": "planned", "start_date": "2026-01-01"}
   POST /api/p/testproject/releases  {"name": "v2", "status": "planned", "start_date": "2026-01-08"}
   ```
2. `GET /api/p/testproject/releases/graph`
3. Find the timeline edge from `release:<v1-id>` to `release:<v2-id>`.
4. Inspect the `label` field.

## Expected Behaviour

The edge `label` should be `"1 week"` for a 7-day gap.

Per the spec (test artifact §Milestone 2):
- 7-day gap → `"1 week"`
- 14-day gap → `"2 weeks"`
- 30-day gap → `"4 weeks"`
- 35-day gap → `"1 month"`

## Actual Behaviour

The edge `label` is `"7 days"` instead of `"1 week"`.

## Logs / Output

```
releases_graph_test.go:503: 7-day gap edge label: want "1 week", got "7 days"
--- FAIL: TestRoadmapGraph_EdgeLabel7Days (0.12s)
```

**Root cause location**: `internal/http/releases.go` line 603 — the `humanDuration` function uses `if days < 8` as the threshold for the "days" bucket, so 7 days falls through to `fmt.Sprintf("%d days", days)` and returns `"7 days"`.

The fix is to change the threshold to `if days < 7`: any gap of 7 or more days with a whole-week count falls into the weeks bucket (7/7 = 1 week). A gap of 1–6 days continues to display as `"N day(s)"`.
