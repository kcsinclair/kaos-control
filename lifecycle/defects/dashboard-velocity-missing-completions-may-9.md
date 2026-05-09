---
title: Dashboard Completion Velocity Shows Zero Completions on 9 May 2026
type: defect
status: in-development
lineage: dashboard-velocity-missing-completions-may-9
created: "2026-05-10T08:11:37+10:00"
priority: normal
labels:
    - defect
    - frontend
release: KC-Release0
assignees:
    - role: backend-developer
      who: agent
---

# Dashboard Completion Velocity Shows Zero Completions on 9 May 2026

## Reproduction Steps

1. Open the dashboard in the application.
2. Navigate to the completion velocity chart or widget.
3. Observe the data point for 9 May 2026.

## Expected Behaviour

The completion velocity chart should reflect all artifacts or tasks that were completed on 9 May 2026, showing a non-zero count for that date.

## Actual Behaviour

The completion velocity chart displays zero completions for 9 May 2026, despite multiple items having been completed on that date. The data for that day appears to be missing or not being counted correctly.

## Investigation Findings

Code investigation (`web/src/components/dashboard/widgets/VelocityChartWidget.vue` and `internal/index/index.go`) reveals the root cause is a **backend timezone bug**:

- `internal/index/index.go` line ~1876: `time.Unix(ts, 0)` returns a UTC-based `time.Time`
- Period keys (day/week/month buckets) are built from `time.Now().Location()` (local time)
- In UTC+10 (Sydney), events at `2026-05-09 00:00–09:59 AEST` fall on `2026-05-08` in UTC, so they are bucketed into the wrong day and dropped

The frontend (`VelocityChartWidget.vue`) simply renders what the API returns — there is no client-side date logic to fix.

## Resolved Questions

1. **Who owns this fix?** The artifact is labelled `frontend` but the bug is entirely in `internal/index/index.go` (backend). Should this defect be reassigned to the `backend-developer` agent? Or is there a separate frontend fix expected on top of the backend fix?

> Assigning to backend-developer

2. **No implementation plan is present.** The task asked to implement the defect artifact "milestone by milestone", but the artifact contains only a bug report — there is no `## Milestones` section or frontend implementation plan. Please either:
   - Add a `## Milestones` section describing the frontend work to be done, or
   - Confirm that the fix belongs entirely to the backend and reassign the defect accordingly.

3. **Fix approach confirmation.** If a frontend workaround is desired (e.g., the frontend re-bucketing API timestamps into local-date keys before rendering), please confirm this is acceptable given it would mask a backend bug, and describe the exact behaviour expected.
