---
title: Dashboard Home Screen — Defect Fix Tests (blocked artifact auto-transition)
type: test
status: approved
lineage: dashboard-home-screen
parent: lifecycle/defects/dashboard-home-screen-7-defect.md
---

# Dashboard Home Screen — Defect Fix Tests

Regression fix for dashboard stats and status-distribution tests that failed
because seeded `blocked`-status artifacts were silently auto-transitioned back
to `draft` by `internal/index/autoblock.go` before the HTTP handler ran.

## Root cause recap

`applyOpenQuestionTransition` fires the rule: no open questions AND
`status == blocked` → auto-transition to `draft`. Seed artifacts that used
a plain body (`"Body."`) triggered this rule, so neither endpoint ever
observed a `blocked` ticket.

## Fix

Added `makeBlockedArtifact` helper to `tests/integration/helpers_test.go`.
It wraps `makeArtifact` with `status: blocked` and a body that contains an
open-questions heading, preventing the autoblock rule from firing during
indexing. Callers pass `title`, `typ`, `lineage`, and `parent`; labels are
variadic.

## Scenarios covered

### `TestDashboardStats_MixedStatuses`

File: `tests/integration/dashboard_stats_test.go`

The `stats-blocked-1` seed now uses `makeBlockedArtifact` so the artifact
retains `status: blocked` through indexing. The assertion `blocked == 2`
(one `blocked` + one `clarifying` ticket) now passes correctly.

### `TestStatusDistribution_CorrectCounts`

File: `tests/integration/dashboard_distribution_test.go`

The `dist-blocked-1` seed now uses `makeBlockedArtifact` so the artifact
retains `status: blocked` through indexing. The assertion
`distribution["blocked"] == 1` now passes correctly.
