---
title: Frontend Plan — Release Goal and Description Fields
type: plan-frontend
status: draft
lineage: release-goal-and-description
parent: lifecycle/requirements/release-goal-and-description-2.md
created: "2026-09-03T11:15:00Z"
---

# Frontend Plan — Release Goal and Description Fields

Implements the SPA side of
[lifecycle/requirements/release-goal-and-description-2.md](../requirements/release-goal-and-description-2.md):
make `goal` and `description` editable in the release editor, show `goal` as a
one-line subtitle wherever releases are listed, and render `description` as
sanitised markdown in the release detail view. Depends on the backend contract
in [[release-goal-and-description]] (the `-3-be` plan): the API already accepts
and returns both fields.

## Architecture conformance

Per [architecture-summary.md](../architecture/architecture-summary.md) and
[[adr-0004-embedded-spa-single-binary]], the SPA is embedded in the single
binary — no new runtime dependency is added. `description` is rendered through
the **existing** sanitised `markdown-it` path
(`web/src/components/artifact/MarkdownPreview.vue`, configured `html: false`), so
no new HTML/script-execution surface is introduced (requirement DR-10). `goal` is
soft-capped in the UI at 120 chars (Resolved Question 3) — the only client-side
length rule; the server imposes none.

---

## Milestone 1 — Types (DR-6)

**Description.** Extend the release TypeScript contract with the two optional
fields across read and write payloads.

**Files to change**
- `web/src/types/release.ts`
  - `Release`: add `goal: string` and `description: string` (the backend always
    returns both as strings, empty when unset — see the non-omit JSON tags in the
    `-3-be` plan).
  - `CreateReleasePayload`: add `goal?: string` and `description?: string`.
  - `UpdateReleasePayload`: add `goal?: string` and `description?: string`
    (omitted key = preserve; explicit `''` = clear, matching backend PUT
    semantics).

**Acceptance criteria**
- `web` type-checks (`pnpm -C web build` / `vue-tsc`) with the new fields.
- `normaliseDates` in `web/src/api/releases.ts` still compiles; if needed, extend
  it to default `goal`/`description` to `''` for defensive parity with the
  date/slug normalisation already there.

---

## Milestone 2 — Release editor: goal input + description textarea (DR-6)

**Description.** Add a single-line `goal` input (maxlength 120) and a multi-line
`description` textarea to the create/edit form, wired into submit as optional
fields.

**Files to change**
- `web/src/components/releases/ReleaseFormModal.vue`
  - Add `goal` and `description` refs; in `onMounted`, seed them from
    `props.release?.goal` / `props.release?.description` (edit mode).
  - Template: a `<input type="text" maxlength="120">` for goal (with a short
    helper/counter is optional) and a `<textarea>` for description, styled to
    match the existing form fields.
  - On submit, include `goal` and `description` in the create/update payload.
    Send trimmed values; send `''` (not omit) when the user clears a field in
    **edit** mode so the backend clears it. On **create**, empty values may be
    omitted or sent as `''` — either leaves the release without them.

**Acceptance criteria**
- Creating a release with a goal and description persists both (visible after
  reload / in the detail view).
- Editing a release can set, change, and **clear** either field; clearing an
  existing value removes it from the file (backend clear-on-`''`).
- Submitting with both empty leaves the release without them (no error).
- `goal` input enforces the 120-char maxlength in the browser.

---

## Milestone 3 — Release detail view: goal subtitle + description markdown (DR-7)

**Description.** Show `goal` as a one-line subtitle under the release name and
render `description` as markdown in the detail modal; omit each section when the
value is absent.

**Files to change**
- `web/src/components/releases/ReleaseDetailModal.vue`
  - Render `detail.goal` as a one-line subtitle beneath the release title, shown
    only via `v-if="detail?.goal"` (no empty placeholder).
  - Render `detail.description` (when non-empty) through the existing
    `MarkdownPreview` component (`:source="detail.description"` `:project`),
    reusing the sanitised `markdown-it` path and wiki-link handling. Wrap in
    `v-if="detail?.description"` so an absent value omits the whole section.
  - Import `MarkdownPreview` from `@/components/artifact/MarkdownPreview.vue`.

**Acceptance criteria**
- With `goal` set, a one-line subtitle appears under the release name in the
  detail view; absent → nothing rendered.
- With `description` set, it renders as markdown (headings, lists, links) via the
  existing sanitised renderer; absent → section omitted.
- No raw HTML/script from `description` executes (inherits `html: false` from
  `MarkdownPreview`).

---

## Milestone 4 — Goal subtitle in list / roadmap / kanban contexts (DR-7)

**Description.** Surface `goal` as a one-line subtitle under the release name in
at least the release **list** and **roadmap/kanban** release contexts. Absent →
nothing rendered. `description` is **detail-view only** (Non-goal: no list/kanban
column for it).

**Files to change**
- `web/src/components/releases/GanttChart.vue` — where each release row/label
  shows `release.name`, add a conditional one-line `goal` subtitle (truncated /
  `text-overflow: ellipsis` to guarantee single-line rendering in the timeline).
- `web/src/views/project/RoadmapView.vue` — if release names are surfaced outside
  the Gantt rows (e.g. detail badge cache / headers), add the same conditional
  subtitle there.
- `web/src/views/project/KanbanBoardView.vue` — if releases appear as a kanban
  swimlane/column header or chip, add the `goal` subtitle under the name;
  otherwise no change (confirm during implementation which release surface the
  kanban actually renders).

> Implementation note: the release list/roadmap/kanban surfaces read from
> `useReleasesStore().releases` (type `Release`), which now carries `goal` after
> Milestone 1 — no extra fetch needed for the subtitle. The `RoadmapView` graph
> node payload (`/releases/graph`) is **not** required to carry `goal`; the
> subtitle is driven from the store's release list.

**Acceptance criteria**
- `goal` (when set) appears as a one-line subtitle under the release name in the
  roadmap (Gantt) and the release list; absent → nothing rendered.
- The subtitle never wraps to a second line or breaks row layout (CSS
  single-line truncation).
- No `description` is shown in list/roadmap/kanban contexts (detail-view only).

---

## Out of scope (per requirement Non-goals)

- No new list/kanban column for `description`.
- No `goal` sorting/filtering control (display-only, Resolved Question 2).
- No realtime change — the fields ride the existing releases WebSocket payload
  (`useReleasesSocket` / `useReleasesStore`); no new socket surface.
