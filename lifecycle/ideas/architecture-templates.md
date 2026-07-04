---
title: Architecture Templates for Project Bootstrapping
type: idea
status: draft
lineage: architecture-templates
created: "2026-05-15T13:10:31+10:00"
priority: normal
labels:
    - architecture
    - feature
    - onboarding
release: KC-Release5
parent: lifecycle/ideas/kaos-control-devops-cli.md
---

# Architecture Templates for Project Bootstrapping

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
- How much scaffolding do we generate vs. reference? (repo skeleton? or just
  config + pipelines + ADRs?)
- Where does the embedded catalog get copied on init, and how are project-local
  edits reconciled with catalog updates in later kaos-control versions?
- Which initial templates ship as *named bundles* (the original idea listed
  Go+SQLite, PHP/Symfony/Postgres, Python/Mongo — note PHP/Symfony and Mongo
  aren't yet in the stack catalog; add them, or start from the current nine)?
