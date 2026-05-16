---
title: E2E stub-agent missing prompt_templates causes 409 on triggerRun (TC1, TC3, TC4)
type: defect
status: done
lineage: artefacts-agent-run-count-column
parent: lifecycle/tests/artefacts-agent-run-count-column-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

# E2E stub-agent missing prompt_templates causes 409 on triggerRun (TC1, TC3, TC4)

## Reproduction Steps

1. Run `cd tests/e2e && pnpm exec playwright test flows/10-artefact-run-count-column.spec.ts --reporter=list`
2. Observe TC1, TC3, and TC4 all fail at the first `triggerRun()` call.

## Expected Behaviour

The `triggerRun()` helper in `tests/e2e/flows/10-artefact-run-count-column.spec.ts` posts to
`POST /api/p/testproject/agents/stub-agent/run` and receives a `run_id` in response.
Subsequent tests (TC3, TC4) can start and observe agent runs through their lifecycle.

## Actual Behaviour

All three tests fail immediately with HTTP 409:

```
Error: triggerRun failed (409): {"error":{"code":"run_error",
"message":"agent \"stub-agent\" has no prompt template for role \"product-owner\""}}
```

## Root Cause

`internal/agent/agent.go:StartRun()` looks up `ag.PromptTemplates[role]` for the selected
role before dispatching to any driver. If the map is empty or missing the role key it returns
an error — regardless of driver type (including `shell-stub`).

The E2E fixture config at `tests/e2e/fixtures/lifecycle/config.yaml` configures `stub-agent`
with `role: [product-owner]` but omits a `prompt_templates` block entirely:

```yaml
agents:
  - name: stub-agent
    role:
      - product-owner
    driver: shell-stub
    shell_command: "sleep 1 && printf ..."
    # prompt_templates is absent — causes 409 on every StartRun call
```

## Fix Required

Add a `prompt_templates` entry for the `product-owner` role to the stub-agent in
`tests/e2e/fixtures/lifecycle/config.yaml`:

```yaml
    prompt_templates:
      product-owner: "Process {target_path}"
```

## Logs / Output

```
  ✘  1 flows/10-artefact-run-count-column.spec.ts:94:3 › TC1: Runs column ... (196ms)
  ✘  3 flows/10-artefact-run-count-column.spec.ts:182:3 › TC3: "Agent Running" pill ... (355ms)
  ✘  4 flows/10-artefact-run-count-column.spec.ts:219:3 › TC4: run count increments ... (271ms)

Error: triggerRun failed (409):
  {"error":{"code":"run_error",
   "message":"agent \"stub-agent\" has no prompt template for role \"product-owner\""}}
```
