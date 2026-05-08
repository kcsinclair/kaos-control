---
title: "Blocked-questions banner uses wrong CSS class name in ArtifactEditorView"
type: defect
status: in-development
lineage: artefact-status-blocked-on-questions
parent: lifecycle/tests/artefact-status-blocked-on-questions-5.md
labels:
  - defect
assignees:
  - role: frontend-developer
    who: agent
---

# Blocked-questions banner uses wrong CSS class name in ArtifactEditorView

## Reproduction Steps

1. Run the frontend Vitest suite for the blocked-questions feature:
   ```
   cd tests/web && pnpm vitest run artifact-blocked-questions
   ```
2. Observe test 3 (`renders the blocked-questions banner when status is "blocked" and body has Open Questions`) fail.
3. Alternatively, mount `ArtifactEditorView` with an artifact whose `status` is `"blocked"` and whose body contains an `## Open Questions` section with at least one bullet point.
4. Inspect the rendered DOM — search for an element with class `blocked-questions-banner`.

## Expected Behaviour

`ArtifactEditorView` should render an element with CSS class `blocked-questions-banner` (as specified by the test plan and frontend plan) when the artifact status is `"blocked"` and `hasOpenQuestions` is `true`.

## Actual Behaviour

The banner element is rendered with CSS class `open-questions-banner` instead of `blocked-questions-banner` (`web/src/views/project/ArtifactEditorView.vue:311`). The test's `wrapper.find('.blocked-questions-banner')` finds nothing, causing `banner.exists()` to return `false`.

Test 4 (`does NOT render the blocked-questions banner when status is not "blocked"`) passes because `wrapper.find('.blocked-questions-banner')` correctly returns nothing for a `draft` artifact — the wrong class name happens to satisfy that negative assertion.

## Logs / Output

```
 FAIL  artifact-blocked-questions.test.ts > ArtifactEditorView — blocked-questions banner visibility > renders the blocked-questions banner when status is "blocked" and body has Open Questions
AssertionError: Expected .blocked-questions-banner to be rendered when artifact is blocked with open questions: expected false to be true // Object.is equality

- Expected
+ Received

- true
+ false

 ❯ artifact-blocked-questions.test.ts:321:7
    319|       banner.exists(),
    320|       'Expected .blocked-questions-banner to be rendered when artifact…
    321|     ).toBe(true)
       |       ^
    322|   })

 Test Files  1 failed (1)
       Tests  1 failed | 3 passed (4)
    Duration  823ms
```

**Root cause:** `web/src/views/project/ArtifactEditorView.vue` line 311:

```html
<!-- actual -->
<div v-if="artifact.status === 'blocked' && hasOpenQuestions" class="open-questions-banner">

<!-- expected -->
<div v-if="artifact.status === 'blocked' && hasOpenQuestions" class="blocked-questions-banner">
```

**Fix:** Rename the CSS class from `open-questions-banner` to `blocked-questions-banner` in the template and any associated style rules in `ArtifactEditorView.vue`.
