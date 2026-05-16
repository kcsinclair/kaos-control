---
title: '"New Docs" creation modal submit never fires POST /artifacts'
type: defect
status: done
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# "New Docs" creation modal submit never fires POST /artifacts

## Reproduction Steps

1. Navigate to `/p/testproject/dashboard` or `/p/testproject/artifacts`.
2. Confirm the "New Docs" button is present (TC1 and TC2 pass).
3. Click "New Docs" to open the creation modal/form.
4. Fill in a title or brief (the form field rendered by the modal).
5. Click the submit button (`button[type="submit"]:has-text("Create")` or `button:has-text("Submit")`).
6. Wait for a `POST /api/p/testproject/artifacts` response.

## Expected Behaviour

Submitting the form sends `POST /api/p/testproject/artifacts` with `stage: "docs"`, a valid slug, and draft frontmatter. The server creates the file at `lifecycle/docs/<slug>.md` and returns HTTP 201. The UI navigates to the new artifact's detail page.

## Actual Behaviour

The `page.waitForResponse` for `POST .../artifacts` times out after 20 s. Either:
- The submit button click has no effect (the form does not submit), OR
- The POST is sent but the server fails to create the file (see `tech-writer-agent-7-defect.md` — `lifecycle/docs` directory not auto-created).

The test ends with:
```
TimeoutError: page.waitForResponse: Timeout 20000ms exceeded while waiting for event "response"
  at flows/07-doc-new.spec.ts:41:40
Error: page.click: Test ended.
  - waiting for locator('button[type="submit"]:has-text("Create"), button:has-text("Submit")')
  at flows/07-doc-new.spec.ts:60:16
```

The submit button itself was also not found when the test timed out, suggesting the modal either does not open or the submit button selector is wrong.

Note: even if the frontend form submission is fixed, the backend `lifecycle/docs` directory issue (`tech-writer-agent-7-defect.md`) must also be resolved for a 201 response to be returned.

**Failing test:** `Flow 07 TC3` (`tests/e2e/flows/07-doc-new.spec.ts:30`).
