---
title: "Git Context Display — Backend Plan"
type: plan-backend
status: in-development
lineage: git-context-display
parent: lifecycle/requirements/git-context-display-2.md
---

# Git Context Display — Backend Plan

This plan covers the REST endpoint and WebSocket event broadcasting needed to expose git repository state to the frontend. The implementation builds on the existing `internal/git` package (`go-git` wrapper) and the `internal/hub` WebSocket broadcast hub — no new dependencies are required.

## Milestone 1: Git Status Query Method

**Description:** Add a method to `internal/git.Repo` that returns the working-tree status summary needed by the endpoint: current branch, dirty flag, and HEAD commit metadata.

**Files to change:**
- `internal/git/git.go` — add `type StatusSummary struct` and `func (repo *Repo) Status() (*StatusSummary, error)`.

**Details:**

```go
type StatusSummary struct {
    Branch      string `json:"branch"`
    Dirty       bool   `json:"dirty"`
    HeadSHA     string `json:"head_sha"`
    HeadMessage string `json:"head_message"`
    HeadAuthor  string `json:"head_author"`
    HeadWhen    string `json:"head_when"` // ISO 8601
}
```

The method must:
1. Call `repo.r.Head()` to get the branch ref and HEAD hash.
2. Resolve the commit object for the HEAD hash to extract SHA (abbreviated to 7 chars), first-line message, author name, and author timestamp (formatted as ISO 8601).
3. Call `repo.r.Worktree().Status()` and check if any entry has a non-unmodified state to derive the `dirty` boolean. Note: the existing `ModifiedFiles` method filters by allowed paths — the new method should check the *entire* worktree without path filtering, matching FR1's definition.

**Acceptance criteria:**
- [ ] `StatusSummary.Branch` returns the short branch name (e.g. `main`, `feature/login`).
- [ ] `StatusSummary.Dirty` is `true` when any file is modified, added, or untracked in the working tree.
- [ ] `StatusSummary.HeadSHA` is exactly 7 characters.
- [ ] `StatusSummary.HeadMessage` contains only the first line of the commit message.
- [ ] `StatusSummary.HeadWhen` is a valid ISO 8601 timestamp.
- [ ] Method completes in under 100 ms for a repository with realistic history (NFR1) — it must not walk commit history.

## Milestone 2: REST Endpoint

**Description:** Add `GET /api/p/{project}/git/status` that returns the git status JSON or a graceful `{ "available": false }` response for non-git projects.

**Files to change:**
- `internal/http/git_status.go` (new) — handler `handleGetGitStatus` that reads `project.Git`, calls `Status()`, and serialises JSON.
- `internal/http/server.go` — register route `r.Get("/git/status", s.handleGetGitStatus)` inside the `/p/{project}` sub-router, alongside the existing routes (e.g. near the dashboard block).

**Details:**

The handler reads the `*project.Project` from request context (via the existing `projectFromCtx` helper). If `p.Git == nil`, return HTTP 200 with:
```json
{ "available": false }
```

Otherwise call `p.Git.Status()`, wrap in a response envelope with `"available": true`, and return HTTP 200:
```json
{
  "available": true,
  "branch": "main",
  "dirty": false,
  "head_sha": "a1b2c3d",
  "head_message": "fix: resolve login redirect",
  "head_author": "Alice",
  "head_when": "2026-05-13T10:30:00+10:00"
}
```

**Acceptance criteria:**
- [ ] `GET /api/p/{project}/git/status` returns HTTP 200 with correct branch, dirty flag, and HEAD commit info for a git-backed project.
- [ ] Returns `{ "available": false }` for a project whose directory is not a git repository (FR4).
- [ ] Response time < 100 ms on a repository with up to 100,000 commits (NFR1).
- [ ] No new Go dependencies introduced (NFR2).

## Milestone 3: WebSocket Event After API Commits

**Description:** Broadcast a `git.status` WebSocket event after every successful `AddAndCommit` call made through the API, so the [[git-context-display]] frontend component receives live updates.

**Files to change:**
- `internal/http/write.go` (or whichever handler calls `AddAndCommit`) — after a successful commit, call `p.Git.Status()` and broadcast via `p.Hub.Broadcast(hub.Event{Type: "git.status", Payload: statusSummary})`.

**Details:**

Locate all call sites where `p.Git.AddAndCommit(...)` is invoked in the HTTP handlers. After a successful commit, insert:

```go
if summary, err := p.Git.Status(); err == nil {
    p.Hub.Broadcast(hub.Event{Type: "git.status", Payload: summary})
}
```

This is a minimal, event-driven approach — no polling (NFR3).

**Acceptance criteria:**
- [ ] A `git.status` WebSocket event is broadcast after a commit via `AddAndCommit` (FR2).
- [ ] The event payload matches the `StatusSummary` struct shape.
- [ ] No timer-based polling is introduced (NFR3).

## Milestone 4: WebSocket Event on .git/HEAD and .git/index Changes

**Description:** Extend the file-system watcher to detect changes to `.git/HEAD` and `.git/index`, triggering a `git.status` WebSocket broadcast when the repository state changes externally (e.g. branch checkout from terminal).

**Files to change:**
- `internal/watcher/watcher.go` — add `.git/HEAD` and `.git/index` to the watched paths. When either file changes, call through to a git-status broadcast callback rather than the artifact-indexing path.
- `internal/project/project.go` — wire the git-status broadcast into the watcher (or pass a callback/reference to the `Repo` and `Hub` so the watcher can trigger it).

**Details:**

The current watcher only processes files under `lifecycle/` that end in `.md`. To support git-state events:

1. In `New()` or `Start()`, add explicit watches on `filepath.Join(projectRoot, ".git", "HEAD")` and `filepath.Join(projectRoot, ".git", "index")` (fsnotify can watch individual files).
2. In the event loop, when the changed path matches either `.git/HEAD` or `.git/index`, bypass `handleChange` (artifact indexing) and instead call a new `handleGitChange` method that:
   - Obtains the `*git.Repo` (passed in at construction or via callback).
   - Calls `repo.Status()`.
   - Broadcasts `hub.Event{Type: "git.status", Payload: summary}` via the hub.
3. Apply the same 150 ms debounce used for artifact changes to prevent rapid-fire events during rebases or merges.

The watcher will need access to the `*git.Repo` — either pass it as a constructor parameter to `New()`, or pass a callback `func()` that the project wires up to perform the status + broadcast. The callback approach keeps the watcher decoupled from the git package.

**Acceptance criteria:**
- [ ] A `git.status` WebSocket event is broadcast when `.git/HEAD` changes (e.g. branch checkout) (FR2).
- [ ] A `git.status` WebSocket event is broadcast when `.git/index` changes (e.g. `git add`).
- [ ] Events are debounced (150 ms) to avoid flooding during multi-file operations.
- [ ] The watcher does not poll; it reacts to fsnotify events only (NFR3).
- [ ] No new Go dependencies (NFR2).
