---
title: ProjectQueuePanel tests incorrectly assert on job ID in component text
type: defect
status: approved
lineage: project-queue-view
parent: lifecycle/tests/project-queue-view-5-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the Vitest suite for `ProjectQueuePanel.test.ts`:
   ```sh
   cd tests/web && pnpm test ProjectQueuePanel.test.ts
   ```
2. Observe 4 test case failures in `ProjectQueuePanel component`:
   - `shows only jobs belonging to the current project`
   - `shows running section when job belongs to project`
   - `shows pending jobs in position order`
   - `filters out jobs from other projects`

## Expected Behaviour

The test helper `makeJob` should generate jobs with unique `artifact_path` properties if the test needs to assert that a specific job is present in the DOM text, since `ProjectQueuePanel.vue` renders the `artifact_path` and `agent_name`, but does not render the internal `id` (e.g., `p1`, `p3`, `run-1`).

## Actual Behaviour

The tests call `makeJob` without overriding `artifact_path`, so all jobs default to `lifecycle/ideas/test.md`. The tests then assert `expect(wrapper.text()).toContain('p3')` (where `p3` is the job ID), which fails because job IDs are not rendered to the text content of the panel.

## Logs / Output

```
 FAIL  ProjectQueuePanel.test.ts > ProjectQueuePanel component > shows only jobs belonging to the current project
AssertionError: expected 'Project Queue1 pendingNothing running…' to contain 'p3'

Expected: "p3"
Received: "Project Queue1 pendingNothing running for this projectPendinglifecycle/ideas/test.mdrequirements-analyst · enqueued 20621d agoView global queue"
```
