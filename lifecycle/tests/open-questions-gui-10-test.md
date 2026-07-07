---
title: "Test Suite: Open-Questions Awaiting-Answers Query & Permissions"
type: test
status: draft
lineage: open-questions-gui
parent: lifecycle/defects/open-questions-gui-8-defect.md
---

# Test Suite Coverage

Fixes [[open-questions-gui]] defect `open-questions-gui-8-defect`: the missing
integration test file for Milestone 3 of
[[open-questions-gui]]-5-test (`tests/open_questions_awaiting_test.go`).

## Test File

- `tests/open_questions_awaiting_test.go`

## Scenarios Covered

1. **`TestOpenQuestionsAwaiting_ListReturnsExactBlockedSet`** — seeds two
   blocked artifacts with a non-empty `## Open Questions` section plus a
   draft decoy and a done decoy with no open questions; asserts
   `GET /api/p/testproject/artifacts?awaiting_answers=true` returns exactly
   the two blocked-with-open-questions artifacts and nothing else.
2. **`TestOpenQuestionsAwaiting_CountOnly`** — seeds three matching artifacts
   plus a decoy; asserts `awaiting_answers=true&count_only=true` returns
   `{"count":3}` with no `items` field.
3. **`TestOpenQuestionsAwaiting_ResolvingLastDropsCountToZero`** — seeds a
   single blocked-with-open-questions artifact, confirms the count starts at
   `1`, resolves it completely via `POST .../open-questions/preview`
   (`complete=true`) followed by the existing `PUT /artifacts/*path`, and
   confirms the count drops to `0` once the artifact auto-unblocks to
   `draft`.
4. **`TestOpenQuestionsAwaiting_WebSocketEventOnResolve`** — connects to the
   project WebSocket before resolving a blocked artifact the same way as
   scenario 3, and asserts the existing `artifact.indexed` event fires for
   that artifact's path after the resolving PUT (NFR2).
5. **`TestOpenQuestionsAwaitingPermissions_PreviewRequiresProductOwner`** —
   asserts a session held by a non-product-owner user (`dev@test.local`, which
   only carries developer roles) gets HTTP 403 from
   `POST .../open-questions/preview` and that the artifact is left unchanged
   (still `blocked`, heading untouched), then asserts the same request
   succeeds (HTTP 200) for the product-owner session (`admin@test.local`)
   (FR8, Resolved Q2).

All scenarios use the existing `testEnv` integration harness (`tests/integration/helpers_test.go`):
auto-login as admin, `makeBlockedArtifact`/`makeArtifact` seeding, and the
`buildCookieHeader`/`coder/websocket` pattern already used by
`tests/integration/priority_patch_test.go` for WebSocket assertions.

Run with:

```sh
go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsAwaiting
```
