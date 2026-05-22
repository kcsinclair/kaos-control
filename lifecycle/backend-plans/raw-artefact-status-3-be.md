---
title: Backend Plan — Add 'raw' Artefact Status Before Draft
type: plan-backend
status: in-development
lineage: raw-artefact-status
parent: lifecycle/requirements/raw-artefact-status-2.md
---

# Backend Plan — Add `raw` Artefact Status Before Draft

Implements the backend side of [[raw-artefact-status]]: extend the status
vocabulary, extend the workflow state machine, ensure the indexer and
API round-trip `raw` cleanly, and confirm auto-block-on-open-questions
still fires from a `raw` source.

Cross-links: integration scenarios live in [[raw-artefact-status]] test
plan; the frontend defaults / surfaces live in
[[raw-artefact-status]] frontend plan.

## Milestone 1 — Extend the status vocabulary

**Description.** Add `raw` to `KnownStatuses` so the parser, validator,
indexer, and API stop treating it as an unknown status. Update the
authoritative spec and CLAUDE.md status vocabulary line to describe the
new state.

**Files to change.**
- `internal/artifact/artifact.go` — add `"raw": true` to `KnownStatuses`.
- `internal/artifact/artifact_test.go` — add a sub-test that parses an
  artefact with `status: raw` and asserts no `unknown status` parse
  error is emitted.
- `CLAUDE.md` — extend the "Status vocabulary" sentence with `raw` and
  a brief description ("unprocessed quick-capture input").
- `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md`
  §4.2 — add `raw` to the status vocabulary table with its description.

**Acceptance criteria.**
- `go test ./internal/artifact/...` passes with the new test green.
- Parsing a markdown file whose frontmatter sets `status: raw` produces
  an `Artifact` with `FM.Status == "raw"` and an empty `ParseErrs`
  slice.
- `KnownStatuses["raw"]` evaluates to `true` from any package.

## Milestone 2 — Add workflow transitions involving `raw`

**Description.** Extend the default rule matrix in
`internal/workflow/workflow.go` so the lifecycle has explicit
transitions into and out of `raw`. Universal escape hatches
(`any → rejected`, `any → abandoned`, `any → blocked`) already use an
empty `from` matcher and require no new rules.

Transitions to add:

| from   | to    | roles                                      |
|--------|-------|--------------------------------------------|
| `raw`  | `draft` | `product-owner`, `analyst`, `system`     |
| `draft`| `raw`   | `product-owner`                          |

The `system` actor is required so the analyst agent (running as
`system` for machine-initiated promotions) can lift a captured fragment
to `draft` without product-owner intervention — matches the
existing pattern for `system` in `blocked ↔ draft`.

**Files to change.**
- `internal/workflow/workflow.go` — append the two new entries to
  `defaultRules`. Place them near the top of the matrix so the lifecycle
  reads top-to-bottom; mirror the existing comment style.
- `internal/workflow/workflow_test.go` — add table-driven cases:
  - `raw → draft` allowed for each of `product-owner`, `analyst`,
    `system`; denied for `backend-developer`, `qa`, `reviewer`.
  - `draft → raw` allowed for `product-owner`; denied for `analyst`,
    `system`, and the developer roles.
  - `raw → rejected` (`reviewer`), `raw → abandoned`
    (`product-owner`), `raw → blocked` (`system`, any agent role) all
    allowed (cover the escape hatches without adding rules).
  - `AllowedTargets("raw", ["analyst"], "idea")` returns at least
    `["draft", "blocked"]` and never `"raw"` itself (self-transition
    guard).

**Acceptance criteria.**
- `go test ./internal/workflow/...` passes with new cases green and no
  existing cases regressed.
- Calling `Engine.CanTransition("raw", "draft", []string{"analyst"}, "idea")`
  returns `true`.
- Calling `Engine.CanTransition("draft", "raw", []string{"analyst"}, "idea")`
  returns `false`.
- Product-owner remains a superuser (all transitions return `true` for
  product-owner regardless of source).

## Milestone 3 — API round-trip for `raw`

**Description.** Confirm the HTTP layer accepts `raw` end-to-end with
no extra changes: `POST /artifacts` and `PUT /artifacts/{path}` already
delegate to the parser + indexer, both of which become permissive once
Milestone 1 lands. This milestone audits the handlers, adds one
defensive check if missing, and adds an integration test scenario
(coordinated with the test plan).

**Files to change.**
- `internal/http/artifacts.go` (and any sibling status-aware handler) —
  read through every place that switches on status; remove any
  hard-coded status whitelist that would still reject `raw`. Search
  pattern: `\"draft\"|\"clarifying\"|\"planning\"`. No code change is
  expected, but file the audit findings in the commit message.
- `internal/http/artifacts_test.go` (or whatever the existing
  REST integration file is) — add a sub-test that:
  1. `POST /artifacts` with frontmatter `status: raw, type: idea`.
  2. Asserts a 201 response and a present `path` in the body.
  3. `GET /artifacts/{path}` returns `status: raw` with no warnings.
  4. `GET /artifacts/{path}/allowed-targets` as role `analyst` returns
     a set containing `draft` and `blocked` but not `raw` itself.

**Acceptance criteria.**
- `go test ./internal/http/...` passes including the new scenario.
- A round-trip `POST → GET` of a `raw` artefact produces no
  `ParseErrs`, no `unknown status` log line, and the SQLite index row
  stores `status='raw'` (verified by inspecting the index in the test).
- `allowed-targets` returns the expected set for each role:
  - `analyst` from `raw` → contains `draft`, `blocked`; excludes `raw`.
  - `reviewer` from `raw` → contains `rejected`.
  - `product-owner` from `raw` → returns the full set (superuser).

## Milestone 4 — Auto-block from `raw` on open questions

**Description.** The auto-block reactor in
`internal/index/autoblock.go` currently transitions any non-blocked
artefact to `blocked` when an `## Open Questions` section is detected,
delegated through the workflow engine as the `system` actor. With
Milestone 2 the universal `any → blocked` rule for `system` matches
`raw` already, so no logic change is required — but we must verify and
lock the behaviour with a test.

The reverse path (`blocked → draft`) is already permitted for `system`;
auto-unblock from a previously-raw-now-blocked artefact will land in
`draft`, which is the intended behaviour per resolved question #5.

**Files to change.**
- `internal/index/autoblock_test.go` — add a sub-test that:
  1. Writes a `raw` artefact whose body contains an `## Open Questions`
     section with one bullet.
  2. Indexes it.
  3. Asserts the on-disk frontmatter is rewritten to `status: blocked`
     and the assignee `role: product-owner, who: agent` is appended.
  4. Removes the `## Open Questions` section, re-indexes, asserts the
     artefact is auto-transitioned to `draft` (not back to `raw` — the
     existing reactor only knows how to unblock to `draft`).

**Acceptance criteria.**
- `go test ./internal/index/...` passes with the new sub-test.
- Manual reproduction in `make run`: creating a `raw` artefact with
  `## Open Questions` flips it to `blocked` within ~150 ms (debounce)
  and broadcasts an `artifact.indexed` event.
- The reactor produces no log warning when the source state is `raw`
  (the workflow check passes).

## Milestone 5 — Audit hard-coded status arrays

**Description.** Non-functional requirement #5 asks that `raw` appears
everywhere an enumerated list of statuses appears. This milestone walks
the codebase, identifies hard-coded status arrays, and updates them.
Frontend arrays are handled in the frontend plan; this milestone covers
Go-side hard-coded arrays.

**Search targets** (run with Grep, not by hand):
- `\"draft\".*\"clarifying\"` — list literals.
- `\"approved\".*\"done\"` — terminal-bucket literals.
- `KnownStatuses` references — every call site.
- `statuscheck` package — confirm `raw` does not produce a spurious
  "needs status update" suggestion.

**Files to change.** Determined by the audit. Expected hits include:
- `internal/initcmd/templates/config.yaml.tmpl` — if the template lists
  statuses anywhere (likely not, but verify).
- Any dashboard / aggregation that buckets statuses (e.g. velocity,
  status distribution backend feeders).
- `internal/statuscheck/statuscheck.go` — if `raw` is treated as a
  "needs update" candidate, exclude it (a raw artefact is allowed to
  sit indefinitely until a human or analyst promotes it).

**Acceptance criteria.**
- A grep for hard-coded status string literals in `internal/...`
  returns no list that omits `raw` (where inclusion makes semantic
  sense — pure transition matrices are exempt).
- `make lint` and `make test-unit` are both green.
- The commit message includes the list of files audited with a one-
  line note on whether each needed a change.

## Out of scope (for clarity)

- The frontend status badge, brain-dump default, graph palette entry,
  dashboard widget and list/kanban filter — see the frontend plan.
- End-to-end integration scenarios (API + WS round-trip with frontend
  expectations) — see the test plan.
- Auto-promotion of `raw → draft` by any AI process — explicitly
  ruled out in the requirement's non-goals.
