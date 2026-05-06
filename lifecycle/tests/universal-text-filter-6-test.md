---
title: Universal Text Filter — Integration Tests
type: test
status: draft
lineage: universal-text-filter
parent: lifecycle/test-plans/universal-text-filter-5-test.md
---

# Universal Text Filter — Integration Tests

Integration test suite for the Universal Text Filter feature. Tests are split across six files in `tests/integration/`.

## Milestone 1 — Backend API (fully implemented)

**File:** `tests/integration/universal_text_filter_api_test.go`

Run with: `go test ./tests/integration/ -tags integration -run TestUniversalTextFilterAPI`

All 11 test cases from the test plan are implemented:

| Test function | Scenario |
|---|---|
| `TestUniversalTextFilterAPI_BasicSubstringMatch` | Seeds known artifacts; asserts `q=<substring>` returns only matching rows and `total` reflects filtered count |
| `TestUniversalTextFilterAPI_CaseInsensitivity` | `q=KANBAN` returns "Kanban View" (case-folded SQLite LIKE) |
| `TestUniversalTextFilterAPI_MatchesOnSlug` | `q=kanban-view` returns artifact with slug `kanban-view` |
| `TestUniversalTextFilterAPI_MatchesOnLineage` | `q=login-flow` returns all artifacts in that lineage |
| `TestUniversalTextFilterAPI_MatchesOnType` | `q=ticket` returns only ticket-type artifacts |
| `TestUniversalTextFilterAPI_MatchesOnStatus` | `q=draft` returns only artifacts whose indexed fields contain "draft" |
| `TestUniversalTextFilterAPI_CompositionWithDropdownFilters` | `q=kanban&status=draft` applies AND logic; planning-status artifact excluded |
| `TestUniversalTextFilterAPI_NoMatches` | `q=zzz_nonexistent_zzz` returns `items=[]` and `total=0` |
| `TestUniversalTextFilterAPI_EmptyQ` | `q=` returns same count as no `q` parameter |
| `TestUniversalTextFilterAPI_SpecialCharacters` | `q=100%25` matches literal `%`; `q=_idea` does not use `_` as a wildcard |
| `TestUniversalTextFilterAPI_PaginationReset` | `limit=3&offset=0` always starts from the first page |

The backend `q` filter is implemented in `internal/index/index.go` (`buildWhere`, line 1427) and unit-tested in `internal/index/filter_test.go`.

## Milestones 2–7 — UI / Browser tests (skeletons pending automation)

The following files contain one skeleton test function per test-plan scenario. All functions call `t.Skip` with a description of what the test must do once a browser automation framework (e.g. Playwright, chromedp, or rod) is integrated.

| File | Milestone | Scenarios |
|---|---|---|
| `tests/integration/universal_text_filter_list_test.go` | 2 — Artifact List view | 7 scenarios: input presence, real-time filtering, `<mark>` highlighting, clear button, dropdown composition, pagination reset, empty state |
| `tests/integration/universal_text_filter_kanban_test.go` | 3 — Kanban Board view | 5 scenarios: input presence, card hiding, empty-column indicator, dropdown composition, clear restores |
| `tests/integration/universal_text_filter_graph_test.go` | 4 — Graph view | 7 scenarios: input presence, node dimming, node highlighting, edge visibility, camera focus, clear restores, sidebar composition |
| `tests/integration/universal_text_filter_feed_test.go` | 5 — Project Feed view | 4 scenarios: input presence, entry filtering, event-type toggle composition, clear restores |
| `tests/integration/universal_text_filter_keyboard_test.go` | 6 — Keyboard & a11y | 6 scenarios: `/` focuses filter, `/` non-steal, Escape clears+blurs, aria-label on input, aria-label on clear button, keyboard-accessible clear |
| `tests/integration/universal_text_filter_perf_test.go` | 7 — Performance | 2 scenarios: 500-artifact ≤16 ms budget, debounce prevents jank |

Run all skeletons to confirm they compile and skip gracefully:

```
go test ./tests/integration/ -tags integration -run 'TestUniversalTextFilterList|TestUniversalTextFilterKanban|TestUniversalTextFilterGraph|TestUniversalTextFilterFeed|TestUniversalTextFilterKeyboard|TestUniversalTextFilterPerf'
```
