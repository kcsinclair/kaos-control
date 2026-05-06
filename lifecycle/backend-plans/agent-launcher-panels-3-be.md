---
title: 'Backend Plan: Agent Launcher Panels'
type: plan-backend
status: done
lineage: agent-launcher-panels
parent: lifecycle/requirements/agent-launcher-panels-2.md
assignees:
    - role: product-owner
      who: agent
---

## Overview

Expose `model` and `active_status` fields in the `GET /agents` API response so the frontend can render agent panels with full metadata and compute artifact eligibility for the launch flow. This is a small, additive change — the data already exists in `AgentConfig`; it just needs to be surfaced through the HTTP layer.

Related: [[agent-launcher-panels]]

## Milestone 1 — Extend `agentSummary` in the agent list handler

### Description

Add `model` and `active_status` to the inline `agentSummary` struct in `handleListAgents` and populate them from `AgentConfig`. This satisfies FR-5 of the requirement.

### Files to change

- `internal/http/agents.go` — Modify the `agentSummary` struct (currently lines 21-26) to add two fields:
  ```go
  type agentSummary struct {
      Name         string   `json:"name"`
      Roles        []string `json:"roles"`
      Driver       string   `json:"driver"`
      Model        string   `json:"model,omitempty"`
      ActiveStatus string   `json:"active_status,omitempty"`
      AllowedPaths []string `json:"allowed_write_paths,omitempty"`
  }
  ```
- In the same file, update the loop that builds the `out` slice (lines 28-34) to populate the two new fields from `ag.Model` and `ag.ActiveStatus`.

### Acceptance criteria

- [ ] `GET /api/p/:project/agents` returns `model` and `active_status` for each agent in the JSON response.
- [ ] Agents without a model or active_status omit those fields (per `omitempty`).
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] Existing agent-related endpoints (`POST /agents/{name}/run`, `GET /agents/runs`, etc.) are unaffected.

## Milestone 2 — Verify artifact status filtering works end-to-end

### Description

The requirement (FR-6) states the frontend must be able to retrieve artifacts filtered by status. The existing `GET /artifacts` handler already reads `?status=` and `?type=` query parameters and passes them to `index.Filter`. No code change is expected here — this milestone is a verification step to confirm the existing behaviour is correct and complete.

### Files to check (no changes expected)

- `internal/http/artifacts.go` — `handleListArtifacts` (lines 18-51): confirm `status` and `type` are wired into `index.Filter`.
- `internal/index/index.go` — `Filter` struct and `List()` method: confirm both fields are applied in the SQL query.

### Acceptance criteria

- [ ] `GET /api/p/:project/artifacts?status=draft` returns only artifacts with `status: draft`.
- [ ] `GET /api/p/:project/artifacts?status=draft&type=idea` returns only ideas in draft status.
- [ ] If filtering already works correctly, no code changes are committed for this milestone.
- [ ] If filtering is broken or missing for either parameter, fix it and commit.

## Resolved Questions

1. The requirement's Open Question 2 asks whether the artifact list should also filter by `type`. The backend already supports `?type=` filtering, so the frontend can opt in without backend changes. No backend decision needed — this is a [[agent-launcher-panels]] frontend plan concern.

> an approved artifact is ready for work.
