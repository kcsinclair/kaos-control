# Plan: migrate FEATURES.md into `feature` artifacts (tech-writer)

**Role:** tech-writer · **Type of work:** documentation migration · **Status:** ready

## Objective

Convert the archived subsystem sections still in [FEATURES.md](../FEATURES.md)
into first-class `type: feature` lifecycle artifacts under
[`lifecycle/features/`](../lifecycle/features/), then reduce FEATURES.md to a
pure deprecation pointer. The Features view (left menu) reads these artifacts and
groups them by their `function:` field.

## Why

`feature` is now a standing-reference lifecycle type (like the architecture
zone): indexed, browsable, searchable, and linkable — so shipped capability is
tracked *inside* the lifecycle instead of in a flat file that drifts (FEATURES.md
went stale for 3+ months). Six features are already migrated; this finishes the
job.

## Context you need before starting

- **The model.** A feature is a standing-reference artifact under
  `lifecycle/features/`, exempt from lineage-index rules (clean slug filename, no
  `-N` index). It carries a `function:` field (the Features-view grouping key)
  and links back to the artifacts that delivered it via `related_to:`.
- **Copy the shape from an existing one.** Use
  [`lifecycle/features/architecture-wizard-and-catalog.md`](../lifecycle/features/architecture-wizard-and-catalog.md)
  and [`agent-permission-mediation.md`](../lifecycle/features/agent-permission-mediation.md)
  as templates — match their frontmatter and body structure exactly.
- **Do NOT re-migrate** what already exists. Already done (skip these sections):
  Architecture (wizard/catalog + overview/zone), Agent directives generation,
  Agent permission mediation, Reports (→ agent usage reports), and the
  create/add-existing part of onboarding (→ project onboarding).

## Artifact template

```markdown
---
title: <Human name of the capability>
type: feature
status: approved
lineage: feature-<kebab-slug>
created: "<RFC3339 current date-time, e.g. 2026-08-21T13:00:00+10:00>"
summary: <one factual sentence — shown on the feature card>
function: <grouping bucket, see the worklist>
labels:
    - feature
    - <area label(s)>
related_to:
    - <path to the idea/requirement/release/ADR that delivered it, if found>
---

# <Human name of the capability>

<one-line framing>

## What it does

- <bullet lifted/tightened from the FEATURES.md section>
- ...

Reachable at **<view/menu>**; API under `<route prefix>` (where applicable).
```

Rules:
- `status: approved` (shipped and in force — this is what the current six use, and
  it keeps features visible in the list by default rather than hidden behind the
  terminal-status filter). Do not use `done`.
- `lineage: feature-<slug>` where `<slug>` matches the filename stem.
- `function:` must be **reused consistently** so the Features view groups cleanly
  — prefer an existing bucket over a near-duplicate (`Agents`, not `Agent`).
- Keep bodies factual and short. Do not invent capabilities: every claim must
  trace to the FEATURES.md text or to the code/running app. If a FEATURES.md
  bullet is stale (feature changed/removed), verify against the code before
  writing it, and correct or drop it.

## Worklist (remaining sections → target artifacts)

One feature per subsystem section; split only where a section clearly holds two
distinct capabilities. Reuse existing `function` buckets where noted.

| FEATURES.md section | File (`lifecycle/features/…`) | `function` |
| --- | --- | --- |
| Lifecycle & artifacts | `markdown-lifecycle-artifacts.md` | Lifecycle & artifacts |
| — RICE prioritisation (split out) | `rice-prioritisation.md` | Lifecycle & artifacts |
| Workflow & state machine | `workflow-state-machine.md` | Workflow |
| Graph & visualisation | `graph-visualisation.md` | Visualization |
| Agents (core orchestration) | `agent-orchestration.md` | Agents |
| Idea capture | `idea-capture.md` | Idea capture |
| Releases & roadmap | `releases-and-roadmap.md` | Releases & roadmap |
| Dashboard | `dashboard.md` | Dashboard |
| Kanban board | `kanban-board.md` | Boards & views |
| DevOps pipelines | `devops-pipelines.md` | DevOps |
| Scheduler | `scheduler.md` | Scheduler |
| Ollama (local LLMs) | `ollama-local-llms.md` | Agents |
| Project feed | `project-feed.md` | Activity & feed |
| Multi-project | `multi-project.md` | Projects & onboarding |
| Auth & authorisation | `auth-and-authorisation.md` | Auth & access |
| Git integration | `git-integration.md` | Git |
| Operations | `operations.md` | Operations |

(Function buckets `Architecture`, `Agents`, `Reports & analytics`, and
`Projects & onboarding` already exist — match their spelling.)

## Finding `related_to` links

For each feature, try to link the artifact(s) that produced it:
- `grep -ril "<keyword>" lifecycle/ideas lifecycle/requirements` for the
  originating idea/requirement;
- link a release under `lifecycle/releases/` if the section maps to one;
- link a relevant ADR under `lifecycle/architecture/decisions/`.
Omit `related_to` rather than guessing a wrong link.

## Steps

1. Read FEATURES.md and this plan fully. Confirm the "already done" list against
   `lifecycle/features/` so nothing is duplicated.
2. For each worklist row, author the artifact from the template, lifting the
   section's bullets into `## What it does` and tightening them. Verify each
   claim against the code/app; correct stale bullets.
3. Add `related_to` links found via the grep above.
4. After all rows are done, **trim FEATURES.md**: keep only the deprecation
   banner (the block above the first `---`) and delete the archived sections
   below it. Update the banner's "Already migrated" line to "fully migrated".
5. Do not edit code, config, or any non-feature artifact.

## Verification (definition of done)

- Every worklist section has a corresponding `lifecycle/features/*.md`; none
  duplicate the six pre-existing features.
- All new artifacts parse cleanly: the **Parse Errors** view is empty for them
  (or `go run ./cmd/kaos-control … ` indexing logs no parse errors for
  `lifecycle/features/`).
- The **Features** view shows every feature, grouped by `function`, with search
  and the function/status filters working; each card's summary reads well.
- `function` values are consistent (no near-duplicate buckets).
- FEATURES.md is reduced to the deprecation banner only.
- Commit with a message like
  `docs(features): migrate remaining FEATURES.md sections to feature artifacts`.

## Guardrails

- Scope is documentation only — no code/config changes.
- Never fabricate a capability or a `related_to` link; verify against the repo.
- Preserve the six existing feature artifacts as-is.
- Keep summaries factual and one sentence; bodies concise.
