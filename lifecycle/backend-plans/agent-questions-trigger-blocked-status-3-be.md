---
title: 'Backend plan: auto-transition artifacts to blocked on open questions'
type: plan-backend
status: in-development
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/requirements/agent-questions-trigger-blocked-status-2.md
---

## Overview

Move the auto-block/unblock logic from the HTTP write handler (`internal/http/write.go:261-279`) down into the indexer layer so it fires on every path that indexes an artifact: API writes, watcher file-change events, and the startup full-scan. Add the reverse auto-unblock path (`blocked → draft`) when the `## Open Questions` section is removed.

Interactions with [[agent-questions-trigger-blocked-status]] frontend plan: the backend emits the same `artifact.indexed` and `feed.new` WebSocket events already consumed by all views, so no new event types are required. The frontend plan covers any UI-side adjustments.

Interactions with [[agent-questions-trigger-blocked-status]] test plan: the test plan covers unit and integration tests for every milestone below.

---

## Milestone 1 — Add a system actor to the workflow engine

**Description:** Introduce a `system` role that is authorised for `any → blocked` and `blocked → draft` transitions. This avoids co-opting the `product-owner` role for machine-initiated changes and keeps audit logs unambiguous.

**Files to change:**

- `internal/workflow/workflow.go` — add two rules:
  - `{from: "", to: "blocked", roles: ["system"]}` (any → blocked)
  - `{from: "blocked", to: "draft", roles: ["system"]}` (blocked → draft)
- `internal/config/config.go` — add `"system"` to the supported-roles list so validation passes.

**Acceptance criteria:**

- [ ] `engine.CanTransition("draft", "blocked", []string{"system"})` returns `true`.
- [ ] `engine.CanTransition("blocked", "draft", []string{"system"})` returns `true`.
- [ ] `engine.CanTransition("draft", "approved", []string{"system"})` returns `false` (system is not a superuser).
- [ ] Existing role-based transitions are unchanged.

---

## Milestone 2 — Implement `applyOpenQuestionTransition` in the index package

**Description:** Create a function in `internal/index/` (or a new file `internal/index/autoblock.go`) that encapsulates the detect → validate → rewrite → re-index → event → broadcast cycle.

```
func (idx *Index) applyOpenQuestionTransition(a *artifact.Artifact, absPath string) error
```

Logic:

1. Call `artifact.HasOpenQuestions(a.Body)`.
2. If `true` and `a.FM.Status != "blocked"`:
   a. Validate `CanTransition(a.FM.Status, "blocked", ["system"])`.
   b. If not permitted, log a warning and return nil.
   c. Patch frontmatter: set `status: blocked`, ensure `assignees` contains `{role: product-owner, who: agent}`.
   d. Write patched file atomically (write to `.tmp`, rename).
   e. Re-parse and `Upsert` the updated artifact.
   f. Insert a `status_changed` event via `InsertEvent`.
   g. Broadcast `artifact.indexed` (with `blocked_reason: "open_questions_detected"`) and `feed.new` via hub.
   h. Log at INFO: path, old status, new status, reason `open_questions_detected`.
3. If `false` and `a.FM.Status == "blocked"`:
   a. Validate `CanTransition("blocked", "draft", ["system"])`.
   b. Patch frontmatter: set `status: draft`.
   c. Atomic write, re-parse, Upsert.
   d. Insert `status_changed` event.
   e. Broadcast events.
   f. Log at INFO: path, old status `blocked`, new status `draft`, reason `open_questions_resolved`.
4. Otherwise, no-op (idempotent).

**Files to change:**

- `internal/index/autoblock.go` (new) — the function above.
- `internal/index/index.go` — add fields for `hub *hub.Hub` and `wf *workflow.Engine` to the `Index` struct; update the constructor to accept them.

**Acceptance criteria:**

- [ ] An artifact with `## Open Questions` content and status `draft` is rewritten to `blocked` on disk and re-indexed.
- [ ] An artifact with status `blocked` and no open questions is rewritten to `draft`.
- [ ] An already-blocked artifact with open questions triggers no disk write and no duplicate event.
- [ ] If the workflow engine rejects the transition, no write occurs and a warning is logged.
- [ ] The on-disk write uses write-then-rename to prevent partial writes.
- [ ] `assignees` includes `{role: product-owner, who: agent}` after auto-block.

---

## Milestone 3 — Wire `applyOpenQuestionTransition` into IndexFile and Scan

**Description:** Call `applyOpenQuestionTransition` after every successful `Upsert` inside `IndexFile`. Also call it during `Scan` for each artifact processed. This ensures auto-block/unblock fires on watcher events, API writes, and server startup.

**Files to change:**

- `internal/index/index.go`
  - In `IndexFile` (around line 295): after the existing `Upsert` call, invoke `applyOpenQuestionTransition`.
  - In `Scan` (around line 240): after each per-file index, invoke `applyOpenQuestionTransition`.

**Acceptance criteria:**

- [ ] A file written externally (e.g. by an agent) that contains `## Open Questions` is auto-blocked within one watcher cycle (~150 ms debounce + processing).
- [ ] On server startup, artifacts with stale status are corrected during the initial scan.
- [ ] The flow does not add measurable latency to the startup scan beyond the existing parse time.

---

## Milestone 4 — Prevent circular re-index triggers

**Description:** The on-disk rewrite from milestone 2 will fire a second `fsnotify` event. The indexer must detect that the content has not meaningfully changed and skip re-processing.

**Approach:** After the atomic write in `applyOpenQuestionTransition`, compute the SHA-256 of the new file content. On the next `IndexFile` call triggered by the watcher, compare the incoming SHA-256 against the stored value in the DB. If they match, skip the `Upsert` and `applyOpenQuestionTransition` entirely.

**Files to change:**

- `internal/index/index.go` — in `IndexFile`, add a content-hash guard before `Upsert`: query the existing `body_sha256` from the DB and compare with the newly parsed artifact's SHA-256. If identical, return early.

**Acceptance criteria:**

- [ ] Writing `## Open Questions` into a file triggers exactly one auto-block cycle, not an infinite loop.
- [ ] The SHA-256 comparison adds negligible overhead (single DB row lookup).

---

## Milestone 5 — Remove redundant auto-block logic from the HTTP handler

**Description:** The existing auto-block code in `internal/http/write.go` (lines 261-279) is now redundant because `IndexFile` (called after the write) will handle it. Remove the handler-level override to avoid double-processing and keep the single source of truth in the indexer.

**Files to change:**

- `internal/http/write.go` — remove the `HasOpenQuestions` check, the status override to `blocked`, and the assignee injection from `handleUpdateArtifact`. The `blocked_reason` payload in the broadcast can also be removed since the indexer now handles it.

**Acceptance criteria:**

- [ ] `PUT /artifacts/*` with a body containing `## Open Questions` still results in a `blocked` artifact (via the indexer path).
- [ ] No duplicate status-change events are emitted for a single API write.
- [ ] Existing API behaviour is preserved from the caller's perspective.

---

## Milestone 6 — Logging and observability

**Description:** Ensure every automatic transition is logged at INFO level with structured fields.

**Files to change:**

- `internal/index/autoblock.go` — add `slog.Info` calls at each transition point with fields: `path`, `old_status`, `new_status`, `reason`.

**Log format (example):**

```
level=INFO msg="auto-transition: open questions detected" path=lifecycle/requirements/login-2.md old_status=draft new_status=blocked reason=open_questions_detected
level=INFO msg="auto-transition: open questions resolved" path=lifecycle/requirements/login-2.md old_status=blocked new_status=draft reason=open_questions_resolved
level=WARN msg="auto-transition: workflow rejected" path=lifecycle/releases/v1.md old_status=done new_status=blocked reason=transition_not_permitted
```

**Acceptance criteria:**

- [ ] Each automatic status change produces exactly one INFO log line.
- [ ] Rejected transitions produce a WARN log line.
- [ ] No log output for idempotent no-ops.
