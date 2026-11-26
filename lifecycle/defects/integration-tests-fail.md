---
title: Integration tests failed in internal/agent
type: defect
status: draft
lineage: adr-mediated-agent-driver-permission-model
parent: lifecycle/tests/agent-permission-precheck-6-test.md
labels: [defect]
created: 2026-09-01T19:00:00Z
assignees:
  - role: backend-developer
    who: agent
---

## Reproduction Steps
1. Run the integration test suite:
   `make test-integration`

## Expected Behaviour
All integration tests in `github.com/kaos-control/kaos-control/tests/integration` should pass.

## Actual Behaviour
The integration test suite failed (`FAIL github.com/kaos-control/kaos-control/tests/integration`).

## Logs / Output
```
FAIL
FAIL	github.com/kaos-control/kaos-control/tests/integration	307.461s
?   	github.com/kaos-control/kaos-control/tests/integration/testutil	[no test files]
?   	github.com/kaos-control/kaos-control/tests/integration/testutil/fake_precheck_claude	[no test files]
?   	github.com/kaos-control/kaos-control/web	[no test files]
FAIL
```
