---
title: "Backend Plan — Architectural Artefacts On-Disk Model"
type: plan-backend
status: done
lineage: architectural-artefacts
parent: lifecycle/requirements/architectural-artefacts-2.md
labels:
    - backend
    - architecture
    - artefacts
release: KC-Release5
---

# Backend Plan — Architectural Artefacts On-Disk Model

This plan implements the server-side model for [[architectural-artefacts-2]]: the on-disk
zones under `lifecycle/architecture/`, the **promotion** primitive that copies chosen catalog
items to the directory root, the **ADR** numbering/authoring service, registration of the new
artefact types, and the relaxation of lineage/index validation for everything under
`lifecycle/architecture/`. It also completes the agent prompt-template directives.

**Scope boundary.** This lineage owns the *artefact model on disk* and the *reusable promotion
and ADR primitives* (callable Go services + HTTP endpoints). It does **not** own the wizard UX
that triggers promotion — that is [[onboarding-architecture-selection]], which calls the
primitives defined here. Catalog seed content (candidate architectures, stacks, standards
seeds) is [[architecture-templates]]. UI rendering is [[architecture-overview-view]] /
[[architecture-relationship-map]]. Concrete directive prose is [[agent-directives-generation]];
this plan only ensures the directive slot exists in every agent template.

Cross-references:
- [[architectural-artefacts-4-fe]] — Frontend plan (type vocab, ADR-create affordance, clean-slug rendering).
- [[architectural-artefacts-5-test]] — Test plan.
- [[onboarding-architecture-selection]] — wizard; consumer of the promotion/ADR-0001 primitives.
- [[architecture-templates]] — catalog seed; de-duplicated standards seed set (FR-16, OQ-5).

---

## Milestone 1 — Register new types and relax lineage validation for `lifecycle/architecture/`

### Description

Register `adr` as a `KnownType` (`architecture` and `tech-stack` are already present in
[internal/artifact/artifact.go](internal/artifact/artifact.go)) and relax the required-`lineage`
and lineage-index expectations for files whose repo-relative path is under
`lifecycle/architecture/`, so clean-slug files (no `-N` index) index without validation errors
and a promoted copy whose `parent:` is a catalog entry is accepted (FR-18, FR-19, FR-20).

### Files to change

- **Edit** `internal/artifact/artifact.go`:
  - Add `"adr": true` to `KnownTypes` (keep the existing `architecture` / `tech-stack` entries;
    extend the doc comment to mention promoted choices, ADRs, and the summary).
  - Add a helper `func IsArchitecturePath(relPath string) bool` returning true when
    `filepath.ToSlash(relPath)` has prefix `lifecycle/architecture/`. Use `filepath.ToSlash`
    so Windows separators do not defeat the check.
  - In `Parse`, gate the `missing required field: lineage` ParseErr (currently
    artifact.go:176–179) on `!IsArchitecturePath(relPath)`. For architecture-zone files, when
    `FM.Lineage == ""` leave `Lineage` empty (do **not** backfill to slug) and record **no**
    ParseErr — these are standing reference artefacts (FR-19/FR-20). `title`, `type`, `status`
    remain required for all paths.
  - Confirm the existing filename parse (`indexSuffixRe`, artifact.go:308) already yields
    `Index == 0` for a clean slug like `postgres-modular-monolith.md`; no change needed — assert
    this in the unit test below rather than adding logic.
- **Edit** `internal/artifact/artifact.go` (or wherever parent-continuity is checked — grep for
  any "parent" lineage-step validation): audit for a check that a non-originating artefact's
  `parent:` must point to the previous lineage index. If one exists, exempt
  `IsArchitecturePath` files so a promoted copy pointing at a catalog entry
  (`architectures/foo.md`) is accepted (FR-20). Current grep shows `parent:` only drives edge
  creation (artifact.go:357) with no continuity assertion — if that holds, document "no change
  required" in the commit and cover it with the test.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/artifact/artifact_test.go`:
  - A file at `lifecycle/architecture/postgres-modular-monolith.md` with `type: architecture`,
    no `lineage:`, no `-N` index parses with **zero** ParseErrs, `Index == 0`, `Lineage == ""`.
  - A promoted copy with `parent: lifecycle/architecture/architectures/postgres-modular-monolith.md`
    parses with zero ParseErrs and emits a `parent` edge to that target.
  - `type: adr` validates (no "unknown type" ParseErr); `type: doc` still validates.
  - A file **outside** `lifecycle/architecture/` with no `lineage:` still records the
    `missing required field: lineage` ParseErr (regression guard — relaxation is path-scoped).

---

## Milestone 2 — `internal/architecture` package: promotion primitive

### Description

Create `internal/architecture/` owning the deterministic, idempotent promotion operation
(FR-4–FR-7, NFR-3). Promotion copies the chosen architecture and tech-stack catalog artefacts to
the `lifecycle/architecture/` root, stamps a `parent:` pointing back to the catalog source, and
on a *changed* selection moves the previously promoted root copies to an archive location
(OQ-3 resolved: **archive, not hard-delete**). This package owns no HTTP and no wizard logic.

### Files to change

- **New** `internal/architecture/promote.go`:
  - `type PromotionRequest struct { ArchitectureCatalogPath string; TechStackCatalogPath string }`
    — both are repo-relative paths under `architectures/` / `tech-stacks/`.
  - `type PromotionResult struct { PromotedArchitecture string; PromotedTechStack string; Archived []string }`
    — repo-relative destination paths and any archived prior copies.
  - `func Promote(projectRoot string, req PromotionRequest) (PromotionResult, error)`:
    - Resolves catalog sources through `sandbox.Resolve(filepath.Join(projectRoot,
      "lifecycle/architecture"), rel)`; errors on traversal or missing source.
    - Destination filenames are the **clean basename** of the source (e.g.
      `architectures/postgres-modular-monolith.md` → `postgres-modular-monolith.md`) placed at
      `lifecycle/architecture/<basename>`.
    - **Idempotent same-choice (FR-7):** if the destination already exists *and* its stamped
      `parent:` equals the requested source, overwrite in place — no archive, no duplicate.
    - **Changed-choice replacement (FR-7, OQ-3):** identify the *currently promoted* root copy
      of the same kind (by scanning root-level `type: architecture` / `type: tech-stack` files)
      whose source differs; move each to `lifecycle/architecture/archive/<basename>` before
      writing the new copy. Create `archive/` on demand. If an archive target already exists,
      suffix with the shortest numeric disambiguator (`-1`, `-2`) — never overwrite an archived
      file (history preservation).
    - Stamps `parent: <source repo-relative path>` into the destination frontmatter via
      `artifact.PatchFrontmatterField` (add the field if absent — extend the patcher or set it
      through a small `SetFrontmatterField` helper if `PatchFrontmatterField` only replaces
      existing keys; verify at artifact.go:195).
    - Writes atomically (temp file in the same dir + `os.Rename`), mirroring the docs-write
      pattern used elsewhere.
    - Returns the result; leaves catalog entries untouched (FR-3).
  - `func currentlyPromoted(projectRoot, kind string) ([]string, error)` — helper listing
    root-level (non-`archive/`, non-subdir) files of the given `type:`.

### Acceptance criteria

- `go build ./internal/architecture/... && go vet ./...` clean.
- Package unit test `internal/architecture/promote_test.go`:
  - Promote into an empty architecture dir → two root copies exist, each with `parent:` pointing
    at its catalog source; catalog sources byte-identical to before.
  - Re-promote the **same** selection → still exactly one root copy per kind (no `-2` duplicate),
    `archive/` not created.
  - Promote a **different** architecture → prior root copy now under `archive/`, new root copy in
    place, catalog untouched.
  - Traversal attempt (`../../etc/x`) as a source → error wrapping `sandbox.ErrPathTraversal`.
  - Two archived generations of the same basename coexist (`archive/foo.md`, `archive/foo-1.md`).

---

## Milestone 3 — ADR numbering + ADR-0001 authoring service

### Description

Add ADR support to `internal/architecture`: allocate monotonic zero-padded 4-digit numbers that
are never reused, author ADR files under `lifecycle/architecture/decisions/`, and provide the
ADR-0001 "Adopt <architecture> with <tech-stack>" authoring helper the wizard calls (FR-11,
FR-12, FR-14). Agent-proposed and human-created ADRs land as `status: draft` (OQ-4 resolved).

### Files to change

- **New** `internal/architecture/adr.go`:
  - `var adrFileRe = regexp.MustCompile(^adr-(\d{4})-.*\.md$)`.
  - `func NextADRNumber(projectRoot string) (int, error)` — scans
    `lifecycle/architecture/decisions/`, parses the numeric group of every matching file
    (including `status: superseded`/`rejected` ones — numbers are never reused, FR-14), returns
    `max+1`, or `1` when the dir is empty/absent.
  - `type ADRRequest struct { Slug string; Title string; Status string; Body string }` — `Slug`
    is a kebab-case slug for the filename; `Status` defaults to `draft` when empty.
  - `func CreateADR(projectRoot string, req ADRRequest) (path string, err error)`:
    - Allocates `n := NextADRNumber(...)`, builds filename
      `adr-<zero-padded-4>-<slug>.md`, writes frontmatter
      `title`, `type: adr`, `status: <req.Status|draft>`, plus `date` omitted (deterministic —
      no clock in the primitive; the caller/UI stamps a date if wanted) and the body.
    - Creates `decisions/` on demand; refuses to overwrite an existing file for the allocated
      number (allocate-then-write is not atomic across processes, so re-check existence and, on
      collision, re-allocate once before erroring — keeps NFR-3 idempotency honest).
  - `func WriteADR0001(projectRoot string, arch, stack string, qaTrail string, rejected []string) (string, error)`:
    - Deterministic wrapper: title `"Adopt <arch> with <stack>"`, slug derived from
      `adopt-<arch-slug>`, body containing the Q&A trail and a "Rejected alternatives" section
      listing `rejected` (ranked). **Idempotent (FR-12, NFR-3):** if an `adr-0001-*.md` already
      exists, overwrite that same file rather than allocating `0002`.
    - `status: accepted`? No — the wizard's own decision is a fait accompli; author ADR-0001 as
      `status: approved` (it records a decision already made). Agent-*proposed* later ADRs use
      `draft` (OQ-4). Document this distinction in the function comment.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Package unit test `internal/architecture/adr_test.go`:
  - Empty `decisions/` → `NextADRNumber == 1`; `CreateADR` writes `adr-0001-*.md`.
  - With `adr-0001`, `adr-0002` present → `NextADRNumber == 3`; deleting `adr-0002` leaves
    `NextADRNumber == 2`? No — assert numbers derive from files present, so after deletion next
    is 2 (documented behaviour: numbering is monotonic over *existing* files; superseded files
    stay on disk so numbers are not reused in practice).
  - A `status: superseded` `adr-0003` still counts → `NextADRNumber == 4`.
  - `WriteADR0001` twice with the same inputs → exactly one `adr-0001-*.md`, no `adr-0002`.
  - `CreateADR` default status is `draft` when `req.Status == ""`.

---

## Milestone 4 — HTTP endpoints for promotion and ADR creation

### Description

Expose the M2/M3 primitives over the existing chi router so the wizard
([[onboarding-architecture-selection]]) and the editor ([[architectural-artefacts-4-fe]]) can
drive them. Reuse the existing project-context and role-gate helpers. After each write, the
existing fsnotify watch + API-write re-index paths pick the files up (NFR-2) — trigger a
synchronous re-index of the written files before responding, mirroring `PUT /artifacts/*`.

### Files to change

- **New** `internal/http/architecture.go`:
  - `handlePromoteArchitecture(w, r)` — `POST /api/p/{project}/architecture/promote`:
    - Role gate: `requireRole(w, r, p, RolesArtifactEditors...)` (same gate used by artifact
      mutations; verify helper name).
    - Body: `{ "architecture_path": "architectures/...", "tech_stack_path": "tech-stacks/..." }`.
    - Calls `architecture.Promote(p.Entry.Path, req)`; on success synchronously re-indexes the
      promoted (and archived) paths, then returns `200` with the `PromotionResult` JSON.
    - 400 on traversal/missing source; 409 is not needed (promotion is idempotent).
  - `handleCreateADR(w, r)` — `POST /api/p/{project}/architecture/adrs`:
    - Role gate as above (humans create; agents propose via their own write-path — both allowed).
    - Body: `{ "slug": "...", "title": "...", "status": "draft", "body": "..." }`.
    - Calls `architecture.CreateADR`; re-indexes the new file; returns `201` with
      `{ "path": "lifecycle/architecture/decisions/adr-0004-...md", "number": 4 }`.
  - `handleNextADRNumber(w, r)` — `GET /api/p/{project}/architecture/adrs/next` → `200 {"number": 4}`
    so the frontend can preview the number before submitting.
- **Edit** `internal/http/server.go`: mount the three routes inside the `/p/{project}` group,
  after the artifacts routes.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Integration coverage in [[architectural-artefacts-5-test]]: promote happy-path, idempotent
  re-promote, changed-choice archive, create-ADR increments number, next-number preview, role
  gate returns 403 for a read-only user, traversal returns 400.
- After a promote call, `GET /api/p/{project}/artifacts` includes the two promoted root copies
  with their resolved `parent` edges — proving the API-write re-index path fired (NFR-2).

---

## Milestone 5 — Agent directive slots in config templates

### Description

Ensure every agent whose work is design/planning/implementation carries the FR-21/FR-22
directive to read `lifecycle/architecture/` first and to *propose* an ADR (status `draft`, OQ-4)
rather than deviate silently. [lifecycle/config.yaml](lifecycle/config.yaml) already carries this
directive for several agents (analyst templates at ~118/149/185/190; developer templates at
~249/298/346); this milestone audits all six agents for the slot and adds the ADR write-path
where an agent may author a proposed ADR. Concrete prose is owned by
[[agent-directives-generation]] — here we only guarantee the slot and the write scope exist.

### Files to change

- **Edit** `lifecycle/config.yaml`:
  - Confirm each of `requirements-analyst`, `planning-analyst`, `backend-developer`,
    `frontend-developer`, `test-developer`, `qa` has a "read `lifecycle/architecture/` before …"
    line and a "propose an ADR in `lifecycle/architecture/decisions/` on deviation" line. Add any
    that are missing.
  - Add `lifecycle/architecture/decisions` to `allowed_write_paths` for the analyst/developer
    agents that may author a *proposed* ADR (the `backend-developer` already lists
    `lifecycle/architecture/decisions` at ~176 — mirror that for the other design/build agents
    that FR-13 names).
- **Edit** `CLAUDE.md`: no change required — the "Architecture artifacts (KC-Release5)" section
  already states the read-first + propose-ADR rule; verify it still matches FR-21/FR-22.

### Acceptance criteria

- `go build ./...` clean (config is loaded and validated at startup — `make run` boots without a
  config-parse error).
- Integration/config test in [[architectural-artefacts-5-test]] asserts each design/build agent
  template contains the read-architecture directive and the propose-ADR directive, and that the
  ADR write-path is present for agents permitted to author proposed ADRs.

---

## Risk notes

- **Promotion/archive race** — `Promote` is not transactional across the archive-move + write.
  Single-user wizard/editor semantics make this acceptable for v1; documented for future
  hardening (mirror the existing optimistic-concurrency stance on artifact writes).
- **ADR number allocation race** — allocate-then-write re-checks existence and re-allocates once
  on collision; two truly-simultaneous creators are out of scope for v1 (single-user editing).
- **Lineage-relaxation blast radius** — the relaxation is strictly path-scoped via
  `IsArchitecturePath`; the M1 regression test guards that files outside
  `lifecycle/architecture/` keep the `lineage` requirement.

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean (new `internal/artifact` + `internal/architecture` unit tests).
3. `make test-integration` clean (new tests in [[architectural-artefacts-5-test]]).
4. Manual smoke via `make run`: `POST …/architecture/promote` with a catalog architecture +
   stack → two root copies appear with `parent:` set; re-run same → no duplicate; run a different
   architecture → prior copies land in `archive/`. `POST …/architecture/adrs` twice → `adr-0002`
   then `adr-0003`. All files appear in the graph/index without lineage-validation errors.
