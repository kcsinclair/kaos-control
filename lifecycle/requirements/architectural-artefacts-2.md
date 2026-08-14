---
title: Architectural Artefacts — On-Disk Model
type: requirement
status: blocked
lineage: architectural-artefacts
parent: lifecycle/ideas/architectural-artefacts.md
labels:
    - architecture
    - artefacts
    - enhancement
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architectural Artefacts — On-Disk Model

## Problem

Agents (analysts and developers) design and build without a durable, machine-readable
statement of *how this system is built* and *what rules it must follow*. Non-functional
concerns — secrets handling, minimum account/user options, usability rules (e.g. "always
sort lists alphabetically"), testing and security-scanning expectations — live only in
people's heads or scattered prose, so a less-technical user cannot reliably get an
enterprise-class result. There is also no consistent home for the chosen architecture and
tech stack, no recorded rationale tying them to the requirements that could break the
solution, and no standing record of architecture/technology decisions (ADRs).

This requirement defines the **artefact model on disk**: what lives in
`lifecycle/architecture/` once a project has chosen its architecture and stack, how those
files are produced and maintained, and what agents are directed to read. It does **not**
define the wizard UX ([[onboarding-architecture-selection]]), the catalog seed content
([[architecture-templates]]), or the visualisations ([[architecture-overview-view]],
[[architecture-relationship-map]]) — those are owned by their own lineages and are
consumers/producers of the model defined here.

## Goals / Non-goals

### Goals

- Define a single on-disk home — `lifecycle/architecture/` — with two zones: the *catalog*
  (reference menu) and the *project's own architecture* (promoted choices, summary,
  decisions, standards).
- Specify the **promotion** operation that copies chosen catalog items to the directory root
  while preserving traceability back to the catalog.
- Specify the **Architecture Summary** document: content, provenance, and maintenance rules.
- Specify **ADRs** as first-class artefacts (numbering, type, who writes them, when).
- Specify the **standards** set: what it holds, how it is seeded and extended.
- Register the new artefact types (`architecture`, `tech-stack`, `adr`) and ratify the
  clean-slug (no lineage index) filename exception for everything under
  `lifecycle/architecture/`.
- Direct agents to read `lifecycle/architecture/` before designing or building, and to
  propose an ADR when they deviate from or extend the recorded architecture.

### Non-goals

- The Architecture Wizard's screens, question flow, and scoring — [[onboarding-architecture-selection]].
- The shipped catalog contents (candidate architectures, stacks, standards seeds) —
  [[architecture-templates]].
- UI rendering of the summary, ADRs, and relationships — [[architecture-overview-view]],
  [[architecture-relationship-map]].
- Generating the concrete per-agent prompt directives — [[agent-directives-generation]]
  (this requirement only states the directive that must exist).
- Enforcing conformance automatically (linting standards against code); v1 relies on agents
  reading and honouring the artefacts.

## Detailed Requirements

### Functional — directory layout

- **FR-1** `lifecycle/architecture/` is the single home for architectural artefacts. No
  second lifecycle directory is introduced for this feature.
- **FR-2** The directory has two zones that coexist:
  - *Catalog*: `README.md` (catalog index), `architectures/` (`type: architecture`),
    `tech-stacks/` (`type: tech-stack`) — reference material, unchanged by promotion.
  - *Project's own architecture*: promoted `<chosen-architecture>.md` and
    `<chosen-tech-stack>.md` at the root, `architecture-summary.md`, `decisions/`, and
    `standards/`.
- **FR-3** The catalog zone remains present and editable after a project has chosen its
  architecture; promotion never deletes or mutates catalog entries.

### Functional — promotion

- **FR-4** When the Architecture Wizard completes, the chosen architecture artefact and the
  chosen tech-stack artefact are **copied** from `architectures/` / `tech-stacks/` into the
  `lifecycle/architecture/` root.
- **FR-5** Each promoted root copy keeps a `parent:` frontmatter field pointing back to the
  originating catalog entry (relative repo path) for traceability.
- **FR-6** The promoted root copies are the project's editable reference; edits to a root
  copy do not propagate to the catalog entry and vice versa.
- **FR-7** Promotion is idempotent for a given choice: re-running it with the same selection
  overwrites the existing root copies rather than creating duplicates. Choosing a *different*
  architecture/stack replaces the previously promoted root copies (superseded content remains
  in git history).

### Functional — Architecture Summary

- **FR-8** `architecture-summary.md` is the primary architecture document. It records:
  - the **critical / architecture-breaking requirements** surfaced by the wizard or later
    requirements work, and for each, *how it maps to* the chosen architecture and tech stack
    (why this shape and these tools satisfy it);
  - the wizard **questions and answers** (the selection rationale);
  - pointers (links) to the promoted architecture, promoted stack, ADRs, and standards.
- **FR-9** The summary is created by the wizard and maintained thereafter. When a new
  architecture-breaking requirement appears, the summary is updated (and, per FR-13, usually
  an ADR is raised).
- **FR-10** The summary is a first-class indexed artefact so it is discoverable and
  link-resolvable; `type: doc` is acceptable for v1 (no dedicated type required).

### Functional — ADRs

- **FR-11** ADRs live in `lifecycle/architecture/decisions/`, one file per decision,
  `type: adr`, named `adr-<NNNN>-<slug>.md` with a zero-padded 4-digit monotonic number
  starting at `adr-0001`.
- **FR-12** **ADR-0001 is written automatically by the wizard**: title "Adopt
  <architecture> with <tech-stack>", body containing the Q&A trail and the ranked
  alternatives that were rejected.
- **FR-13** Subsequent ADRs are raised whenever an architecture/design/technology decision is
  made — by humans in the editor or by agents. Analyst and developer agents are prompted to
  *propose* an ADR when they deviate from or extend the recorded architecture.
- **FR-14** ADRs are first-class artefacts: indexed, graphable, and available to
  [[architecture-overview-view]]. ADR numbering is monotonic and never reused; a superseded
  ADR is marked (e.g. `status:` / a "Superseded by" pointer) rather than deleted.

### Functional — standards

- **FR-15** `lifecycle/architecture/standards/` holds the rules-for-the-robots (and humans):
  non-functional baselines, secrets handling, minimum account/user options, usability rules
  (e.g. "always sort lists alphabetically"), and testing/security-scanning expectations.
- **FR-16** A chosen architecture+stack **seeds an initial standards set** (sourced from
  [[architecture-templates]] §4); teams may add to or edit the seeded standards afterwards.
- **FR-17** Standards files are indexed artefacts; `type: doc` is acceptable for v1.

### Functional — types & filenames

- **FR-18** `architecture`, `tech-stack`, and `adr` are added to `KnownTypes`
  ([internal/artifact/artifact.go](internal/artifact/artifact.go)); existing type validation,
  frontmatter editing, and indexing accept them.
- **FR-19** Everything under `lifecycle/architecture/` (catalog entries, promoted copies,
  ADRs, standards, the summary) uses **clean slug filenames with no lineage `-N` index** —
  these are standing reference artefacts, not steps in an idea→release lineage. This ratifies
  the exception already stated in CLAUDE.md and [[architecture-templates]] §7.
- **FR-20** The lineage/index validation and any "parent points to previous lineage step"
  checks are relaxed for `lifecycle/architecture/`: a `parent:` on a promoted copy points to a
  catalog entry (not a prior lineage index), and files here are not required to carry a
  `lineage:`/monotonic index.

### Functional — agent directives

- **FR-21** Agents are directed (via their prompt templates) to read `lifecycle/architecture/`
  — the summary, promoted architecture and stack, ADRs, and standards — **before** any
  substantive design, planning, or implementation work, and to conform to them.
- **FR-22** When a change genuinely requires deviating from or extending the recorded
  architecture, agents must not deviate silently: they propose a new ADR (FR-13) rather than
  changing course without a record. (The concrete directive text is produced by
  [[agent-directives-generation]].)

### Non-functional

- **NFR-1** All artefacts under `lifecycle/architecture/` are plain markdown with YAML
  frontmatter, readable and editable outside the tool; disk remains authoritative and the
  SQLite index remains a cache.
- **NFR-2** New/promoted/edited architecture artefacts are picked up by the existing indexing
  paths (startup scan, fsnotify live watch, API writes) with no special-casing beyond type
  registration and the filename-exception rule.
- **NFR-3** Promotion and ADR-0001 authoring must be deterministic and safe to re-run
  (idempotent per FR-7) without leaving orphaned or duplicate files.
- **NFR-4** Links between architecture artefacts use the existing `[[slug]]` / relative-path
  convention so they resolve in the graph and editor.

## Acceptance Criteria

- [ ] `lifecycle/architecture/` supports both zones simultaneously: catalog (`README.md`,
      `architectures/`, `tech-stacks/`) and project-own (promoted copies, `architecture-summary.md`,
      `decisions/`, `standards/`). *(FR-1, FR-2)*
- [ ] Completing the wizard copies the chosen architecture and tech-stack artefacts to the
      `lifecycle/architecture/` root; catalog entries remain untouched. *(FR-3, FR-4)* — see [[onboarding-architecture-selection]]
- [ ] Each promoted root copy has `parent:` pointing to its catalog source. *(FR-5)*
- [ ] Re-running promotion with the same choice overwrites (no duplicates); a different choice
      replaces the prior root copies. *(FR-7)*
- [ ] `architecture-summary.md` contains: architecture-breaking requirements with per-requirement
      mapping to architecture+stack, the wizard Q&A, and links to promoted choices/ADRs/standards. *(FR-8)*
- [ ] Adding a new architecture-breaking requirement results in an updated summary (and an ADR
      where a decision was made). *(FR-9, FR-13)* — see [[architecture-relationship-map]]
- [ ] `decisions/adr-0001-<slug>.md` exists after the wizard runs, titled "Adopt <architecture>
      with <tech-stack>", containing the Q&A trail and rejected alternatives. *(FR-11, FR-12)*
- [ ] New ADRs can be created by humans and proposed by agents; numbering is monotonic,
      zero-padded, never reused. *(FR-11, FR-13, FR-14)*
- [ ] `standards/` is seeded from the chosen architecture+stack and is editable afterwards. *(FR-15, FR-16)* — see [[architecture-templates]]
- [ ] `architecture`, `tech-stack`, and `adr` validate as known types in the parser,
      frontmatter editor dropdowns, and index. *(FR-18)*
- [ ] Files under `lifecycle/architecture/` with clean slug filenames (no `-N` index) index
      without lineage-validation errors; a promoted copy whose `parent:` is a catalog entry is
      accepted. *(FR-19, FR-20)*
- [ ] Agent prompt templates instruct reading `lifecycle/architecture/` before design/build and
      proposing an ADR on deviation. *(FR-21, FR-22)* — see [[agent-directives-generation]]
- [ ] All new/promoted architecture artefacts are re-indexed by startup scan, live watch, and
      API writes without special-casing beyond FR-18/FR-19. *(NFR-2)*
- [ ] Promotion and ADR-0001 authoring are idempotent and leave no orphaned/duplicate files on
      re-run. *(NFR-3)*

## Open Questions

- **OQ-1** Where in the model does a *doc/diagram for humans* (mentioned in the idea) live —
  under `standards/`, a new `docs/` sub-folder, or the existing docs panel? v1 assumes the
  summary + standards cover the human-facing narrative; a dedicated diagram home is out of
  scope pending [[architecture-overview-view]].

> kaos-control would keep lifecycle related docs in lifecycle/docs, end user docs in a docs directory the project root directory, this is the Documentation link in the gui,  implemented now in kaos-control, any diagrams can live in that docs directory.

- **OQ-2** Should the summary and standards eventually get dedicated `type:` values, or is
  `type: doc` sufficient long-term? v1 uses `doc`.

> type doc for now works.

- **OQ-3** When a *different* architecture is chosen (re-promotion), should the previously
  promoted root copies be hard-deleted (relying on git history) or moved to an archive
  location? FR-7 currently assumes deletion via overwrite/replacement.

> moved to archive location

- **OQ-4** Should agent-*proposed* ADRs land as `status: draft` awaiting human approval, or be
  written directly? Assumed `draft` pending confirmation, aligning with the normal transition
  gating.

- **OQ-5** Is there a maximum/duplicate-detection rule for standards seeded from both the
  architecture *and* the tech-stack (e.g. two secrets-handling standards)? Assumed the seed set
  is de-duplicated by [[architecture-templates]] before promotion.
