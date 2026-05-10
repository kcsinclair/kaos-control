---
title: 'Backend Plan: Agent Ready-Count Endpoint'
type: plan-backend
status: done
lineage: agent-panel-status-and-ready-count
parent: lifecycle/requirements/agent-panel-status-and-ready-count-2.md
---

## Overview

Add a `GET /api/p/:project/agents/ready-counts` endpoint that returns per-agent artifact counts based on each agent's `active_status` configuration. This enables [[agent-panel-status-and-ready-count]] frontend badges showing queue depth per agent.

## Milestone 1: Ready-Counts Handler

**Description:** Implement the HTTP handler that iterates configured agents, queries the index for each agent's `active_status`, and returns a counts map.

**Files to change:**
- `internal/http/agents.go` — add `handleGetReadyCounts` handler function
- `internal/http/server.go` — register route `GET /api/p/:project/agents/ready-counts` (add near existing agent routes, lines 156-162)

**Implementation details:**
- Load the project's agent configs from `project.Config().Agents`
- For each agent with a non-empty `ActiveStatus`, call `project.Index().Count(index.Filter{Status: agent.ActiveStatus})`
- Return JSON: `{"counts": {"agent-name": N, ...}}`
- Agents with empty `ActiveStatus` are omitted from the response

**Acceptance criteria:**
- [ ] `GET /api/p/:project/agents/ready-counts` returns 200 with correct shape
- [ ] Only agents with non-empty `active_status` appear in response
- [ ] Counts match actual artifacts with matching status in the index
- [ ] Response time under 50 ms for 10,000 indexed artifacts (simple COUNT queries)
- [ ] `go build ./...` and `go vet ./...` pass with no new errors

## Milestone 2: Badge Click Navigation Support

**Description:** Per resolved Q1, the badge should be clickable to navigate to a filtered artifact list. The backend already supports `?status=` filtering on `GET /api/p/:project/artifacts` via `index.Filter{Status: ...}`. No new endpoint needed, but verify the existing list endpoint correctly handles status filtering for all configured `active_status` values.

**Files to change:**
- None (existing endpoint sufficient) — verify only

**Acceptance criteria:**
- [ ] `GET /api/p/:project/artifacts?status=<active_status>` returns correct filtered results for each agent's configured status
- [ ] Response includes all artifacts matching the status regardless of type

## Milestone 3: Running-Count in Agent Runs Response

**Description:** Per resolved Q2, the running-state highlight should show a run count (not just boolean). The existing `GET /api/p/:project/agents/runs?status=running` endpoint already returns all running runs. The frontend can group by `agent_name` to derive per-agent counts. No backend change needed — this milestone confirms the existing endpoint is sufficient.

**Files to change:**
- None — existing `handleListAgentRuns` with `?status=running` filter is sufficient

**Acceptance criteria:**
- [ ] `GET /api/p/:project/agents/runs?status=running` returns runs with `agent_name` field populated
- [ ] Frontend can derive per-agent running count by grouping response by `agent_name`
