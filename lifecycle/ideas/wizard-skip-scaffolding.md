---
title: Architecture Wizard — first-class "Skip scaffolding" for already-set-up projects
type: idea
status: planning
lineage: wizard-skip-scaffolding
created: "2026-08-21T11:20:00+10:00"
priority: normal
labels:
    - architecture
    - wizard
    - onboarding
    - ux
release: KC-Release5
---

# Architecture Wizard — first-class "Skip scaffolding" for already-set-up projects

The wizard's final step (`ScaffoldStep`, FR-17/FR-18) offers
config / pipelines / agent-directives / repo-skeleton scaffolding. It is already
opt-in — nothing runs until "Run scaffolding" is clicked — but there is no clear
**Skip** path, so on a **retrofit** (a project that already has its config,
directives, pipelines, and repo skeleton — e.g. kaos-control itself) the step
reads as an expected action rather than an optional one, and it isn't obvious
how to finish the wizard without running it.

## Thoughts

- Add an explicit **"Skip scaffolding / Finish"** action that completes the
  wizard cleanly without invoking the scaffolder — distinct from, and as
  prominent as, "Run scaffolding".
- Make the step **detect "already scaffolded"** and default to skip: the
  availability endpoint (`GET .../wizard/scaffold`) already knows what exists;
  when every offered artefact is present, present the step as "Everything's
  already in place — nothing to scaffold" with Skip as the primary action.
- Surface per-item state so a partially-scaffolded project can scaffold only the
  missing pieces and skip the rest, rather than an all-or-nothing run.

## Why it matters

Retrofitting the architecture process onto an existing codebase is a first-class
use case (it's how kaos-control adopts its own lifecycle). The scaffold step is
built for greenfield onboarding; on a retrofit it should get out of the way.
