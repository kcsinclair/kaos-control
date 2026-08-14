---
title: Architecture Wizard — Guided Architecture & Stack Selection
type: idea
status: draft
lineage: onboarding-architecture-selection
priority: normal
parent: lifecycle/ideas/architecture-templates.md
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

*(Named the **Architecture Wizard** — confirmed 2026-08-14.)*

The onboarding UX that sits on top of the [architecture catalog](../architecture/README.md)
and the [[architecture-templates]] design. kaos-control walks a person from
"I have an idea" to a scaffolded project by helping them **choose an
architecture, then a compatible stack** — either by free browsing or via a
short guided questionnaire that *recommends* a fit.

## Goal

Let a person — including a less-technical one — end up with a proven,
consistent foundation without having to already know the difference between a
modular monolith and event-driven microservices. Two paths, same destination:

1. **Browse** — explore the catalog (cards / comparison table / the
   [[architecture-relationship-map]]), read pros/cons, pick directly.
2. **Guided** — answer a handful of questions; get a ranked recommendation with
   the reasoning shown, then confirm or override.

## The questionnaire — hunting architecture-breaking requirements

The wizard's questions are not a style quiz: their job is to **flesh out the
high-level requirements and surface the *architecture-breaking* ones** — the
requirements that could break a solution, either the architecture shape or
the chosen technology (offline operation, multi-user collaboration, realtime,
scale, mobile-first, cost ceiling, team skills, compliance…). Each question
maps to the decision-signal `labels` already on the catalog artifacts.
Indicative questions → signals:

| Question | Drives |
| --- | --- |
| Must it work offline / without a network? | `offline-capable` |
| Do multiple people share and edit the same data? | `collaborative` |
| Is real-time / streaming behaviour core? | `realtime` |
| Expected scale — a team, a company, or internet-wide? | `high-scale` vs. small |
| Is it phone-first? | `mobile` |
| Is AI/ML central to the product? | `ai-ml` |
| How much ops/complexity can the team take on? | `low-complexity` … `high-complexity` |
| Minimise fixed running cost / start tiny? | `low-cost-start` |
| Team's strongest languages? | filters/ranks stacks by language labels |

### Scoring

- Filter architectures by hard constraints (offline → desktop/edge/mobile;
  phone-first → mobile-native).
- Score the rest by signal overlap; present the **top 2–3** with a one-line
  "why", not a single black-box answer.
- Then filter stacks to those `related_to` the chosen architecture, ranked by
  the team's language answer.
- Always allow "show me everything anyway" — the questionnaire recommends, it
  doesn't gate.

Default bias: when signals are weak/ambiguous, lean toward the
[Modular Monolith](../architecture/architectures/modular-monolith.md) + a stack
matching the team's language — the safe, low-regret starting point.

## Re-run behaviour

The wizard can be started **at any time**, not just at project creation. On
start it detects whether it has been run before in this project (presence of
the promoted architecture / `architecture-summary.md` / ADR-0001 — see
[[architectural-artefacts]]) and tells the user so; they can choose to
continue. A re-run that changes the selection records the change as a **new
ADR** (superseding ADR-0001's choice) and updates the Architecture Summary —
it never silently overwrites history.

## Flow

```
Start wizard (any time; from New Project, or the Architecture menu)
  → detect prior run? ──yes──▶ inform user ─▶ continue or exit
  → (idea/name)
  → Guided?  ──yes──▶ questionnaire ─▶ ranked archs ─▶ pick ─▶ ranked stacks ─▶ pick
        └──no──▶ browse catalog / relationship map ────────────────────────────┘
  → confirm selection (architecture + stack [+ seed standards])
  → persist: promote chosen artifacts, write architecture-summary.md (Q&A +
    critical requirements), write ADR-0001            ← [[architectural-artefacts]]
  → OFFER scaffolding (opt-in): config, pipelines incl. testing + security
    scanning, agent directives, repo skeleton — naming choices prompted, with
    "decide for me"                                    ← [[architecture-templates]] §4,
                                                         [[agent-directives-generation]]
```

## Scope / relationship to other work

- **Consumes** the catalog + compatibility edges (already seeded).
- **Hands off** to the selection→config scaffolding in [[architecture-templates]] §4
  and directive generation in [[agent-directives-generation]].
- **Complemented by** the visual [[architecture-relationship-map]] as the
  "browse" surface, and by [[architecture-overview-view]] which displays the
  outcome (chosen architecture, Q&A trail, NFRs, ADRs) afterwards.
- Both a **UI** flow (project-create wizard + Architecture menu) and a **CLI**
  flow (`kaos-control init` guided prompts) should be considered — the CLI
  matters for the "help me start a new project" story.

## Resolved Questions

- Do we persist the questionnaire answers on the project (as rationale / an ADR
  input) so the *why* behind the choice is traceable?

> Yes — persisted twice: in `architecture-summary.md` (the critical-requirements
> record) and in ADR-0001. The Q&A trail is displayed in
> [[architecture-overview-view]].

- Should recommendations be rule-based (label scoring) to start, with an
  optional LLM-assisted "describe your project in a sentence" mode later?

> Rule-based label scoring for v1. The conversational-AI mode ("have a
> discussion with it") is the intended follow-up, not v1 scope.

- Wizard in the web UI, the CLI, or both first?

> **Resolved (2026-08-14):** **UI wizard first.** The CLI flow
> (`kaos-control init` guided prompts) goes on the roadmap, to be built later
> if needed — v1 scope is web UI only.

## Resolved Questions

- How many questions is too many? (Target: ≤ 8, skippable.)

> Up to 10 is OK.
