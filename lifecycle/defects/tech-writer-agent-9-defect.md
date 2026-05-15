---
title: Transitioning doc to in-qa does not populate assignees with qa/agent
type: defect
status: approved
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: backend-developer
      who: agent
---

# Transitioning doc to in-qa does not populate assignees with qa/agent

## Reproduction Steps

1. Create a `doc` artifact in `in-development` status.
2. Authenticate as a user with the `tech-writer` role.
3. Send `POST /api/p/testproject/artifacts/lifecycle/docs/<slug>/transition` with `{"to": "in-qa"}`.
4. Inspect the returned artifact's `assignees` field.

## Expected Behaviour

The transition succeeds (HTTP 200) and the returned artifact contains:

```json
"assignees": [{"role": "qa", "who": "agent"}]
```

This mirrors the existing behaviour for other artifact types where entering `in-qa` auto-assigns the QA agent.

## Actual Behaviour

The transition succeeds but `assignees` is empty (or absent). The QA agent is not assigned.

## Logs / Output

```
api_doc_transition_test.go:133: expected non-empty 'assignees' after transitioning doc to in-qa
--- FAIL: TestDocTransition_AssigneesOnInQA (0.15s)
```

**Failing test:** `TestDocTransition_AssigneesOnInQA` (`tests/integration/api_doc_transition_test.go:107`).
