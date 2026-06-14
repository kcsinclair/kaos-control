---
title: Integration test suite build failure due to duplicate helper declarations
type: defect
status: done
lineage: raw-artefact-status
parent: lifecycle/tests/raw-artefact-status-6-test.md
labels:
  - defect
assignees:
  - role: test-developer
    who: agent
---

# Integration test suite build failure due to duplicate helper declarations

The command to run the regression suite or integration tests for `raw` status (e.g., `go test ./tests/integration/... -tags=integration -run TestRawStatus`) fails to build because of multiple redeclared helper functions in the `integration` package.

## Reproduction Steps

1. Run the integration test suite:
   ```bash
   go test ./tests/integration/... -tags=integration
   ```
2. Observe the compilation failure.

## Expected Behaviour

The integration test suite compiles cleanly without errors.

## Actual Behaviour

The build fails with multiple redeclaration errors:
- `setupFakeClaudeWithOutput` redeclared in `tests/integration/agent_ws_test.go:95` (also in `agent_metrics_test.go:16`)
- `setupFakeClaudeWithScript` redeclared in `tests/integration/queue_helpers_test.go:402` (also in `agent_metrics_test.go:29`)
- `seedAgentRun` redeclared in `tests/integration/reports_api_test.go:17` (also in `agents_api_test.go:307`)

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
tests/integration/reports_api_test.go:77:35: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
tests/integration/reports_api_test.go:83:32: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
tests/integration/reports_api_test.go:147:32: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
tests/integration/reports_api_test.go:151:33: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
tests/integration/reports_api_test.go:152:32: too many arguments in call to seedAgentRun
	have (*testing.T, *testEnv, string, string, "time".Time, string)
	want (*testing.T, *testEnv, *"github.com/kaos-control/kaos-control/internal/index".AgentRunRow)
tests/integration/reports_api_test.go:152:32: too many errors
FAIL	github.com/kaos-control/kaos-control/tests/integration [build failed]
```
