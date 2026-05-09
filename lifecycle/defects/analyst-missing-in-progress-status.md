---
title: Analyst Agent Does Not Set In-Progress Status During Execution
type: defect
status: done
lineage: analyst-missing-in-progress-status
priority: normal
labels:
    - defect
    - agent
    - workflow
    - artefacts
release: KC-OG-Sprint
---

# Analyst Agent Does Not Set In-Progress Status During Execution

## Reproduction Steps

1. Trigger an analyst agent run (e.g. `requirements-analyst` or `planning-analyst`) against an artifact in a qualifying state.
2. Observe the artifact status while the agent is actively working.
3. Compare behaviour to the `backend-developer` or `frontend-developer` agent runs, which were recently updated to set `in-development` status, and the `qa` agent which sets `in-qa`.

## Expected Behaviour

While an analyst agent is executing, the target artifact's status should be updated to `in-progress` to reflect that work is actively underway — consistent with the pattern used by `in-development` (developer agents) and `in-qa` (qa agent).

## Actual Behaviour

The analyst agent does not set any in-progress status on the artifact during its run. The artifact remains in its prior status (e.g. `draft` or `clarifying`) until the agent completes, giving no indication that work is in progress.
