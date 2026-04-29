---
title: Project Feed
type: requirement
status: draft
lineage: project-feed
parent: lifecycle/ideas/project-feed.md
---

## Problem

Users currently have no single place to see what has happened in a project recently. To understand activity they must manually browse the graph, open individual artifacts, or check agent run history. This makes it easy to miss status transitions, new defects, or completed agent runs — especially when multiple agents are working concurrently. There is no chronological narrative of project events.

## Goals / Non-goals

### Goals

- Provide a chronological activity stream (the "Project Feed") that surfaces all significant lifecycle events in one view.
- Each feed entry must be actionable: clicking it navigates directly to the relevant artifact or agent run detail.
- The feed must update in real time via the existing WebSocket infrastructure (no polling).
- Recent feed history must survive page refreshes by persisting events in the SQLite index.
- The feed must be accessible from the main navigation alongside existing views (Graph, Kanban, etc.).

### Non-goals

- The feed is **not** a notification system — there are no badges, toasts, or push notifications.
- No per-user read/unread tracking or personalised filtering in this iteration.
- No email or external webhook integration.
- The feed does not replace the existing agent run detail panel or artifact editor; it links to them.

## Detailed Requirements

### Functional

#### FR-1: Event taxonomy

The feed must capture and display these event categories:

| Category | Trigger | Key data |
|---|---|---|
| **Status transition** | `artifact.indexed` where status changed | artifact title, old → new status, who/what triggered it |
| **Artifact created** | `artifact.indexed` for a new path | artifact title, type, lineage |
| **Agent run started** | `agent.started` | agent name, target artifact, run ID |
| **Agent run finished** | `agent.finished` | agent name, status (success), artifacts produced |
| **Agent run failed** | `agent.failed` | agent name, status, stderr excerpt |
| **Defect raised** | `artifact.indexed` where type = defect and path is new | defect title, assigned role, parent artifact |
| **Git commit** | `git.committed` | short SHA, message excerpt, agent name |

#### FR-2: Feed entry structure

Each feed entry must contain:
- **Timestamp** — when the event occurred (ISO 8601, displayed as relative time e.g. "3 min ago").
- **Event type icon/badge** — visually distinguishes the category (e.g. status transition vs. agent run).
- **Summary line** — human-readable description (e.g. "`login-2` transitioned from `planning` → `in-development`").
- **Actor** — the role, agent name, or user who caused the event.
- **Navigation target** — clicking the entry routes to the artifact detail or agent run panel.

#### FR-3: Backend persistence

- Add an `events` table to the SQLite index with columns: `id` (INTEGER PRIMARY KEY AUTOINCREMENT), `event_type` (TEXT), `timestamp` (INTEGER, Unix seconds), `actor` (TEXT), `artifact_path` (TEXT, nullable), `run_id` (TEXT, nullable), `summary` (TEXT), `payload_json` (TEXT).
- The table must be created via `ensureEventsTable()` following the same pattern as `ensureAgentRunsTable()` — outside the versioned schema so it survives rebuilds.
- Expose a REST endpoint `GET /api/projects/{project}/feed` returning events in reverse-chronological order.
  - Query parameters: `limit` (default 50, max 200), `before` (event ID for cursor-based pagination), `types` (comma-separated filter, e.g. `types=transition,agent.finished`).
  - Response: JSON array of event objects plus a `next_cursor` field (the smallest event ID in the page, or null).

#### FR-4: Backend event recording

- Events must be inserted into the `events` table at the same points where `hub.Broadcast` is called today, for the event types listed in FR-1.
- Insertion must not block the broadcast; use a buffered channel or perform the insert in the same goroutine that already does the broadcast (no new goroutines per event).
- Events older than 30 days should be pruned on startup (configurable via `feed.retention_days` in project config, default 30).

#### FR-5: Real-time frontend updates

- The Vue SPA must listen for WebSocket events matching the FR-1 taxonomy and prepend new entries to the feed in real time.
- A new event arriving while the feed view is open must appear at the top with a brief highlight animation.
- If the feed view is not active, a small unread-count indicator on the feed nav item is acceptable but not required for this iteration.

#### FR-6: Feed view UI

- Add a new route `/feed` rendered by a `FeedView.vue` component.
- Add a nav entry (icon + label) in the sidebar/header alongside Graph, Kanban, etc.
- The feed must be a vertical, scrollable list ordered newest-first.
- Support infinite scroll: when the user scrolls near the bottom, fetch the next page via the cursor from FR-3.
- Provide a filter bar at the top with toggle chips for event categories (all enabled by default).
- Each entry row must show: icon, relative timestamp, summary line, actor badge. Clicking anywhere on the row navigates to the target.
- Empty state: display a message like "No activity yet" when the events table is empty.

### Non-functional

#### NFR-1: Performance

- The `GET /api/projects/{project}/feed` endpoint must respond in < 50 ms for the default page size (50 events) on a project with up to 10 000 stored events.
- Index the `events` table on `(timestamp DESC)` and `(event_type)`.

#### NFR-2: Storage

- With a 30-day retention and typical project activity (~200 events/day), storage should remain under 5 MB.

#### NFR-3: Accessibility

- Feed entries must be keyboard-navigable (arrow keys to move, Enter to open).
- Use semantic HTML (`<ol>`, `<li>`, `<time>`, ARIA labels on icons).

#### NFR-4: Responsiveness

- The feed view must be usable on viewports ≥ 768 px wide (tablet and above).

## Acceptance Criteria

- [ ] An `events` table exists in the SQLite index and survives schema rebuilds.
- [ ] Status transitions, artifact creations, agent starts/finishes/failures, defect raises, and git commits are recorded as rows in `events`.
- [ ] `GET /api/projects/{project}/feed` returns paginated events in reverse-chronological order with correct `limit`, `before`, and `types` filtering.
- [ ] The `/feed` route renders a scrollable, newest-first activity stream.
- [ ] Each feed entry displays event icon, relative timestamp, summary, and actor.
- [ ] Clicking a feed entry navigates to the correct artifact detail or agent run panel.
- [ ] New events appear in real time via WebSocket without page refresh.
- [ ] Infinite scroll loads additional pages when the user scrolls near the bottom.
- [ ] Filter chips toggle event categories on and off.
- [ ] Events older than the configured retention period are pruned on startup.
- [ ] The feed endpoint responds in < 50 ms for 50 events on a 10 000-row table.
- [ ] Feed entries are keyboard-navigable.
- [ ] Related artifacts: [[project-feed]]

## Open Questions

- Should the feed show lock/unlock events (`lock.acquired`, `lock.released`)? They are noisy but may be useful for debugging contention. Recommend: omit for v1, add later if requested.
- Should `file.changed` events (raw FS changes) be included, or only the higher-level `artifact.indexed` that follows? Recommend: only `artifact.indexed` to avoid duplicate/noisy entries.
- Is 30 days the right default retention, or should it be event-count-based (e.g. keep last 5 000 events)?
