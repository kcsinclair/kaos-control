---
title: Agent Emits 'Open Questions' Header With No Questions, Triggering False Workflow Block
type: defect
status: draft
lineage: agent-empty-open-questions-triggers-workflow-block
created: "2026-08-24T19:26:13+10:00"
priority: normal
labels:
    - defect
    - agent
    - workflow
    - lifecycle
---

# Agent Emits 'Open Questions' Header With No Questions, Triggering False Workflow Block

## Reproduction Steps

1. Trigger the analyst agent to produce a requirements artifact for an idea where all design questions have already been resolved.
2. Observe the generated requirements markdown file (e.g. `requirements/frontend-lint-gap-2.md`).
3. Note that the agent includes a `## Open Questions` section populated only with a prose statement such as *"None. All design questions … have been resolved."* rather than omitting the section entirely.
4. Open the artifact in the kaos-control UI.
5. Observe that the workflow engine detects the `## Open Questions` heading and treats the artifact as having unresolved questions, blocking the requirements from advancing.

## Expected Behaviour

When an agent has no open questions, it should either omit the `## Open Questions` section entirely or emit a sentinel value (e.g. an empty list or a machine-readable `none` marker) that the workflow engine recognises as "no questions outstanding". The workflow block for unresolved open questions must not be triggered in this case.

## Actual Behaviour

The agent includes the `## Open Questions` heading with a human-readable "None" prose note. The workflow engine detects the heading and incorrectly concludes there are open questions, blocking the requirements artifact from advancing. A maintainer must manually edit the file to remove or rewrite the section before the workflow unblocks.
