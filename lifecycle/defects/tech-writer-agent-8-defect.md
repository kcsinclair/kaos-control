---
title: "Required-plans gate not enforced for requirement when plans are absent"
type: defect
status: draft
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Required-plans gate not enforced for requirement when plans are absent

## Reproduction Steps

1. Configure a project with `required_plans: ticket: [plan-backend, plan-frontend, plan-test]`.
2. Create a `requirement` artifact in `planning` status (no plan artifacts present for its lineage).
3. Authenticate as a user with only the `approver` role (no `product-owner`).
4. Send `POST /api/p/testproject/artifacts/lifecycle/requirements/gate-req-2.md/transition` with `{"to": "in-development"}`.
5. Observe the response.

## Expected Behaviour

The server returns HTTP 409 Conflict with `error.code = "gate_not_ready"`, blocking the `planning → in-development` transition because the required plan artifacts are absent.

## Actual Behaviour

The server returns HTTP 200 and allows the transition to `in-development` despite no plans being present. The gate is effectively bypassed.

## Logs / Output

```
workflow_doc_test.go:213: expected status 409, got 200: {"artifact":{"path":"lifecycle/requirements/gate-req-2.md","slug":"gate-req","lineage":"gate-req","index":2,"stage":"requirements","type":"requirement","status":"in-development",...}}
--- FAIL: TestDocGate_RequirementStillGated (0.19s)
```

**Failing test:** `TestDocGate_RequirementStillGated` (`tests/integration/workflow_doc_test.go:201`).
