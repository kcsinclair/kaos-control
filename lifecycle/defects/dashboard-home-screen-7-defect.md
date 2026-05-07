---
title: "Dashboard stats/distribution tests fail: blocked artifacts auto-transitioned to draft on index"
type: defect
status: done
lineage: dashboard-home-screen
parent: lifecycle/tests/dashboard-home-screen-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# Dashboard stats/distribution tests fail: blocked artifacts auto-transitioned to draft on index

## Reproduction Steps

1. Run the dashboard integration tests with the integration build tag:
   ```
   go test -tags=integration ./tests/integration/ \
     -run "TestDashboardStats_MixedStatuses|TestStatusDistribution_CorrectCounts" -v -count=1
   ```
2. Observe both tests fail.

## Expected Behaviour

- `TestStatusDistribution_CorrectCounts`: the `distribution` array includes an entry `{"status":"blocked","count":1}`.
- `TestDashboardStats_MixedStatuses`: the `blocked` field in the stats response equals `2` (one `blocked` ticket + one `clarifying` ticket).

## Actual Behaviour

- `TestStatusDistribution_CorrectCounts`: the `blocked` entry is absent from the distribution; count is `0`.
- `TestDashboardStats_MixedStatuses`: `blocked` count is `1` (only the `clarifying` ticket; the seeded `blocked` ticket is gone).

## Root Cause

`makeArtifact` seeds `blocked`-status artifacts with a plain body (`"Body."`) and no `## Open Questions` section. When the index scans the file, `internal/index/autoblock.go:applyOpenQuestionTransition` fires the rule:

> "No open questions AND status == `blocked` → auto-transition to `draft`"

The artifact is rewritten to `status: draft` before the HTTP handler is called, so neither the distribution endpoint nor the stats endpoint ever observes a `blocked` ticket.

The same auto-unblock fires for the `stats-blocked-1.md` seed in `TestDashboardStats_MixedStatuses`, leaving only the `clarifying` artifact contributing to the `blocked` count (expected 2, got 1).

## Logs / Output

```
# TestStatusDistribution_CorrectCounts
INFO auto-transition: open questions resolved
      path=lifecycle/requirements/dist-blocked-1.md
      old_status=blocked new_status=draft reason=open_questions_resolved
    dashboard_distribution_test.go:74: blocked count: want 1, got 0
--- FAIL: TestStatusDistribution_CorrectCounts (0.14s)

# TestDashboardStats_MixedStatuses
    dashboard_stats_test.go:79: field "blocked": want 2, got 1
--- FAIL: TestDashboardStats_MixedStatuses (0.13s)
```

## Fix Guidance

In `tests/integration/dashboard_stats_test.go` and `tests/integration/dashboard_distribution_test.go`, any seed artifact that must remain in `blocked` status throughout the test must include an `## Open Questions` section in its body (so the auto-unblock rule does not fire). Either:

- Extend `makeArtifact` to accept an optional body string that callers can pass containing `## Open Questions\n- Why is the sky blue?`, or
- Add a dedicated `makeBlockedArtifact` helper that injects the section automatically.
