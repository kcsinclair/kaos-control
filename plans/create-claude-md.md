# Plan: Create CLAUDE.md for kaos-control

## Context

`kaos-control` (working name, also known as "Innovation Maker") is the user's new project. A CLAUDE.md does not yet exist. The project has a single git commit, no code, no build tooling, no dependencies — it is currently in a **requirements-and-planning stage**, captured entirely in markdown files organised under a `lifecycle/` directory tree.

The repository is meta in an important way: the project is **building a tool that enforces a specific requirements-to-release lifecycle**, and the user is **applying that same lifecycle to the repo itself** to produce the tool. The `lifecycle/` directory structure and the indexed-filename scheme (`<slug>-<N>-<suffix>.md`) described in the spec are already live in this repo.

A future Claude Code instance opening this repo needs to understand three non-obvious things quickly:
1. There is no code yet — don't go looking for `src/`, `go.mod`, tests, or a build.
2. The authoritative spec lives at a specific long path inside `lifecycle/requirements/`, and the lineage naming convention matters.
3. The user's planned next step is prompting Opus to produce three plan artifacts (BE, FE, test) in the empty `lifecycle/{backend,frontend,test}-plans/` directories.

Without CLAUDE.md, a future instance will waste turns re-discovering the state and may violate the lineage conventions when creating new artifacts.

## Approach

Create a single file at the repo root: `CLAUDE.md`. Keep it compact (target under 100 lines) — this repo has very little to say about "build/test/lint" because none of that exists yet. Focus on the architecture of **documents and workflow**, not code.

### Content outline

1. **Mandatory prefix** (verbatim from /init instructions).

2. **Current state** — one short paragraph: pre-code, requirements-driven, meta-project. Names the two working titles (kaos-control, Innovation Maker).

3. **Authoritative spec** — point at `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md` as the source of truth for product scope, tech stack, and conventions. Tell future Claude to read it before making substantive suggestions.

4. **Directory semantics** — brief description of what each `lifecycle/` subdirectory holds, noting which are populated and which are awaiting the next step. Explicitly call out that the top-level `src/` and similar don't yet exist and should not be invented.

5. **Lineage naming convention** — the rule that matters: `<slug>.md` → `<slug>-2.md` → `<slug>-3-be.md` etc., monotonic index per lineage, stage suffix for plan splits. Reference §3.3 and §4.4 of the detailed spec for the full rule. This is the convention most likely to be violated by a Claude instance creating new artifacts.

6. **Current workflow stage and next prompt** — note that the next planned action is generating three plan artifacts via Opus, per the prompt stored in `project-notes.md`. This helps future Claude understand *why* empty plan directories exist.

7. **Planned tech stack** — one-liner noting that when code arrives it will be Go (chi + stdlib net/http + go-git + fsnotify + modernc.org/sqlite + goldmark) + Vite + Vue 3 + 3d-force-graph + Cytoscape.js. Reference §12 of the spec for details. This steers any early scaffolding suggestions.

8. **What's intentionally absent** — no README (yet), no CI, no package manifests, no code. Don't propose creating these unless the user asks; they belong to later lifecycle stages.

### Content deliberately excluded

- Any "how to build" / "how to test" section — nothing to build or test yet.
- Generic developer practices (per /init rules).
- Full file trees or re-statement of content that's already in the spec — link instead.
- Role/state-machine details — they live in the spec; CLAUDE.md just points there.
- Anything about Cursor/Copilot — none exist in this repo.

## Critical files

- **Create**: `CLAUDE.md` at the repo root (`/Users/keith/Library/Mobile Documents/com~apple~CloudDocs/Projects/kaos-control/CLAUDE.md`).
- **Reference from CLAUDE.md** (do not modify):
  - `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md` — authoritative spec.
  - `project-notes.md` — user's workflow log, including the next-step Opus prompt.
  - `lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md` — original idea.

## Verification

After writing `CLAUDE.md`:

1. **Read it back** and check it passes the /init rules: prefix present; no generic practices; no made-up sections; nothing duplicating the spec; no invented file trees.
2. **Mental test**: if a fresh Claude instance opens this repo and reads only CLAUDE.md, can it answer "what stage is this project in, and what should I do if the user asks me to start writing code?" — the answer should be "read the spec, check project-notes.md for the next prompt, and confirm with the user before scaffolding, because planned tech stack is Go + Vue."
3. **Length check**: target < 100 lines. If longer, prune — CLAUDE.md should be scannable, not comprehensive.
4. **No new files beyond CLAUDE.md**. Do not create `.gitignore`, `README.md`, or scaffolding in this step.
