---
title: Documentation for the agent-usage-analytics-report lineage
type: doc
status: done
lineage: agent-usage-analytics-report
created: "2026-06-27T14:47:30+10:00"
priority: normal
parent: lifecycle/requirements/agent-usage-analytics-report-2.md
release: KC-Release3
---

Documentation for the agent-usage-analytics-report lineage.

## Output

`docs/agent-usage-analytics-report.md` — production documentation covering:

- Dashboard walkthrough (filter bar, summary tiles, five charts, per-model table, CSV export)
- Full API reference for `GET /api/p/:project/reports/agent-usage` including all query parameters, response shape with annotated JSON examples, metric definitions, and error responses
- Pricing table and explanation of the input/output cost split
- Backfill command reference (`kaos-control backfill agent-run-metrics`) with flags and sample output
- Data availability section explaining `metrics_unavailable_count` and how incomplete runs are handled

Audience: operators using the dashboard and developers integrating with or extending the feature.
