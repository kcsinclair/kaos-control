---
title: "Test Plan — Architectural Artefacts On-Disk Model"
type: plan-test
status: in-development
lineage: architectural-artefacts
parent: lifecycle/requirements/architectural-artefacts-2.md
labels:
    - test
    - architecture
    - artefacts
release: KC-Release5
---

# Test Plan — Architectural Artefacts On-Disk Model

Verifies the acceptance criteria of [[architectural-artefacts-2]] against the implementations in
[[architectural-artefacts-3-be]] and [[architectural-artefacts-4-fe]]: type registration, the
two coexisting zones, the promotion primitive (idempotency + archive-on-replace), ADR numbering
and ADR-0001 authoring, clean-slug/lineage-relaxed indexing, agent-directive presence, and
re-indexing across all three index paths.

Integration tests live in `tests/` per the repository convention; package-internal unit tests are
owned by the backend/frontend plans and are referenced here only where they satisfy an acceptance
criterion. Recall (project memory): `testEnv` auto-logins as admin; devops-style URL helpers
return full URLs for `http.Get`; write endpoints re-index synchronously before responding.

Cross-references:
- [[architectural-artefacts-3-be]] — endpoints and primitives under test.
- [[architectural-artefacts-4-fe]] — UI behaviours under test.
- [[architecture-templates]] — supplies catalog fixtures (architectures/tech-stacks) the promotion tests copy from.

---

## Milestone 1 — Type registration & vocabulary (FR-18)

### Description

Prove `architecture`, `tech-stack`, and `adr` validate end-to-end: parser, index, and API.

### Files to change

- **New** `tests/architecture_types_test.go`:
  - Seed an artefact of each new type under `lifecycle/architecture/`; assert `GET
    /api/p/{project}/artifacts` returns them with the correct `type` and **no** parse/validation
    error flag.
  - Assert an unknown type (`type: bogus`) still surfaces a validation error (guard that the
    vocabulary was extended, not disabled).

### Acceptance criteria

- Each of `architecture`, `tech-stack`, `adr` indexes cleanly and is retrievable via the API.
- Frontend vitest (owned by [[architectural-artefacts-4-fe]] M1) confirms the three types appear
  in the type filter and have defined graph colours — referenced, not duplicated here.

---

## Milestone 2 — Two coexisting zones (FR-1, FR-2, FR-3)

### Description

Verify catalog and project-own zones coexist and that catalog entries are never mutated by
project-own activity.

### Files to change

- **New** `tests/architecture_zones_test.go`:
  - Fixture: `lifecycle/architecture/README.md`, `architectures/*.md`, `tech-stacks/*.md`
    (catalog) plus promoted root copies, `architecture-summary.md`, `decisions/`, `standards/`.
  - Assert all are indexed; assert catalog `README.md` is excluded from indexing per the existing
    ignore-readme rule; assert the catalog source bytes are unchanged after a promotion runs.

### Acceptance criteria

- Catalog and project-own artefacts index simultaneously.
- Catalog source files are byte-identical before and after promotion (FR-3, FR-6).

---

## Milestone 3 — Promotion: idempotency & archive-on-replace (FR-4–FR-7, OQ-3, NFR-3)

### Description

Exercise `POST …/architecture/promote` for first promotion, idempotent re-promotion of the same
choice, and replacement with a different choice (prior copies archived, not deleted).

### Files to change

- **New** `tests/architecture_promote_test.go`:
  - **First promotion:** POST with a catalog architecture + tech-stack; assert two root copies
    exist, each with `parent:` pointing at its catalog source (FR-5), and both appear via the
    artifacts API (proves API-write re-index, NFR-2).
  - **Idempotent same-choice:** POST the identical selection again; assert exactly one root copy
    per kind (no `-2`/duplicate) and `archive/` was **not** created (FR-7, NFR-3).
  - **Changed-choice replacement:** POST a *different* architecture; assert the prior root copy
    now exists under `lifecycle/architecture/archive/`, the new copy is at the root, catalog
    untouched, and two archived generations of the same basename coexist without overwrite
    (`archive/foo.md` + `archive/foo-1.md`).
  - **Traversal:** POST with `architecture_path: "../../etc/passwd"` → 400.
  - **Role gate:** POST as a read-only user → 403.

### Acceptance criteria

- All promotion sub-cases pass; no orphaned or duplicate files on any re-run (NFR-3).
- Archived files are never overwritten; git history retains superseded content (verified by
  presence under `archive/`, not deletion).

---

## Milestone 4 — ADR numbering & ADR-0001 authoring (FR-11, FR-12, FR-14, OQ-4)

### Description

Verify monotonic zero-padded numbering, never-reused numbers, ADR-0001 idempotent authoring, and
that human/agent-created ADRs default to `status: draft`.

### Files to change

- **New** `tests/architecture_adr_test.go`:
  - `GET …/architecture/adrs/next` on an empty `decisions/` → `{"number":1}`.
  - `POST …/architecture/adrs` (title/slug/body) → creates `adr-0001-<slug>.md`, `type: adr`,
    `status: draft` by default; response includes `number: 1` and the path.
  - Create a second ADR → `adr-0002-*.md`; `next` now returns 3.
  - Add a `status: superseded` `adr-0003` fixture on disk → `next` returns 4 (superseded numbers
    are counted, never reused, FR-14).
  - **ADR-0001 authoring idempotency:** invoke the wizard's ADR-0001 authoring path (the backend
    `WriteADR0001` primitive, exercised via the promotion/wizard hook or a direct primitive test)
    twice with identical inputs → exactly one `adr-0001-*.md`, titled "Adopt <architecture> with
    <tech-stack>", body containing the Q&A trail and ranked rejected alternatives; no `adr-0002`
    spawned (FR-12, NFR-3).
  - Filename format guard: created ADRs match `^adr-\d{4}-.+\.md$`.

### Acceptance criteria

- Numbering is monotonic, zero-padded 4-digit, never reused across superseded/rejected files.
- ADR-0001 authoring is idempotent and carries the required title/Q&A/rejected-alternatives.
- New ADRs default to `status: draft` (OQ-4); ADR-0001 (a recorded decision) is `approved`.

---

## Milestone 5 — Clean-slug indexing & lineage relaxation (FR-19, FR-20, NFR-2)

### Description

Prove clean-slug (no `-N`) files under `lifecycle/architecture/` index without lineage-validation
errors and that a promoted copy whose `parent:` is a catalog entry is accepted — and that the
relaxation does **not** leak to other lifecycle directories.

### Files to change

- **New** `tests/architecture_lineage_test.go`:
  - Index a `lifecycle/architecture/postgres-modular-monolith.md` (no `lineage:`, no `-N`); assert
    it appears via the API with **no** validation/parse error and `Index == 0`.
  - Index a promoted copy with `parent: lifecycle/architecture/architectures/…`; assert accepted
    and that a `parent` graph edge resolves to the catalog target.
  - **Regression:** a file under `lifecycle/requirements/` with no `lineage:` still reports the
    missing-lineage validation error (relaxation is path-scoped).
- Also assert the three index paths pick up architecture artefacts (NFR-2):
  - **Startup scan:** files present before boot appear after the initial scan.
  - **Live watch:** a file written to `lifecycle/architecture/` after boot triggers
    `artifact.indexed` / `file.changed` and becomes retrievable.
  - **API write:** covered by the promotion/ADR endpoint tests (Milestones 3–4).

### Acceptance criteria

- Clean-slug architecture files index without lineage errors; catalog-parent `parent:` accepted.
- Non-architecture files retain the lineage requirement (regression guard).
- All three index paths surface architecture artefacts with no special-casing beyond FR-18/FR-19.

---

## Milestone 6 — Agent directives present (FR-21, FR-22)

### Description

Assert the config the server loads directs each design/build agent to read
`lifecycle/architecture/` first and to propose an ADR on deviation, and that agents permitted to
author proposed ADRs have the `decisions/` write scope.

### Files to change

- **New** `tests/architecture_directives_test.go` (or extend an existing config test):
  - Load `lifecycle/config.yaml` via the same config loader the server uses; for each of
    `requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`,
    `test-developer`, `qa`, assert the prompt template contains a "read `lifecycle/architecture/`"
    directive and a "propose an ADR … `lifecycle/architecture/decisions/`" directive.
  - For agents that may author a proposed ADR, assert `lifecycle/architecture/decisions` is in
    `allowed_write_paths`.

### Acceptance criteria

- Every design/build agent template carries both directives; the ADR write-path is present where
  FR-13 requires it. Concrete prose is validated only for the directive's presence — its wording
  is owned by [[agent-directives-generation]].

---

## Test data / fixtures

- Catalog fixtures (one architecture + one tech-stack, and a second architecture for the
  changed-choice case) come from [[architecture-templates]]; if unavailable at test time, the
  tests seed minimal valid catalog artefacts in a temp project root.
- A read-only user fixture for the role-gate assertions (reuse the existing auth test helpers).

## Verification (end-to-end)

1. `make test-unit` clean (parser/promotion/ADR unit tests from the backend plan).
2. `make test-integration` clean (all `tests/architecture_*_test.go`).
3. `pnpm test` clean (frontend vitest from [[architectural-artefacts-4-fe]]).
4. Full acceptance sweep: every checkbox in [[architectural-artefacts-2]] §Acceptance Criteria maps
   to at least one milestone above and passes.
