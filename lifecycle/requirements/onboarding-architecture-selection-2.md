---
title: Architecture Wizard — Guided Architecture & Stack Selection
type: requirement
status: planning
lineage: onboarding-architecture-selection
parent: lifecycle/ideas/onboarding-architecture-selection.md
labels:
    - architecture
    - onboarding
    - ux
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architecture Wizard — Guided Architecture & Stack Selection

## Problem

A person starting a project — including a less-technical one — must choose an
architecture and a compatible technology stack before any durable design work
can begin. Today there is no guided path: the [architecture catalog](../architecture/README.md)
exists ([[architecture-templates]]) and the on-disk model for a project's own
architecture is defined ([[architectural-artefacts]]), but nothing walks a user
from "I have an idea" to a *chosen, promoted, and rationalised* architecture +
stack. Without that, users either copy a default blindly or make an uninformed
choice, and the critical / **architecture-breaking requirements** (offline,
collaboration, realtime, scale, mobile-first, cost, compliance, team skills)
are never systematically surfaced or recorded against the choice.

This requirement defines the **Architecture Wizard**: the onboarding UX (web UI,
v1) that lets a user reach a proven foundation by two paths — free **Browse** of
the catalog, or a short **Guided** questionnaire that *recommends* a fit — and
then persists the outcome by driving the promotion, summary, and ADR-0001
authoring defined in [[architectural-artefacts]].

## Goals / Non-goals

### Goals

- Provide a web-UI wizard, launchable **at any time** (from New Project and from
  the Architecture menu), that produces a chosen architecture + compatible stack.
- Offer two paths to the same destination: **Browse** the catalog and pick
  directly, or answer a **Guided** questionnaire (≤ 10 questions, each skippable)
  that surfaces architecture-breaking requirements and returns a ranked
  recommendation with a visible "why".
- Score recommendations by **rule-based label overlap** against the catalog
  artefacts' decision-signal `labels`; recommend, never gate — always allow
  "show me everything anyway".
- On completion, persist the outcome by invoking the model in
  [[architectural-artefacts]]: promote the chosen architecture + stack, write
  `architecture-summary.md` (Q&A + architecture-breaking requirements), and write
  ADR-0001.
- Detect a **prior run** in the project and inform the user before continuing; a
  re-run that changes the selection records a **new superseding ADR** and updates
  the summary — never a silent overwrite.
- Offer, as an **opt-in** follow-up step, scaffolding (config, pipelines, agent
  directives, repo skeleton) by handing off to [[architecture-templates]] §4 and
  [[agent-directives-generation]].

### Non-goals

- The on-disk artefact model — directory zones, promotion mechanics, ADR
  numbering, standards seeding. Owned by [[architectural-artefacts]]; the wizard
  is a *caller* of it.
- The shipped catalog contents (candidate architectures, stacks, standards
  seeds) and their `labels`. Owned by [[architecture-templates]].
- The visual browse/relationship surface and the post-selection display of the
  outcome. Owned by [[architecture-relationship-map]] and
  [[architecture-overview-view]]; the wizard *embeds/links* to them.
- Generating the concrete per-agent directive text. Owned by
  [[agent-directives-generation]].
- A **CLI** wizard (`kaos-control init` guided prompts). Explicitly deferred to
  the roadmap; v1 is web UI only.
- A **conversational / LLM-assisted** "describe your project in a sentence"
  recommendation mode. Intended follow-up, not v1 scope.

## Detailed Requirements

### Functional — entry & prior-run detection

- **FR-1** The wizard is launchable from at least two places: the New Project
  flow ([[new-project-init-directory-options]]) and an "Architecture" menu entry,
  and can be started at any time in a project's life, not only at creation.
- **FR-2** On start the wizard detects whether it has run before in this project
  by the presence of the promoted architecture, `architecture-summary.md`, and/or
  `decisions/adr-0001-*.md` (per [[architectural-artefacts]]).
- **FR-3** When a prior run is detected, the wizard informs the user of the
  existing selection and offers to **continue (re-run)** or **exit**; it does not
  proceed to overwrite anything without that explicit choice.

### Functional — path selection & browse

- **FR-4** After start, the user picks a path: **Browse** or **Guided**. Either
  path must be able to reach the same confirm step, and the user may switch from
  Guided to Browse ("show me everything anyway") at any point.
- **FR-5** The Browse path presents the catalog — architecture cards / comparison
  table and a link/embed to the [[architecture-relationship-map]] — showing each
  candidate's pros/cons and decision-signal labels, and lets the user pick an
  architecture directly.
- **FR-6** After an architecture is chosen (either path), the wizard presents the
  **compatible stacks** — those `related_to` the chosen architecture per the
  catalog compatibility edges — for selection.

### Functional — guided questionnaire

- **FR-7** The questionnaire asks **at most 10 questions**; every question is
  **skippable**. Questions map to the decision-signal `labels` on the catalog
  artefacts (e.g. offline → `offline-capable`, multi-user edit → `collaborative`,
  realtime/streaming → `realtime`, scale → `high-scale`, phone-first → `mobile`,
  AI/ML-central → `ai-ml`, ops tolerance → `low-complexity … high-complexity`,
  cost → `low-cost-start`, strongest languages → stack language ranking).
- **FR-8** Answers are treated as **hard constraints** or **soft signals**:
  hard constraints **filter** the candidate set (e.g. offline →
  desktop/edge/mobile architectures only; phone-first → mobile-native); soft
  signals **score** the remaining candidates by label overlap.
- **FR-9** The wizard presents the **top 2–3** architecture recommendations, each
  with a one-line "why" (which answers/signals drove it) — never a single
  black-box answer — and lets the user confirm the top choice or override with any
  other candidate.
- **FR-10** After the architecture is chosen, stacks are filtered to those
  compatible with it (FR-6) and **ranked by the team's language answer**; the user
  confirms or overrides.
- **FR-11** When signals are weak or ambiguous, the ranking applies a documented
  **default bias** toward the Modular Monolith plus a stack matching the team's
  language, as the low-regret starting point.
- **FR-12** Scoring is **deterministic and rule-based** for v1 (label scoring,
  no LLM); given the same catalog and the same answers it yields the same ranking.

### Functional — confirm & persist

- **FR-13** A **confirm** step shows the final selection — architecture + stack
  (+ any seeded standards to be created) — before anything is written to disk.
- **FR-14** On confirm, the wizard invokes the [[architectural-artefacts]]
  operations: **promote** the chosen architecture and stack to the
  `lifecycle/architecture/` root, **write `architecture-summary.md`** capturing the
  architecture-breaking requirements surfaced by the questionnaire and their
  mapping to the chosen architecture + stack, plus the full **Q&A trail**, and
  **write ADR-0001** ("Adopt <architecture> with <tech-stack>") containing the Q&A
  trail and the ranked alternatives that were rejected.
- **FR-15** The complete questionnaire **answers are persisted** as the selection
  rationale — in both `architecture-summary.md` and ADR-0001 — so the *why* behind
  the choice is traceable and displayable by [[architecture-overview-view]].
- **FR-16** On a **re-run** that changes the selection, the wizard records the
  change as a **new ADR** that supersedes the prior choice's ADR and updates
  `architecture-summary.md`; it never silently overwrites decision history.
  (Superseded promoted copies are archived per [[architectural-artefacts]] FR-7.)

### Functional — scaffolding hand-off

- **FR-17** After persistence, the wizard **offers (opt-in, never automatic)** a
  scaffolding step: config, pipelines (including testing + security scanning),
  agent directives, and a repo skeleton, by handing off to
  [[architecture-templates]] §4 and [[agent-directives-generation]].
- **FR-18** Where scaffolding requires naming choices, the wizard prompts for them
  and offers a **"decide for me"** default so a less-technical user can proceed.

### Non-functional

- **NFR-1** The wizard performs **no writes** to `lifecycle/architecture/` until
  the user confirms (FR-13); abandoning or exiting the wizard before confirm
  leaves the project unchanged.
- **NFR-2** Persistence (promotion + summary + ADR-0001) is **deterministic and
  idempotent** to re-run per [[architectural-artefacts]] FR-7 / NFR-3; a completed
  wizard leaves no orphaned or duplicate files.
- **NFR-3** All artefacts the wizard produces are plain markdown + YAML
  frontmatter, picked up by the existing indexing paths (startup scan, fsnotify
  watch, API writes) with no special-casing beyond the type/filename rules already
  defined in [[architectural-artefacts]].
- **NFR-4** Recommendation reasoning is **transparent**: for every recommended
  candidate the driving answers/signals are shown; the questionnaire recommends
  and does not gate (a user can always reach any catalog item).
- **NFR-5** The wizard is usable by a **less-technical user**: questions are in
  plain language, every question is skippable, and a "decide for me" / default
  path exists at each decision point (path choice, ambiguous scoring, naming).

## Acceptance Criteria

- [ ] The wizard can be launched from both the New Project flow and an
      Architecture menu entry, at any time in a project's life. *(FR-1)* — see [[new-project-init-directory-options]]
- [ ] Starting the wizard in a project that already has a promoted architecture /
      `architecture-summary.md` / ADR-0001 detects the prior run, informs the user,
      and offers continue-or-exit before writing anything. *(FR-2, FR-3, NFR-1)*
- [ ] The user can choose Browse or Guided, and can switch from Guided to Browse
      ("show me everything anyway") at any point. *(FR-4)*
- [ ] Browse presents catalog cards / comparison + the relationship map with
      pros/cons and labels, and lets the user pick an architecture directly.
      *(FR-5)* — see [[architecture-relationship-map]]
- [ ] The questionnaire asks ≤ 10 questions, each skippable, and each maps to a
      catalog decision-signal label. *(FR-7)*
- [ ] Hard-constraint answers filter the candidate set (e.g. offline excludes
      non-offline architectures); soft signals score the rest by label overlap.
      *(FR-8)*
- [ ] The wizard shows the top 2–3 architectures, each with a one-line "why", and
      lets the user confirm or override. *(FR-9, NFR-4)*
- [ ] After architecture selection, only compatible stacks are offered, ranked by
      the team's language answer, with confirm/override. *(FR-6, FR-10)*
- [ ] With weak/ambiguous signals the ranking biases toward Modular Monolith + a
      language-matched stack. *(FR-11)*
- [ ] Re-running the questionnaire with the same catalog and the same answers
      produces the same ranking (deterministic, rule-based, no LLM). *(FR-12)*
- [ ] A confirm step shows architecture + stack (+ seeded standards) before any
      disk write; abandoning before confirm leaves the project unchanged.
      *(FR-13, NFR-1)*
- [ ] Confirming promotes the chosen architecture + stack, writes
      `architecture-summary.md` (architecture-breaking requirements + mapping +
      Q&A), and writes ADR-0001 titled "Adopt <architecture> with <tech-stack>"
      with the Q&A trail and rejected alternatives. *(FR-14, FR-15)* — see [[architectural-artefacts]]
- [ ] The full questionnaire answers are persisted in both the summary and
      ADR-0001 and are available for display. *(FR-15)* — see [[architecture-overview-view]]
- [ ] A re-run that changes the selection writes a new superseding ADR and
      updates the summary without erasing prior history. *(FR-16)*
- [ ] After persistence, the wizard offers (opt-in) scaffolding — config,
      pipelines with testing + security scanning, agent directives, repo skeleton —
      with naming prompts that include a "decide for me" default. *(FR-17, FR-18)* — see [[architecture-templates]], [[agent-directives-generation]]
- [ ] Persistence is idempotent and leaves no orphaned/duplicate files on re-run;
      all produced artefacts are re-indexed by the existing paths. *(NFR-2, NFR-3)*

## Resolved Questions

- **OQ-1** The catalog compatibility edges are `related_to` links between
  architecture and stack artefacts. Is that edge data guaranteed present and
  bidirectionally queryable at wizard time, or does the wizard need a fallback
  when a chosen architecture has no `related_to` stacks recorded?

> Yes, the related_to will be in place.

- **OQ-2** For hard-constraint filtering (FR-8), what happens when constraints
  filter the candidate set to **zero** architectures (e.g. an over-constrained
  answer combination)? Options: relax the weakest constraint automatically and
  say so, or surface "no exact match — here is the closest" with the constraints
  that were dropped.

> “no exact match — here is the closest” with the constraints that were dropped

- **OQ-3** Should the wizard support **partial completion / resume** — a user
  abandoning midway and returning later — or is each run atomic (start → confirm
  or discard)? NFR-1 currently implies atomic-until-confirm with no saved
  in-progress state.

> Yes, it should support partial completion and resume.

- **OQ-4** Where do the questionnaire's question set and its question→label
  mapping live — hard-coded in the SPA, in the per-project `lifecycle/config.yaml`,
  or derived from the catalog labels themselves — given catalogs may add new
  signals over time?

> in the lifecyce/config.yaml

- **OQ-5** Is there an API/permission model consideration — which role(s) may run
  the wizard and thereby promote architecture + author ADR-0001 — or is that
  covered by existing mutation auth ([[auth-role-checks-mutations]])?

> product owner will be running the wizard.
