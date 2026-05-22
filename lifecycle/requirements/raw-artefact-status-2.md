---
title: Add 'raw' Artefact Status Before Draft
type: requirement
status: blocked
lineage: raw-artefact-status
created: "2026-05-22T10:30:00+10:00"
priority: normal
parent: lifecycle/ideas/raw-artefact-status.md
labels:
    - artefacts
    - workflow
    - feature
    - backend
    - frontend
assignees:
    - role: product-owner
      who: agent
---

# Add 'raw' Artefact Status Before Draft

## Problem

The `draft` status currently serves a dual purpose: it is both the initial state of a freshly captured idea and the state of a deliberately shaped document awaiting review. This conflation prevents tools, dashboards, and agents from distinguishing unprocessed input (e.g. brain dumps, voice-note transcriptions, hastily-typed jots) from content that has received editorial attention.

Without an explicit "unprocessed" state, reviewers see every captured fragment as if it were a considered draft, agents cannot prioritise which inputs need a first pass of structuring, and the workflow has no place to park content that exists but is not yet ready for clarification.

## Goals / Non-goals

### Goals

- Introduce a new artefact status `raw` that sits before `draft` in the lifecycle vocabulary.
- Allow new artefacts (especially those created via brain-dump / quick-capture flows) to start in `raw` rather than `draft`.
- Permit a `raw → draft` transition so a captured fragment can be promoted once a human or analyst has shaped it.
- Surface `raw` distinctly in the UI (filters, status badges, graph nodes, kanban/dashboard widgets) so unprocessed items are easy to find and triage.
- Keep `raw` non-terminal and non-blocking: it must coexist cleanly with existing `rejected` / `abandoned` / `blocked` escape hatches.

### Non-goals

- Auto-classifying or auto-promoting `raw` artefacts to `draft` (no AI-driven content shaping in this change).
- Migrating existing `draft` artefacts to `raw` (existing data is left untouched).
- Adding new capture entry-points (e.g. new voice-note ingest pipelines) beyond what already exists — those are separate ideas.
- Changing the meaning of any other existing status.
- Introducing a separate `raw` artefact *type* — only the status vocabulary changes.

## Detailed Requirements

### Functional

1. **Status vocabulary** — `raw` is added to `KnownStatuses` in [internal/artifact/artifact.go](../../internal/artifact/artifact.go). It is accepted by the parser without producing an `unknown status` parse error.
2. **Workflow transition: raw → draft** — The state machine in [internal/workflow/workflow.go](../../internal/workflow/workflow.go) permits the transition `raw → draft` for roles `product-owner`, `analyst`, and `system` (so the analyst agent can promote a captured fragment once it has been shaped, and the system actor can perform machine-initiated promotions).
3. **Workflow transition: draft → raw** — A `draft → raw` reverse transition is permitted for `product-owner` only, so a mis-classified item can be sent back to the unprocessed pool. (Optional / open question — see below.)
4. **Universal escape hatches** — Existing `any → rejected`, `any → abandoned`, and `any → blocked` rules continue to apply when the current state is `raw`. No new rules required, because they already use an empty `from` matcher.
5. **Default status for quick-capture flows** — The brain-dump quick-capture entry point (see `web/src/stores/brainDump.ts` and `web/src/components/idea/BrainDumpModal.vue`) creates new artefacts with status `raw` by default instead of `draft`. Other artefact-creation flows (full artefact editor, agent-produced artefacts, CLI scaffolds) continue to default to `draft`.
6. **API acceptance** — `POST /artifacts` and `PUT /artifacts/*` accept `raw` in the `status` field with no extra validation errors. `GET .../allowed-targets` returns the correct set of next statuses (`draft`, `rejected`, `abandoned`, `blocked`) when the current status is `raw`, gated by role.
7. **Status badges** — The status badge component (used in artefact detail view, list view, kanban/testing boards) renders `raw` with a distinct, neutral colour (e.g. a desaturated grey or slate tone) that visually de-emphasises it relative to `draft` and active-work statuses.
8. **Graph & dashboard surfaces** —
   - The active-status colour palette in [web/src/components/map/graphConstants.ts](../../web/src/components/map/graphConstants.ts) is extended with a `raw` entry so graph nodes in `raw` state render with the same colour as the badge.
   - The status-distribution dashboard widget (`StatusDistributionWidget.vue`) includes `raw` as a tracked status bucket.
   - The artefact list view and kanban board filters expose `raw` as a selectable status.
9. **Hide-done filtering** — `raw` is treated as a non-terminal, non-done status for the purposes of the existing "hide done" filter; it remains visible by default.
10. **Indexing** — The SQLite indexer stores and returns `raw` like any other status; no schema migration is required (status is stored as a free-text column).

### Non-functional

1. **Backward compatibility** — All existing artefacts (none of which use `raw`) continue to parse, index, and transition exactly as before. No data migration is required.
2. **Documentation** — `CLAUDE.md` (status vocabulary line) and the authoritative spec (§4.2 status vocabulary) are updated to list `raw` and describe its role.
3. **Test coverage** — Unit tests in [internal/workflow/workflow_test.go](../../internal/workflow/workflow_test.go) cover the new transition rule(s). An integration test exercises creating an artefact with status `raw` via the API, transitioning it to `draft`, and verifying allowed-targets behaviour.
4. **Accessibility** — The new badge colour passes WCAG AA contrast against both light and dark theme backgrounds.
5. **Consistency** — `raw` appears in every place an enumerated list of statuses appears in the codebase (no missed call sites). Specifically, every file matching `KnownStatuses` references or hard-coded status arrays must be reviewed.

## Acceptance Criteria

- [ ] `raw` is present in `KnownStatuses` in `internal/artifact/artifact.go` and a parser unit test confirms it produces no `unknown status` error.
- [ ] The workflow state machine allows `raw → draft` for `product-owner`, `analyst`, and `system` roles, and rejects it for unauthorised roles. Verified by `workflow_test.go`.
- [ ] `any → rejected`, `any → abandoned`, and `any → blocked` continue to work from a `raw` source state.
- [ ] An artefact can be created via `POST /artifacts` with `status: raw` and round-trips through the indexer and `GET /artifacts/{path}` unchanged.
- [ ] `GET /artifacts/{path}/allowed-targets` returns the correct set of next statuses when the current status is `raw`, scoped by the caller's role.
- [ ] The brain-dump quick-capture modal creates new artefacts with `status: raw` by default; other creation flows still default to `draft`. Verified by frontend unit/integration tests for `BrainDumpModal.vue` and `brainDump.ts`.
- [ ] The status badge component renders `raw` with a distinct visible colour and label in both light and dark themes; passes WCAG AA contrast.
- [ ] The 2D and 3D graph views render `raw`-status nodes with the new active-status palette entry.
- [ ] The status-distribution dashboard widget and the artefact list / kanban filters list `raw` as a selectable bucket.
- [ ] CLAUDE.md and the spec (§4.2) list `raw` in the status vocabulary with a one-line description.
- [ ] An integration test creates a `raw` artefact, transitions it to `draft` as an analyst, and asserts the change is reflected in the index and via WebSocket `artifact.indexed` events. Related: [[artefact-inline-status-change]].
- [ ] No regression in existing transition tests or end-to-end workflow tests.

## Open Questions

1. **Reverse transition `draft → raw`** — Is the reverse transition actually useful, or does the existing `any → rejected` / `clarifying → draft` machinery already cover the "I made a mistake" recovery path? If not required, drop requirement #3.
2. **Agent visibility** — Should the `analyst` agent be allowed to *read* `raw` artefacts as input (analogous to how it currently reads `draft` ideas), and if so, should it produce a `clarifying` artefact directly or first promote `raw → draft`? This intersects with [[analyst-agent-sees-draft-ideas]].
3. **Default for agent-produced artefacts** — Should any agent-generated content ever default to `raw` (e.g. transcribed voice notes from a future ingest agent), or is `raw` exclusively reserved for human quick-capture input?
4. **Badge colour token** — Which exact colour token in `web/src/styles/tokens.css` should represent `raw`? A new token may be needed; alternatively reuse an existing muted-neutral token.
5. **Auto-block interaction** — When an artefact in `raw` status has open questions in its body, should the existing auto-block-on-open-questions behaviour fire, or is auto-block deferred until the artefact has been promoted to `draft`?
