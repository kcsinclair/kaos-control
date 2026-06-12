---
title: 'Triage in-flight coalescing broken: rapid duplicate calls create separate runs'
type: defect
status: in-development
lineage: triage-coalesce-duplicate-runs
created: "2026-06-12T00:00:00+10:00"
labels:
    - defect
release: KC-Release3
assignees:
    - role: backend-developer
      who: agent
---

# Triage in-flight coalescing broken: rapid duplicate calls create separate runs

## Reproduction Steps

1. Block the LLM call so a triage run stays in-flight (e.g. hold a channel).
2. POST `/api/p/testproject/ideas/<slug>/triage` twice in rapid succession for the same artifact.
3. Check the two response `run_id` values.
4. Count `agent_runs` rows for the artifact after both calls complete.

## Expected Behaviour

Both calls return the same `run_id` (the in-flight run). Exactly 1 `agent_runs` row is created.

## Actual Behaviour

Each call gets a different `run_id`. 3 `agent_runs` rows are created (instead of 1).

```
triage_api_test.go:199: expected coalesced run IDs to match;
    got "c1b27b3b-5757-49ad-b706-d88eaae8b237" and "b99c9234-ea30-417d-b0ef-89ca9c1e932a"
triage_api_test.go:211: expected exactly 1 agent_runs row, got 3
```

The triage `Manager.Trigger` method is not checking whether a run for the same artifact is already in flight before starting a new one. It should detect the in-progress state and return the existing run ID.

## Failing Test

- `TestTriageAPI_InFlightCoalesce` (`triage_api_test.go:199,211`)

## Fix

`Manager.Trigger` should maintain an in-flight map keyed by `relPath` (protected by the existing mutex). If a run is in progress for the same path, return its `runID` without starting a new goroutine.
