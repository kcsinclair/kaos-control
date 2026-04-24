# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Current state

This project — working name **kaos-control**, product name **Innovation Maker** (to be finalised) — is in a **pre-code, requirements-driven stage**. There is no source code, no build system, no tests, no package manifests. The entire repository is markdown artifacts organised under `lifecycle/`.

The project is **meta**: it is building a tool that enforces a specific requirements-to-release lifecycle, and the same lifecycle structure is being applied to this repo to produce the tool itself.

## Authoritative spec

Before making any substantive suggestions about scope, design, or implementation, read:

[lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md)

It is the source of truth for product scope, tech stack, workflow states, roles, file format, directory layout, and configuration. Do not restate it here.

Original idea and Q&A history:
- [lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md](lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md)
- [lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md](lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md)

## Lifecycle directory semantics

Artifacts live in stage-named subdirectories under `lifecycle/`. A directory being empty means *"this stage hasn't happened yet"*, not that it's unused.

- `lifecycle/ideas/` — original idea docs. Populated.
- `lifecycle/requirements/` — detailed spec. Populated.
- `lifecycle/backend-plans/`, `lifecycle/frontend-plans/`, `lifecycle/test-plans/` — **empty**, awaiting Opus to generate plans (see next section).
- `lifecycle/dev-plans/`, `lifecycle/tests/`, `lifecycle/prototypes/`, `lifecycle/releases/`, `lifecycle/sprints/` — per the spec, will exist when the relevant stage runs.

Top-level `src/`, `go.mod`, `package.json`, CI configs etc. **do not exist yet** and should not be invented. If the user asks to scaffold code, confirm intent first and follow the planned tech stack below.

## Lineage filename convention

Artifacts for a single idea share a **slug** and carry a **monotonic index** across stages, with optional stage suffix. Example:

```
lifecycle/ideas/login.md            (originating, no suffix)
lifecycle/requirements/login-2.md
lifecycle/backend-plans/login-3-be.md
lifecycle/frontend-plans/login-4-fe.md
lifecycle/test-plans/login-5-test.md
```

Rules:
- First file in a lineage has **no suffix**. Indices start at `-2`.
- Index is monotonic **per lineage, across stages** — never reused.
- Rejected-and-replanned artifacts get the **next** index; superseded files stay in place and in git history.
- Every non-originating artifact has `parent:` in its YAML frontmatter pointing to the previous file.

See §3.3 and §4.4 of the spec for the full rule.

## Current workflow stage

The **next planned action** is to prompt Opus to read the detailed requirements and produce three plan artifacts:

- `lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md`
- `lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md`
- `lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md`

The exact prompt the user intends to run is recorded in [project-notes.md](project-notes.md). After those three plans exist, the user plans to use Sonnet agents to implement code from each.

## Planned tech stack

When code does arrive, per §12 of the spec:

- **Backend**: Go 1.22+, stdlib `net/http`, `go-chi/chi`, `goldmark` (markdown + frontmatter), `gopkg.in/yaml.v3`, `fsnotify`, `nhooyr.io/websocket` or `gorilla/websocket`, `modernc.org/sqlite` (pure-Go, no cgo), `go-git/go-git`.
- **Frontend**: Vite build pipeline, Vue 3 SFCs, 3d-force-graph, Cytoscape.js, `markdown-it`.
- **Distribution**: single Go binary with embedded frontend via `embed.FS`.

Use these choices for any early scaffolding suggestions; do not propose alternatives unless the user asks.

## What's intentionally absent

No README, no CI, no `.gitignore`, no package manifests, no code. These belong to later lifecycle stages and should not be created as a side effect of other work.

## Commit Conventions

- **Plans**: Every git commit must include an updated `plans/PROJECT_PLAN.md` reflecting what changed and the current project state. Update the "Recent Changes" section and any affected "Completed" or "Planned" items before committing.
- **Implementation plans**: When a Claude Code plan file (`~/.claude/plans/*.md`) was used for implementation, copy it into `plans/` with a descriptive name (e.g., `plans/geolite2-country-lookup.md`) and include it in the commit.


