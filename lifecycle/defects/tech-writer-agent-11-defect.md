---
title: '"Request docs" button not implemented on done artifact view'
type: defect
status: approved
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels: [defect]
assignees:
  - role: frontend-developer
    who: agent
---

# "Request docs" button not implemented on done artifact view

## Reproduction Steps

1. Navigate to a `done` artifact in the artifact detail view (e.g., `lifecycle/requirements/smoke-req-done.md`).
2. Wait for the artifact page to finish loading.
3. Look for a `button` with text "Request docs".

## Expected Behaviour

A "Request docs" button is visible on `done` artifacts. Clicking it opens a modal where the user can supply a brief description. Submitting the modal sends `POST /api/p/:project/artifacts` with `stage: "docs"`, `lineage`, and `parent` set to the current artifact, then navigates to the newly created doc artifact.

The button must NOT appear on non-done artifacts.

## Actual Behaviour

No "Request docs" button exists anywhere on the artifact detail view, regardless of status. The button selector `button:has-text("Request docs")` is never found.

## Logs / Output

```
Error: expect(locator).toBeVisible() failed
Locator: locator('button:has-text("Request docs")')
Expected: visible
Timeout: 10000ms
Error: element(s) not found
  at flows/06-doc-request.spec.ts:46:67
```

Note: this failure is partially masked by defect `tech-writer-agent-10-defect.md` (artifact detail shows "project not found"), so both defects must be resolved to fully validate TC1 and TC3 of Flow 06.

**Failing test:** `Flow 06 TC3` (`tests/e2e/flows/06-doc-request.spec.ts:39`); TC1 verifies button visibility, also covered by this defect.
