---
title: 'Backend Plan: Update Recent Ideas and Defects Widget Limit to 7'
type: plan-backend
status: rejected
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/requirements/dashboard-recent-ideas-defects-widget-10.md
assignees:
    - role: product-owner
      who: agent
---

# Backend Plan: Update Recent Ideas and Defects Widget Limit to 7

**need to revisit this problem.....**

This plan covers backend-side changes required by [[dashboard-recent-ideas-defects-widget-10]]. The backend API itself requires no code changes — the `limit` query parameter already accepts arbitrary integers and the widget component already sends `limit: 7`. The work here is limited to updating the Go integration tests that assert the old `limit=6` value.

---

## Milestone 1: Update Go integration test — widget query limit assertions

### Description

The integration tests in `tests/integration/api_artifacts_widget_query_test.go` hard-code `limit=6` in query strings and assertion messages. These must be updated to `limit=7` to match the current widget behaviour.

### Files to change

- `tests/integration/api_artifacts_widget_query_test.go`
  - **`TestWidgetQuery_LimitApplied`**: Change query string from `limit=6` to `limit=7`. Update assertion that checks `len(items) <= 6` to `<= 7`, and the exact-count check from `6` to `7`. Update error message strings accordingly.
  - **`TestWidgetQuery_OnlyIdeasAndDefects`**: Change query string from `limit=6` to `limit=7`.
  - **`TestWidgetQuery_SortedByCreatedDesc`**: Change query string from `limit=6` to `limit=7`.
  - **`TestWidgetQuery_TotalIsFullMatchCount`**: Change query string from `limit=6` to `limit=7`. Update comment referencing `limit=6`.
  - **`TestWidgetQuery_FewerThanLimit`**: Change query string from `limit=6` to `limit=7`. Update comment referencing "limit of 6".
  - **`TestWidgetQuery_ZeroResults`**: Change query string from `limit=6` to `limit=7`.

### Acceptance criteria

- All six `TestWidgetQuery_*` test functions use `limit=7` in their HTTP request URLs.
- All assertion messages and comments reference `limit=7` (not `limit=6`).
- `go test ./tests/integration/ -run TestWidgetQuery` passes with all assertions green.
- No other integration tests are affected.

---

## Milestone 2: Update requirement artifact to reflect limit of 7

### Description

The original requirement [[dashboard-recent-ideas-defects-widget-2]] references "6 most recent" items in multiple places. These references must be updated to "7" so that the specification, code, and tests are all consistent.

### Files to change

- `lifecycle/requirements/dashboard-recent-ideas-defects-widget-2.md`
  - **Non-goals section**: "No filtering, searching, or pagination within the widget — it shows only the most recent 6 items." → Change `6` to `7`.
  - **Non-goals section**: "No backend API changes beyond what is needed to fetch the 6 most recent ideas and defects" → Change `6` to `7`.
  - **Functional requirement 1**: "Fetches the 6 most recent artifacts" → Change `6` to `7`.
  - **Data source**: `limit=6` in the example query parameter → Change to `limit=7`.
  - **Acceptance criteria**: "The widget displays up to 6 items" → Change `6` to `7`.

### Acceptance criteria

- Every occurrence of "6" referring to the item limit in [[dashboard-recent-ideas-defects-widget-2]] is replaced with "7".
- No unrelated content in the artifact is modified.
- The artifact remains valid markdown with correct frontmatter.

---

## Milestone 3: Verify no regression in existing backend tests

### Description

Run the full Go integration test suite to confirm that the limit change does not cause regressions in any other test.

### Files to change

- None (verification only).

### Acceptance criteria

- `go test ./tests/integration/...` passes with zero failures.
- `go test ./... -short` (unit tests) passes with zero failures.

---

## Rejected Questions

1. **Scope conflict — no `internal/**` or `cmd/**` changes exist in this plan.** The plan's own preamble states "the backend API itself requires no code changes." All milestones target files outside the backend developer agent's write scope:
   - Milestone 1 writes to `tests/integration/api_artifacts_widget_query_test.go` — `tests/` is out of scope for the backend developer.
   - Milestone 2 writes to `lifecycle/requirements/dashboard-recent-ideas-defects-widget-2.md` — `lifecycle/` is out of scope for backend developer writes.
   - Milestone 3 is verification only (no writes).

   **Questions for product-owner:**
   - Should Milestone 1 be reassigned to the `test-developer` agent, which owns `tests/`?
   - Should Milestone 2 be handled by the `analyst` agent or folded into an existing lifecycle-artifact update task?
   - Is there any `internal/**` or `cmd/**` change that was omitted from this plan? If so, please describe it so the plan can be corrected before the backend developer proceeds.
   - Alternatively, should this plan be rejected and its milestones absorbed by the test plan (`dashboard-recent-ideas-defects-widget-13-test`) and a separate analyst task?

2. **`status: blocked` is not a valid status value.** The valid status vocabulary (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`) does not include `blocked`. The system normalises the status field back to `draft` when `blocked` is written. Should a new `blocked` status be added to the vocabulary, or should a different mechanism (e.g., a `blocked_reason` field) be used to record that an artifact is waiting on product-owner input?

---

## Cross-references

- [[dashboard-recent-ideas-defects-widget-12-fe]] — Frontend plan (no widget code change; layout verification only).
- [[dashboard-recent-ideas-defects-widget-13-test]] — Test plan (Vitest assertion update + test-plan artifact update).
