---
title: 'Frontend Plan: Run All Tests and Auto-file Defects'
type: plan-frontend
status: in-development
lineage: test-everything
parent: lifecycle/requirements/test-everything-2.md
---

## Overview

The frontend work for the test-everything feature is minimal. The existing artifact list, defect views, and agent launcher panel already support the workflows this feature introduces. The primary changes are: ensuring the agent launcher panel supports invoking the `test-runner` agent without a `target_path`, displaying the run summary in the agent run history, and adding a coverage gap indicator for orphaned tests.

Related: [[test-everything]], [[agent-launcher-panels]], [[agent-run-summary-panel]]

## Milestone 1 — Agent Launcher: Support Target-less Invocation

### Description

Ensure the agent launcher panel allows invoking the `test-runner` agent without requiring a `target_path` selection. Currently, some agents require a target artifact; the `test-runner` agent should show a simple "Run All Tests" trigger without a target picker.

### Files to change

- `web/src/components/agents/AgentLauncherPanel.vue` — Conditionally hide the target artifact picker when the selected agent has no `source_types` configured (empty array). Show an informational message like "This agent runs all test suites — no target artifact required." The "Run" button should be enabled without a target selection.
- `web/src/stores/agents.ts` — Ensure `launchAgent(agentName, targetPath?)` handles an undefined/empty `targetPath` by omitting it from the API request body rather than sending an empty string.

### Acceptance criteria

- [ ] Selecting `test-runner` in the agent launcher hides the target artifact picker.
- [ ] An informational message explains the agent runs without a target.
- [ ] The "Run" button is enabled without selecting a target.
- [ ] Clicking "Run" invokes `POST /api/p/:project/agents/test-runner/run` without a `target_path` field.
- [ ] Agents that do require a target (e.g., `qa`) are unaffected — the picker still appears for them.

## Milestone 2 — Agent Run Summary Display

### Description

Display the `test-runner` agent's run summary in the agent run history panel. The summary includes per-suite totals, defects created, duplicates found, orphaned failures, and coverage gaps.

### Files to change

- `web/src/components/agents/AgentRunHistory.vue` (or equivalent) — When displaying a run for the `test-runner` agent, render the `RunSummary` data in a structured format:
  - A table or grid showing per-suite stats: total, passed, failed, skipped.
  - A summary line: "X defects created, Y duplicates found, Z orphaned failures."
  - A collapsible section for coverage gaps (tests without `lifecycle/tests/*.md` artifacts).
  - Wall-clock duration of the run.
- `web/src/types/agent.ts` (or equivalent type file) — Add `RunSummary` TypeScript interface matching the backend struct:
  ```typescript
  interface RunSummary {
    suites: Array<{
      name: string
      total: number
      passed: number
      failed: number
      skipped: number
      elapsed: number
    }>
    defectsCreated: number
    duplicatesFound: number
    orphanedFailures: number
    coverageGaps: string[]
    elapsed: number
  }
  ```

### Acceptance criteria

- [ ] Run history for `test-runner` displays per-suite statistics in a readable table.
- [ ] Defects created, duplicates found, and orphaned failures are shown prominently.
- [ ] Coverage gaps are listed in a collapsible section.
- [ ] Wall-clock duration is displayed.
- [ ] Other agents' run history is unaffected.

## Milestone 3 — Defect List: Auto-filed Indicator

### Description

Add a visual indicator in the artifact list view for defects that were auto-filed by the `test-runner` agent. This helps users quickly distinguish manually filed defects from automated ones.

### Files to change

- `web/src/components/artifacts/ArtifactListItem.vue` (or equivalent) — When a defect artifact's `labels` include `auto-filed`, render a small badge or icon (e.g., a robot icon from lucide-vue-next: `Bot`) next to the title. Use a tooltip: "Auto-filed by test-runner agent."
- `web/src/components/artifacts/ArtifactListItem.vue` — Style the badge to be unobtrusive (muted colour, small size) so it doesn't dominate the list view.

### Acceptance criteria

- [ ] Defects with the `auto-filed` label show a bot icon badge.
- [ ] The badge has a tooltip explaining its meaning.
- [ ] The badge does not appear on defects without the `auto-filed` label.
- [ ] The badge does not appear on non-defect artifact types.
- [ ] The list layout is not disrupted by the additional badge.

## Milestone 4 — DevOps Pipeline: Test-All Card

### Description

Ensure the `test-all.yaml` pipeline ([[devops-pipelines]]) renders correctly in the DevOps UI with appropriate status indicators and output display for test results.

### Files to change

- `web/src/components/devops/PipelineCard.vue` — No structural changes expected; the existing card component should render the `test-all` pipeline correctly since it follows the standard pipeline YAML format. Verify that:
  - The pipeline appears under a "Test" type group/column.
  - The step description ("Execute all test suites and file defects for failures") renders in the step list.
  - The long timeout (30 minutes) does not cause UI issues with progress display.
- `web/src/views/DevOpsView.vue` — Verify the "Test" type column renders if it is a new type not previously used.

### Acceptance criteria

- [ ] The `test-all` pipeline appears in the DevOps view under a "Test" type group.
- [ ] The pipeline card shows the pipeline name "Run All Tests" and step count.
- [ ] Running the pipeline from the DevOps UI triggers the test-runner agent.
- [ ] Pipeline output (step logs) stream correctly during execution.
- [ ] The 30-minute timeout does not cause progress bar or timer display issues.
