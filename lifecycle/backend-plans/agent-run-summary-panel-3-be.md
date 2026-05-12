---
title: 'Backend Plan: Agent Run Summary Panel'
type: plan-backend
status: in-development
lineage: agent-run-summary-panel
parent: lifecycle/requirements/agent-run-summary-panel-2.md
release: KC-Release1
---

# Backend Plan: Agent Run Summary Panel

relates-to: [[agent-run-summary-panel]]

## Overview

Enrich the backend to extract and serve the `type:result` summary data from completed agent run logs. The [[agent-run-summary-panel-4-fe]] frontend plan depends on the API changes delivered here. The existing log endpoint and WebSocket events need targeted additions — the agent runner and log format are **not** modified.

---

## Milestone 1 — Result line parser utility

### Description

Create a utility function that accepts raw log content (string) and extracts the last JSON line containing `"type":"result"`. Return the parsed fields as a typed struct. This keeps parsing logic in one place, reusable by the HTTP handler and (optionally) the supervisor.

### Files to change

- `internal/agent/result.go` — new file:
  ```go
  // RunResult holds the parsed fields from a Claude Code type:result JSON line.
  type RunResult struct {
      Subtype                  string             `json:"subtype"`
      TotalCostUSD             float64            `json:"total_cost_usd"`
      DurationMs               int64              `json:"duration_ms"`
      DurationApiMs            int64              `json:"duration_api_ms"`
      NumTurns                 int                `json:"num_turns"`
      Usage                    RunResultUsage     `json:"usage"`
      PermissionDenials        []json.RawMessage  `json:"permission_denials"`
      SessionID                string             `json:"session_id"`
  }

  type RunResultUsage struct {
      InputTokens              int64 `json:"input_tokens"`
      CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
      CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
      OutputTokens             int64 `json:"output_tokens"`
  }

  // ParseResultLine scans log content from the end and returns the parsed
  // RunResult, or nil and an error description if none is found/parseable.
  func ParseResultLine(logContent string) (*RunResult, error)
  ```
  - Scan backwards line-by-line for efficiency (result is the last or near-last line).
  - Use `encoding/json` for parsing; reject lines where `"type"` is not `"result"`.
  - Return `nil, errNoResultLine` when no result line exists (not a hard error — expected for Ollama runs).
  - Capture all fields from the result line, including `permission_denials` as raw JSON to allow for arbitrary structures.

### Acceptance criteria

- `ParseResultLine` correctly extracts the result from a real Claude Code log.
- Returns `nil` with a descriptive error for logs with no result line.
- Handles malformed JSON gracefully (returns error, never panics).
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — REST endpoint: run result summary

### Description

Add a new endpoint `GET /api/p/{project}/agents/runs/{run_id}/result` that reads the run log, parses the result line, and returns the structured summary JSON. This keeps the summary fetch separate from the raw log endpoint and avoids burdening the frontend with raw log parsing for the API-driven code path.

### Files to change

- `internal/http/agents.go` — add handler `handleGetAgentRunResult`:
  ```go
  func (s *Server) handleGetAgentRunResult(w http.ResponseWriter, r *http.Request)
  ```
  1. Get the run ID from the URL (`chi.URLParam(r, "run_id")`).
  2. Load the run record to verify it exists and is in terminal status. If still `running`, return `409 Conflict` with `{"error": "run is still in progress"}`.
  3. Read the log file via `p.Agents.LogPath(runId)`.
  4. Call `agent.ParseResultLine(logContent)`.
  5. If parsing fails (no result line), return `200` with `{"result": null, "reason": "no result line found"}`.
  6. If successful, return `200` with `{"result": { ... }}` containing the `RunResult` fields.

- `internal/http/routes.go` — register the new route:
  ```go
  r.Get("/agents/runs/{run_id}/result", s.handleGetAgentRunResult)
  ```
  Place it alongside the existing `/agents/runs/{run_id}/log` route.

### Acceptance criteria

- `GET /api/p/{project}/agents/runs/{run_id}/result` returns the parsed result for a completed Claude Code run.
- Returns `{"result": null, "reason": "..."}` for Ollama runs or runs with missing/corrupt result lines — never a 500 error.
- Returns `409` for runs still in `running` status.
- Returns `404` for unknown run IDs.
- Response shape is documented in-code with struct tags.
- `go vet ./...` passes.

---

## Milestone 3 — Include result summary in WebSocket finish events

### Description

When the supervisor broadcasts `agent.finished` or `agent.failed`, include the parsed `RunResult` in the event payload so the frontend can display the summary immediately without an additional API call (FR-5).

### Files to change

- `internal/agent/agent.go` — in the `supervise()` goroutine (around line 741–751 where finish events are broadcast):
  1. After the process exits, read the log file.
  2. Call `ParseResultLine(logContent)`.
  3. If successful, add a `"result"` key to the broadcast payload map containing the serialised `RunResult`.
  4. If parsing fails, add `"result": null` — do not block the broadcast.

### Acceptance criteria

- `agent.finished` events for Claude Code runs include a `result` object with `total_cost_usd`, `usage`, `duration_ms`, etc.
- `agent.failed` events also include `result` when a result line exists (agent may have written results before failing).
- `agent.finished`/`agent.failed` events for Ollama runs include `"result": null` — no error.
- Existing payload fields (`run_id`, `agent`, `lineage`, `status`, `artifacts`, `target_path`) remain unchanged.
- Parsing errors do not delay or prevent the broadcast.

---

## Milestone 4 — Unit tests for result parser

### Description

Add unit tests for `ParseResultLine` covering the expected inputs and edge cases.

### Files to change

- `internal/agent/result_test.go` — new file:

  1. **`TestParseResultLine_ValidResult`** — provide a multi-line log ending with a valid `type:result` JSON line. Assert all fields are parsed correctly.
  2. **`TestParseResultLine_NoResultLine`** — log with no `type:result` line. Assert `nil` result and non-nil error.
  3. **`TestParseResultLine_MalformedJSON`** — log ending with an invalid JSON line containing `"type":"result"`. Assert `nil` result and error.
  4. **`TestParseResultLine_ResultNotLastLine`** — result line appears mid-log with other lines after it. Assert the result is still found (scan backwards).
  5. **`TestParseResultLine_EmptyLog`** — empty string input. Assert `nil` result.
  6. **`TestParseResultLine_ZeroUsage`** — valid result line with all usage fields at zero. Assert parsed successfully with zero values.
  7. **`TestParseResultLine_PermissionDenials`** — result line with non-empty `permission_denials` array. Assert the array is preserved.

### Acceptance criteria

- All tests pass with `go test ./internal/agent/ -run TestParseResultLine`.
- Tests cover both happy path and edge cases identified in FR-1 and NFR-2.
- No external dependencies (no disk I/O, no network).
