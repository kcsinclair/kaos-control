---
title: "Backend Plan: Artefacts Agent Run Count Column"
type: plan-backend
status: approved
lineage: artefacts-agent-run-count-column
parent: lifecycle/requirements/artefacts-agent-run-count-column-2.md
---

# Backend Plan: Artefacts Agent Run Count Column

This plan implements FR1 (aggregate query) and FR2 (API response enrichment) from the requirement. The backend must expose an `agent_run_count` integer on every artefact in the list response, plus an `active_agent_status` string to support the "Agent Running" / "Work Queued" pill described in Q1.

## Milestone 1 — Add `AgentRunCountsByTargetPath` to the index package

### Description

Add a new method to `internal/index/index.go` that returns a `map[string]int` of `target_path -> total run count` across all statuses. Uses a single `GROUP BY` query against the already-indexed `agent_runs.target_path` column.

### Files to change

- `internal/index/index.go` — add method `AgentRunCountsByTargetPath() (map[string]int, error)`

### Implementation detail

```sql
SELECT target_path, COUNT(*) FROM agent_runs
WHERE target_path IS NOT NULL AND target_path != ''
GROUP BY target_path
```

Return a `map[string]int`; callers treat a missing key as 0.

### Acceptance criteria

- [ ] Method exists and compiles.
- [ ] Returns correct counts for artefacts with 0, 1, and N runs.
- [ ] Executes a single SQL statement (no N+1).
- [ ] `go vet` and `staticcheck` pass.

---

## Milestone 2 — Add `ActiveAgentStatusByTargetPath` to the index package

### Description

Add a method that returns a `map[string]string` of `target_path -> active status` where the status is `"running"` if any run for that path has `status = 'running'`, or `"queued"` if any run has `status = 'queued'` (and none are running), or empty string otherwise. This supports the pill indicator from Q1.

### Files to change

- `internal/index/index.go` — add method `ActiveAgentStatusByTargetPath() (map[string]string, error)`

### Implementation detail

```sql
SELECT target_path,
       MAX(CASE WHEN status = 'running' THEN 2
                WHEN status = 'queued'  THEN 1
                ELSE 0 END) AS priority
FROM agent_runs
WHERE target_path IS NOT NULL AND target_path != ''
  AND status IN ('running', 'queued')
GROUP BY target_path
```

Map priority 2 -> `"running"`, 1 -> `"queued"`, absent -> `""`.

### Acceptance criteria

- [ ] Method exists and compiles.
- [ ] Returns `"running"` when at least one run is running.
- [ ] Returns `"queued"` when runs are queued but none running.
- [ ] Returns empty string (missing key) when no active runs exist.
- [ ] Single SQL statement.

---

## Milestone 3 — Extend `ArtifactRow` and enrich the list response

### Description

Add `AgentRunCount int` and `ActiveAgentStatus string` fields to `ArtifactRow`. In the `handleListArtifacts` handler, call the two new methods after fetching the artefact list and populate the fields before serialising the JSON response.

### Files to change

- `internal/index/index.go` — add fields to `ArtifactRow`:
  ```go
  AgentRunCount     int    `json:"agent_run_count"`
  ActiveAgentStatus string `json:"active_agent_status,omitempty"`
  ```
- `internal/http/artifacts.go` — in `handleListArtifacts`, after the `idx.List()` call:
  1. Call `idx.AgentRunCountsByTargetPath()` to get counts map.
  2. Call `idx.ActiveAgentStatusByTargetPath()` to get active-status map.
  3. Loop over `items` and set `item.AgentRunCount = counts[item.Path]` and `item.ActiveAgentStatus = activeStatuses[item.Path]`.

### Acceptance criteria

- [ ] `GET /api/p/:project/artifacts` includes `agent_run_count` (integer, default 0) on every item.
- [ ] `GET /api/p/:project/artifacts` includes `active_agent_status` (`"running"`, `"queued"`, or omitted) on items with active runs.
- [ ] An artefact with no runs returns `agent_run_count: 0` and no `active_agent_status`.
- [ ] An artefact with 3 completed runs returns `agent_run_count: 3`.
- [ ] Counts include all statuses (done, failed, killed, killed-timeout, running, queued).
- [ ] The two batch queries do not cause perceptible latency (NFR1).
- [ ] `go vet` and `staticcheck` pass.
- [ ] Interacts with [[artefacts-agent-run-count-column]] frontend plan: the frontend reads `agent_run_count` and `active_agent_status` from the response.

---

## Milestone 4 — Verify `agent.finished` WebSocket event carries enough data

### Description

Confirm that the existing `agent.finished` event (broadcast from `internal/agent/agent.go`) already triggers the frontend to re-fetch the artefact list (via the `artifact.indexed` event chain). No backend changes should be needed here — the existing flow where an agent run completes, the artefact is re-indexed, and `artifact.indexed` is broadcast already causes the frontend to call `fetchList()` again, which will now include the updated count.

### Files to change

- None expected. If the flow is broken, fix the event broadcast in `internal/agent/agent.go` or `internal/queue/dispatcher.go`.

### Acceptance criteria

- [ ] After an agent run finishes, the next `GET /api/p/:project/artifacts` call returns the incremented `agent_run_count`.
- [ ] The `artifact.indexed` WebSocket event fires after the agent run completes, triggering the frontend refresh (NG3 — no real-time mid-run updates needed).
