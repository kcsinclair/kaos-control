---
title: "Frontend Plan: Display Analyst In-Progress Statuses"
type: plan-frontend
status: in-development
lineage: analyst-missing-in-progress-status
parent: lifecycle/defects/analyst-missing-in-progress-status.md
labels:
    - agent
    - workflow
    - frontend
---

# Frontend Plan: Display Analyst In-Progress Statuses

The analyst agents will now set `clarifying` and `planning` as active statuses on target artifacts during execution (see [[analyst-missing-in-progress-status-2-be]]). The frontend already handles these statuses in most places — they are part of the known status vocabulary — but this plan verifies that all UI surfaces render them with appropriate visual treatment, consistent with how `in-development` and `in-qa` are displayed.

## Milestone 1: Verify Status Badge Rendering for `clarifying` and `planning`

### Description

The `tokens.css` file defines badge colour tokens for several statuses (`done`, `approved`, `in-progress`, `blocked`, `rejected`, `in-qa`, `in-dev`) but does not have explicit tokens for `clarifying` or `planning`. Verify how the badge component handles statuses without dedicated tokens and confirm they receive a reasonable default style. If they fall through to an unstyled default, add dedicated tokens.

### Files to Change

- `web/src/styles/tokens.css` — add `--badge-clarifying-bg`, `--badge-clarifying-text`, `--badge-planning-bg`, `--badge-planning-text` tokens if not already handled by a fallback. Use colours that convey "active work" without conflicting with existing status colours:
  - `clarifying`: light blue tones (e.g. `#eff6ff` / `#1e40af` light, `#1e3a5f` / `#93c5fd` dark) to suggest investigation/analysis
  - `planning`: light indigo/violet tones (e.g. `#eef2ff` / `#4338ca` light, `#2e1a4a` / `#c4b5fd` dark) to suggest planning/design
- Badge component (likely in `web/src/components/artifact/` or a shared component) — verify the CSS class mapping covers `clarifying` and `planning`

### Acceptance Criteria

- [ ] `clarifying` status badges render with a distinct, readable colour scheme in both light and dark mode
- [ ] `planning` status badges render with a distinct, readable colour scheme in both light and dark mode
- [ ] Badge colours for `clarifying` and `planning` are visually distinguishable from each other and from `in-development`, `in-qa`, and `draft`
- [ ] No visual regressions to existing status badges

## Milestone 2: Verify Graph Node Colours for `clarifying` and `planning`

### Description

The 3D and 2D graph views colour nodes by status using the map in `web/src/components/graph/graphConstants.ts`. Verify that `clarifying` and `planning` have entries in the `statusColors` map. If they fall through to a default colour, add explicit entries.

### Files to Change

- `web/src/components/graph/graphConstants.ts` — add entries to the status colour map:
  - `'clarifying': '#60a5fa'` (blue, matching badge theme)
  - `'planning': '#a78bfa'` (violet, matching badge theme)

### Acceptance Criteria

- [ ] Graph nodes for artifacts in `clarifying` status render with a blue colour
- [ ] Graph nodes for artifacts in `planning` status render with a violet colour
- [ ] Colours are visually distinct from `draft` (grey), `in-development` (green), and `in-qa` (amber)
- [ ] Both 3D (three.js) and 2D (Cytoscape) graphs pick up the new colours

## Milestone 3: Verify Artifact List and Workspace Views

### Description

The `ArtifactListView.vue` and `WorkspaceView.vue` display artifact status in tables and cards. Verify that `clarifying` and `planning` appear correctly in filters, sort ordering, and status columns. These views likely use the same badge component verified in Milestone 1.

### Files to Change

- `web/src/views/project/ArtifactListView.vue` — verify status filter dropdown includes `clarifying` and `planning` (these are likely populated dynamically from the backend's status vocabulary)
- `web/src/views/project/WorkspaceView.vue` — verify agent run status display shows the real-time status change when an analyst agent starts

### Acceptance Criteria

- [ ] `clarifying` and `planning` appear in the status filter dropdown on the artifact list
- [ ] Filtering by `clarifying` or `planning` returns the correct artifacts
- [ ] When an analyst agent starts and the WebSocket broadcasts `artifact.indexed`, the artifact's status updates in real-time in the list and workspace views

## Milestone 4: Verify Transition Dialog

### Description

The `TransitionDialog.vue` component shows allowed status transitions for an artifact. When an artifact is in `clarifying` or `planning` status, the dialog should show the correct set of allowed transitions based on the user's roles. Verify no changes are needed (the dialog fetches allowed targets from the backend workflow engine).

### Files to Change

- `web/src/components/artifact/TransitionDialog.vue` — verify only; likely no changes needed since transitions are computed server-side

### Acceptance Criteria

- [ ] An artifact in `clarifying` status shows valid transition targets (e.g. `draft`, `planning`, `blocked`, `rejected`, `abandoned`)
- [ ] An artifact in `planning` status shows valid transition targets (e.g. `in-development`, `blocked`, `rejected`, `abandoned`)
- [ ] The dialog renders correctly with no layout issues for the new statuses

## Cross-References

- [[analyst-missing-in-progress-status-2-be]] — backend config changes that enable these statuses; must be deployed first
- [[analyst-missing-in-progress-status-4-test]] — integration tests verify the full round-trip from agent start → status change → UI update
