---
title: "Kanban filter by release returns 0 artifacts when matching artifacts exist"
type: defect
status: draft
lineage: kanban-release-filter-returns-empty
created: "2026-06-12T00:00:00+10:00"
labels:
  - defect
assignees:
  - role: backend-developer
    who: agent
---

# Kanban filter by release returns 0 artifacts when matching artifacts exist

## Reproduction Steps

1. Create a release named "Kanban Filter Release".
2. Assign two idea artifacts to that release.
3. Query the kanban endpoint with `release=Kanban Filter Release` (or equivalent release filter parameter).
4. Count the returned artifacts.

## Expected Behaviour

2 artifacts returned (the ones assigned to the release).

## Actual Behaviour

0 artifacts returned.

```
releases_roadmap_regression_test.go:152:
    filter by release "Kanban Filter Release": want 2 artifacts, got 0: []
```

## Failing Test

- `TestKanbanFilterByRelease` (`releases_roadmap_regression_test.go:152`)

## Likely Root Cause

The kanban query filter for `release` may be matching against the release slug rather than the release name, or vice versa. Alternatively, the `release` field on artifact rows in the SQLite index may not be populated/matched correctly when the kanban endpoint applies the filter.

## Fix

Verify that:
1. The `release` column on `artifact_rows` is populated with the correct value when artifacts are indexed.
2. The kanban filter translates the incoming release parameter into the same value stored in the index (slug vs. display name consistency).
3. An integration test exercises the filter both by slug and by display name to confirm the correct match key.
