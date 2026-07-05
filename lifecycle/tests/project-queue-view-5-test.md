---
title: Project Queue View — Test Suite
type: test
status: draft
lineage: project-queue-view
parent: lifecycle/test-plans/project-queue-view-5-test.md
---

# Project Queue View — Test Suite

This test suite covers the implementation of the Project Queue View feature. It verifies that:

1. The backend queue API provides the necessary data for the frontend to filter by project
2. The frontend component correctly displays project-scoped queue data
3. Real-time updates work properly for the current project
4. The panel integrates correctly with the Agents view layout

## Files Covered

- `tests/integration/queue_api_test.go` - Backend API tests
- `tests/web/ProjectQueuePanel.test.ts` - Component unit tests  
- `tests/web/ProjectQueuePanel.realtime.test.ts` - Real-time update tests
- `tests/web/AgentsRunsView.projectQueue.test.ts` - Integration tests with Agents view

## Test Coverage

### Milestone 1 — Backend verification / regression guard
- GET /api/queue returns jobs with non-empty project field
- DELETE /api/queue/{id} cancels pending jobs and emits queue.cancelled
- Global queue integration tests continue to pass unchanged
- (Contingency only) GET /api/p/{project}/queue endpoint tests

### Milestone 2 — ProjectQueuePanel unit/component tests
- Panel shows correct project-scoped data for pending jobs
- Running section shows only jobs belonging to the current project
- Pending count indicator is accurate and accessible
- Artifact navigation works correctly
- Cancel action works properly
- Empty state is displayed when appropriate
- Paused state is handled correctly
- Link to global queue is present

### Milestone 3 — Real-time update tests (store-driven)
- Simulating queue.added for the current project adds job to panel
- Simulating queue.started moves job to running section when belongs to project
- Simulating queue.finished/cancelled removes job from waiting sections
- Events for different projects don't appear in the panel
- queue.paused/resumed toggles panel's paused affordance

### Milestone 4 — Layout, integration, and global-queue regression
- AgentsRunsView renders ProjectQueuePanel alongside existing elements
- Panel receives correct project route param
- Responsive layout tests
- Existing global-queue tests pass unchanged

## Test Files

- `tests/integration/project_queue_test.go` - Backend tests for project-scoped queue functionality
- `tests/web/ProjectQueuePanel.test.ts` - Unit tests for ProjectQueuePanel component
- `tests/web/ProjectQueuePanel.realtime.test.ts` - Real-time update tests for the panel
- `tests/web/AgentsRunsView.projectQueue.test.ts` - Integration tests with Agents view