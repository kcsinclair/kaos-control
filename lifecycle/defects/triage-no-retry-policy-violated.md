---
title: Triage retries a failed run when no-retry policy is expected
type: defect
status: in-development
lineage: triage-no-retry-policy-violated
created: "2026-06-12T00:00:00+10:00"
labels:
    - defect
release: KC-Release3
assignees:
    - role: backend-developer
      who: agent
---

# Triage retries a failed run when no-retry policy is expected

## Reproduction Steps

1. Configure the triage manager so the LLM call always fails (e.g. inject an error).
2. Trigger triage for a `raw` idea artifact.
3. Wait for the run to complete.
4. Count the `agent_runs` rows for the artifact.

## Expected Behaviour

Exactly 1 `agent_runs` row — triage does not retry on failure.

## Actual Behaviour

2 `agent_runs` rows are created, indicating a retry was attempted after the first failure.

```
triage_failure_test.go:225: expected exactly 1 run (no retry), got 2
```

The triage manager is triggering a second execute after the first fails. This may occur because the watcher or startup scan re-observes the artifact (still `status: raw` after failure) and enqueues it again before the first run's failure is fully recorded, or because the failure-path exception handling in the manager itself issues a second attempt.

## Failing Test

- `TestTriageFailure_NoRetry` (`triage_failure_test.go:225`)

## Fix

After a triage run fails, the manager must suppress re-triggering for the same artifact for at least the duration of the test observation window. This could be achieved via:
- Recording the failure in a backoff map keyed by `relPath` with a cooldown period.
- Ensuring the lineage lock (if held during the run) is not released until after watcher debounce settles.
- Verifying the watcher/startup re-scan does not re-enqueue an artifact whose last run status is `failed`.
