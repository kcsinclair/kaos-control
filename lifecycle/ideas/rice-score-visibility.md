---
title: Display RICE Scores in More Places
type: idea
status: approved
lineage: rice-score-visibility
created: "2026-08-15T09:49:28+10:00"
priority: normal
labels:
    - feature
    - config
    - ui
    - artifacts
    - frontend
    - ux
release: KC-Release6
---

# Display RICE Scores in More Places

Add a configuration option (e.g. `show_rice_scores: true/false`) that controls whether RICE scores are surfaced throughout the interface. Since RICE scoring provides default values, every artifact that has gone through the scoring workflow will always have a complete score available to display.

When enabled, RICE scores should appear in as many relevant contexts as possible: artifact cards in list views, Kanban columns and cards, the artifact detail/edit view, and any other location where artifact metadata is shown. The score could be displayed as a single composite number or broken into its Reach, Impact, Confidence, and Effort components depending on available space.

This visibility improvement helps teams prioritise work at a glance without needing to open individual artifacts, making the scoring data actionable across the full UI rather than only visible in dedicated scoring views.
