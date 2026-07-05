---
title: "Backend Plan: Open-Questions Resolution GUI Support"
type: plan-backend
status: in-development
lineage: open-questions-gui
parent: lifecycle/requirements/open-questions-gui-2.md
---

# Backend Plan — Open-Questions Resolution GUI Support

Backend support for the guided open-questions resolution flow described in
[[open-questions-gui]] (requirement `open-questions-gui-2`). The auto-block /
auto-unblock state machine in `internal/index/autoblock.go` is **already built
and must not be re-implemented** (Non-goal). This plan adds only the read-model
and pure body-transform helpers the frontend needs, plus the configuration and
authorisation surface. Persistence of answers remains the existing
`PUT /api/p/:project/artifacts/*path` body write (NFR1) — no new write path
sets `status`.

The companion plans are [[open-questions-gui]] frontend (`-4-fe`) and test
(`-5-test`).

## Milestone 1 — Answer-format configuration (per-project)

**Description.** Resolved Q1 fixes the answer format as a **per-project**
setting living in `lifecycle/config.yaml`, with the blockquote default seeded
into the onboarding template. Add a typed config section, a safe default, and a
parsed-config endpoint the frontend can read (mirroring the existing kanban
config endpoint).

**Files to change.**
- `internal/config/config.go` — add `OpenQuestions OpenQuestionsConfig` to the
  `Project` struct (`yaml:"open_questions,omitempty"`). New struct:
  `OpenQuestionsConfig{ AnswerFormat string `yaml:"answer_format" json:"answer_format"` }`.
  Provide a `func (c OpenQuestionsConfig) EffectiveFormat() string` returning
  `"blockquote"` when empty. Accepted values: `"blockquote"` (default); the
  format is a named strategy, not free-form, so unknown values fall back to
  blockquote with a logged warning.
- `internal/http/config.go` — add `handleGetOpenQuestionsConfig` that reloads
  the project config from disk (same pattern as `handleGetKanbanConfig`) and
  returns `{"answer_format":"blockquote"}`.
- `internal/http/server.go` — register `GET /api/p/:project/config/open-questions`.
- `internal/initcmd/templates/config.yaml.tmpl` — add a commented+default
  `open_questions:\n  answer_format: blockquote` block so new projects onboard
  with the default present (Resolved Q1).

**Acceptance criteria.**
- With no `open_questions` block, `EffectiveFormat()` returns `"blockquote"`
  and the endpoint returns `{"answer_format":"blockquote"}`.
- Setting `open_questions.answer_format` in `lifecycle/config.yaml` changes the
  endpoint response without any code change (NFR4).
- Newly scaffolded projects contain the `open_questions.answer_format:
  blockquote` default.
- Unit test covers: default, explicit override, unknown value → blockquote +
  warning.

## Milestone 2 — Question parser + read endpoint

**Description.** Provide a pure, well-tested parser that turns the
`## Open Questions` section into an ordered question model, surfacing any
answers already written in the configured format so partially-answered sessions
resume in place (FR3 pre-population, FR5 resume). This is new parsing and does
not touch the `HasOpenQuestions` block/unblock trigger.

**Files to change.**
- `internal/artifact/questions.go` (new) — `ParseOpenQuestions(body string)
  ([]Question, bool)` where `Question{ Index int; Text string; Answer string }`.
  Rules: locate the `## Open Questions` heading (case-sensitive, matching
  `HasOpenQuestions` semantics); each **top-level list item** (`- ` / `1. `)
  under it is one question (Non-goal: agent list contract unchanged); text runs
  until the next top-level item, blank-separated blockquote answer, or next
  `## ` heading. An immediately-following blockquote (`> …`) is captured as that
  question's existing `Answer`. Returns `(nil, false)` when the section is
  absent, empty, or malformed (NFR6 graceful parsing — never errors).
- `internal/http/artifacts.go` (or new `open_questions.go`) —
  `handleGetOpenQuestions` for `GET
  /api/p/:project/artifacts/*path/open-questions` returning
  `{"heading":"## Open Questions","format":"blockquote","questions":[{"index":0,
  "text":"…","answer":"…"}]}`. When there is nothing to resolve, return
  `questions: []` with HTTP 200 (not an error) so the frontend can hide the
  action.
- `internal/http/server.go` — register the route.

**Acceptance criteria.**
- A body with N top-level list items under `## Open Questions` yields exactly N
  questions in document order.
- A question already followed by a `> …` blockquote surfaces that text in
  `answer`.
- Malformed/empty/absent section → `questions: []`, HTTP 200, no panic.
- Sub-items and prose under a question are attached to that question, not
  counted as separate questions.
- Unit tests in `internal/artifact/questions_test.go` cover list markers,
  multi-line questions, pre-existing answers, and the empty/malformed cases.

## Milestone 3 — Answer write-back builder (pure transform, no disk write)

**Description.** Centralise the byte-exact, idempotent, loss-safe body
transform in Go so the house format is produced identically to a correct manual
edit (FR4, NFR3). This helper only *builds* the new body; the frontend persists
it via the existing `PUT /artifacts/*` (NFR1). It also enforces the safe-
completion rule server-side: the heading is renamed only when every question
has a non-empty answer (FR6, Goal "Safe completion").

**Files to change.**
- `internal/artifact/questions.go` — `ApplyAnswers(body string, answers
  map[int]string, format string, complete bool) (string, error)`. Behaviour:
  for each question, write/replace its answer immediately after the question in
  the configured format (default blockquote `> …` with blank-line spacing
  matching the current manual convention); operate only on the
  `## Open Questions` section, leaving frontmatter and all other sections
  byte-for-byte unchanged; re-applying identical answers yields an identical
  body (idempotent). When `complete == true`, require all questions answered,
  then rename `## Open Questions` → `## Resolved Questions` in the same returned
  body; if `complete == true` but any answer is empty, return an error
  (`ErrIncompleteAnswers`) and do **not** rename.
- `internal/http/open_questions.go` — `handlePreviewOpenQuestions` for `POST
  /api/p/:project/artifacts/*path/open-questions/preview` accepting
  `{"answers":{"0":"…"},"complete":false}` and returning `{"body":"…"}`
  (a **compute-only** endpoint — it does not write to disk). This keeps the
  format logic testable and shared while the actual write stays on PUT.
- `internal/http/server.go` — register the route.

**Acceptance criteria.**
- Partial (`complete=false`): answers inserted, heading stays
  `## Open Questions`; a subsequent normal PUT of that body leaves the artefact
  `blocked` via existing autoblock (verified in [[open-questions-gui]] tests).
- Complete with all answers: heading renamed to `## Resolved Questions`;
  PUTting the result triggers the existing auto-unblock to `draft` with no
  status field in the payload.
- `complete=true` with any empty answer → `ErrIncompleteAnswers`, no rename.
- Idempotency: `ApplyAnswers` applied twice with the same inputs returns equal
  output; unrelated sections and frontmatter are preserved exactly.
- The preview endpoint never writes to disk (verified by mtime/no-index-event).

## Milestone 4 — Awaiting-answers query + authorisation

**Description.** Make the "N awaiting your answers" count and filtered list
cheap and live, and enforce the resolution permission. Resolved Q3 says the
badge navigates to the artefact list filtered to current blocked items, so the
list endpoint must expose an efficient blocked-with-open-questions filter.
Resolved Q2 scopes authorisation to **product-owner for now**.

**Files to change.**
- `internal/index/` (schema + upsert) — add an indexed boolean column
  `has_open_questions`, populated from `artifact.HasOpenQuestions(body)` on
  every upsert (the design tension noted in `autoblock.go`'s comment). Bump the
  schema version so startup re-scans.
- `internal/http/artifacts.go` — extend the list query with
  `awaiting_answers=true` (blocked AND `has_open_questions=1`), composable with
  the existing `count_only=true` so the badge can fetch just the count.
- `internal/http/open_questions.go` — require the caller to hold the
  `product-owner` role on the request (existing auth/role check) for both the
  preview endpoint and, at minimum, surface the same rule so the frontend can
  gate the action; return HTTP 403 otherwise (Resolved Q2, FR8). Add a small
  `isProductOwner(ctx)` helper reusing the existing role plumbing.

**Acceptance criteria.**
- `GET …/artifacts?awaiting_answers=true` returns only artefacts that are
  `blocked` with a non-empty `## Open Questions` section; adding
  `count_only=true` returns just `{count:N}`.
- The count reflects changes within one indexer cycle (WS `artifact.indexed`
  fires on the underlying write), so the frontend badge can update live (NFR2).
- A request from a user without the `product-owner` role to the preview
  endpoint returns HTTP 403 and performs no work.
- Startup re-scan populates `has_open_questions` for all existing artefacts.

## Notes on routing (Resolved Q5)

Routing after resolution is a deliberate, frontend-initiated action (FR7) and
reuses existing transition/queue APIs — no new backend write is required. Per
Resolved Q5, the two roles associated with an artefact are the role that
**creates** it and the role that **actions** it; the frontend derives the
requeue target from the artefact `type` (requirement vs `plan-*`) and
`assignees`. If a creator-role field is not already discoverable from the
existing artefact/run data, expose it read-only from run history — but do not
trigger any approval or requeue automatically.
