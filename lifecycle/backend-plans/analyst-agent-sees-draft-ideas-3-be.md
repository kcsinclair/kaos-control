---
title: "Agent Launcher Input Status Filtering — Backend Plan"
type: plan-backend
status: approved
lineage: analyst-agent-sees-draft-ideas
parent: lifecycle/requirements/analyst-agent-sees-draft-ideas-2.md
---

# Agent Launcher Input Status Filtering — Backend Plan

The requirement (NFR-1) explicitly states "No new API calls. The fix should not require additional API endpoints or backend changes. The existing `listArtifacts` query parameter `status=approved` is sufficient."

This plan documents the backend non-changes and confirms that the existing API surface supports the frontend fix described in [[analyst-agent-sees-draft-ideas]] frontend plan.

## Milestone 1: Confirm Existing API Supports Approved-Status Filtering

### Description

Verify that the existing `GET /p/:project/artifacts?status=approved` query parameter correctly filters artifacts to only those with `status: approved`. The frontend fix will switch from querying variable statuses (e.g. `status=draft` for the analyst agent) to always querying `status=approved`. No backend code changes are required — this milestone confirms the existing behaviour is sufficient.

### Files to Change

None.

### Acceptance Criteria

- [ ] `GET /p/:project/artifacts?status=approved` returns only artifacts whose frontmatter status is `approved`.
- [ ] `GET /p/:project/artifacts?status=approved&type=idea` correctly combines both filters, returning only approved ideas.
- [ ] `GET /p/:project/artifacts?status=approved&type=defect` returns only approved defects (used by developer agents for defect inclusion per FR-3).
- [ ] No backend Go code is modified for this feature.

## Milestone 2: Confirm Defect Filtering by Type and Status

### Description

Verify that the existing API supports the defect-inclusion query that developer agents need (FR-3). The frontend will make a second call `GET /p/:project/artifacts?status=approved&type=defect` to fetch approved defects. The assignee filtering (by role) is performed client-side from the `assignees` field in the artifact frontmatter. Confirm the API returns the `assignees` field in the response.

### Files to Change

None.

### Acceptance Criteria

- [ ] `GET /p/:project/artifacts?status=approved&type=defect` returns defect artifacts with `status: approved`.
- [ ] Each artifact in the API response includes the `frontmatter.assignees` array, allowing the frontend to filter by role.
- [ ] The existing response shape is unchanged — no new fields or envelope changes are introduced.

## Milestone 3: Document Future Hardening Opportunity

### Description

The requirement's Open Questions section notes that server-side validation of agent input status at `POST /agents/runs` is a desirable defence-in-depth measure, deferred from this fix. This milestone documents the current state: the backend does not validate that the artifact passed to an agent run has `status: approved`. This is tracked for a future release and requires no code changes now.

### Files to Change

None.

### Acceptance Criteria

- [ ] No server-side input status validation is added to `POST /agents/runs` in this change.
- [ ] The existing agent run endpoint continues to accept any artifact path without status checks (current behaviour preserved).
