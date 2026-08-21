---
title: Agent usage & cost reports
type: feature
status: approved
lineage: feature-agent-usage-reports
created: "2026-08-21T12:14:00+10:00"
summary: Analytics over agent runs — tokens, cache usage, and USD cost — aggregated from the run history.
function: Reports & analytics
labels:
    - feature
    - reports
    - agent
---

# Agent usage & cost reports

Understand what the agents are costing and doing.

## What it does

- **Aggregated agent usage** from the persisted run history: input / output /
  cache-creation / cache-read tokens and derived USD cost, totalled and broken
  down.
- **Pure aggregation over the live index** — no separate store to keep in sync.
- Reachable at the **Reports** view; API under `/reports/*`.
