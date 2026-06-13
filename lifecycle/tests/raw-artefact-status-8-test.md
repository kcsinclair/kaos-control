---
title: "Tests: fix duplicate helper declarations causing integration build failure"
type: test
status: draft
lineage: raw-artefact-status
parent: lifecycle/defects/raw-artefact-status-7-defect.md
---

# Tests: fix duplicate helper declarations causing integration build failure

Fixes three symbol redeclaration errors that prevented the integration test package
from compiling (`go test ./tests/integration/... -tags=integration`).

## Changes made

No new test code was added. The existing code was renamed to eliminate duplicate
declarations that had accumulated as tests were added across multiple agent runs:

### 1. `setupFakeClaudeWithOutput` / `setupFakeClaudeWithLines`

- **Conflict**: `agent_metrics_test.go:16` declared `setupFakeClaudeWithOutput(t, ndjsonOutput string)`.  
  `agent_ws_test.go:95` declared another `setupFakeClaudeWithOutput` with a different signature: `(t, []string, int)`.
- **Fix**: The `agent_ws_test.go` version was renamed to `setupFakeClaudeWithLines(t, outputLines []string, exitCode int)`,
  which better describes its behaviour (writes each line to a temp file and cats it).  
  All callers in `agent_ws_test.go` were updated accordingly.

### 2. `setupFakeClaudeWithScript` / `setupFakeClaudeWithRawScript`

- **Conflict**: `queue_helpers_test.go:402` declared `setupFakeClaudeWithScript(t, script string)`.  
  `agent_metrics_test.go:29` declared another `setupFakeClaudeWithScript` with the same signature.
- **Fix**: The `agent_metrics_test.go` version was renamed to `setupFakeClaudeWithRawScript` to distinguish
  it from the queue variant (the metrics version prepends `#!/bin/sh\n` automatically).  
  All callers in `agent_metrics_test.go` were updated.

### 3. `seedAgentRun` / `seedAgentRunRow`

- **Conflict**: `reports_api_test.go:17` declared `seedAgentRun(t, env, id, agent string, startedAt time.Time, status string)`.  
  `agents_api_test.go:307` declared `seedAgentRun(t, env, *index.AgentRunRow)` — a struct-based variant.
- **Fix**: The `agents_api_test.go` version was renamed to `seedAgentRunRow` to clarify that it
  accepts a fully-populated `*index.AgentRunRow`.  
  All callers in `agents_api_test.go` were updated.

## Verification

```bash
go build -tags=integration ./tests/integration/...   # no errors
go test ./tests/integration/... -tags=integration -list '.*'  # all tests listed
```

Both commands succeed. The package builds and all tests are enumerable.

## Affected files

- `tests/integration/agent_ws_test.go` — renamed `setupFakeClaudeWithOutput` → `setupFakeClaudeWithLines`
- `tests/integration/agent_metrics_test.go` — renamed `setupFakeClaudeWithScript` → `setupFakeClaudeWithRawScript`
- `tests/integration/agents_api_test.go` — renamed `seedAgentRun` → `seedAgentRunRow`
