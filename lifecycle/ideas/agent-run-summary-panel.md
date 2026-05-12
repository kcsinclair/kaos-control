---
title: Agent Run Summary Panel with Token Efficiency Metrics
type: idea
status: clarifying
lineage: agent-run-summary-panel
created: "2026-05-12T16:59:39+10:00"
priority: high
labels:
    - agent
    - agent-runner
    - frontend
    - vue
    - feature
---

# Agent Run Summary Panel with Token Efficiency Metrics

When an agent finishes running, parse the final log line (type `result`) and display a scrollable summary box showing what the agent did. Include token usage stats: input tokens, cache creation, cache read, output tokens, cost, and number of turns.

Calculate and display token efficiency metrics as described in `claude-token-usage.md`. The primary metric is cache hit ratio: `cache_read / (cache_read + cache_creation + input)`, expressed as a percentage with a quality label (e.g. excellent ≥ 90%, good ≥ 75%, fair ≥ 50%, poor below that). This gives the user an at-a-glance signal on whether the agent is working efficiently or re-reading large files unnecessarily.

When the user clicks the raw log button, open the full log content in a full-height modal (not a small panel) so long logs are easy to read and scroll through without leaving the current page context.
