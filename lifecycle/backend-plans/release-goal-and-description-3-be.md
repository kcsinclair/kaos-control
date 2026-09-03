---
title: Backend Plan — Release Goal and Description Fields
type: plan-backend
status: draft
lineage: release-goal-and-description
parent: lifecycle/requirements/release-goal-and-description-2.md
created: "2026-09-03T11:15:00Z"
---

# Backend Plan — Release Goal and Description Fields

Implements the server side of
[lifecycle/requirements/release-goal-and-description-2.md](../requirements/release-goal-and-description-2.md):
two **optional** release fields — `goal` (single-line intent) and `description`
(multi-line markdown) — carried end-to-end through the release write path:
frontmatter → parser/marshaller → in-memory `Release` → `releases` cache → REST
API. Pairs with [[release-goal-and-description]] (frontend + test plans share
this lineage).

## Architecture conformance

Assessed against
[architecture-summary.md](../architecture/architecture-summary.md) and confirmed
**non-breaking** (requirement §Architecture-Breaking Requirements):

- **Single pure-Go binary** — two `TEXT` columns and two YAML frontmatter keys;
  no new dependency, datastore, or cgo. Consistent with
  [[adr-0003-pure-go-sqlite-index]] and [[adr-0004-embedded-spa-single-binary]].
- **Disk is authoritative; the index is a cache** — the markdown file is written
  first, the cache row follows. Per [[index-is-a-cache]] and
  [[adr-0003-pure-go-sqlite-index]], the new columns are delivered as a
  **schema-version bump that forces a rebuild-from-disk**, never an in-place data
  migration (Milestone 3). No new ADR required.

The recommended design (both fields in **frontmatter**, per the requirement's
Resolved Question 1) sidesteps the pre-existing `DiskSync.Write` body-drop gap
entirely: `description` never touches `File.Body`.

---

## Milestone 1 — File parser & marshaller round-trip (DR-1, DR-2)

**Description.** Teach the on-disk file model to read and write `goal` and
`description` as optional frontmatter keys, with lossless round-tripping.

**Files to change**
- `internal/release/file.go`
  - Add `Goal string` and `Description string` to `File`.
  - Add `goal` / `description` to the anonymous `fm` struct in `Parse`; assign
    into `File` with `strings.TrimSpace`-based empty handling so absent, empty,
    or whitespace-only values become `""` and never raise a validation error.
  - Add `Goal string \`yaml:"goal,omitempty"\`` and
    `Description string \`yaml:"description,omitempty"\`` to `releaseFM` (declared
    after `status`, before `start_date`, to keep deterministic key ordering) and
    populate them in `Marshal`. `omitempty` guarantees unchanged files stay
    byte-stable — the key is emitted only when the field is non-empty.
  - Confirm the YAML marshaller emits a block scalar for multi-line
    `description`; `gopkg.in/yaml.v3` does this automatically for strings
    containing newlines. No manual scalar-style handling needed.

**Acceptance criteria**
- A release file with `goal:` and a multi-line `description:` block scalar parses
  into populated `File.Goal` / `File.Description`; re-`Marshal`-ing round-trips
  both without loss (newlines preserved).
- A file with **neither** key parses to empty `Goal`/`Description` and succeeds
  (no error, no validation failure).
- `Marshal` of a `File` with both fields empty emits **no** `goal`/`description`
  keys — an existing file re-marshalled is byte-identical to before this change
  (aside from any pre-existing normalisation).
- `goal` supplied as multi-line is accepted verbatim (no newline validation).

---

## Milestone 2 — Carry fields through the in-memory model & write path (DR-3)

**Description.** Add the fields to the `Release` struct and ensure
`DiskSync.Write` populates them on the `File` it marshals, so an API edit never
silently drops either field.

**Files to change**
- `internal/release/release.go`
  - Add `Goal string \`json:"goal"\`` and
    `Description string \`json:"description"\`` to `Release`. Use plain (non-omit)
    JSON tags so `GET` responses always include the keys as `""` when unset
    (DR-5 requires empty-string, not absent).
  - No change to `Validate()` — both fields are free-form and optional (no length
    gate server-side; `goal` is soft-capped in the UI only, per DR-10).
- `internal/release/disksync.go`
  - In `Write`, set `Goal: r.Goal` and `Description: r.Description` on the `File`
    literal it constructs before `Marshal`.

**Acceptance criteria**
- `DiskSync.Write` of a `Release` with `Goal`/`Description` set produces a file
  that round-trips those values (verified against Milestone 1 parse).
- `Rename` (which delegates to `Write`) preserves both fields across a slug
  change.
- A `Release` marshalled to JSON always contains `"goal"` and `"description"`
  keys (empty string when unset).

---

## Milestone 3 — Cache columns, schema bump & rehydrate (DR-4, DR-8, DR-9)

**Description.** Add `goal` and `description` columns to the `releases` cache and
populate them on every read/write path. Deliver as a **schema-version bump** that
triggers a full rebuild-from-disk — not an `ALTER`/migration — honouring
[[index-is-a-cache]].

**Files to change**
- `internal/index/index.go`
  - In the `releases` DDL (~line 2130), add
    `goal        TEXT NOT NULL DEFAULT ''` and
    `description TEXT NOT NULL DEFAULT ''`.
  - Bump `const schemaVersion` (currently `7` → `8`). This is the mechanism that
    forces `dropAndRecreate` on next open, rebuilding the cache from disk. **Do
    not** add an `ALTER TABLE` — the bump + rebuild is the sanctioned path
    ([[adr-0003-pure-go-sqlite-index]]).
- `internal/release/store.go`
  - Add `goal, description` to every release `SELECT`: `List`, `Get`,
    `GetBySlug`, `GetByName` (and their scan helpers `scanReleases`,
    `scanReleaseWithCounts`, and the inline scans in `GetBySlug`/`GetByName`).
  - Add the two columns + bindings to the `UpsertBySlug` INSERT and its
    `ON CONFLICT ... DO UPDATE SET` clause.
  - Add `goal = ?, description = ?` to the in-place `UPDATE` in `Store.Update`
    (bind `r.Goal`, `r.Description`).
- `internal/release/rehydrate.go`
  - Populate `Goal: f.Goal` and `Description: f.Description` on the `Release`
    built from each parsed `File` before `UpsertBySlug`.

**Acceptance criteria**
- Opening an index whose stored `schema_version` is `< 8` drops and rebuilds the
  cache from disk; the new columns exist and are populated (verified via
  `TestSchemaUpgrade`-style path).
- `List`/`Get`/`GetBySlug`/`GetByName` return populated `Goal`/`Description`.
- `Rehydrate` of a `lifecycle/releases/` dir reproduces `goal`/`description` from
  disk into the cache (DR-9).
- A pre-existing release directory with no `goal`/`description` rehydrates with
  empty fields, **no error, no file rewrite** (DR-8).
- No `ALTER TABLE releases` appears anywhere; the diff is DDL + `schemaVersion`
  bump only.

---

## Milestone 4 — REST API accept & return (DR-5)

**Description.** Accept and return `goal`/`description` on the four release
endpoints, with PUT preserve-on-omit / clear-on-empty semantics.

**Files to change**
- `internal/http/releases.go`
  - `createReleaseRequest`: add `Goal *string \`json:"goal"\`` and
    `Description *string \`json:"description"\``. In `handleCreateRelease`, set
    `rel.Goal`/`rel.Description` from the pointers when non-nil (nil → stays
    `""`).
  - `updateReleaseRequest`: add `Goal *string` and `Description *string`. In
    `handleUpdateRelease`, implement **merge semantics** against the already
    fetched `current` release:
    - `req.Goal == nil` → `rel.Goal = current.Goal` (preserve).
    - `req.Goal != nil` → `rel.Goal = *req.Goal` (set, including `""` to clear).
    - Same for `Description`.
    This requires `current` (from `store.Get`) to carry the stored values — met
    by Milestone 3's `scanReleaseWithCounts` change.
  - No changes to response marshalling: `GET`/`POST`/`PUT` already write the
    `*Release` through `writeJSON`, so the new non-omit JSON tags (Milestone 2)
    surface `goal`/`description` automatically.

**Acceptance criteria**
- `POST /releases` with `goal`/`description` persists them (file first, then
  cache) and echoes them in the 201 body.
- `GET /releases` and `GET /releases/{slug}` include `"goal"` and `"description"`
  (empty string when unset).
- `PUT /releases/{slug}` **omitting** `goal` leaves the stored value unchanged;
  sending `"goal": ""` clears it. Same for `description`.
- Writes persist to the markdown file **before** the cache row is refreshed
  (read-after-write consistency; disk authoritative) — no new DB-origin write is
  introduced, keeping alignment with [[release-artefacts]] DR-1.

---

## Out of scope (per requirement Non-goals)

- No `goal` sorting/filtering — `buildOrderBy` and the list query are untouched
  (Resolved Question 2: display-only).
- No full-text search / indexing of `description` beyond cache read-back.
- No backfill onto historical releases; no `status`/date/slug/lineage changes.
