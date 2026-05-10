---
title: Agent Panel Status and Ready Count — Integration Tests
type: test
status: in-qa
lineage: agent-panel-status-and-ready-count
parent: lifecycle/test-plans/agent-panel-status-and-ready-count-5-test.md
---

## Summary

Integration tests for the agent panel status and ready-count feature, covering the backend `handleGetReadyCounts` endpoint and the agent start/finish WebSocket event lifecycle.

## Scope

These tests implement **Milestone 4** of the test plan (E2E integration smoke tests).
Milestones 1–3 (internal unit tests and frontend component tests) are outside the
`tests/**` write scope and are not included here.

## Test file

`tests/integration/agent_panel_status_test.go`

## Scenarios covered

### Ready-count endpoint correctness

| Test | What it verifies |
|------|-----------------|
| `TestReadyCounts_ReflectsIndexedArtifacts` | Agents with `active_status` return counts matching seeded artifacts; agents without `active_status` are absent from the response. Response shape is `{"counts": {...}}` with numeric values. |
| `TestReadyCounts_ZeroCountReturned` | Agents with `active_status` but no matching artifacts appear in the response with count `0`. |
| `TestReadyCounts_MultipleArtifactsSameStatus` | Five artifacts at the same status are correctly aggregated into a single count. |
| `TestReadyCounts_NoAgentsConfigured` | When no agents are defined in config, the endpoint returns `{"counts": {}}` (HTTP 200, empty object). |

### Real-time update via WebSocket

| Test | What it verifies |
|------|-----------------|
| `TestReadyCounts_RealtimeUpdateAfterArtifactIndexed` | Transitioning an artifact to an agent's `active_status` triggers an `artifact.indexed` hub event; re-fetching counts reflects the incremented value. |
| `TestReadyCounts_StatusChangeReflected` | Transitioning an artifact away from an agent's `active_status` decrements that agent's count and increments the count for the agent whose `active_status` matches the new status. |

### Agent start/finish lifecycle

| Test | What it verifies |
|------|-----------------|
| `TestAgentPanel_StartFinishLifecycle` | Starting an agent run broadcasts `agent.started` with correct `run_id` and `agent` fields; after completion, a terminal event (`agent.finished`) is broadcast with the matching `run_id`. Final run status via REST API is not `"running"`. |
| `TestAgentPanel_AgentStartedEventWhileRunning` | After `agent.started` is received, the run appears in `GET /agents/runs` with `status: "running"`. Test waits for completion before teardown. |

## Configuration used

Tests use `agentPanelCfgYAML` (defined in `tests/integration/agents_api_test.go`), which configures:

- `agent-with-model` — `active_status: clarifying`, driver `claude-code-cli`
- `agent-no-model` — `active_status: planning`, driver `claude-code-cli`
- `agent-no-active-status` — no `active_status` (must be absent from counts)
- `idea-capture` — `driver: inline`, no `active_status` (must be absent from counts)

Tests that exercise agent runs use `setupFakeClaude(t, 0)` to inject a stub `claude` binary that exits immediately, keeping test runtime low.
