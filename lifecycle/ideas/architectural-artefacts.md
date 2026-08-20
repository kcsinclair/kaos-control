---
created: "2026-08-14T12:17:40+10:00"
title: Architectural Artefacts
type: idea
status: done
lineage: architectural-artefacts
priority: normal
labels:
    - architecture
    - artefacts
    - enhancement
release: KC-Release5
parent: lifecycle/ideas/kaos-control-devops-cli.md
---

# Architectural Artefacts

See new related [[architecture-templates]]

The goal of architecture artefacts is that the agents should be referring to these when designing, etc.  This should cover some of the non-functional things, and how secrets are handled, minimum options for user accounts, etc.

There should also be a set of standards, policies, rules for usability, etc, like always sort lists alphabetically

People with less technical experience can still get an enterprise class application (get closer to).

New directory in lifecycle which defines any architectural artefacts.

These may be determined during development or defined up front.

e.g. programming language, frameworks, database, operating system, decisions.

Maintain ADR's

Some of these may also be documents and diagrams for humans, and/or the robots.

---

## Converged Design (2026-08-14)

Rationalised with [[architecture-templates]], [[onboarding-architecture-selection]]
(the **Architecture Wizard**), [[architecture-relationship-map]],
[[architecture-overview-view]] and [[agent-directives-generation]]. This idea
owns the *artefact model on disk*: what lives in `lifecycle/architecture/`
once a project has chosen its architecture and stack, and what the agents read.

### Directory layout — `lifecycle/architecture/` is the single home

The "new directory in lifecycle" above **is** the existing
`lifecycle/architecture/` — no second directory is introduced. It has two
zones: the *catalog* (shipped/copied reference material) and the *project's
own architecture* (the chosen items, promoted to the root, plus decisions and
standards):

```
lifecycle/architecture/
├── README.md                    catalog index (ships with kaos-control)
├── architectures/               catalog: candidate architectures (type: architecture)
├── tech-stacks/                 catalog: candidate stacks        (type: tech-stack)
├── <chosen-architecture>.md     PROMOTED copy of the chosen architecture
├── <chosen-tech-stack>.md       PROMOTED copy of the chosen stack
├── architecture-summary.md      the Architecture Summary (see below)
├── decisions/                   ADRs (type: adr) — adr-0001, adr-0002, …
└── standards/                   standards, policies, usability rules the agents follow
```

**Promotion**: when the Architecture Wizard completes, the chosen architecture
and tech-stack artifacts are *copied from the catalog subdirectories into the
`lifecycle/architecture/` root* (keeping `parent:` pointing back at the
catalog entry for traceability). The root copies are the project's editable
reference for *how this system is built*; the catalog below remains the
untouched menu of alternatives.

### The Architecture Summary

`architecture-summary.md` is the main architecture document. It records:

- the **critical / architecture-breaking requirements** surfaced by the
  wizard (or by later requirements work) and *how each one maps to the chosen
  architecture and tech stack* — i.e. why this shape and these tools satisfy
  the things that could break a solution;
- the wizard **questions and answers** (the selection rationale);
- pointers to the promoted architecture, stack, ADRs and standards.

It is created by the wizard and maintained thereafter — when a new
architecture-breaking requirement appears, the summary (and usually an ADR)
is updated. Surfaced in the UI by [[architecture-overview-view]].

### ADRs

- Live in `lifecycle/architecture/decisions/`, one file per decision,
  `type: adr`, numbered `adr-0001-<slug>.md` onward.
- **ADR-0001 is written automatically by the wizard**: "Adopt <architecture>
  with <tech-stack>", body containing the Q&A trail and the ranked
  alternatives that were rejected.
- Subsequent ADRs are raised whenever an architecture/design/technology
  decision is made — by humans in the editor, or by agents (analysts and
  developers are prompted to *propose* an ADR when they deviate from or
  extend the recorded architecture; see [[agent-directives-generation]] and
  the agent prompt updates).
- ADRs are first-class artifacts: indexed, graphable, and visualised in
  [[architecture-overview-view]].

### Standards

`standards/` holds the rules-for-the-robots (and humans): non-functional
baselines, secrets handling, minimum account/user options, usability rules
("always sort lists alphabetically"), testing and security-scanning
expectations. A chosen architecture+stack **seeds an initial standards set**
(from [[architecture-templates]] §4); teams extend it. Agents are directed to
read `lifecycle/architecture/` (summary, promoted choices, ADRs, standards)
before designing or building — this is how less-technical users still get an
enterprise-class result.

### New artifact types

`architecture`, `tech-stack` (already planned in [[architecture-templates]] §1)
plus **`adr`** are added to `KnownTypes`. The summary and standards files can
be `type: doc` initially; a dedicated type is not required for v1.

### Filenames

Catalog entries, promoted copies, ADRs and standards all use **clean slug
filenames** (no lineage `-N` index) — they are standing reference artifacts,
not steps in an idea→release lineage (the exception already flagged in
[[architecture-templates]] §7; to be ratified in requirements).
