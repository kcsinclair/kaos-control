---
title: "Test Plan — RICE Scoring for Ideas and Defects"
type: plan-test
status: in-development
lineage: rice-scoring
parent: lifecycle/requirements/rice-scoring-2.md
---

# Test Plan — RICE Scoring for Ideas and Defects

## Overview

Verifies the RICE feature end-to-end across the [[rice-scoring]] backend and
frontend plans, with particular emphasis on the requirement's cross-cutting
guarantees: **single source of truth** for the formula (requirement §22),
**persistence fidelity** on write (requirement §23), **no migration**
(requirement §21), and the **`N/A`-grouped sort** (requirement §12). Backend
Go tests live under `internal/` (owned by [[rice-scoring]] be Milestone 5);
integration tests exercising HTTP + index + disk live under `tests/`; frontend
component tests live under `web/src/**/__tests__/`. This artifact describes what
that test code covers.

Test env note (see project memory): the integration `testEnv` auto-logins as
admin; devops-style URL helpers return full URLs; the run-log endpoint returns
NDJSON — reuse those conventions for any new HTTP integration tests.

---

## Milestone 1 — Derivation & validation truth table (backend unit)

### Description

Exhaustively cover `artifact.RiceScore` / `ValidateRice` — the single formula
source.

### Files to change

- `internal/artifact/rice_test.go`

### Acceptance criteria

- [ ] All-four-valid returns the exact unrounded formula value; `RoundRice` = 2 dp.
- [ ] Each missing component (and all combinations of ≥1 missing) → `N/A`.
- [ ] `rice_effort ≤ 0` → `N/A`, never `Inf`/`NaN`/panic (requirement §8).
- [ ] `confidence` boundaries: `0` and `100` valid; `-0.1` and `100.1` invalid.
- [ ] `reach = 0` / `impact = 0` (others valid) → score `0`, **not** `N/A`.
- [ ] Each validator error carries the offending field name (requirement §19).

---

## Milestone 2 — Frontmatter round-trip & backward compatibility (backend)

### Description

Verify parse → `buildMarkdown` fidelity and no-migration behaviour.

### Files to change

- `internal/artifact/artifact_test.go`
- `tests/` — a fidelity integration test writing via the API.

### Acceptance criteria

- [ ] A pre-existing `idea`/`defect` with **no** RICE fields parses, indexes, and
      exposes `rice_score` absent/null; the file is untouched (requirement §21).
- [ ] Unset components decode to `nil`, not `0`; a present `0` decodes to
      non-nil (requirement §4).
- [ ] After `PATCH .../rice`, the on-disk file is byte-for-byte identical to the
      original except for the RICE lines — all other frontmatter fields, ordering,
      and the body unchanged (requirement §23).
- [ ] Clearing a component (`null`) removes the YAML line entirely — no `0`/`""`
      residue (requirement §20).

---

## Milestone 3 — Index column & grouped sort (backend integration)

### Description

Verify the derived score is indexed and that sort groups `N/A` correctly.

### Files to change

- `internal/index/index_test.go` (or `internal/index/rice_test.go`)
- `tests/` — list-sort integration test over the HTTP API.

### Acceptance criteria

- [ ] After a `schemaVersion` rebuild + scan, `rice_score` is back-filled for every
      scored `idea`/`defect` and null elsewhere (requirement §24).
- [ ] `sort=rice:asc` and `sort=rice:desc` return scored rows in numeric order with
      **all null rows grouped after** them in **both** directions (requirement §12).
- [ ] Sorting a large list touches only indexed values — no artifact body reparse
      (assert via query behaviour / timing budget) (requirement §25).

---

## Milestone 4 — `PATCH .../rice` endpoint (backend integration)

### Description

Exercise the write path: happy path, clear, validation rejection, locking, auth,
and the broadcast.

### Files to change

- `tests/` — new `rice_patch` integration test.

### Acceptance criteria

- [ ] Valid PATCH persists, returns `200` with re-indexed row incl. `rice_score`,
      and broadcasts `artifact.indexed` `action:"updated"` (requirement §15).
- [ ] Invalid input (non-numeric, negative reach/impact, confidence outside 0–100,
      effort ≤ 0) → `400`/`422` with a field-level message and the file unchanged
      (requirement §19).
- [ ] Foreign lineage lock → `423`; unauthenticated → `401`; unauthorised role →
      `403`.
- [ ] Identical bodies to `PATCH .../rice` twice (idempotent) leave the same file.

---

## Milestone 5 — Cross-tier parity: list == detail == API (integration)

### Description

The load-bearing single-source-of-truth check (requirement §18, §22): identical
inputs must yield identical stored results regardless of entry point, and the Go
and TS formulas must agree.

### Files to change

- `tests/` — parity integration test.
- `web/src/lib/__tests__/rice.spec.ts` — TS-vs-fixtures parity.

### Acceptance criteria

- [ ] A shared fixture set of component tuples produces identical scores from the Go
      `RiceScore` and the TS `riceScore` (fixtures committed and consumed by both;
      any divergence fails) (requirement §22).
- [ ] Saving the same components via the list editor and via the detail editor
      yields byte-identical files and identical `rice_score` (requirement §18).

---

## Milestone 6 — Frontend component behaviour (Vitest)

### Description

Cover the editor and column UI defined in [[rice-scoring]] (fe).

### Files to change

- `web/src/components/artifact/__tests__/RiceEditor.spec.ts`
- `web/src/views/project/__tests__/ArtifactListView.spec.ts` (extend)

### Acceptance criteria

- [ ] Live preview recomputes on each component change before save, matching
      `formatRice(riceScore(...))` (requirement §14, §17).
- [ ] Invalid input shows a field-level message and blocks Save; nothing is sent
      (requirement §19).
- [ ] Clearing a field sends `null` and the preview reverts to `N/A` when fewer
      than four valid components remain (requirement §20).
- [ ] Opening the editor on an item with no RICE fields seeds inputs with the
      defaults (Reach 100 / Impact 0.25 / Confidence 25 / Effort 1) as pre-fill,
      while an unscored row still renders `N/A` in the column (requirement §21).
- [ ] The RICE column renders scores/`N/A` and sorts with `N/A` grouped last in
      both directions (requirement §11, §12).
- [ ] A failed save reverts the optimistic value and surfaces an error (parity with
      `PriorityDropdown`).

---

## Milestone 7 — Full-suite gate

### Acceptance criteria

- [ ] `make test-unit`, the `tests/` integration suite, and `pnpm test` (web) all
      pass with the new coverage.
- [ ] `make lint` passes.
