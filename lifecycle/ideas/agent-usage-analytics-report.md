---
title: Agent Usage Analytics Report
type: idea
status: clarifying
lineage: agent-usage-analytics-report
created: "2026-06-12T15:19:52+10:00"
priority: normal
labels:
    - agent
    - runs
    - observability
    - feature
    - frontend
release: KC-Release3
---

# Agent Usage Analytics Report

Add a new **Reports** section (left navigation menu entry) that aggregates all agent run logs and presents a single-page analytics dashboard. The page should surface timing metrics (run duration, queue wait time), cost metrics (token counts, estimated API spend), and token efficiency indicators (tokens per minute, input/output ratios) across all historical runs.

The dashboard should support trend visualisation over time — e.g. average run duration per agent type, cost-per-run trends, and comparative cost/time scatter plots — so that operators can identify degradation or efficiency gains as the system evolves. Filtering by agent, date range, and status (success/failure/truncated) will make the data actionable.

On the backend, a new reporting endpoint should aggregate run records from the SQLite index (or run log storage), returning pre-computed summary statistics. The frontend component should render charts (reusing existing three.js/Cytoscape or adding a lightweight charting library) alongside a sortable summary table.
