---
title: Roadmap Gantt Period Display Options
type: idea
status: clarifying
lineage: roadmap-gantt-period-options
created: "2026-05-09T16:16:52+10:00"
priority: high
labels:
    - roadmaps
    - frontend
    - enhancement
    - usability
release: KC-Release0
---

# Roadmap Gantt Period Display Options

The Roadmap Gantt view currently displays all time columns regardless of whether they contain any data, resulting in empty columns that waste space and make the chart harder to read. A period display option should be added to give users control over the visible time range.

Two options should be offered: Option 1 is an autoscale mode that automatically fits the Gantt to only the time range spanned by actual items, eliminating all empty leading and trailing columns. Option 2 is a fixed-period mode where the user selects a predefined window — Month, Quarter, Half-Year, or Year — with the chart rendered at that scale and horizontal scrolling enabled when content overflows.

These two modes should be surfaced as a control on the Roadmap Gantt toolbar or filter bar, defaulting to autoscale for a clean out-of-the-box experience while allowing users to switch to a fixed period when they need a stable, predictable time axis.
