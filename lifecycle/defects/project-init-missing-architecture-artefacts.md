---
title: Project initialisation does not include architecture artefacts
type: defect
status: in-development
lineage: project-init-missing-architecture-artefacts
created: "2026-08-15T13:28:13+10:00"
priority: normal
labels:
    - defect
    - architecture
    - artefacts
    - onboarding
    - lifecycle
    - backend
assignees:
    - role: product-owner
      who: agent
---

# Project initialisation does not include architecture artefacts

## Reproduction Steps

1. Create a new kaos-control project via the initialisation flow.
2. Inspect the generated project structure under `lifecycle/`.
3. Observe that `lifecycle/architecture/` and its expected contents (catalog entries, architecture summary, decisions/, standards/) are absent.
4. Also inspect existing kaos-control projects — the architecture artefacts are likewise missing.

## Expected Behaviour

Project initialisation should scaffold the `lifecycle/architecture/` directory structure and seed it with the shipped architecture catalog (candidate architectures + tech-stacks), plus empty tracked `decisions/` and `standards/` directories. Existing projects should be retrofitted with the same structure on project open. See **Scope (decided)** below for the exact, narrowed boundary — the original wording overreached into decisions owned by other lineages.

## Actual Behaviour

The initialisation flow does not create `lifecycle/architecture/` or any of its required sub-directories and seed files. Existing kaos-control projects also lack this structure, meaning agents and the Architecture Wizard have no canonical location to read from or write to, and the architecture governance workflow cannot function correctly.

## Scope (decided 2026-08-15)

Narrowed to the **directory skeleton + shipped-catalog copy**, implementable end-to-end by
`backend-developer` (`internal/**`, `cmd/**`) with the milestones below — no separate
`plan-backend` needed. Catalog *content* seeding is deliberately pulled forward into this fix
(it was deferred to [[architecture-templates]] as a non-goal of [[architectural-artefacts-2]]):
the catalog now exists in this repo under `lifecycle/architecture/`, and shipping it into new
and existing projects is exactly the point of this defect. `architecture-templates` remains the
home for the richer wizard-driven seeding/refresh engine; this fix is its minimal, standalone
first slice.

**In scope**

- Embed the shipped catalog — `lifecycle/architecture/README.md`, `architectures/*.md`,
  `tech-stacks/*.md` — into the binary via `go:embed`.
- On `init` **and** on project `Open`: if `lifecycle/architecture/` is missing or incomplete,
  scaffold `architectures/`, `tech-stacks/`, `decisions/`, `standards/` and write the embedded
  catalog files. **Idempotent, skip-if-exists** (matching existing `scaffoldDirs` behaviour):
  a populated tree — like this repo's own — is left untouched.
- `decisions/` and `standards/` are created as **empty, tracked** directories (`.gitkeep`).

**Out of scope (resolves the questions this defect was blocked on)**

- **No `architecture-summary.md` placeholder** — the summary is wizard-created once an
  architecture is chosen ([[architectural-artefacts-2]] FR-8/FR-9); a project with no chosen
  architecture has nothing to summarise.
- **No `standards/` / `decisions/` content** at init — content is seeded post-wizard by the
  chosen architecture+stack ([[architectural-artefacts-2]] FR-16). Init creates the empty dirs
  only.
- **No new migration framework** — retrofit reuses the reconcile-on-`Open` path with
  skip-if-exists; no new CLI subcommand required.

## Milestones

1. **Embed the catalog.** `go:embed` the shipped `lifecycle/architecture/` catalog files
   (README + `architectures/*` + `tech-stacks/*`) into an `internal/` package exposing them as
   an `fs.FS`.
2. **Reconcile function.** `EnsureArchitectureScaffold(root)` — ensure the four subdirectories
   exist and every embedded catalog file is present, writing only missing files (skip-if-exists,
   idempotent); create `decisions/`/`standards/` with `.gitkeep`.
3. **Wire in.** Call it from `init` (`internal/initcmd`) and from project `Open`
   (`internal/project/project.go`) so both new and existing projects converge.
4. **Tests.** New project gets the full catalog + empty `decisions/`/`standards/`; an
   already-populated tree is untouched; a partial tree is completed; re-running is a no-op.
