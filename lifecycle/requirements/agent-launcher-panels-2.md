---
title: Agent Launcher Panels
type: requirement
status: planning
lineage: agent-launcher-panels
parent: lifecycle/ideas/agent-launcher-panels.md
---

## Problem

The Agents screen currently shows only a flat list of past/active runs. Users have no visibility into which agents are configured for the project, what role or model each uses, or which lifecycle stage each targets. Starting a run requires clicking a generic "Run Agent" button and manually filling in agent name and target path. This makes agent discovery poor and the launch flow high-friction.

## Goals / Non-goals

### Goals

- **G1** — Give users an at-a-glance overview of every configured agent (name, role, model) directly on the Agents screen.
- **G2** — Let users launch an agent run in two clicks: click a panel, select an eligible artifact, confirm.
- **G3** — Clearly indicate which agents are manually launchable vs externally driven (e.g. idea-capture).
- **G4** — Surface only artifacts that are eligible for the selected agent's lifecycle stage, reducing user error.

### Non-goals

- Agent configuration editing from the UI (agents are defined in `lifecycle/config.yaml`).
- Real-time status on each panel (e.g. "running" badge) — this is a separate enhancement.
- Drag-and-drop or reordering of panels.
- Filtering or searching within the panel row.

## Detailed Requirements

### Functional

**FR-1 — Agent panel row**
A horizontal row of compact panels is rendered above the runs table on the Agents screen. One panel per agent returned by `GET /agents`. Panels are ordered as they appear in the API response (which mirrors `config.yaml` order).

**FR-2 — Panel content**
Each panel displays:
- Agent name (primary label).
- Role(s) — comma-separated if multiple.
- Model identifier (e.g. "opus", "sonnet") when present; omit line if empty.

**FR-3 — Disabled panel for non-launchable agents**
Agents whose `driver` is `inline` must render as visually distinct (muted/greyed, no pointer cursor) and must not be clickable. A tooltip or subtitle should read "Externally driven" or similar.

**FR-4 — Launch flow on click**
Clicking a launchable panel opens a modal or popover containing:
1. A filtered list of artifacts eligible for that agent. Eligibility is determined by the agent's `active_status` field: list artifacts whose current `status` matches the value that would trigger this agent (i.e. one step before `active_status` in the workflow). If `active_status` is empty, fall back to listing all artifacts and let the user choose.
2. Each list item shows the artifact's title, lineage slug, current status, and relative path.
3. A "Run" confirmation button that calls `POST /agents/:name/run` with the selected artifact's path.
4. A "Cancel" button or click-outside to dismiss.

**FR-5 — Backend: expose model and active_status in agent list**
The `GET /agents` response must include `model` and `active_status` fields for each agent so the frontend can render them and compute eligibility. Update the `AgentSummary` struct and TypeScript type accordingly.

**FR-6 — Artifact eligibility query**
The frontend must be able to retrieve artifacts filtered by status. The existing `GET /artifacts` endpoint already supports `?status=` filtering. If it does not, add support. The frontend uses this to populate the launch modal's artifact list.

**FR-7 — Empty state**
If no agents are configured for the project, the panel row is not rendered. If a launchable agent has no eligible artifacts, the modal shows a message: "No eligible artifacts for this agent."

### Non-functional

**NFR-1 — Responsiveness**
The panel row must wrap gracefully on narrow viewports (< 768px) without horizontal scrolling.

**NFR-2 — Performance**
Panel rendering must add no extra API calls on page load beyond the existing `GET /agents` call. The artifact list in the modal is fetched on-demand when a panel is clicked.

**NFR-3 — Accessibility**
Panels must be keyboard-navigable (focusable, activatable with Enter/Space). Disabled panels must have `aria-disabled="true"`.

## Acceptance Criteria

- [ ] The Agents screen renders one panel per configured agent above the runs table.
- [ ] Each panel shows the agent's name, role(s), and model.
- [ ] Inline-driver agents (e.g. `idea-capture`) render as disabled and are not clickable.
- [ ] Clicking a launchable panel opens a modal listing eligible artifacts filtered by the agent's expected input status.
- [ ] Selecting an artifact and confirming triggers `POST /agents/:name/run` and the new run appears in the runs table.
- [ ] The `GET /agents` response includes `model` and `active_status` for each agent.
- [ ] `AgentSummary` TypeScript type includes `model` and `active_status` fields.
- [ ] When no eligible artifacts exist, the modal shows an appropriate empty-state message.
- [ ] Panels wrap on narrow screens without horizontal overflow.
- [ ] Panels are keyboard-accessible (Tab, Enter/Space).
- [ ] The existing `RunAgentDialog` remains functional as a fallback (not removed).
- [ ] Related: [[agent-launcher-panels]]

## Open Questions

1. **Artifact eligibility mapping** — The idea says "approved or ready state for that agent's stage." The agent config has `active_status` (the status the agent *sets* on the target when it starts a run), but there is no explicit "input status" field. Should eligibility be defined as artifacts one workflow step before `active_status`, or should a new `input_status` field be added to agent config? For this requirement, we assume the mapping is: show artifacts whose status is the workflow predecessor of `active_status` (e.g. if `active_status` is `clarifying`, show `draft` artifacts; if `active_status` is `in-development`, show `planning` artifacts). This should be validated against the workflow state machine.
2. **Scope of artifact list** — Should the modal list *all* artifacts matching the status, or only those whose `type` aligns with the agent's expected input type (e.g. `analyst-planner` only sees `requirement` type artifacts)? Recommend filtering by type as well for precision, but this needs confirmation.
