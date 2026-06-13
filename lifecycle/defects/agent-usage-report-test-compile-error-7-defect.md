---
title: Integration tests fail to compile due to duplicate helper declarations
type: defect
status: approved
lineage: agent-usage-analytics-report
parent: lifecycle/tests/agent-usage-analytics-report-6-test.md
labels:
  - defect
assignees:
  - role: test-developer
    who: agent
---

# Integration tests fail to compile due to duplicate helper declarations

The integration test suite fails to compile because helper functions (like `setupFakeClaudeWithOutput`, `setupFakeClaudeWithScript`, and `seedAgentRun`) are declared multiple times within the same package directory (`tests/integration/`).

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/... -run="TestReportsAgentUsage|TestSupervisor|TestBackfill"` in the repository root.
2. Observe Go compilation failure due to duplicate function declarations.

## Expected Behaviour

The integration test suite compiles successfully. Helper functions that are shared across files in the same `integration` package should only be defined once, or have unique names to avoid namespace collisions.

## Actual Behaviour

The compilation fails with redeclaration errors because:
- `setupFakeClaudeWithOutput` is declared in both `agent_metrics_test.go` and `agent_ws_test.go` with conflicting signatures.
- `setupFakeClaudeWithScript` is declared in both `agent_metrics_test.go` and `queue_helpers_test.go`.
- `seedAgentRun` is declared in both `reports_api_test.go` and `agents_api_test.go` with conflicting signatures.

## Logs / Output

```
# github.com/kaos-control/kaos-control/tests/integration [github.com/kaos-control/kaos-control/tests/integration.test]
tests/integration/agent_ws_test.go:95:6: setupFakeClaudeWithOutput redeclared in this block
	tests/integration/agent_metrics_test.go:16:6: other declaration of setupFakeClaudeWithOutput
tests/integration/agent_ws_test.go:123:53: too many arguments in call to setupFakeClaudeWithOutput
	have (*testing.T, []string, number)
	want (*testing.T, string)
tests/integration/queue_helpers_test.go:402:6: setupFakeClaudeWithScript redeclared in this block
	tests/integration/agent_metrics_test.go:29:6: other declaration of setupFakeClaudeWithScript
tests/integration/reports_api_test.go:17:6: seedAgentRun redeclared in this block
	tests/integration/agents_api_test.go:307:6: other declaration of seedAgentRun
tests/integration/reports_api_test.go:48:27: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
```
