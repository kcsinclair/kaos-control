---
title: YAML-Configurable Panels and Cards
type: idea
status: draft
lineage: yaml-configurable-panels-and-cards
created: "2026-05-10T09:23:42+10:00"
priority: medium
labels:
    - frontend
    - feature
    - vue
    - architecture
---

# YAML-Configurable Panels and Cards

Standardise the UI panels and cards used throughout the application by defining their structure, content slots, and layout rules in YAML configuration. This would give a single source of truth for what data each card or panel displays, how it is labelled, and in what order fields appear — making it trivial to adjust the UI without touching component code.

Support at least two card sizes: a compact "small" card suitable for list or grid views with only the most essential fields, and a "large" card for detail or expanded views that surfaces the full field set. Both sizes should be driven by the same YAML definition, with the size variant controlling which fields are visible and how they are laid out.

The YAML config should be composable so that common field groups (e.g. status badge, lineage link, timestamps) can be defined once and referenced across multiple panel types, reducing duplication and keeping visual consistency as new artifact types are added.
