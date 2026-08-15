---
title: "Post-resolution routing behavior for open-questions resolve (Milestone 4)"
type: test
status: draft
lineage: open-questions-gui
parent: lifecycle/defects/open-questions-gui-9-defect.md
---

# Test Suite Coverage

This artifact documents `tests/integration/open_questions_routing_test.go`,
closing [open-questions-gui-9-defect](../defects/open-questions-gui-9-defect.md):
the Milestone 4 ("Post-resolution routing") test file listed in
[open-questions-gui-5-test.md](open-questions-gui-5-test.md) was missing.

## Scenarios covered

### `TestOpenQuestionsRouting_RequirementNoAutoApprove`
- **ResolveRoutesToDraftNotApproved** — a `requirement` artifact seeded
  `blocked` with a single open question is resolved via the standard
  preview(complete=true) + PUT flow. Asserts the resulting status is
  `draft` (awaiting approval), not `approved`.
- **ApprovalOnlyHappensViaExplicitTransition** — re-fetches the artifact to
  confirm it is still `draft` (no approval side effect from the resolve),
  then issues a separate `POST .../transition {"to":"approved"}` call and
  asserts that *this* deliberate call is what moves the status to
  `approved`.

### `TestOpenQuestionsRouting_DeveloperArtifactNoAutoRequeue`
- **ResolveRoutesToDraftWithNoRunStarted** — a `plan-backend` artifact
  (a developer-raised artifact type) seeded `blocked` with a single open
  question is resolved the same way. Asserts the resulting status is
  `draft` and that `GET .../agents/runs?target_path=...` returns zero runs
  — no agent run is started automatically by the resolve.
- **ExplicitRequeueTargetsOriginatingRoleAndStartsRun** — issues a separate
  `POST .../agents/backend-developer/run` call (the explicit requeue) using
  the `startAgentRun`/`waitForRunCompletion`/`setupFakeClaude` harness from
  `agent_helpers_test.go`, and asserts the resulting run's `role` and
  `agent_name` are `backend-developer` (the originating developer role for
  a `plan-backend` artifact), and that exactly one run now exists for the
  target path.

Both tests share a `resolveOpenQuestions` helper that drives the standard
preview+PUT resolve flow already exercised in
`open_questions_resolve_e2e_test.go`, and confirm — end to end through the
real HTTP API — that neither the approve nor the requeue side effect is
observable until the corresponding deliberate API call is made, matching
the Expected Behaviour in
[open-questions-gui-9-defect.md](../defects/open-questions-gui-9-defect.md).

## Implementation note

No product code changed: `internal/index/autoblock.go`'s
`applyOpenQuestionTransition` already only ever auto-transitions
blocked ↔ draft and never auto-approves or auto-starts a run, for any
artifact type. This defect was solely a missing-test-coverage gap; the new
test file makes that existing (correct) behavior verifiable and
regression-proof.
