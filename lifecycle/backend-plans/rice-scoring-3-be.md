---
title: "Backend Plan — RICE Scoring for Ideas and Defects"
type: plan-backend
status: draft
lineage: rice-scoring
parent: lifecycle/requirements/rice-scoring-2.md
---

# Backend Plan — RICE Scoring for Ideas and Defects

## Overview

Adds the four optional RICE components to `idea`/`defect` frontmatter, a **single
canonical Go derivation** for the score, a dedicated `PATCH .../rice` write path
(mirroring the existing `handlePatchPriority` at `internal/http/write.go:466`),
and an **indexed, nullable `rice_score` column** so list/sort queries never
reparse artifact bodies. The score is *derived, never stored in frontmatter*
(requirement §6): it is written only into the SQLite cache column and returned in
API responses.

Design decisions that gate the milestones below:

- **Pointer fields, not plain floats.** `rice_reach = 0` and `rice_impact = 0`
  are *valid* values, so a plain `float64` with `omitempty` would silently drop a
  legitimate zero. RICE fields are therefore `*float64` — `nil` (omitted) is the
  unset state (requirement §4, §20), any present value (including `0`) is set.
- **`buildMarkdown` re-marshals the whole `Frontmatter` struct** (`write.go:691`),
  so adding the fields to the struct is what makes them round-trip; setting a
  pointer back to `nil` is what removes the field on clear (requirement §20, §23).
- **Single source of truth.** `artifact.RiceScore` is the one formula used by the
  indexer (column value) and by every API response (requirement §22). The
  frontend live-preview mirror in [[rice-scoring]] (fe plan) is verified against
  it by the [[rice-scoring]] test plan.

Relates to [[artefact-priority-inline-edit]] and [[artefact-inline-status-change]]
(the PATCH-single-field + re-index + broadcast pattern reused here) and
[[sortable-table-columns]] (the sortable-column behaviour the fe plan builds on).

---

## Milestone 1 — RICE frontmatter fields on the artifact model

### Description

Add the four optional components to the `Frontmatter` struct as nullable pointers
so unset is distinct from zero, and so `yaml.Marshal` (via `buildMarkdown`) omits
them when `nil`.

### Files to change

- `internal/artifact/artifact.go` — add to `Frontmatter`:
  ```go
  RiceReach      *float64 `yaml:"rice_reach,omitempty"      json:"rice_reach,omitempty"`
  RiceImpact     *float64 `yaml:"rice_impact,omitempty"     json:"rice_impact,omitempty"`
  RiceConfidence *float64 `yaml:"rice_confidence,omitempty" json:"rice_confidence,omitempty"`
  RiceEffort     *float64 `yaml:"rice_effort,omitempty"     json:"rice_effort,omitempty"`
  ```

### Acceptance criteria

- [ ] An `idea`/`defect` file with any subset of the four fields parses without
      `ParseErrs`; absent fields decode to `nil`, not `0` (requirement §3, §4).
- [ ] A field present with value `0` decodes to a non-nil pointer to `0`.
- [ ] Round-trip through `buildMarkdown`: a `nil` field is absent from the emitted
      YAML; a set field is emitted with its numeric value.
- [ ] Files with no RICE fields still parse and index exactly as before
      (requirement §21, backward compatibility).

---

## Milestone 2 — Canonical score derivation + validation (single source of truth)

### Description

Add `internal/artifact/rice.go` with the one formula and the one validator used
everywhere on the backend.

- `func RiceScore(fm Frontmatter) (score float64, ok bool)` —
  `ok` is true only when **all four** pointers are non-nil and each satisfies its
  constraint; then `score = (reach × impact × (confidence/100)) / effort`
  (requirement §5, §7). `ok` is false — i.e. `N/A` — when any component is unset,
  or `reach < 0`, `impact < 0`, `confidence` outside `0–100`, or `effort ≤ 0`
  (requirement §7, §8). No `Inf`/`NaN` ever returned.
- `func ValidateRiceComponent(field string, v *float64) error` and a
  `ValidateRice(fm Frontmatter) error` covering the same constraints, returning a
  field-level message (used by the PATCH handler in Milestone 4).
- `func RoundRice(score float64) float64` — 2-dp rounding used only for display
  callers; sorting/storage uses the unrounded value (requirement §9).

### Files to change

- `internal/artifact/rice.go` — new file (`RiceScore`, `ValidateRice`,
  `ValidateRiceComponent`, `RoundRice`).

### Acceptance criteria

- [ ] All-four-valid → `(score, true)` equal to the formula (unrounded).
- [ ] Any missing component → `(_, false)`.
- [ ] `rice_effort` present and `≤ 0` → `(_, false)`, no panic, no `Inf`
      (requirement §8).
- [ ] `rice_confidence` of `-1` or `101` → invalid; `0` and `100` → valid.
- [ ] `reach = 0` or `impact = 0` with the other three valid → `(0, true)` (a real
      zero score, distinct from `N/A`).
- [ ] `RoundRice` yields 2-dp values; `RiceScore` itself is unrounded.

---

## Milestone 3 — Index the derived score (nullable column + grouped sort)

### Description

Persist the derived score in the artifact cache so list/sort is O(n log n) over
indexed values without reparsing bodies (requirement §24, §25), and expose it on
`ArtifactRow`.

- Add `rice_score REAL` (nullable) to the `artifacts` DDL and
  `CREATE INDEX idx_artifacts_rice_score`.
- **Bump `schemaVersion` 6 → 7** so the startup `dropAndRecreate` back-fills the
  column for every existing artifact (the mtime/SHA guards would otherwise skip
  unchanged files).
- In `Upsert`, call `artifact.RiceScore(a.FM)`; bind the column to the score when
  `ok`, else SQL `NULL` (both to `N/A` and non-idea/defect types → `NULL`).
- Add `RiceScore *float64 \`json:"rice_score,omitempty"\`` to `ArtifactRow`; scan
  it in `scanRows` / the `Get` and `List` SELECTs.
- Add `"rice"` to the `buildOrderBy` allowlist mapping to an expression that pins
  `NULL` last in **both** directions:
  `ORDER BY (rice_score IS NULL), rice_score <DIR>` (requirement §12).

### Files to change

- `internal/index/index.go` — DDL (`CREATE TABLE artifacts`, new index),
  `schemaVersion`, `Upsert` INSERT column + value, `ArtifactRow`, the `List`/`Get`
  SELECT column list, `scanRows`, and `buildOrderBy`.

### Acceptance criteria

- [ ] After a schema-version bump + startup scan, every `idea`/`defect` with four
      valid components has a non-null `rice_score` equal to `RiceScore`; all other
      rows are `NULL`.
- [ ] `GET .../artifacts` and `GET .../artifacts/*` include `rice_score` when set
      and omit it when null (`omitempty`).
- [ ] `sort=rice:asc` and `sort=rice:desc` both return all scored rows ordered
      numerically with **all `NULL` rows grouped together after** them
      (requirement §12).
- [ ] Re-index on write (Milestone 4) and on external file change recomputes the
      column; no artifact body is reparsed to sort (requirement §24, §25).

---

## Milestone 4 — `PATCH .../rice` write path

### Description

Add `handlePatchRice`, modelled on `handlePatchPriority` (`write.go:466`):
resolve path via `sandbox.Resolve`, read + `artifact.Parse`, enforce the lineage
lock (423 on foreign holder), **validate with `artifact.ValidateRice`**, apply the
four pointers (a JSON `null` for a field clears it → pointer `nil`), re-serialise
with `buildMarkdown`, write, `IndexFile`, and broadcast `artifact.indexed`.

Request body (each key optional; `null` = clear, absent = leave unchanged):
```json
{ "rice_reach": 100, "rice_impact": 0.25, "rice_confidence": 25, "rice_effort": 1 }
```

Wire the route in the greedy-wildcard dispatch in `internal/http/server.go`
(the `r.Patch("/artifacts/*", …)` block near line 248) with a
`strings.HasSuffix(param, "/rice")` branch alongside `/priority` and `/release`.
Gate with a `RolesRiceEditors` set (product-owner, analyst — reuse
`RolesPriorityEditors` if the membership matches).

### Files to change

- `internal/http/write.go` — new `handlePatchRice`; `buildMarkdown` reused
  unchanged.
- `internal/http/server.go` — `/rice` suffix branch in the PATCH wildcard
  dispatch; `RolesRiceEditors` if a new set is needed.

### Acceptance criteria

- [ ] `PATCH .../rice` with four valid values persists them to frontmatter,
      re-indexes, returns `200` with the re-indexed row (including `rice_score`),
      and broadcasts `artifact.indexed` `action:"updated"`.
- [ ] A field sent as `null` is **removed** from frontmatter (not written as `0`
      or `""`); if fewer than four valid components remain, `rice_score` becomes
      null (requirement §20).
- [ ] Invalid input (non-numeric, `reach`/`impact` `< 0`, `confidence` outside
      `0–100`, `effort ≤ 0`) returns `400`/`422` with a field-level message and
      **does not write the file** (requirement §19).
- [ ] All non-RICE frontmatter fields and the body are unchanged except the RICE
      fields (requirement §23) — at parity with the existing priority/release
      PATCH round-trip.
- [ ] A locked lineage (foreign holder) returns `423`; unauthenticated → `401`;
      unauthorised role → `403`.
- [ ] Identical request bodies to this endpoint and to the detail-view save path
      in [[rice-scoring]] (fe) yield byte-identical files (requirement §18, §22).

---

## Milestone 5 — Regression + parity guards

### Description

Backend unit/integration coverage owned here (the end-to-end and cross-tier parity
assertions live in the [[rice-scoring]] test plan).

### Files to change

- `internal/artifact/rice_test.go`, `internal/artifact/artifact_test.go`
- `internal/index/index_test.go` (or a new `rice_test.go` in `index`)

### Acceptance criteria

- [ ] Table-driven `RiceScore`/`ValidateRice` tests cover every branch in
      Milestone 2's acceptance criteria.
- [ ] Index test: sort grouping (`NULL` last both directions) and column
      back-fill after a `schemaVersion` rebuild.
- [ ] Backward-compat test: an artifact with no RICE fields loads, indexes, and
      returns `rice_score` absent/null (requirement §21).
- [ ] `make lint` and `make test-unit` pass.
