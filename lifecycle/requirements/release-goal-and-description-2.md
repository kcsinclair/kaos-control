---
title: Release Goal and Description Fields
type: requirement
status: blocked
lineage: release-goal-and-description
created: "2026-08-26T11:45:00+10:00"
priority: normal
parent: lifecycle/ideas/release-goal-and-description.md
labels:
    - releases
    - feature
    - enhancement
    - ui
    - frontend
    - backend
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Release Goal and Description Fields

## Problem

Release artifacts today carry only `title`, `status`, `start_date`, and
`end_date` (see the frontmatter parsed at
[internal/release/file.go](../../internal/release/file.go) and the `releases`
cache columns at [internal/index/index.go](../../internal/index/index.go#L2111)).
There is nowhere to record **what a release is trying to achieve** or **why**.
Stakeholders scanning the roadmap, kanban, or release list get a name and a set
of dates but no statement of intent, and there is no home for scope rationale,
non-goals, or delivery notes on the release itself.

We want two **optional** fields:

- **`goal`** — a short, one-line intent statement (e.g. *"More and better LLM
  support"*) visible wherever releases are listed.
- **`description`** — a longer free-form markdown narrative for richer context
  (scope rationale, non-goals, links to related requirements, delivery notes),
  shown in the release detail view.

Both fields MUST be optional so every existing release file remains valid with
**no migration** — consistent with [[index-is-a-cache]] (a schema change to the
`releases` cache is a rebuild-from-disk, never a data migration; see
[[adr-0003-pure-go-sqlite-index]]).

## Goals / Non-goals

**Goals**

- Add optional `goal` and `description` to the release model end-to-end:
  markdown frontmatter → release parser/marshaller → `releases` cache → REST API
  → frontend types, editor, and detail view.
- Preserve disk-as-authoritative: the markdown file is written first, the cache
  follows (carried forward from [[release-artefacts]] DR-1).
- Surface `goal` in release list and kanban/roadmap contexts; surface both
  `goal` and `description` in the release detail view; make both editable in the
  release editor.
- Keep existing releases valid and unchanged when neither field is set.

**Non-goals**

- No change to `status`, date, slug, lineage, or assignment semantics.
- No required-field or validation-gate changes; both new fields are optional and
  free-form.
- No new list/kanban column purely for `description` (it is detail-view only).
- No full-text search or indexing of `description` beyond storing it in the
  cache for read-back (search is out of scope for this requirement).
- No backfill of `goal`/`description` onto historical releases.

## Detailed Requirements

### Functional

**DR-1 — Frontmatter fields (disk is authoritative).**
The release markdown frontmatter MAY carry two new optional keys:

```yaml
goal: "More and better LLM support"
description: |
  Free-form **markdown** narrative. Multiple lines allowed.
  Scope rationale, non-goals, links to [[related-requirements]].
```

- Both keys are optional. Absent, empty, or whitespace-only values MUST parse to
  an empty field and MUST NOT produce a validation error.
- `goal` is a single-line string. If a multi-line value is supplied it is
  accepted but treated as free text (no newline validation); the UI presents it
  as one line.
- `description` is a (potentially multi-line) markdown string, marshalled using
  a YAML block scalar so round-tripping preserves newlines.

**DR-2 — Parser & marshaller round-trip.**
[internal/release/file.go](../../internal/release/file.go) `Parse` MUST read
`goal` and `description` into new `File` fields, and `Marshal` MUST emit them
(omitting each key entirely when empty, so unchanged files are byte-stable except
where the field is actually set). Parsing an existing release file with neither
key present MUST yield empty values and succeed.

**DR-3 — Fields carried through the write path.**
The in-memory `Release` struct
([internal/release/release.go](../../internal/release/release.go)) gains `Goal`
and `Description` fields, and `DiskSync.Write`
([internal/release/disksync.go](../../internal/release/disksync.go#L79)) MUST
populate them on the `File` it marshals. **Note:** `Write` currently constructs
its `File` without carrying `File.Body`; whichever design is chosen for
`description` (frontmatter vs. markdown body — see Open Questions), the write
path MUST NOT silently drop the field on an API edit. The recommended design
(description in frontmatter) sidesteps the existing body-drop gap entirely.

**DR-4 — Cache columns & rehydrate.**
The `releases` cache table
([internal/index/index.go](../../internal/index/index.go#L2111)) gains `goal` and
`description` TEXT columns (default `''`). `Rehydrate`
([internal/release/rehydrate.go](../../internal/release/rehydrate.go)) and the
`UpsertBySlug` insert ([internal/release/store.go](../../internal/release/store.go#L253))
MUST populate them from the parsed file. Because the cache is rebuilt from disk
on every load and a schema/driver mismatch triggers a rebuild rather than a
migration ([[adr-0003-pure-go-sqlite-index]], [[index-is-a-cache]]), adding these
columns requires **no data migration** — only the schema-version bump that forces
a rebuild-from-disk.

**DR-5 — REST API.**
`GET /releases`, `GET /releases/{slug}`, `POST /releases`, and
`PUT /releases/{slug}` ([internal/http/releases.go](../../internal/http/releases.go))
MUST accept and return `goal` and `description`:

- Response bodies include `"goal"` and `"description"` (empty string when unset).
- Create/Update request bodies accept optional `goal` and `description`; omitting
  a key on `PUT` leaves the stored value unchanged, and explicitly sending `""`
  clears it.
- Writes persist to the markdown file first, then refresh the cache row
  synchronously before responding (read-after-write consistency, per
  [[release-artefacts]] DR-1).

**DR-6 — Frontend model & editor.**
`web/src/types/release.ts` `Release`/`ReleaseDetail`/`CreateReleasePayload`/
`UpdateReleasePayload` gain optional `goal` and `description`. The release editor
([web/src/components/releases/ReleaseFormModal.vue](../../web/src/components/releases/ReleaseFormModal.vue))
gains:

- a single-line `goal` input (with a sensible maxlength, e.g. 120 chars), and
- a multi-line `description` textarea.

Both are optional; submitting empty leaves the release without them.

**DR-7 — Frontend display.**

- The `goal` (when present) is shown in the **release detail view**
  ([ReleaseDetailModal.vue](../../web/src/components/releases/ReleaseDetailModal.vue))
  and in at least the release **list** and **roadmap/kanban** release contexts as
  a one-line subtitle under the release name; when absent, nothing is rendered
  (no empty placeholder).
- The `description` (when present) is rendered as **markdown** in the release
  detail view, reusing the existing `markdown-it` rendering path used elsewhere
  in the SPA. Absent → section omitted.

### Non-functional

**DR-8 — Backward compatibility / no migration.** Loading a `lifecycle/releases/`
directory of pre-existing release files (no `goal`/`description`) MUST succeed and
present empty fields, with no error, no file rewrite, and no migration step.

**DR-9 — Round-trip stability.** Creating or editing a release via the API and
then wiping the `releases` **table** (not the files) and restarting MUST
reproduce identical `goal`/`description` values from disk (disk is authoritative;
the cache is rebuildable).

**DR-10 — Length & safety.** `description` is stored and rendered as untrusted
markdown through the SPA's existing sanitised markdown renderer; this requirement
introduces no new HTML/script execution surface beyond what release/artifact
markdown already uses. No server-side length cap is imposed on `description`
beyond practical file-size limits; `goal` is soft-capped in the UI (DR-6).

### Architecture-Breaking Requirements

Assessed against
[lifecycle/architecture/architecture-summary.md](../architecture/architecture-summary.md).
**None of this requirement is architecture-breaking.** Specifically:

- **Single self-contained binary / pure-Go stack** — satisfied. Two TEXT columns
  and two frontmatter fields; no new dependency, datastore, or cgo. No conflict
  with [[adr-0003-pure-go-sqlite-index]] or [[adr-0004-embedded-spa-single-binary]].
- **Local filesystem is the source of truth** — satisfied and reinforced. The
  markdown file is authoritative; the added cache columns are rebuildable from
  disk (DR-4, DR-9). Adding columns is a **rebuild, not a migration**, exactly as
  [[index-is-a-cache]] mandates — this is the one place a naïve implementation
  could break the invariant (e.g. by ALTER-migrating instead of bumping the
  schema version and rebuilding), so it is called out explicitly rather than
  worked around silently.
- **Realtime / collaboration** — unchanged. Release create/update already
  broadcasts over the existing releases WebSocket
  ([useReleasesSocket.ts](../../web/src/composables/useReleasesSocket.ts)); the new
  fields ride the existing payload with no new realtime surface.
- **Offline / scale / security-compliance / cost** — no change. Fields are
  optional metadata; no auth, tenancy, or scale characteristic shifts.

No new ADR is required. If, contrary to the recommendation in Open Questions, the
team elects to store `description` in the markdown **body** (rather than
frontmatter), that still fits the architecture but would require fixing the
existing `DiskSync.Write` body-drop behaviour (DR-3) and is noted below rather
than resolved here.

## Acceptance Criteria

- [ ] A release file with `goal:` and a multi-line `description:` block scalar
      parses correctly, and re-marshalling round-trips both values without loss.
- [ ] A pre-existing release file with **neither** key loads without error and
      exposes empty `goal`/`description` (no file rewrite, no migration).
- [ ] `POST /releases` and `PUT /releases/{slug}` accept `goal`/`description`;
      `GET /releases` and `GET /releases/{slug}` return them (empty string when
      unset). Omitting a key on `PUT` preserves the stored value; sending `""`
      clears it.
- [ ] After a create/edit via the API, wiping the `releases` **table** and
      restarting reproduces identical `goal`/`description` from disk (verifies
      [[index-is-a-cache]] compliance).
- [ ] The release editor
      ([ReleaseFormModal.vue](../../web/src/components/releases/ReleaseFormModal.vue))
      lets a user set, edit, and clear both fields; empty submissions leave the
      release without them.
- [ ] `goal` (when set) appears as a one-line subtitle in the release list and
      roadmap/kanban release contexts and in the detail view; absent → nothing
      rendered.
- [ ] `description` (when set) renders as markdown in the release detail view via
      the existing sanitised `markdown-it` path; absent → section omitted.
- [ ] Adding the two cache columns is delivered as a schema-version bump that
      triggers a rebuild-from-disk, not an in-place data migration
      ([[adr-0003-pure-go-sqlite-index]]).
- [ ] Related: aligns with [[release-artefacts]] (markdown-authoritative release
      write path) and does not reintroduce a DB-origin write.

## Open Questions

1. **Where does `description` live — frontmatter or markdown body?**
   *Recommendation:* **frontmatter** (a YAML block scalar), because (a) it matches
   the originating idea's wording ("included in the release artifact's YAML
   frontmatter"), (b) the existing DB read path (`Store.Get`) can serve it from a
   cache column with no new file-read on the detail endpoint, and (c) it avoids
   the current `DiskSync.Write` behaviour that drops `File.Body` on API edits.
   *Alternative:* store `description` as the markdown **body** (its natural home
   for long-form markdown), which would additionally require fixing the body-drop
   in `Write` (DR-3) and deciding how the detail endpoint sources body text. Needs
   a product-owner/architect decision before implementation.

2. **Should `goal` be filterable/sortable in the artifact/release list, or
   display-only?** This requirement scopes it as display-only; promoting it to a
   sortable/filterable column would extend `buildOrderBy` and the list query.

3. **Soft cap for `goal` length** — is 120 characters (matching the release
   `name` bound) the right UI maxlength, or should it be shorter to guarantee
   single-line rendering across the list/kanban chips?
