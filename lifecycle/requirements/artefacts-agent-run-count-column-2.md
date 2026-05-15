---
title: 'Artefacts View: Agent Run Count Column'
type: requirement
status: approved
lineage: artefacts-agent-run-count-column
created: "2026-05-16"
priority: normal
parent: lifecycle/ideas/artefacts-agent-run-count-column.md
labels:
    - artefacts
    - frontend
    - agent
    - enhancement
    - feature
assignees:
    - role: product-owner
      who: agent
---

# Artefacts View: Agent Run Count Column

## Problem

The artefacts list view (`ArtifactListView.vue`) shows eight columns (path, stage, status, priority, release, type, created, modified) but provides no indication of how much agent activity has occurred against each artefact. Users must navigate into each artefact's detail view and expand the run-history panel to see whether any agent work has been performed.

This forces product owners and QA to click through artefacts one by one to identify items that have had zero agent runs — a common workflow when triaging progress or spotting workflow gaps.

## Goals / Non-goals

### Goals

- G1: Surface the total agent-run count for each artefact as a column in the artefacts table.
- G2: Make the column sortable so users can surface untouched artefacts (count = 0) or heavily-iterated items quickly.
- G3: Keep the implementation consistent with the existing table column patterns (sorting, formatting, responsiveness).

### Non-goals

- NG1: Filtering the table by run-count ranges (e.g. "show only artefacts with 0 runs") — may be added later.
- NG2: Showing per-agent-name breakdowns in the column (the detail view already has `ArtifactRunHistory`).
- NG3: Real-time live-updating of the count while an agent run is in progress — the count updates on the next artefact list refresh or WebSocket `agent.finished` event, which is sufficient.

## Detailed Requirements

### Functional

- **FR1 — Backend: aggregate query.** Add a method to the index package (e.g. `AgentRunCountsByTargetPath`) that returns a `map[string]int` of `target_path → run count` for all artefacts, using a single `SELECT target_path, COUNT(*) FROM agent_runs GROUP BY target_path` query. This avoids N+1 queries when the artefact list is loaded.
- **FR2 — Backend: API response enrichment.** The `GET /api/p/:project/artifacts` endpoint must include an `agent_run_count` integer field on each artefact object in the response. The value is 0 when no runs exist for that artefact's path. The counts must be fetched in a single batch query per request, not per-artefact.
- **FR3 — Frontend: new table column.** Add an "Agent Runs" column to the artefacts table in `ArtifactListView.vue`. The column displays the integer count. A count of 0 should be displayed as `0` (not blank or a dash).
- **FR4 — Frontend: column sorting.** The "Agent Runs" column must be sortable (ascending / descending) using the same sorting mechanism as the existing columns (e.g. Created, Modified).
- **FR5 — Frontend: column position.** The "Agent Runs" column should appear after the "Type" column and before the "Created" column, keeping date columns together at the right edge.

### Non-functional

- **NFR1 — Performance.** The aggregate query must not add perceptible latency to the artefact list load. A single `GROUP BY` query on the indexed `target_path` column satisfies this.
- **NFR2 — Consistency.** The column styling (font, alignment, padding) must match existing numeric columns (if any) or use right-aligned text consistent with numeric data conventions.
- **NFR3 — Responsiveness.** On narrow viewports the column may be hidden or truncated following the same responsive behaviour as other lower-priority columns.

## Acceptance Criteria

- [ ] `GET /api/p/:project/artifacts` response includes `agent_run_count` (integer) on every artefact object.
- [ ] An artefact with no agent runs returns `agent_run_count: 0`.
- [ ] An artefact with 3 completed agent runs returns `agent_run_count: 3` (counts all statuses: done, failed, killed, etc.).
- [ ] The artefacts table displays an "Agent Runs" column showing the integer count for each row.
- [ ] Clicking the "Agent Runs" column header sorts the table by run count ascending; clicking again sorts descending.
- [ ] After an agent run finishes (WebSocket `agent.finished` event triggers list refresh), the count increments without a full page reload.
- [ ] The aggregate query uses a single SQL statement (no N+1 per artefact).
- [ ] `go vet` and `staticcheck` pass after backend changes.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass after frontend changes.
- [ ] Related: [[artefacts-agent-run-count-column]]

## Resolved Questions

- Q1: Should the count include all run statuses (done, failed, killed, killed-timeout, running) or only completed runs? The idea says "total number of times an agent has been run" which implies all statuses. Recommend: count all statuses.

> It should be the total of all runs.  If there is a job running or queued for an artefact, something in the row should indicate that.  A pill beside the name work work, "Agent Running" or "Work Queued"

- Q2: Should the column header be "Agent Runs", "Runs", or something shorter? Recommend: "Runs" for compactness with a tooltip showing "Agent Run Count".

> Runs works.
