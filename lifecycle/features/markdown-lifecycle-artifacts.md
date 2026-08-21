---
title: Markdown lifecycle artifacts
type: feature
status: approved
lineage: feature-markdown-lifecycle-artifacts
created: "2026-08-21T15:00:00+10:00"
summary: Every project artifact is a markdown file with YAML frontmatter on disk, indexed into a queryable, browsable, agent-driveable SQLite cache.
function: Lifecycle & artifacts
labels:
    - feature
    - lifecycle
    - artifacts
related_to:
    - lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md
---

# Markdown lifecycle artifacts

The core idea: every project's work — from initial sketch through to shipped
release — is markdown files on disk with YAML frontmatter, and kaos-control
indexes them into something queryable, browsable, and agent-driveable.

## What it does

- **Markdown-on-disk source of truth.** All artifacts live as `*.md` files
  under `<project>/lifecycle/`. The SQLite index is a cache; disk wins every
  reconciliation.
- **Stage directories.** `ideas/`, `requirements/`, `backend-plans/`,
  `frontend-plans/`, `test-plans/`, `tests/`, `defects/`, `releases/`,
  `sprints/`, `prototypes/`, `devops/` — one directory per artifact stage.
- **Type vocabulary.** `idea`, `requirement`, `plan-backend`, `plan-frontend`,
  `plan-test`, `test`, `prototype`, `defect`, `release`, `sprint`, `feature`,
  the architecture types `architecture`, `tech-stack`, `adr`, plus
  configurable types per project.
- **Frontmatter parser.** Required fields: `title`, `type`, `status`,
  `lineage`. Optional: `priority`, `release`, `parent`, `assignees`,
  `labels`, `created`, `summary`, `function`, `depends_on`, `blocks`,
  `related_to`, and the RICE fields. Parse errors surface in the **Parse
  Errors** view rather than silently breaking.
- **Lineage tracking.** Every artifact in a chain shares a slug and carries a
  monotonic per-lineage index across stages
  (`login.md → login-2.md → login-3-be.md → login-4-fe.md`). Indexes never
  reuse, even after rejection — full history is preserved.
- **Live indexing.** `fsnotify` watcher with 150 ms debounce; external edits
  (e.g. from a text editor or an agent) re-index within milliseconds and
  broadcast `artifact.indexed` to connected clients.
- **Markdown editor.** CodeMirror 6 with frontmatter dropdowns for enum
  fields (status, type, role, who, priority), live preview via
  `markdown-it`, optional line-wrap toggle, and external-change detection
  (warns if the file changed under you).
- **Artifact list.** Filter by stage, status, type, label, priority, release,
  or full-text. Sortable on every column. Paginated. Show / hide done items
  toggle. Hide tests by default.
- **Inline status & priority changes.** Click the status pill in the list to
  transition without opening the editor. Same for priority.
- **Defect creation.** "New defect" button captures stack/log/repro in a
  single submit; auto-routes to the right developer role based on labels.

Reachable at **Artifacts** (list, board, graph); API under `/artifacts`.
