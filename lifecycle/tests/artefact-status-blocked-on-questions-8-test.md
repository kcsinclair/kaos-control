---
created: "2026-07-14T19:34:44+10:00"
title: "Tests: fix unmocked getOpenQuestions in blocked-banner Vitest suite"
type: test
status: draft
lineage: artefact-status-blocked-on-questions
parent: lifecycle/defects/artefact-status-blocked-on-questions-7-defect.md
---

# Tests: fix unmocked getOpenQuestions in blocked-banner Vitest suite

Fixes the defect described in
`lifecycle/defects/artefact-status-blocked-on-questions-7-defect.md`: the
`@/api/artifacts` mock in `tests/web/artifact-blocked-questions.test.ts` did
not include `getOpenQuestions`, so `ArtifactEditorView`'s `loadOpenQuestions()`
call threw, was silently swallowed, and always reset `openQuestions` to `[]` —
the `.blocked-questions-banner` element could never render.

## Change

File: `tests/web/artifact-blocked-questions.test.ts`

- The top-level `vi.mock('@/api/artifacts', ...)` factory now includes
  `getOpenQuestions`, defaulting to a resolved empty-questions payload
  (`{ heading, format, questions: [], can_resolve: false }`) so every test
  that doesn't care about open questions gets a safe, non-throwing default.
- The banner-visibility test ("renders the blocked-questions banner when
  status is `blocked` and body has Open Questions") now queues a
  `mockResolvedValueOnce` on `getOpenQuestions`, before mounting, returning a
  payload with one question matching the fixture body (`## Open
  Questions\n\n- Q1\n`). This lets `ArtifactEditorView`'s
  `hasOpenQuestions` computed evaluate `true` so the banner renders.
- No change was needed for the companion negative test ("does NOT render...
  when status is not blocked") — it relies on the new default empty mock,
  which correctly keeps `hasOpenQuestions` false.

## Scenarios covered (all 4 passing)

1. Info toast shown when save's returned status differs (blocked) from the
   submitted status.
2. No extra toast when the returned status matches the submitted status.
3. `.blocked-questions-banner` renders when `getOpenQuestions` returns a
   non-empty questions array for a blocked artifact.
4. `.blocked-questions-banner` is absent when `getOpenQuestions` returns no
   questions (default mock), regardless of body content.

Verified with:

```sh
cd tests/web && pnpm test artifact-blocked-questions.test.ts
```

All 4 tests pass. Full `tests/web` suite run alongside this change shows no
new failures — the 19 pre-existing failures in `ProjectQueuePanel*` and
`AgentsRunsView.projectQueue.test.ts` are unrelated and unchanged by this fix.
