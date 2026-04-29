---
title: "Backend Plan: Artifact Agent Run History"
type: plan-backend
status: done
lineage: artifact-agent-run-history
parent: lifecycle/requirements/artifact-agent-run-history-2.md
created: "2026-04-28"
---

# Backend Plan: Artifact Agent Run History

relates-to: [[artifact-agent-run-history]]

## Overview

Add backend support for querying agent runs by target path: a new SQLite index, a new index-layer method, an extended REST handler, and enriched WebSocket payloads. The [[artifact-agent-run-history-4-fe]] frontend plan depends on the API and WS changes delivered here.

---

## Milestone 1 — SQLite index on `agent_runs(target_path)`

### Description

Add a secondary index so target-path queries avoid full table scans and meet the <50 ms NFR.

### Files to change

- `internal/index/index.go` — inside `ensureAgentRunsTable()` (around line 1071), append a `CREATE INDEX IF NOT EXISTS` statement for `agent_runs(target_path)`.

### Acceptance criteria

- `EXPLAIN QUERY PLAN SELECT … FROM agent_runs WHERE target_path = ?` shows `USING INDEX idx_agent_runs_target_path`.
- Existing runs are unaffected (the index is additive; no migration required).
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Index-layer query method

### Description

Add `ListAgentRunsByTargetPath` to the `Index` type, returning runs whose `target_path` matches a given path, ordered by `started_at DESC`.

### Files to change

- `internal/index/index.go` — new method after `ListAgentRuns` (line ~833):
  ```go
  func (idx *Index) ListAgentRunsByTargetPath(targetPath string) ([]*AgentRunRow, error)
  ```
  Uses the same `scanAgentRunRow` helper. Query:
  ```sql
  SELECT run_id, agent_name, role, target_path, started_at, finished_at, status, exit_code, stderr_tail, artifacts_produced_json
  FROM agent_runs WHERE target_path = ? ORDER BY started_at DESC
  ```

### Acceptance criteria

- Returns all matching runs, newest first.
- Returns an empty slice (not nil) when no runs match.
- No new dependencies introduced.

---

## Milestone 3 — Agent Manager wrapper

### Description

Expose the new index method through the `Manager` so HTTP handlers don't reach into the index directly.

### Files to change

- `internal/agent/agent.go` — new method after `ListRuns` (line ~577):
  ```go
  func (m *Manager) ListRunsByTargetPath(targetPath string) ([]*index.AgentRunRow, error) {
      return m.idx.ListAgentRunsByTargetPath(targetPath)
  }
  ```

### Acceptance criteria

- Method delegates to the index layer without additional logic.
- `go build ./...` passes.

---

## Milestone 4 — REST endpoint: filter runs by `target_path`

### Description

Extend the existing `GET /api/p/{project}/agents/runs` handler to accept an optional `target_path` query parameter. When provided, it returns only runs targeted at that path instead of all project runs.

### Files to change

- `internal/http/agents.go` — in `handleListAgentRuns` (line 84–99):
  1. Read `target_path` from `r.URL.Query().Get("target_path")`.
  2. If non-empty, call `p.Agents.ListRunsByTargetPath(targetPath)` and return early.
  3. Otherwise fall through to the existing `ListRuns(status, limit)` path.

### Acceptance criteria

- `GET /api/p/{project}/agents/runs?target_path=lifecycle/requirements/foo-2.md` returns only runs whose `target_path` equals the given value.
- Response shape remains `{"runs": [...]}` with the same `AgentRunRow` JSON fields.
- Empty result returns `{"runs": []}`, not an error.
- Existing behaviour without `target_path` is unchanged (status/limit filtering still works).
- `go vet ./...` passes.

---

## Milestone 5 — Enrich WebSocket payloads with `target_path`

### Description

The `agent.started`, `agent.finished`, and `agent.failed` hub broadcasts currently include `run_id`, `agent`, `lineage`, and `status` but omit `target_path`. The [[artifact-agent-run-history-4-fe]] frontend needs `target_path` to decide whether to refresh the run list for the currently viewed artifact.

### Files to change

- `internal/agent/agent.go`:
  - Line ~433 (`agent.started` broadcast) — add `"target_path": targetPath` to the payload map.
  - Line ~545 (`agent.finished` / `agent.failed` broadcast) — add `"target_path": run.TargetPath` to the payload map.

### Acceptance criteria

- All three event types (`agent.started`, `agent.finished`, `agent.failed`) include a `target_path` string field in their JSON payload.
- Existing payload fields (`run_id`, `agent`, `lineage`, `status`, `artifacts`) remain unchanged.
- Frontend WS consumers that ignore unknown fields are unaffected.
