---
title: Agent picker list overflows the viewport with many agents — lower options unreachable
type: defect
status: draft
lineage: agent-picker-overflow-unreachable-options
created: "2026-08-11T00:00:00+10:00"
priority: medium
labels:
    - defect
    - ui
    - agents
    - frontend
assignees:
    - role: frontend-developer
      who: agent
---

# Agent picker list overflows the viewport with many agents — lower options unreachable

## Source

GitHub issue [#15](https://github.com/kcsinclair/kaos-control/issues/15) —
reported by **aburow**, 2026-06-21.

## Summary

When many agents/roles are configured, the agent-selection list renders as one
long column that extends past the bottom of the screen. The lower entries (in
the report: the `ollama-*` agents at the end of a ~20-agent list) — and
presumably any confirm/action control below the list — are cut off by the
bottom of the viewport and cannot be reached, so the user is **unable to
progress**.

## Steps to reproduce

1. Configure a large agent roster (the reporter had claude-code-cli, inline,
   codex-cli, and ollama agents — ~20 entries).
2. Open the agent picker.
3. The list exceeds the viewport height; entries below the fold and the action
   control are inaccessible (no internal scroll).

## Likely root cause

The agent-picker list/menu has no bounded height or internal scroll — it grows
to the number of agents and overflows the viewport instead of capping its height
and scrolling within. (Frontend: the agent-picker component in `web/src`; needs
a `max-height` + `overflow-y: auto`, and the action button kept visible/sticky.)

## Expected

The picker constrains its height to the viewport and scrolls internally so every
agent is reachable, with the confirm/action control always visible.
