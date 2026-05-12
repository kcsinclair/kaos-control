---
title: Global Agent Work Queue with Rate-Limit Auto-Pause
type: idea
status: planning
lineage: agent-rate-limit-queue
created: "2026-05-12T10:30:00+10:00"
priority: high
labels:
    - agent
    - queue
    - backend
    - frontend
    - operability
release: KC-Release1
---

# Global Agent Work Queue with Rate-Limit Auto-Pause

## Problem

Today, agent runs are kicked off one at a time by clicking each artefact's
launch button. Two consequences:

1. **No batching.** If a user wants to run agents through five backlog
   items, they have to babysit each one, watch it finish, then launch the
   next.

2. **No rate-limit recovery.** If an agent run hits the Claude weekly
   subscription cap or a per-minute rate limit, the run fails and stays
   failed. Anything else the user was planning to launch is blocked until
   they come back hours (or days) later and start it manually.

There's also no protection against the user accidentally launching two
runs that touch the same git repo at the same time — even though the
agent supervisor has a per-lineage lock, nothing prevents two agents
from picking up two different artefacts in the same project and
producing competing commits.

## Vision

A **global FIFO queue** the user can enqueue work into from any artefact
view, plus a dispatcher that:

- Processes one job at a time, server-wide. Strict serial; no two
  agent runs from the queue overlap, across any project.
- Detects rate-limit errors in the Claude Code stream, parses the reset
  time from the error message, pauses the queue, and auto-resumes a few
  minutes after the reset point, requeueing the failed job at the head
  of the queue.
- Survives server restart — queue contents persist to disk.
- Exposes a queue view in the UI: see what's running, what's pending,
  what was just paused and when it'll resume. Pause / resume manually.
  Remove queued items.

The "one at a time" rule serialises every agent that runs through the
queue. It does NOT block manual one-off launches via the existing agent
panel button — those keep their current semantics so power users can
side-step the queue if they need to.

## Sample error evidence

Captured from `~/.kaos-control/data/kaos-control/runs/22de65f53d82bf14.log`:

```json
{"type":"assistant","message":{...,"content":[{"type":"text","text":"You're out of extra usage · resets 8pm (Australia/Brisbane)"}],...},...,"error":"rate_limit"}
```

The reliable signal is the top-level `"error":"rate_limit"` field on the
assistant event. The reset time is in the human-readable text — parseable
with a small grammar (see the requirement for spec).

## Out of scope (for KC-Release1)

- **Per-project sub-queues.** Single global queue ships first; if multi-
  project users complain that one project's queue blocks another's,
  revisit.
- **Smart scheduling / priority weighting.** Strict FIFO at v1; priority
  fields, fairness, or "skip the head if it can't run" can come later.
- **Auto-requeue with retries on non-rate-limit failures.** Failed runs
  stay failed; user manually decides whether to retry.
- **Queue-as-service to external callers.** No public API beyond the
  in-app UI and the same WS hub the rest of the SPA uses.

## Related

- Existing scheduler (`internal/scheduler/`) handles **cron-style** recurring
  jobs — a different surface. This queue is for ad-hoc FIFO work.
- The auth-role-checks-mutations work introduces `requireRole`; this
  feature reuses it for permission gating.
- The per-lineage lock in `internal/lock/` continues to apply at agent
  start time — the queue's serial guarantee is additional, not a
  replacement.
