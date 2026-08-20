---
created: "2026-07-14T19:34:44+10:00"
title: "Test Plan: Open-Questions Resolution GUI"
type: plan-test
status: done
lineage: open-questions-gui
parent: lifecycle/requirements/open-questions-gui-2.md
---

# Test Plan — Open-Questions Resolution GUI

Integration tests (in repo-root `tests/`) exercising the guided open-questions
resolution flow from [[open-questions-gui]] (requirement `open-questions-gui-2`),
covering the [[open-questions-gui]] backend (`-3-be`) and frontend (`-4-fe`)
plans. The overriding invariant to protect: **status changes are only ever a
side effect of a body edit through the existing indexer/autoblock** — no test
path (and no product code under test) writes `status` directly (NFR1).

Test-infra notes: the integration `testEnv` auto-logins as admin; devops-style
URL helpers return full URLs suitable for `http.Get`; run-log endpoints return
NDJSON. Reuse those patterns. Where a permission test needs a non-product-owner
user, provision a second user/session rather than relying on the auto-admin
login.

## Milestone 1 — Configuration & parser/builder API

**Description.** Verify the answer-format config surface and the pure
parse/build endpoints from backend Milestones 1–3.

**Files to change.**
- `tests/open_questions_config_test.go`, `tests/open_questions_parse_test.go`.

**Acceptance criteria.**
- `GET …/config/open-questions` returns `{"answer_format":"blockquote"}` by
  default; after writing `open_questions.answer_format` to
  `lifecycle/config.yaml` and reloading, the endpoint reflects the override
  (NFR4) — no restart/code change.
- `GET …/artifacts/*path/open-questions` on a body with N top-level list items
  returns N questions in order; a question already followed by a `> …`
  blockquote surfaces that text as its `answer`.
- Malformed/empty/absent `## Open Questions` → `questions: []`, HTTP 200.
- `POST …/open-questions/preview` with `complete=false` returns a body with
  answers inserted and the heading unchanged; calling it twice with identical
  answers returns byte-identical bodies (idempotent, NFR3); frontmatter and
  unrelated sections are preserved exactly.
- `preview` with `complete=true` and all answers present renames the heading to
  `## Resolved Questions`; with any empty answer it returns an error and does
  not rename.
- The `preview` endpoint does not modify the artefact on disk (no index event /
  mtime change).

## Milestone 2 — End-to-end resolve → unblock

**Description.** The full happy path: questions appear → auto-block → partial
save stays blocked → complete → auto-unblock to `draft`, driven only through the
public write API (mirrors the requirement's final acceptance criterion).

**Files to change.**
- `tests/open_questions_resolve_e2e_test.go`.

**Acceptance criteria.**
- Creating/PUTting an artefact whose body has a non-empty `## Open Questions`
  section results in `status: blocked` with a `product-owner` assignee (existing
  autoblock; [[agent-questions-trigger-blocked-status]]).
- A partial resolve (preview `complete=false` → PUT body) leaves the artefact
  `blocked` and the heading `## Open Questions`; re-fetching pre-populates the
  saved answers (FR5).
- A complete resolve (preview `complete=true` → PUT body) renames the heading to
  `## Resolved Questions` and the artefact auto-transitions to `draft`
  ([[agent-questions-trigger-blocked-status]]).
- The captured PUT request body contains only `frontmatter` (unchanged) + `body`
  and **no** `status` mutation authored by the client (NFR1, inspected on the
  payload).
- Re-saving the same completed answers is idempotent (body unchanged, artefact
  stays `draft`).
- Re-opening: moving an item back under a non-empty `## Open Questions` heading
  and PUTting re-blocks the artefact via existing logic (FR9).

## Milestone 3 — Awaiting-answers count/list & permissions

**Description.** Verify the badge/list query surface and the product-owner
authorisation rule (backend Milestone 4; Resolved Q2, Q3).

**Files to change.**
- `tests/open_questions_awaiting_test.go`.

**Acceptance criteria.**
- `GET …/artifacts?awaiting_answers=true` returns exactly the blocked artefacts
  with a non-empty `## Open Questions` section; adding `count_only=true` returns
  `{count:N}` matching that set.
- Resolving the last such artefact drops the count to `0`.
- The count changes are observable via the existing `artifact.indexed` WS event
  after a write (NFR2) — assert the event fires on the underlying PUT.
- A request to `…/open-questions/preview` from a session **without** the
  `product-owner` role returns HTTP 403 and performs no work; a product-owner
  session succeeds (FR8, Resolved Q2).

## Milestone 4 — Post-resolution routing

**Description.** Confirm resolution leaves routing to a deliberate human action
and offers the correct option per originating role (FR7, Resolved Q5). No
follow-up fires automatically.

**Files to change.**
- `tests/open_questions_routing_test.go`.

**Acceptance criteria.**
- For a `type: requirement` artefact, after a complete resolve the status is
  `draft` (awaiting approval) and **no** approval/transition to the
  planning-analyst has occurred without an explicit approve action.
- For a developer-raised artefact (`plan-*` carrying the `## Open Questions`),
  after resolve no requeue/run is started automatically; an explicit requeue
  call targets the originating developer role and starts a run.
- Neither the approve nor the requeue side effect is observable until the
  corresponding deliberate API call is made.

## Companion test artifact

On completion, record a `test` artifact in `lifecycle/tests/` (lineage
`open-questions-gui`) summarising the suites above and pointing to the specific
`tests/open_questions_*_test.go` files, per the test-developer contract.
