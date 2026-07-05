---
title: "Frontend Plan: Recursive Subdirectory Support for Artifact Directories"
type: plan-frontend
status: draft
lineage: idea-archiving
parent: lifecycle/requirements/idea-archiving-2.md
created: "2026-07-05T16:00:00+10:00"
priority: high
labels:
    - frontend
    - artifacts
    - enhancement
---

# Frontend Plan: Recursive Subdirectory Support for Artifact Directories

Implements the frontend half of [idea-archiving-2](../requirements/idea-archiving-2.md).
Consumes the `rel_path` field added by the backend plan [[idea-archiving-3-be]];
verified by the test plan [[idea-archiving-5-test]].

## Context from the code

- The router already uses a catch-all `pathMatch` array for artifact paths
  (`ArtifactEditorView.vue:42-45`) and the API interpolates the multi-segment
  path raw into the URL (`api/artifacts.ts:56,89`), so nested paths already load
  and save without routing changes.
- `LineageBreadcrumb.vue` already splits a path on `/` into segments
  (`LineageBreadcrumb.vue:17-28`) and renders `artifacts / <seg> / … / current`,
  but intermediate folder segments are plain non-clickable `<span>`s
  (LineageBreadcrumb.vue:41). This is the natural home for the folder breadcrumb
  (Resolved Q2 = "a breadcrumb").
- `ArtifactRow`/`ArtifactDetail` (`types/api.ts:84-104`) carry the full `path`
  but no root-relative field yet.
- The artifact list already shows the full `path`
  (`ArtifactListView.vue:370`, `.artifact-path`).
- Graph nodes use `node.id` (= full path) as identity/route key
  (`ArtifactModal.vue:97,109,203`; `GraphNode` in `types/api.ts:234-247`); they
  carry no path/subdir field and this plan does not add one (Non-goal: no graph
  redesign).

The requirement's only mandated UI change here is **exposing the folder path**;
grouping/filtering UI is deferred (Non-goals + Resolved Q2).

---

## Milestone 1 — Type + API surface for `rel_path`

**Description.** Thread the backend `rel_path` field through the TypeScript
types so components can read it (req #4). No behavioural change yet.

**Files to change.**
- `web/src/types/api.ts`
  - Add `rel_path: string` to `ArtifactRow` (types/api.ts:84-98). It is
    inherited by `ArtifactDetail` (types/api.ts:100-104), so both list and
    detail carry it.
- `web/src/api/artifacts.ts`
  - No transport change needed (the field rides existing JSON), but confirm
    `listArtifacts` (api/artifacts.ts:24) and `getArtifact` (api/artifacts.ts:54)
    pass it through untouched.

**Acceptance criteria.**
- `ArtifactRow.rel_path` is typed and available on store `items`
  (`stores/artifacts.ts:9`) and on `fetchOne` results (`stores/artifacts.ts:32`).
- A flat artifact deserialises with `rel_path` equal to its bare filename; a
  nested one with `sub/dir/file.md` — asserted in a Vitest type/parse test.
- `tsc`/`vue-tsc` build passes with the new field.

---

## Milestone 2 — Folder breadcrumb in the editor

**Description.** Render the artifact's folder location as a breadcrumb derived
from `rel_path`, so a nested/archived artifact visibly shows which folder it
lives in (Resolved Q2 = breadcrumb). Flat artifacts show no extra folder
segments (backward compatible, req #12).

**Files to change.**
- `web/src/components/artifact/LineageBreadcrumb.vue`
  - Prefer building the breadcrumb from `rel_path` (folder segments only) rather
    than the full repo path, so it reads `stage / folder / subfolder / <file>`
    (or just `stage / <file>` when flat). Keep the existing split logic
    (LineageBreadcrumb.vue:17-28) but source the folder segments from the new
    field. Leaf (current file) stays non-navigable text as today.
  - Folder segments remain display-only for this requirement (folder navigation
    /filtering is deferred). Do not make them route links yet — that would imply
    a folder view that is a Non-goal.
- `web/src/views/project/ArtifactEditorView.vue`
  - Pass `artifact.rel_path` (once loaded) into `LineageBreadcrumb`
    (ArtifactEditorView.vue:315-320) in addition to / instead of the raw path,
    guarding for the loading state where the detail isn't fetched yet.

**Acceptance criteria.**
- Opening `lifecycle/ideas/archive/foo.md` shows a breadcrumb containing an
  `archive` folder segment; opening `lifecycle/ideas/foo.md` shows no folder
  segment beyond the stage.
- A deeply nested `2026/q3/foo.md` shows `2026` and `q3` segments in order.
- The breadcrumb renders without error while the artifact is still loading
  (no `rel_path` yet).

---

## Milestone 3 — Surface the folder path in the artifact list

**Description.** Make the folder location visible/scannable in the list so
archived items are recognisable at a glance, reusing the existing path display
slot (req #4). This is display-only — no new filter control (deferred).

**Files to change.**
- `web/src/views/project/ArtifactListView.vue`
  - The row already renders `row.path` (ArtifactListView.vue:370). Keep the full
    path but ensure the folder portion is legible — e.g. render `row.rel_path`
    (root-relative, shorter, forward-slash) as the primary path chip, falling
    back to `row.path`. Confirm sorting/pagination
    (`paginatedItems`, ArtifactListView.vue:345) is unaffected.
  - No new column, facet, or filter — folder filtering is a deferred follow-up.

**Acceptance criteria.**
- The list row for a nested artifact shows its `rel_path`
  (e.g. `archive/foo.md`); a flat artifact shows its bare filename.
- Existing list behaviour (sort, paginate, WS-driven refresh on
  `artifact.indexed`, ArtifactListView.vue:187-204) is unchanged.
- No regression in the existing `ArtifactListView` Vitest specs.

---

## Milestone 4 — Live-refresh & navigation parity for nested artifacts

**Description.** Confirm that create/edit/move of nested artifacts flows through
the existing WS-driven refresh and routing exactly as for flat artifacts (req
#2, #7; AC: "editable via the editor identically"). Primarily verification plus
any small fixes uncovered.

**Files to change.**
- `web/src/stores/artifacts.ts` / `web/src/views/project/ArtifactEditorView.vue`
  - Verify `fetchOne(project, artifactPath)` (stores/artifacts.ts:32) and
    `save()` PUT (ArtifactEditorView.vue:209-247) work with multi-segment paths
    (they should — path is interpolated raw, api/artifacts.ts:89). Fix only if a
    segment needs encoding while preserving `/` separators.
  - Verify the `useExternalChange` auto-refresh composable (referenced by
    [[editor-live-refresh-on-disk-change]]) matches WS `file.changed` paths for
    nested files — its path filtering must compare on the full/rel path
    consistently, not assume a flat filename.
- `web/src/components/map/*` — no change; graph continues to key on `node.id`
  (full path), which already handles nested paths.

**Acceptance criteria.**
- Editing and saving a nested artifact updates it in place (no duplicate,
  correct `expected_sha` flow) — parity with a flat artifact.
- An external on-disk edit to a nested artifact triggers the editor's
  auto-refresh / conflict banner exactly as for a flat file (extends the
  [[editor-live-refresh-on-disk-change]] scenarios).
- Clicking a nested artifact's graph node routes to its editor via `node.id`
  and loads correctly.

---

## Out of scope (deferred per requirement Non-goals + Resolved Q2)

- Folder-based grouping, folder tree, folder filter facet, or a dedicated folder
  view — a follow-up UI requirement.
- Making breadcrumb folder segments navigable (implies a folder view).
- Any graph redesign; graph keeps `node.id`-as-path identity.
