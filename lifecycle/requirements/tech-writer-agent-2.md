---
title: Tech Writer Agent
type: requirement
status: draft
lineage: tech-writer-agent
parent: ideas/tech-writer-agent.md
labels:
    - agent
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

# Tech Writer Agent

## Problem

The current lifecycle produces working code but no user-facing or developer-facing documentation. Knowledge about features, configuration, and APIs lives only inside code, plan artifacts, and requirements -- none of which are written for end-users or operators. Without a dedicated documentation step, every release ships undocumented, and QA has no documentation artifacts to verify for accuracy or completeness.

## Goals / Non-goals

### Goals

- Introduce a **tech-writer** role and a corresponding agent that produces documentation artifacts after backend and frontend development is complete.
- Add a new lifecycle stage directory (`lifecycle/docs/`) to hold documentation artifacts.
- Add a new artifact type `doc` to the type vocabulary.
- Ensure documentation artifacts participate in the existing lineage chain (indexed, parented, linked).
- QA must review documentation artifacts for accuracy, completeness, and consistency with the implemented feature, alongside its existing test-review responsibilities.
- The tech-writer agent should be scoped to write only in `lifecycle/docs/` and a project-level documentation output directory (e.g. `docs/`).

### Non-goals

- Generating API reference docs from code comments (e.g. godoc, typedoc) -- that is a separate tooling concern.
- Replacing or duplicating the requirement or plan artifacts -- documentation targets end-users/operators, not developers.
- Real-time collaborative editing of documentation.
- Hosting or publishing documentation (static-site generation, etc.).

## Detailed Requirements

### Functional

1. **New role: `tech-writer`**
   - Add `tech-writer` to the roles list in `lifecycle/config.yaml`.
   - The role is responsible for reading completed backend plans, frontend plans, and requirement artifacts, then producing documentation artifacts.

2. **New agent: `tech-writer`**
   - Configure a new agent entry in `lifecycle/config.yaml` with:
     - `role: [tech-writer]`
     - `active_status: in-development` (activated alongside or after developer agents complete)
     - `allowed_write_paths: [lifecycle/docs, docs]`
     - A prompt template that instructs the agent to read the requirement and plan artifacts for a lineage, then produce a documentation artifact covering user-facing usage, configuration, and behaviour.
   - The agent must follow the lineage filename convention: `lifecycle/docs/<slug>-<N>-doc.md`.

3. **New lifecycle stage: `docs`**
   - Add a stage entry `{ name: docs, dir: docs }` to the `stages` list in `lifecycle/config.yaml`.
   - Create the `lifecycle/docs/` directory.

4. **New artifact type: `doc`**
   - Add `doc` to the type vocabulary recognised by the indexer and workflow engine.
   - Documentation artifacts use the standard frontmatter (`title`, `type: doc`, `status`, `lineage`, `parent`) and follow the same lineage indexing rules as all other artifacts.

5. **Documentation artifact structure**
   - Each documentation artifact body must include at minimum:
     - `## Overview` -- what the feature does, from an end-user perspective.
     - `## Usage` -- how to use the feature (UI steps, CLI commands, API calls as appropriate).
     - `## Configuration` -- any configurable options, defaults, and environment variables.
     - `## Examples` -- at least one worked example.
   - Additional sections are permitted at the agent's discretion.

6. **QA review of documentation**
   - The QA agent's scope must be extended so that when a lineage enters `in-qa`, the QA agent also reviews the associated `doc` artifact(s).
   - QA should verify: factual accuracy against the implemented code, completeness relative to acceptance criteria in the requirement, and absence of placeholder/stub content.
   - Documentation defects are filed as `type: defect` artifacts in `lifecycle/defects/` with `assignees` routed to `tech-writer`.

7. **Workflow integration**
   - Documentation production occurs during the `in-development` status, after backend and frontend code is committed.
   - The `required_plans` or an equivalent gating mechanism should optionally allow projects to require a `doc` artifact before a lineage can transition from `in-development` to `in-qa`.

### Non-functional

1. The tech-writer agent must complete within the same timeout constraints as other agents (configurable `timeout_minutes`).
2. Documentation artifacts must be indexed by the existing SQLite indexer with no schema changes beyond recognising the new type.
3. The 3D/2D graph must render `doc` artifacts and their lineage edges without additional frontend work beyond recognising the new type in the node/edge renderer.

## Acceptance Criteria

- [ ] `tech-writer` role exists in `lifecycle/config.yaml` roles list.
- [ ] `tech-writer` agent is configured in `lifecycle/config.yaml` with correct `allowed_write_paths`, `active_status`, and prompt template.
- [ ] `lifecycle/docs/` directory exists and is listed as a stage in config.
- [ ] `doc` is a recognised artifact type in the indexer and workflow engine.
- [ ] Running the tech-writer agent against a lineage with completed plans produces a valid documentation artifact in `lifecycle/docs/` with correct frontmatter and lineage index.
- [ ] The documentation artifact body contains at least: Overview, Usage, Configuration, and Examples sections.
- [ ] The QA agent reviews `doc` artifacts during `in-qa` and can raise defects assigned to `tech-writer`.
- [ ] Documentation defects routed to `tech-writer` appear correctly on the Kanban board and in the graph.
- [ ] The graph UI renders `doc` nodes and their parent/lineage edges correctly.
- [ ] Existing agent workflows (analyst, developers, QA) are unaffected by the addition.
- [ ] [[tech-writer-agent]] lineage is preserved end-to-end.

## Resolved Questions

1. Should the tech-writer agent produce one documentation artifact per lineage, or one per plan type (i.e. separate backend docs, frontend docs, etc.)?

> One document per lineage which is associated to an idea

2. Should documentation be gated -- i.e. must a `doc` artifact exist before `in-development` -> `in-qa` is allowed? Or is it optional and advisory?

> Optional and advisory

3. Should the tech-writer also produce content in a project-level `docs/` directory (e.g. markdown files suitable for a static site generator), or only lifecycle artifacts?

> The tech-writer would be writing project document in docs/ which are markdown files suitable for a static site generator

4. What model should the tech-writer agent use (`opus`, `sonnet`, or configurable)?

> Configurable, sonnet to start with.
