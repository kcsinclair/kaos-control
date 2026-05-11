---
title: "Backend: Per-Agent Role-Specific Ready Counts"
type: plan-backend
status: approved
lineage: agent-panel-ready-count-not-role-specific
parent: lifecycle/defects/agent-panel-ready-count-not-role-specific.md
---

# Backend: Per-Agent Role-Specific Ready Counts

## Problem Summary

The `handleGetReadyCounts` endpoint in `internal/http/agents.go` filters artifacts only by each agent's `active_status` field. Agents that share the same `active_status` (e.g. `backend-developer`, `frontend-developer`, and `test-developer` all use `in-development`) produce identical counts because no artifact type filtering is applied.

The fix requires adding a `source_types` field to the agent configuration so that each agent can declare which artifact types it consumes, then using both `Status` and `Type` in the index query.

---

## Milestone 1: Add `source_types` to AgentConfig

### Description

Add a `source_types` YAML field to the `AgentConfig` struct so operators can declare which artifact type(s) an agent consumes. This field is optional — agents without it retain the existing behaviour (count by status only).

### Files to Change

- `internal/config/config.go` — add `SourceTypes []string \`yaml:"source_types,omitempty"\`` field to `AgentConfig`.

### Acceptance Criteria

- [ ] `AgentConfig` has a `SourceTypes []string` field with YAML tag `source_types,omitempty`.
- [ ] Existing configs without the field still unmarshal without error.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2: Update `handleGetReadyCounts` to Filter by Type

### Description

Modify the ready-counts handler to use `index.Filter{Status: ag.ActiveStatus, Type: <type>}` when `SourceTypes` is populated. When an agent has multiple source types, sum the counts across each type.

### Files to Change

- `internal/http/agents.go` — update `handleGetReadyCounts` loop to iterate `ag.SourceTypes` and sum per-type counts, or fall back to status-only when `SourceTypes` is empty.

### Acceptance Criteria

- [ ] Agents with `source_types: [plan-backend]` return a count of artifacts matching `status=<active_status> AND type=plan-backend`.
- [ ] Agents with multiple `source_types` entries return the sum of counts across those types.
- [ ] Agents with no `source_types` fall back to the existing status-only count.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3: Wire `Agents()` Accessor to Expose SourceTypes

### Description

Ensure the `Agents()` method on the project config surfaces `SourceTypes` to callers (it already returns `AgentConfig` values, so this should work automatically once the field is added — verify).

### Files to Change

- `internal/config/config.go` — verify `Agents()` returns full `AgentConfig` including new field.
- `internal/agent/agent.go` — if `Run` or any agent-launch code references source types, add the field there too (likely not needed for counts alone).

### Acceptance Criteria

- [ ] `p.Agents.Agents()` returns configs with `SourceTypes` populated when the YAML has them.
- [ ] No compile errors across the codebase.

---

## Milestone 4: Update Project Config YAML

### Description

Add `source_types` entries to the agent declarations in `lifecycle/config.yaml` so each agent correctly declares its input type(s).

### Files to Change

- `lifecycle/config.yaml` — add `source_types` to each agent:
  - `requirements-analyst`: `[idea]`
  - `planning-analyst`: `[requirement]`
  - `backend-developer`: `[plan-backend]`
  - `frontend-developer`: `[plan-frontend]`
  - `test-developer`: `[plan-test]`
  - `qa`: `[test]`

### Acceptance Criteria

- [ ] Each agent in `lifecycle/config.yaml` has a `source_types` list matching its actual input artefact type(s).
- [ ] Application starts without config errors.
- [ ] The `/api/p/:project/agents/ready-counts` endpoint returns distinct values for `backend-developer`, `frontend-developer`, and `test-developer`.

---

## Cross-References

- [[agent-panel-ready-count-not-role-specific]] frontend plan — the frontend must switch from the shared `approvedCount` to consuming per-agent counts from this endpoint.
- [[agent-panel-ready-count-not-role-specific]] test plan — integration tests verify distinct counts per agent.
