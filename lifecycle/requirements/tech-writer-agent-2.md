---
title: Tech Writer Agent
type: requirement
status: planning
lineage: tech-writer-agent
parent: ideas/tech-writer-agent.md
labels:
    - agent
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Tech Writer Agent

## Problem

The lifecycle today produces working code but no user-facing or operator-facing documentation. Knowledge about features, configuration, and APIs lives only inside code, plan artifacts, and requirements — none of which are written for end-users. Every release ships undocumented.

An earlier draft of this requirement made documentation an *automatic* step inside the main lineage's `in-development` phase. In practice, documentation isn't always wanted (some features are internal-only), isn't always wanted *yet* (rough features ship before docs settle), and the writer's brief is usually different from what a developer or analyst would produce. The author of the work isn't always the right person to scope its documentation.

We need an **explicit, out-of-band request flow**: at any point — including after a feature ships — the product-owner asks for docs, supplies a short brief, and a tech-writer agent produces a documentation artefact. QA reviews. Done.

## Goals / Non-goals

### Goals

- Introduce a **tech-writer** role and a corresponding agent that produces documentation artefacts on demand.
- Documentation production is **explicitly triggered by the product-owner**, not automatically chained after `in-development`.
- Provide a **"Request docs"** button on any artefact view (when `status: done`), and a **top-level "New Docs"** entry that doesn't require a source artefact, so docs requests can also originate cold (e.g. "we should document the install process").
- The PO's brief is captured by the existing `idea-capture` agent under a new `doc-generate` prompt template, producing a structured `type: doc` artefact in `lifecycle/docs/`.
- Documentation artefacts join their source artefact's lineage (when triggered from one) using the standard parent + monotonic-index convention. Docs without a source start their own lineage.
- The docs workflow is a **medium-length pipeline** — `draft → approved → in-development → in-qa → done` — skipping the `clarifying` and `planning` stages that the main feature flow uses.
- QA reviews documentation artefacts when they enter `in-qa` and routes defects back to `tech-writer`.
- The tech-writer agent is scoped to write only in `lifecycle/docs/` and the project-level static-site directory `docs/`.

### Non-goals

- Generating API reference from code comments (godoc, typedoc, etc.) — separate tooling.
- Replacing or duplicating the requirement / plan artefacts.
- Real-time collaborative editing.
- Hosting / publishing the resulting documentation (static-site generation, etc.).
- A separate "docs review" role distinct from QA.

## Detailed Requirements

### Triggers

- **FR1 — "Request docs" button on artefact view.**
  Every artefact view (the main editor view) displays a **Request docs** button when the artefact's status is `done`. Clicking it opens the same modal `idea-capture` uses today, pre-populated with a brief template that includes the source artefact's lineage slug. The PO types a short summary of what documentation is needed and submits. The capture writes a new artefact:
  - `type: doc`
  - `status: draft`
  - `lineage: <source lineage slug>` (inherited from the source artefact)
  - `parent: <source artefact path>`
  - Filename: `lifecycle/docs/<slug>-<N>-doc.md` where `N` is `max(existing index in lineage) + 1` per the standard lineage rules.

- **FR2 — Top-level "New Docs" entry.**
  A "New Docs" item next to the existing "New Idea" affordance (sidebar entry and/or `+ New` menu — see the frontend plan for placement). It opens the same `doc-generate` capture modal, but with no source artefact. The PO supplies a slug and the brief. The capture writes:
  - `type: doc`
  - `status: draft`
  - `lineage: <PO-supplied slug>` (a fresh lineage)
  - `parent:` *(absent — this is an originating artefact)*
  - Filename: `lifecycle/docs/<slug>.md` (no index suffix, per the standard rule for the first artefact in a lineage).

### Capture agent

- **FR3 — New `doc-generate` prompt template on `idea-capture`.**
  The existing `idea-capture` agent (inline driver) gains a second prompt template named `doc-generate`. The template instructs the agent to:
  1. Read the PO's brief.
  2. If a source lineage is supplied, read the parent artefact and any sibling artefacts in the lineage for context.
  3. Write a structured `type: doc` artefact with the body sections defined in FR8.
  4. Set frontmatter `assignees: [{ role: tech-writer, who: agent }]` so the work is queued for the tech-writer.

  No new agent entry is required — the existing `idea-capture` agent is extended with the second template.

### Tech-writer role and agent

- **FR4 — Role.** Add `tech-writer` to the roles list in `lifecycle/config.yaml`. The role is responsible for fleshing out a `draft` documentation artefact (typically a brief written by `idea-capture`) into a complete document.

- **FR5 — Agent.** Configure a new agent in `lifecycle/config.yaml`:
  - `name: tech-writer`
  - `role: [tech-writer]`
  - `driver: claude-code-cli`
  - `model: sonnet` (configurable; per the resolved question)
  - `active_status: in-development`
  - `source_types: [doc]`
  - `allowed_write_paths: [lifecycle/docs, docs]`
  - A prompt template that instructs the agent to read its target `doc` artefact's brief (and any linked source lineage), then expand the brief into a publishable document in *both* `lifecycle/docs/<file>.md` (the lifecycle artefact) and `docs/<corresponding-path>.md` (the static-site-ready output).

### Lifecycle stage and artefact type

- **FR6 — New stage `docs`.** Add `{ name: docs, dir: docs }` to the `stages` list in `lifecycle/config.yaml`. Create `lifecycle/docs/`.

- **FR7 — New artefact type `doc`.** Add `doc` to the type vocabulary in `internal/artifact/artifact.go` (the `KnownTypes` map / equivalent). The indexer treats `doc` like any other type — no schema changes.

- **FR8 — Document body structure.** The tech-writer agent's output must include at minimum:
  - `## Overview` — end-user-perspective summary of the feature.
  - `## Usage` — UI steps / CLI commands / API calls.
  - `## Configuration` — options, defaults, environment variables.
  - `## Examples` — at least one worked example.
  Additional sections at the agent's discretion. Documents produced as briefs by `idea-capture` (status `draft`) may have these sections empty or as TODO placeholders; the tech-writer fills them.

### Workflow

- **FR9 — Stages and transitions.** Documentation artefacts use a shortened workflow distinct from the main feature flow:

  ```
  draft → approved → in-development → in-qa → done
  ```

  - `draft` — `idea-capture` has written the brief.
  - `approved` — the PO has reviewed and accepted the brief; the artefact is ready for the tech-writer.
  - `in-development` — the tech-writer is running, or has finished its run (transitions on agent start, just like other agents).
  - `in-qa` — QA is reviewing.
  - `done` — published.

  The `clarifying` and `planning` stages are skipped. The workflow engine (`internal/workflow/`) must allow these transitions for `type: doc` specifically, or treat documentation as a known special case.

- **FR10 — No `required_plans` gate.** Documentation artefacts do NOT participate in the `required_plans` gating that ticket-type artefacts use. The docs lineage is independent.

### Review

- **FR11 — QA reviews documentation.** When a `doc` artefact transitions to `in-qa`, the QA agent's queue picks it up (existing `source_types` / assignee mechanism). QA verifies:
  - Factual accuracy against the implemented feature.
  - Completeness relative to the source brief and (where applicable) the source lineage's requirement.
  - No placeholder / stub content remaining.
- **FR12 — Documentation defects.** Defects raised by QA against a documentation artefact are filed as `type: defect` in `lifecycle/defects/` with `assignees: [{ role: tech-writer, who: agent }]`. Standard defect flow.

### Non-functional

- **NFR1 — Indexer.** The 3D/2D graph and SQLite indexer must render `doc` nodes and their parent/lineage edges with no work beyond recognising the new type.
- **NFR2 — Timeouts.** Tech-writer agent runs under the same `timeout_minutes` config as other agents.
- **NFR3 — Queue compatibility.** A `doc` artefact in `status: approved` shows the standard "Queue Work" button on its artefact view (routed via `agentForArtifact` → `tech-writer`), so it's enqueueable from the same UI as everything else.

## Acceptance Criteria

- [ ] `tech-writer` role exists in `lifecycle/config.yaml` roles list.
- [ ] `tech-writer` agent is configured with correct `source_types`, `allowed_write_paths`, `active_status`, model, and prompt template.
- [ ] `idea-capture` agent has a new `doc-generate` prompt template alongside its existing one(s).
- [ ] `lifecycle/docs/` directory exists and is listed as a stage in config.
- [ ] `doc` is a recognised artefact type in `KnownTypes` and the workflow engine.
- [ ] Workflow engine permits the medium-length doc-specific transition chain (`draft → approved → in-development → in-qa → done`) for `type: doc`.
- [ ] Clicking **Request docs** on a `done` artefact opens the `doc-generate` capture modal pre-populated with the source lineage and parent. Submitting creates a new artefact in `lifecycle/docs/<slug>-N-doc.md` with the correct frontmatter and lineage index.
- [ ] Clicking the top-level **New Docs** entry opens the same modal with no source, accepting a PO-supplied slug. Submitting creates `lifecycle/docs/<slug>.md` as an originating artefact (no `parent`, no index suffix).
- [ ] After PO transitions a `doc` artefact `draft → approved`, the artefact view shows the **Queue Work** button (or the existing **Run Agent** button works) and clicking it queues a tech-writer run.
- [ ] After the tech-writer run completes, the body contains at least Overview / Usage / Configuration / Examples sections, AND a corresponding markdown file exists under `docs/`.
- [ ] When the doc transitions `in-development → in-qa`, the QA agent's queue/ready count includes the doc artefact.
- [ ] Documentation defects raised by QA route to `tech-writer` on the Kanban board and in the graph.
- [ ] The graph UI renders `doc` nodes and their parent/lineage edges correctly without code changes beyond the new type's colour / label entry.
- [ ] Existing agent workflows (analysts, developers, QA) are unaffected by the addition.
- [ ] [[tech-writer-agent]] lineage is preserved end-to-end through this work itself (i.e. this lineage gains a `doc` artefact via the new flow once shipped).

## Resolved Questions

1. Should the tech-writer agent produce one documentation artifact per lineage, or one per plan type (i.e. separate backend docs, frontend docs, etc.)?

> One document per lineage which is associated to an idea

2. Should documentation be gated -- i.e. must a `doc` artifact exist before `in-development` -> `in-qa` is allowed? Or is it optional and advisory?

> Optional and advisory

3. Should the tech-writer also produce content in a project-level `docs/` directory (e.g. markdown files suitable for a static site generator), or only lifecycle artifacts?

> The tech-writer would be writing project document in docs/ which are markdown files suitable for a static site generator

4. What model should the tech-writer agent use (`opus`, `sonnet`, or configurable)?

> Configurable, sonnet to start with.

5. Lineage relationship — does the docs artefact start its own lineage, or extend the source artefact's lineage?

> Extend the source artefact's lineage. Doc filename follows `lifecycle/docs/<slug>-N-doc.md`, `parent` points at the source. A docs request with no source (top-level "New Docs") starts its own lineage with `lifecycle/docs/<slug>.md`.

6. Workflow shape for the docs lineage — short, medium, or full?

> Medium: `draft → approved → in-development → in-qa → done`. No `clarifying` / `planning`.

7. Where does the trigger live in the UI — per-artefact button, top-level entry, or both?

> Both. Per-artefact **Request docs** button on artefact views with `status: done`, plus a top-level **New Docs** entry for docs that aren't tied to a specific source artefact.

## Resolved Questions

1. **`idea-capture` extension vs new agent.** The current plan extends the existing `idea-capture` agent with a second prompt template named `doc-generate`. Alternative: create a separate `docs-capture` agent. Extending is simpler and matches the user's wording; flagging in case the prompt-template selector at the call site is harder than expected to wire from the new "Request docs" / "New Docs" buttons.

> Lets go with docs-capture agent.

2. **Top-level entry placement.** "New Docs" needs a home. Two candidates:
   - As a sibling of the existing idea-creation entry (likely a sidebar action, depending on where idea-capture lives today).
   - As an item under a new `+ New` dropdown in the header.
   The frontend plan should pick one; both can coexist later if needed.

> On the Dashboard and Artifacts page with New Idea, New Defect add New Docs

3. **QA reviewer assignment.** Today QA picks up work either via `source_types` or via `assignees`. Recommendation: route `doc` artefacts to QA via the same `assignees: [{ role: qa, who: agent }]` mechanism used for defects, set automatically on the `in-development → in-qa` transition. Confirms parity with the defect flow; avoids enlarging the QA agent's `source_types` (which would also drag pre-existing test artefacts into the doc-review scope inadvertently).

> Yes, that works.

4. **`docs/` static-site path layout.** The tech-writer writes both `lifecycle/docs/<slug>.md` (lifecycle artefact, frontmatter-bearing) and `docs/<path>.md` (clean markdown for a static-site generator). What's the mapping? Simplest: `docs/<slug>.md` mirrors the slug. Allow the tech-writer to override via a `doc_path` frontmatter field when nesting is wanted (e.g. `doc_path: agents/queue.md`). Decide in the backend plan.

> Yes, mirrors the slug works.

5. **Source-lineage context size.** When triggered from a `done` artefact, the `doc-generate` template should give the tech-writer the source lineage as context. For lineages with many artefacts (idea, requirement, three plans, defects, tests) that's a lot of input tokens before the writer even starts. Cap to the requirement + the originating idea, or pass everything? Cost vs context trade-off — defer to backend plan with an explicit choice and a config knob if it matters.

> Cap to the requirement + the originating idea

6. **"Done" precondition for Request docs button.** Strict `status: done` only, or also `approved` / `in-qa`? Strict `done` is the most useful default (features that haven't shipped don't need docs yet) but power-users may want to start docs alongside development. Default strict; revisit if it bites.

> When done.
