---
title: Richer Agent Completion Status
type: idea
status: draft
lineage: agent-completion-status-detail
created: "2026-04-27T13:56:41+10:00"
priority: normal
labels:
    - agent
    - enhancement
    - workflow
    - artefacts
---

# Richer Agent Completion Status

When an agent run finishes, the current status update is too sparse — it simply marks the run as done. The completion message should instead reflect what the agent actually accomplished: whether it produced new artifacts, updated existing ones, or (for the QA agent) ran tests and what the outcomes were.

Artifact changes should be surfaced with clear distinction between produced vs updated, listing the relevant file paths. For example: `ARTIFACTS PRODUCED: lifecycle/frontend-plans/prompt-to-idea-12-fe.md` or `ARTIFACTS UPDATED: lifecycle/requirements/login-2.md`. This gives developers immediate visibility into what changed without having to inspect git diff or re-read the run log.

For the QA agent specifically, if all tests pass the completion status should include a summary line (e.g. `ALL TESTS PASSED`) along with the relevant output line from the test runner. If tests fail, the failing test names and error snippets should be surfaced in the status update so the developer can act without opening the full run log.
