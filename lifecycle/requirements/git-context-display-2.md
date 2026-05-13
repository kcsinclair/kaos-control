---
title: Git Context Display in GUI
type: requirement
status: planning
lineage: git-context-display
created: "2026-05-11T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/git-context-display.md
labels:
    - feature
    - frontend
    - backend
    - operability
release: KC-Release1
---

# Git Context Display in GUI

## Problem

Users working in the Innovation Maker GUI have no visibility into the underlying git state of the project repository. When switching between branches for different features or lifecycle stages, they must leave the GUI and check a terminal to confirm which branch is active, whether there are uncommitted changes, or what the last commit was. This context-switching wastes time and increases the risk of editing artifacts on the wrong branch.

## Goals / Non-goals

### Goals

- **G1**: Display the current git branch name persistently in the GUI so users always know which branch they are on.
- **G2**: Show working-tree dirty/clean status so users know whether there are uncommitted changes.
- **G3**: Display the most recent commit summary (short SHA + first-line message) for quick orientation.
- **G4**: Keep the displayed information current — update it automatically when the repository state changes (commits, checkouts, file edits).
- **G5**: Require no additional configuration from the user; the feature works out of the box for any registered project that is a git repository.

### Non-goals

- **NG1**: This feature does not provide any git *operations* (commit, push, pull, checkout) from the GUI — it is read-only context display.
- **NG2**: No display of full commit history or diff views; this is a summary indicator, not a git log viewer.
- **NG3**: No support for non-git version control systems.
- **NG4**: No display of remote tracking state (ahead/behind counts) — this may be a future enhancement.

## Detailed Requirements

### Functional Requirements

**FR1 — Backend endpoint**
A new REST endpoint `GET /api/p/{project}/git/status` shall return a JSON object containing:

| Field | Type | Description |
|---|---|---|
| `branch` | `string` | Short name of the current branch (e.g. `main`, `feature/login`) |
| `dirty` | `bool` | `true` if the working tree has modified, added, or untracked files |
| `head_sha` | `string` | Abbreviated (7-char) SHA of the HEAD commit |
| `head_message` | `string` | First line of the HEAD commit message |
| `head_author` | `string` | Author name of the HEAD commit |
| `head_when` | `string` | ISO 8601 timestamp of the HEAD commit |

If the project directory is not a git repository, the endpoint shall return HTTP 200 with `{ "available": false }` and omit the other fields. This allows the frontend to gracefully degrade.

**FR2 — Real-time updates via WebSocket**
When the repository's git state changes (detected via the existing `fsnotify` watcher or after API-driven commits), the backend shall broadcast a WebSocket event:

```json
{
  "type": "git.status",
  "project": "<project-slug>",
  "data": { /* same shape as FR1 response */ }
}
```

At minimum, a `git.status` event must be emitted:
- After any commit made via the `AddAndCommit` API.
- When the fsnotify watcher detects changes to `.git/HEAD` or `.git/index`.

**FR3 — Frontend status bar component**
A persistent `GitStatusBar` component shall be rendered in the application layout (e.g. the top header bar or a dedicated status strip at the bottom of the viewport). It shall display:

- The branch name, visually distinct (e.g. monospace or with a branch icon).
- A dirty/clean indicator (e.g. a dot, colour change, or label such as "modified").
- The abbreviated HEAD SHA and first-line commit message.

The component shall:
- Fetch `GET /api/p/{project}/git/status` on mount and whenever the active project changes.
- Subscribe to `git.status` WebSocket events for live updates.
- Show a neutral/empty state when `available` is `false` (project is not a git repo).

**FR4 — Graceful handling of non-git projects**
If a project is not backed by a git repository, the git status bar shall either be hidden or show a non-intrusive "no repository" state. It must not produce errors or blank/broken UI.

### Non-functional Requirements

**NFR1 — Performance**
The `GET /api/p/{project}/git/status` endpoint shall respond in under 100 ms for repositories with up to 100,000 commits. It reads only HEAD and working-tree status; it must not walk full history.

**NFR2 — No new dependencies**
The implementation shall use the existing `internal/git` package (which wraps `go-git`) and the existing WebSocket hub. No new Go or JS dependencies are required.

**NFR3 — Minimal footprint**
The backend shall not poll git state on a timer. Updates are event-driven: triggered by file-system events or API-driven commits.

**NFR4 — Accessibility**
The status bar component must be keyboard-navigable and use appropriate ARIA attributes so screen readers can announce the branch name and status.

## Acceptance Criteria

- [ ] `GET /api/p/{project}/git/status` returns correct branch, dirty flag, and HEAD commit info for a git-backed project.
- [ ] `GET /api/p/{project}/git/status` returns `{ "available": false }` for a project directory that is not a git repository.
- [ ] A `git.status` WebSocket event is broadcast after a commit via `AddAndCommit`.
- [ ] A `git.status` WebSocket event is broadcast when `.git/HEAD` changes (e.g. branch checkout).
- [ ] The `GitStatusBar` component renders branch name, dirty indicator, and last commit summary.
- [ ] The component updates in real time when a WebSocket `git.status` event arrives.
- [ ] The component shows a graceful empty/hidden state for non-git projects.
- [ ] The endpoint responds in under 100 ms on a repository with realistic history.
- [ ] No new Go or npm dependencies are introduced.
- [ ] Related: [[git-context-display]]

## Resolved Questions

1. **Placement**: Should the git status bar be in the top header, a bottom status strip, or both? The idea suggests "status bar or header" — a specific decision is needed during design/planning.

> In the left menu bar under the application menu options include a panel for the git information.

2. **Dirty-file count**: Should the dirty indicator show the number of modified files, or just a boolean dirty/clean flag? Showing a count is marginally more useful but adds complexity to the endpoint.

> boolean for now.

3. **Branch icon**: Should the branch name use a git-branch icon from the existing lucide icon set, or plain text with monospace styling?

> git-branch icon is good.
