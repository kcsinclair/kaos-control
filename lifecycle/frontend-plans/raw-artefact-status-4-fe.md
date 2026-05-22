---
title: Frontend Plan — Add 'raw' Artefact Status Before Draft
type: plan-frontend
status: done
lineage: raw-artefact-status
parent: lifecycle/requirements/raw-artefact-status-2.md
---

# Frontend Plan — Add `raw` Artefact Status Before Draft

Implements the UI side of [[raw-artefact-status]]: a new badge style, a
new active-status palette entry, a new default for the brain-dump
quick-capture flow, and consistent inclusion of `raw` across every
status-aware surface (filters, dropdowns, dashboards, kanban).

Cross-links: backend behaviour lives in [[raw-artefact-status]] backend
plan; verification scenarios live in [[raw-artefact-status]] test plan.

## Milestone 1 — Design tokens for the `raw` badge

**Description.** Introduce a new `raw` colour token in `tokens.css`
(per resolved question #4 — a new token, not a reuse). The colour
should be a desaturated neutral so `raw` reads as "not yet shaped"
rather than "active work". Provide light, dark, and prefers-dark
variants matching the existing pattern.

Suggested values (final pick can shift to hit WCAG AA but should stay
in the neutral/slate family):

- Light theme: `--badge-raw-bg: #f1f5f9; --badge-raw-text: #475569;`
  (slate-100 / slate-600).
- Dark theme: `--badge-raw-bg: #1e293b; --badge-raw-text: #cbd5e1;`
  (slate-800 / slate-300).

**Files to change.**
- `web/src/styles/tokens.css` — three new variable pairs, one each in
  the light root block, dark root block, and `prefers-color-scheme:
  dark` block. Keep the ordering consistent with the existing
  alphabetical-by-status pattern within each block.
- `web/src/components/artifact/StatusDropdown.vue` (or wherever the
  badge styles read the token) — confirm the tokens are referenced via
  `var(--badge-raw-bg)` / `var(--badge-raw-text)` selectors matching
  the existing per-status pattern.

**Acceptance criteria.**
- The badge for a `raw` artefact renders with the new colours in both
  light and dark themes, verified manually in `pnpm dev`.
- Computed contrast ratio of `--badge-raw-text` against
  `--badge-raw-bg` is ≥ 4.5:1 in both themes (run a quick check with a
  contrast calculator; record the measured value in the commit
  message).
- No other badge styles change visually (regression check by skimming
  the artefact list view).

## Milestone 2 — Brain-dump quick-capture defaults to `raw`

**Description.** Change the default status produced by the quick-
capture flow from `draft` to `raw`. Only this entry point changes —
the full artefact editor, agent-produced artefacts, and CLI scaffolds
keep `draft`.

**Files to change.**
- `web/src/stores/brainDump.ts` — the literal `status: 'draft'`
  (around line 131 per current state) becomes `status: 'raw'`. If the
  default is centralised (e.g. a constant), update the constant.
- `web/src/components/idea/BrainDumpModal.vue` — confirm no copy in
  the modal claims the captured item lands as "draft"; update copy
  where it does (e.g. a success toast) to refer to "raw" or "captured"
  language without binding the UI to the exact word.

**Acceptance criteria.**
- Submitting the brain-dump modal creates an artefact whose
  frontmatter on disk reads `status: raw`.
- Creating an artefact via `POST /artifacts` from the full editor view
  still defaults to `status: draft`.
- A Vue unit test for `brainDump.ts` (or component test for
  `BrainDumpModal.vue`) asserts the payload sent to the API contains
  `status: 'raw'`.

## Milestone 3 — Active-status palette entry for the graph

**Description.** Extend `activeStatusColors` in `graphConstants.ts` so
2D and 3D graph nodes in `raw` state render with the same colour as
the badge. Add entries to both the dark and light palettes; match the
token hex values from Milestone 1.

**Files to change.**
- `web/src/components/map/graphConstants.ts` — add
  `'raw': '#cbd5e1',` (or chosen hex) to the dark `activeStatusColors`
  block, and `'raw': '#475569',` to the light block. Ordering: place
  next to `clarifying` to keep the early-lifecycle entries grouped.

**Acceptance criteria.**
- Creating an artefact in `raw` state and viewing it in the 2D and 3D
  map renders the node in the new colour.
- The colour matches the badge colour to within a perceptual delta
  (eyeball check is fine; the requirement is "same colour as the
  badge").
- No regression on other node colours — a quick sweep of an existing
  multi-status lineage should show all other statuses unchanged.

## Milestone 4 — Status-distribution dashboard widget

**Description.** Surface `raw` as a tracked bucket in the dashboard
widget. The widget already iterates a status list; add `raw` to that
list and to any colour / label mapping the widget owns.

**Files to change.**
- `web/src/components/dashboard/widgets/StatusDistributionWidget.vue` —
  add `'raw'` to the widget's known-statuses array. If the widget reads
  its colours from `tokens.css` directly, no further change is
  required; if it has its own colour map, add a `raw` entry matching
  the badge tokens.

**Acceptance criteria.**
- With at least one `raw` artefact present, the widget renders a `raw`
  segment in its distribution chart with the new badge colour.
- With zero `raw` artefacts present, the widget renders cleanly (no
  empty slice / division-by-zero bug introduced).
- The widget's bucket count for non-`raw` statuses is unchanged.

## Milestone 5 — List and kanban filters expose `raw`

**Description.** Audit every view that exposes a status filter or
dropdown and add `raw` to the selectable set. Confirm the "hide done"
filter treats `raw` as non-terminal (visible by default).

**Files to change.**
- `web/src/views/project/ArtifactListView.vue` — extend the status
  filter array / option list to include `raw`.
- `web/src/views/project/KanbanBoardView.vue` — add a `raw` column (or
  bucket — whichever the kanban's grouping mechanism uses). Order
  before `draft` so the lifecycle reads left-to-right.
- `web/src/views/project/TestingBoardView.vue` — verify no hard-coded
  status list omits `raw`; update if it does.
- `web/src/components/artifact/TransitionDialog.vue` — verify the
  transition dialog enumerates statuses from the backend (via
  `allowed-targets`) rather than a hard-coded list. If it has a hard-
  coded list, replace with the API-driven approach OR add `raw`.
- `web/src/components/artifact/StatusDropdown.vue` — same audit.

**Acceptance criteria.**
- The list view's status filter shows `raw` as a selectable option;
  selecting it filters to the `raw` artefacts only.
- The kanban view shows a `raw` column (or equivalent bucket) at the
  start of the lifecycle.
- The "hide done" toggle does not hide `raw` artefacts.
- The status dropdown / transition dialog on a `raw` artefact lists
  the legal next states (`draft`, `rejected`, `abandoned`, `blocked`)
  for the current user's role, sourced from the backend's
  `allowed-targets` endpoint.

## Milestone 6 — Cross-surface audit

**Description.** Sweep the frontend for any remaining hard-coded
status arrays or status-aware copy that would silently drop `raw`.
Mirror the backend's Milestone 5 audit on the Vue side.

**Search targets** (run with Grep):
- `\"draft\".*\"clarifying\"` — array literals.
- `STATUS_ORDER`, `statusOrder`, `allStatuses`, `statusList`.
- Any kanban or velocity component that buckets statuses.

**Files to change.** Determined by the audit. Expected hits include:
- `web/src/views/project/ArtifactListView.vue` (already in Milestone 5,
  but may have more than one literal).
- `web/src/views/project/KanbanBoardView.vue`.
- `web/src/views/project/TestingBoardView.vue`.
- `web/src/components/artifact/TransitionDialog.vue`.

**Acceptance criteria.**
- A grep for hard-coded status array literals in `web/src/...` returns
  no list that omits `raw` where inclusion makes semantic sense.
- `pnpm typecheck` and `pnpm lint` are both green.
- The commit message records each audited file with a one-line note
  on whether it needed a change.

## Out of scope

- Backend workflow rules, status vocabulary, indexer behaviour — see
  the backend plan.
- Designing or building a new quick-capture surface — only the
  existing brain-dump flow changes its default.
- Migrating existing `draft` artefacts to `raw`.
