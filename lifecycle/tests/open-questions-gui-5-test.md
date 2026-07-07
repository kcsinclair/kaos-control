---
title: "Test Plan: Open-Questions Resolution GUI"
type: test
status: draft
lineage: open-questions-gui
parent: lifecycle/test-plans/open-questions-gui-5-test.md
---

# Test Suite Coverage

This test suite covers the complete end-to-end integration of the Open-Questions Resolution GUI feature, implementing all requirements from [[open-questions-gui]] (requirement `open-questions-gui-2`). 

## Test Files

### Milestone 1 — Configuration & parser/builder API
- `tests/open_questions_config_test.go` - Tests configuration endpoint and answer format handling
- `tests/open_questions_parse_test.go` - Tests question parsing and build preview functionality

### Milestone 2 — End-to-end resolve → unblock
- `tests/open_questions_resolve_e2e_test.go` - Tests full end-to-end resolution flow including partial saves, completion, and auto-unblocking

### Milestone 3 — Awaiting-answers count/list & permissions  
- `tests/open_questions_awaiting_test.go` - Tests awaiting answers query surface and product-owner authorization rules

### Milestone 4 — Post-resolution routing
- `tests/open_questions_routing_test.go` - Tests post-resolution routing behavior (approve vs requeue)

## Coverage Summary

The tests verify:
1. **Configuration**: Answer format defaults to blockquote, can be overridden, and unknown values fall back gracefully
2. **Parsing**: Questions are correctly parsed from markdown, existing answers are preserved, malformed sections return empty arrays
3. **Preview/Build**: Answers are correctly inserted with proper formatting, complete operation renames heading to "## Resolved Questions"
4. **End-to-end flow**: Partial saves remain blocked, complete saves unblock to draft status, idempotency works correctly
5. **Awaiting answers**: Count query returns correct number of blocked artifacts, list query filters properly, permission checks work
6. **Routing**: Requirement artifacts route to draft (awaiting approval), developer artifacts stay in draft (awaiting requeue), no automatic actions occur

All tests follow the integration test pattern using the testEnv infrastructure with auto-login as admin and proper HTTP request handling.

## Implementation Notes

The implementation covers all acceptance criteria from the test plan:

1. **Configuration & parser/builder API** - Tests verify config endpoint returns correct answer format, parse endpoints return correct questions, and preview/apply functions work correctly
2. **End-to-end resolve → unblock** - Tests verify full flow including partial saves staying blocked, complete saves transitioning to draft, and idempotency
3. **Awaiting-answers count/list & permissions** - Tests verify awaiting count query works, filtering works, and permission checks enforce product-owner role
4. **Post-resolution routing** - Tests verify that resolution properly routes artifacts (requirements go to draft, developer artifacts stay in draft) with no automatic transitions

Each test file contains multiple tests covering different aspects of the feature to ensure comprehensive coverage.