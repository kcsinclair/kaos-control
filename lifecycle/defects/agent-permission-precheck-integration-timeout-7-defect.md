---
title: TestAgentPrecheck integration tests hang and timeout suite
type: defect
status: draft
lineage: agent-permission-precheck
parent: lifecycle/tests/agent-permission-precheck-6-test.md
labels: [defect]
created: 2026-09-01T18:05:00Z
assignees:
  - role: backend-developer
    who: agent
---

# TestAgentPrecheck integration tests hang and timeout suite

The Go integration test suite `make test-integration` times out after 5 minutes because tests in `tests/integration/agent_precheck_test.go` hang waiting for agent runs to terminate.

## Reproduction Steps

1. Run the integration test suite:
   ```bash
   make test-integration
   ```
2. Observe command times out after 5m0s in `tests/integration`.

## Expected Behaviour

`make test-integration` executes all integration tests including `TestAgentPrecheck_*` suites and completes successfully within standard test timeouts.

## Actual Behaviour

The test command exceeds the 5-minute timeout threshold and fails with:
```
command timed out after 5m0s
FAIL	github.com/kaos-control/kaos-control/tests/integration	317.515s
```

## Logs / Output

```
go test ./... -count=1 -tags=integration
ok  	github.com/kaos-control/kaos-control/cmd	0.384s [no tests to run]
ok  	github.com/kaos-control/kaos-control/cmd/kaos-control/devopscmd	0.675s
ok  	github.com/kaos-control/kaos-control/internal/agent	32.099s
ok  	github.com/kaos-control/kaos-control/internal/architecture	1.201s
ok  	github.com/kaos-control/kaos-control/internal/architecture/catalogfs	2.146s
ok  	github.com/kaos-control/kaos-control/internal/artifact	2.569s
ok  	github.com/kaos-control/kaos-control/internal/auth	1.960s
ok  	github.com/kaos-control/kaos-control/internal/config	1.825s
ok  	github.com/kaos-control/kaos-control/internal/devops	3.902s
ok  	github.com/kaos-control/kaos-control/internal/directives	3.882s
ok  	github.com/kaos-control/kaos-control/internal/docs	4.220s
ok  	github.com/kaos-control/kaos-control/internal/git	4.653s
ok  	github.com/kaos-control/kaos-control/internal/http	5.604s
ok  	github.com/kaos-control/kaos-control/internal/ideachat	5.063s
ok  	github.com/kaos-control/kaos-control/internal/index	5.425s
ok  	github.com/kaos-control/kaos-control/internal/initcmd	4.615s
ok  	github.com/kaos-control/kaos-control/internal/project	5.029s
ok  	github.com/kaos-control/kaos-control/internal/queue	6.151s
ok  	github.com/kaos-control/kaos-control/internal/release	5.206s
ok  	github.com/kaos-control/kaos-control/internal/reports	4.650s
ok  	github.com/kaos-control/kaos-control/internal/scheduler	9.231s
ok  	github.com/kaos-control/kaos-control/internal/statuscheck	5.059s
ok  	github.com/kaos-control/kaos-control/internal/testrunner	5.221s
ok  	github.com/kaos-control/kaos-control/internal/triage	4.602s
ok  	github.com/kaos-control/kaos-control/internal/watcher	8.078s
ok  	github.com/kaos-control/kaos-control/internal/workflow	4.570s
ok  	github.com/kaos-control/kaos-control/tests	65.672s
FAIL
FAIL	github.com/kaos-control/kaos-control/tests/integration	317.515s
```
