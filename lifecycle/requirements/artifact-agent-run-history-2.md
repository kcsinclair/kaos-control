---
title: Artifact Agent Run History
type: requirement
status: approved
lineage: artifact-agent-run-history
created: "2026-04-28"
priority: normal
parent: lifecycle/ideas/artifact-agent-run-history.md
labels:
    - agent
    - artefacts
    - feature
    - frontend
    - backend
---

# Artifact Agent Run History

relates-to: [[artifact-agent-run-history]]

## Problem

Users viewing an artifact in the detail panel have no visibility into which agent runs have operated on that artifact. The `agent_runs` table already stores a `target_path` for every run, but there is no API to query runs by artifact and no UI to surface this data. Without this, users must navigate to a separate agent-runs listing and manually correlate runs to artifacts, breaking the auditability and traceability that the lifecycle model demands.

## Goals / Non-goals

### Goals

- Display a chronological list of agent runs associated with the currently viewed artifact, directly within the artifact detail panel.
- Allow users to click a run entry to view full run details in a modal overlay.
- Require no new data collection — use only existing `agent_runs` data joined via `target_path`.

### Non-goals

- Real-time streaming of in-progress agent output (covered by [[improved-agent-handling]]).
- Aggregated agent analytics or dashboards.
- Editing or re-triggering runs from the run-history list.
- Displaying runs for artifacts that were _produced_ by a run (only runs _targeted at_ the artifact are shown).

## Detailed Requirements

### Functional

#### FR-1: Backend — Query runs by target path

Add an index-layer method (e.g. `ListAgentRunsByTargetPath(targetPath string) ([]*AgentRunRow, error)`) that returns all `agent_runs` rows whose `target_path` matches the given path, ordered by `started_at DESC`.

#### FR-2: Backend — REST endpoint

Expose the query via the existing agents API surface. One of:

- A new route `GET /api/p/{project}/agents/runs?target_path={path}`, extending the existing `handleListAgentRuns` with an optional `target_path` query parameter, **or**
- A new dedicated route `GET /api/p/{project}/artifacts/{path}/runs`.

The response schema must match the existing `AgentRunRow` JSON shape. Pagination is not required in the initial implementation (agent run counts per artifact are expected to remain low).

#### FR-3: Frontend — Run history section in artifact detail panel

Add a collapsible section titled "Agent Runs" to the artifact detail panel. When expanded it displays a list of runs with:

- Run ID (truncated or abbreviated for readability)
- Agent name
- Date/time (`started_at`, formatted relative or absolute based on recency)
- Status badge (running / succeeded / failed)

The section must load lazily — fetch runs only when the artifact detail panel is opened or when the section is expanded (implementer's discretion). Show a loading indicator while the request is in flight and an empty-state message ("No agent runs for this artifact") when there are none.

#### FR-4: Frontend — Run detail modal

Clicking a run entry opens a modal displaying the full `AgentRunRow` fields:

- Run ID
- Agent name and role
- Target path
- Started at / Finished at (formatted)
- Status and exit code
- Stderr tail (rendered in a monospace/code block, scrollable)
- Artifacts produced (list of paths, if any)

The modal must be dismissible via close button, Escape key, and clicking the backdrop.

#### FR-5: Frontend — Live updates

If a WebSocket `agent.run.*` event is received for the currently displayed artifact's target path, the run list should update without requiring a manual refresh. A full re-fetch of the list on relevant WS events is acceptable.

### Non-functional

#### NFR-1: Performance

The `target_path` query must use an index. Add a SQLite index on `agent_runs(target_path)` if one does not already exist. Query response time must be under 50 ms for up to 100 runs per artifact.

#### NFR-2: No schema migration

The `agent_runs` table already contains all required columns. No schema changes beyond an optional index addition are permitted.

#### NFR-3: Accessibility

The modal must trap focus while open and restore focus on close. Run status badges must have accessible text (not colour alone).

## Acceptance Criteria

- [ ] `GET /api/p/{project}/agents/runs?target_path=<path>` (or equivalent) returns the correct set of runs for a given artifact, ordered newest-first.
- [ ] Requesting runs for an artifact with no history returns an empty array (`[]`), not an error.
- [ ] The artifact detail panel displays an "Agent Runs" section listing all runs targeted at that artifact.
- [ ] Each run entry shows run ID, agent name, date, and status.
- [ ] Clicking a run entry opens a modal with full run details including stderr tail and artifacts produced.
- [ ] The modal is dismissible via close button, Escape, and backdrop click.
- [ ] The run list updates automatically when a relevant WebSocket event is received for the viewed artifact.
- [ ] A SQLite index exists on `agent_runs(target_path)`.
- [ ] `go build ./...` and `go vet ./...` pass with backend changes.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass with frontend changes.

## Open Questions

- Should the run list also include runs that _produced_ the current artifact (i.e. where the artifact appears in `artifacts_produced_json`), or strictly only runs whose `target_path` matches? The idea specifies target-path matching; producing-run matching would require a more expensive query.

> for now target_path

- Should lineage-level matching be supported (show runs for _any_ artifact in the same lineage), or only exact file-path matching? Lineage matching would be more useful when an artifact has been superseded but the run history is still relevant.

> for now target_path
