---
title: Onboarding — Guided Architecture & Stack Selection
type: idea
status: blocked
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

# Onboarding — Guided Architecture & Stack Selection

The onboarding UX that sits on top of the [architecture catalog](../architecture/README.md)
and the [[architecture-templates]] design. When someone starts a new project,
kaos-control walks them from "I have an idea" to a scaffolded project by helping
them **choose an architecture, then a compatible stack** — either by free
browsing or via a short guided questionnaire that *recommends* a fit.

## Goal

Let a person — including a less-technical one — end up with a proven,
consistent foundation without having to already know the difference between a
modular monolith and event-driven microservices. Two paths, same destination:

1. **Browse** — explore the catalog (cards / comparison table / the
   [[architecture-relationship-map]]), read pros/cons, pick directly.
2. **Guided** — answer a handful of questions; get a ranked recommendation with
   the reasoning shown, then confirm or override.

## The questionnaire

A short set of questions, each mapping to the decision-signal `labels` already
on the catalog artifacts. Indicative questions → signals:

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
| Team's strongest languages? | filters/【ranks】stacks by language labels |

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
[Modular Monolith](../architectures/modular-monolith.md) + a stack matching the
team's language — the safe, low-regret starting point.

## Flow

```
New project
  → (idea/name)
  → Guided?  ──yes──▶ questionnaire ─▶ ranked archs ─▶ pick ─▶ ranked stacks ─▶ pick
        └──no──▶ browse catalog ───────────────────────────────────────────────┘
  → confirm selection (architecture + stack [+ seed standards])
  → scaffold project (config.yaml, devops pipelines, seed ADRs)   ← see [[architecture-templates]] §4
```

## Scope / relationship to other work

- **Consumes** the catalog + compatibility edges (already seeded).
- **Hands off** to the selection→config scaffolding in [[architecture-templates]] §4.
- **Complemented by** the visual [[architecture-relationship-map]] as the
  "browse" surface.
- Both a **UI** flow (project-create wizard) and a **CLI** flow
  (`kaos-control init` guided prompts) should be considered — the CLI matters
  for the "help me start a new project" story.

## Open questions

- Wizard in the web UI, the CLI, or both first?
- Do we persist the questionnaire answers on the project (as rationale / an ADR
  input) so the *why* behind the choice is traceable?
- How many questions is too many? (Target: ≤ 8, skippable.)
- Should recommendations be rule-based (label scoring) to start, with an
  optional LLM-assisted "describe your project in a sentence" mode later?
