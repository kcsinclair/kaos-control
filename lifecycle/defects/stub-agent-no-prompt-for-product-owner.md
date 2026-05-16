---
title: E2E Stub Agent Returns 409 — No Prompt Template for product-owner Role
type: defect
status: draft
lineage: stub-agent-no-prompt-for-product-owner
created: "2026-05-16T14:00:00+10:00"
priority: normal
labels:
    - defect
    - test
    - agents
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# E2E Stub Agent Returns 409 — No Prompt Template for product-owner Role

## Reproduction Steps

1. Start the E2E harness.
2. POST to `/api/p/testproject/agents/stub-agent/run` with a target path under the `rc-pill.md` or `rc-ws.md` fixture (ideas, role `product-owner`).
3. Observe the HTTP response.

## Expected Behaviour

The stub-agent accepts a run request targeting any valid artifact and completes successfully (status `done`), regardless of the artifact's assigned role.

## Actual Behaviour

The E2E tests `Flow 10 TC3` and `Flow 10 TC4` fail immediately with:

```
Error: triggerRun failed (409): {"error":{"code":"run_error","message":"agent \"stub-agent\" has no prompt template for role \"product-owner\""}}
```

The backend rejects the run because the E2E fixture definition of `stub-agent` does not include a prompt template for the `product-owner` role, which is the role assigned to the `rc-pill.md` and `rc-ws.md` idea fixtures.

Failing tests:
- `flows/10-artefact-run-count-column.spec.ts:182` — TC3: "Agent Running" pill appears while run is active and disappears on completion
- `flows/10-artefact-run-count-column.spec.ts:219` — TC4: run count increments without page reload on agent.finished WS event

## Fix

Add a `product-owner` prompt template entry to the `stub-agent` configuration in the E2E test fixtures (the agent definition loaded by the E2E harness). Alternatively, change the `rc-pill.md` and `rc-ws.md` fixtures to use a role for which stub-agent already has a template (e.g. the role already used in Flow 04).
