---
title: Artifact detail view renders 'project not found' when docs-stage fixtures are active
type: defect
status: in-development
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Artifact detail view renders "project not found" when docs-stage fixtures are active

## Reproduction Steps

1. Run the E2E harness with the fixture `config.yaml` that includes the `docs` stage and `tech-writer` agent (`tests/e2e/fixtures/lifecycle/config.yaml`).
2. Log in as `admin@kaos-e2e.local`.
3. Navigate to any artifact detail URL, e.g.:
   - `/p/testproject/artifacts/lifecycle/requirements/smoke-req-done.md`
   - `/p/testproject/artifacts/lifecycle/requirements/smoke-req-01.md`
   - `/p/testproject/artifacts/lifecycle/docs/smoke-doc-approved.md`
4. Wait up to 10 s for the page to load.

## Expected Behaviour

The artifact detail component renders the artifact's content and metadata, including a `.status-badge` or `[data-status]` element.

## Actual Behaviour

The main content area displays `"project not found: testproject"` instead of the artifact. The sidebar shows the project navigation correctly (confirming the project IS registered), but the artifact view component fails to load its data.

The ARIA tree for the main section is:
```
- main:
  - button "← artifacts"
  - text: "project not found: testproject"
```

## Logs / Output

```
Error: expect(locator).toBeVisible() failed
Locator: locator('.status-badge, [data-status]').first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found
```

This blocks all of Flow 06 (TC1, TC2) and Flow 08 (TC1, TC2), preventing verification of the "Request docs" button and the Queue Work button.

**Failing tests:** `Flow 06 TC1`, `Flow 06 TC2` (`tests/e2e/flows/06-doc-request.spec.ts`); `Flow 08 TC1`, `Flow 08 TC2` (`tests/e2e/flows/08-doc-queue.spec.ts`).
