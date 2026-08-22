---
title: Architecture Catalog Shipping & Scaffolding Engine
type: idea
status: done
lineage: architecture-templates
created: "2026-05-15T13:10:31+10:00"
priority: normal
parent: lifecycle/ideas/kaos-control-devops-cli.md
labels:
    - architecture
    - feature
    - onboarding
release: KC-Release5
---

# Architecture Catalog Shipping & Scaffolding Engine

**This idea is an EPIC, many ideas came from this original concept**

*(Retitled 2026-08-14 — was "Architecture Templates for Project Bootstrapping".
The "named template bundles" framing was resolved away during rationalisation;
see the resolved questions below.)*

> **Status note (2026-08-14) — what remains to build.** The catalog itself
> (curated `architecture` and `tech-stack` files under
> `lifecycle/architecture/`) is **already seeded**, and the compatibility
> edges exist as `related_to:` data. The remaining development in this idea is
> the machinery:
>
> 1. **Catalog shipping** — embed `lifecycle/architecture/` in the
>    kaos-control binary and copy it into new projects at init (§2).
> 2. **The scaffolding engine** — invoked when the Architecture Wizard's
>    "initialise scaffolding?" offer is accepted: stack-appropriate
>    `config.yaml` roster and write paths, devops pipelines including the
>    testing and security-scan components, seed standards/ADRs, and the
>    optional repo skeleton, with naming prompts and "decide for me" (§4).
> 3. **`KnownTypes` registration** for `architecture` and `tech-stack` (§1) —
>    to be built early, alongside [[architectural-artefacts]] (which adds
>    `adr`).
>
> Division of labour: the wizard ([[onboarding-architecture-selection]]) is
> the front-of-house UX; this idea is the back-of-house engine it calls.
> [[agent-directives-generation]] is the directive-file slice of the same
> scaffolding step, broken out on its own. [[architectural-artefacts]] defines
> the disk layout everything lands in.

Provide a library of curated architecture templates that define the underlying tech stack and high-level architecture decisions for common project patterns. Templates remove the burden of foundational decision-making from individual contributors, ensuring teams start with proven, consistent foundations.

Initial templates should include: GoLang Web App with Embedded SQLite, PHP with Symfony and PostgreSQL, and Python with MongoDB. Each template captures the canonical choices for language, framework, database, and key architectural patterns so that new projects inherit these decisions automatically.

Templates should be selectable at project creation time and stored as versioned artifacts within the lifecycle system. They should be extensible so teams can fork and customise a base template without losing traceability to the original.

---

## Design Expansion (2026-06-29)

Fleshes out the above into an implementable design, and connects it to the
newly-seeded catalog under [`lifecycle/architecture/`](../architecture/README.md).
The target flow — confirmed with Keith — is:

> **pick architecture → pick a compatible stack → the project is scaffolded
> with matching config, pipelines, and the ADRs/standards the agents follow.**
> *(This is literally how kaos-control itself came to life.)*

### 1. Two new artifact types + a three-layer model

A "template" is not one thing — it's three layers, and separating them keeps
each reusable and comparable:

1. **`architecture`** — the *shape* (Modular Monolith, Microservices, …). New
   type, added to `KnownTypes`.
2. **`tech-stack`** — the *tools* (Go + Vue, Go + gRPC, Flutter, …). New type,
   added to `KnownTypes`.
3. **Standards / ADRs** — the *rules the agents follow* (non-functionals,
   secrets handling, account minimums, "sort lists alphabetically", …). This
   is the sibling idea [[architectural-artefacts]]; a chosen architecture+stack
   seeds an initial ADR set.

The catalog now holds curated `architecture` and `tech-stack` artifacts, one
file each, cross-linked (see §3). A "template" in the original sense =
**architecture + stack + seed standards**, i.e. a saved selection across the
three layers.

### 2. Source of truth: catalog ships in the binary, copied at project init

Decided: the canonical catalog lives at **`kaos-control/lifecycle/architecture/`**
in *this* repo, is **embedded in the binary** (like `web/dist`), and is
**copied into a new project on init** so kaos-control can help bootstrap. It is
*not* per-project reference material — it is shared seed data. A project may
then fork/customise its local copy (the lineage `parent:` mechanism gives
"fork without losing traceability to the original" for free).

### 3. Compatibility model (architecture ↔ stack)

Each `architecture` declares its compatible stacks via `related_to:` (paths
into `architecture/tech-stacks/`). These become real graph edges *today*, and
are the filter that drives stack selection ("show only stacks that suit the
architecture I picked"). Stack files point back in prose. Longer term a typed
relationship (`compatible_with`) would be cleaner than reusing `related_to` —
see [[artefact-relationship-labels-and-links]].

### 4. Selection → config wiring (the high-value part)

Selecting an architecture+stack should **configure**, not just document. On
project init the chosen stack seeds:

- **`lifecycle/config.yaml`** — the agent roster, per-role `driver`/`model`,
  and `allowed_write_paths` appropriate to the stack (e.g. Go+Vue →
  backend-developer writes `internal/`,`cmd/`; frontend-developer writes
  `web/src/`).
- **Devops pipelines** (`lifecycle/devops/*.yaml`) — the toolchain for the
  stack (Go+Vue → `go test`, `go build`, `pnpm build`, lint; Python+FastAPI →
  `pytest`, `ruff`; etc.).
- **Seed ADRs/standards** — an initial [[architectural-artefacts]] set the
  agents read when they design/build.
- **Scaffold** — optional starter directory layout / repo skeleton for the
  stack.

This is what closes the loop: the onboarding choice pre-wires the project so
the agents can immediately work within the right structure and toolchain.

### 5. Guided selection (questionnaire)

Beyond free browsing, a short questionnaire recommends an architecture+stack
from decision signals already encoded as `labels` on each catalog artifact
(`offline-capable`, `collaborative`, `realtime`, `high-scale`, `low-cost-start`,
`ai-ml`, `mobile`, …). Detailed in the companion idea
[[onboarding-architecture-selection]].

### 6. Catalog metadata schema (for the requirements stage)

Today the human-readable decision signals live in each artifact's body table +
a subset as `labels`. To make selection/recommendation fully machine-driven,
promote the key fields into typed frontmatter (extend the `Frontmatter`
struct), e.g. on `architecture`: `scaling`, `offline`, `collaboration`,
`scale`, `complexity`; on `tech-stack`: `languages`, `layer`,
`communication`, `data_options`, `footprint`, `learning_curve`. Until then the
`labels` carry the filterable signals and nothing is lost.

### 7. Note on filenames / lineage

Catalog entries use **clean slug filenames** (`modular-monolith.md`), *not* the
lineage `-N` monotonic index — they are standalone reference artifacts, not
steps in an idea→release lineage. The required `lineage:` field is set to the
slug. Worth ratifying this exception to §3.3 of the spec during requirements.

### Open questions for requirements

- Is a "template" a *saved selection* (architecture + stack + standards) that's
  itself a versioned artifact, or just the act of choosing three separate
  artifacts at init? (Leaning: a lightweight saved selection artifact.)

> **Resolved (2026-08-14):** the saved selection is the **promotion** of the
> chosen architecture + tech-stack artifacts into the `lifecycle/architecture/`
> root, plus the generated `architecture-summary.md` and ADR-0001. No separate
> "template" artifact type is needed — see [[architectural-artefacts]] for the
> full on-disk model.

- How much scaffolding do we generate vs. reference? (repo skeleton? or just
  config + pipelines + ADRs?)

> **Resolved (2026-08-14):** after the stack is chosen, the Architecture Wizard
> **offers** to initialise the scaffolding for the components (opt-in, not
> forced). Scaffolding = `config.yaml` roster, devops pipelines, seed
> ADRs/standards, agent directives ([[agent-directives-generation]]), *and* the
> starter directory skeleton for the stack. Where file/directory naming choices
> exist, the wizard asks the user — with a **"decide for me"** option that
> applies the stack's canonical defaults. The scaffold always includes the
> **testing and security-scanning** components for the stack (e.g. test
> pipeline + [[security-scan-pipeline]]-style scan); where a component needs a
> human decision (e.g. which scanner licence, which CI), the wizard raises an
> **idea artifact** for it instead of guessing.

- Where does the embedded catalog get copied on init, and how are project-local
  edits reconciled with catalog updates in later kaos-control versions?

> **Partially resolved:** copied to `lifecycle/architecture/{architectures,tech-stacks}/`
> on init (decided in §2). Reconciliation with later catalog versions remains
> open for the requirements stage.

- Which initial templates ship as *named bundles*? The original idea listed
  Go+SQLite, PHP/Symfony/Postgres, and Python/Mongo — the stack catalog now
  includes [[php-symfony-postgres]] and [[python-mongodb]] (11 stacks total), so
  named bundles can be assembled from the catalog rather than invented.

> **Resolved (2026-08-14):** no named bundles for v1 — the wizard assembles
> architecture + compatible stack from the catalog; bundles can come later if
> recurring combinations emerge.

---

## Rationalisation with the KC-Release5 set (2026-08-14)

How the pieces divide, so nothing overlaps or conflicts:

| Concern | Owner |
| --- | --- |
| Catalog + 3-layer model + selection→config scaffolding | **this idea** |
| On-disk model after selection: promotion, `architecture-summary.md`, `decisions/` (ADRs), `standards/` | [[architectural-artefacts]] |
| The **Architecture Wizard** (questionnaire, architecture-breaking requirements, re-run behaviour) | [[onboarding-architecture-selection]] |
| Visual browse surface for the catalog (2D/3D map) | [[architecture-relationship-map]] |
| Post-selection **Architecture** menu view (chosen architecture, Q&A, NFRs, ADRs) | [[architecture-overview-view]] |
| Generating CLAUDE.md / AGENTS.md / GEMINI.md / Antigravity directives + stack-tuned agent prompts at init | [[agent-directives-generation]] |
| Auto-generating a diagram of the *current* system from code | [[architecture-auto-diagram]] (backlog) |

Two clarifications folded in from the process notes:

- **§4 config seeding includes agent directives.** The `config.yaml` roster and
  prompts seeded here must be generated *correctly for the chosen stack* —
  right write paths, right build/test commands in each developer prompt — and
  the same init emits the per-agent-CLI directive files. Details in
  [[agent-directives-generation]].
- **§5 questionnaire is framed around architecture-breaking requirements**:
  the questions exist to surface the requirements that could break a solution
  (offline, collaboration, scale, realtime, …), and those answers are
  persisted into the Architecture Summary as the critical-requirements record.
