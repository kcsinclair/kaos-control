---
title: Git integration
type: feature
status: approved
lineage: feature-git-integration
created: "2026-08-21T15:15:00+10:00"
summary: Every artifact write and agent run produces a structured git commit, with per-artifact history and first-commit-date backfill.
function: Git
labels:
    - feature
    - git
related_to:
    - lifecycle/ideas/git-context-display.md
    - lifecycle/requirements/git-context-display-2.md
---

# Git integration

The lifecycle's audit trail is the git history, not a side table.

## What it does

- **Auto-commit on every write.** Each artifact create / edit / transition /
  agent run produces a structured git commit
  (`transition(<lineage>): <from> → <to>`, `agent(<name>): run <id>
  [<status>]`, etc.).
- **Identity per actor.** Agents commit under their configured
  `git_identity`; user actions commit under the logged-in user's email.
- **History API.** `GET /artifacts/.../history` returns the commit chain for
  any artifact.
- **First-commit-date backfill.** Artifacts that lack `created:` frontmatter
  get a `created` value derived from git history; cached in the index.

Reachable via the history panel on any artifact editor.
