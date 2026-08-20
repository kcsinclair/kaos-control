---
created: "2026-07-14T19:34:44+10:00"
title: Blocked questions test fails because getOpenQuestions API endpoint is not mocked
type: defect
status: done
lineage: artefact-status-blocked-on-questions
parent: lifecycle/tests/artefact-status-blocked-on-questions-5.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Run the Vitest suite for `artifact-blocked-questions.test.ts`:
   ```sh
   cd tests/web && pnpm test artifact-blocked-questions.test.ts
   ```
2. Observe the failure in `renders the blocked-questions banner when status is "blocked" and body has Open Questions`.

## Expected Behaviour

The test's mock of `@/api/artifacts` should mock `getOpenQuestions` to return the questions payload associated with the blocked artifact, so that the component can render the banner when there are open questions.

## Actual Behaviour

`getOpenQuestions` is not mocked. The component calls it, resulting in a silent failure inside `loadOpenQuestions()`, resetting `openQuestions.value` to `[]` and rendering `hasOpenQuestions` false. As a result, the `.blocked-questions-banner` element is never rendered in the DOM.

## Logs / Output

```
 FAIL  artifact-blocked-questions.test.ts > ArtifactEditorView — blocked-questions banner visibility > renders the blocked-questions banner when status is "blocked" and body has Open Questions
AssertionError: Expected .blocked-questions-banner to be rendered when artifact is blocked with open questions: expected false to be true // Object.is equality

- Expected
+ Received

- true
+ false

 ❯ artifact-blocked-questions.test.ts:338:7
```
