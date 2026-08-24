---
title: Tech-Writer Feature Extraction and Maintenance from Ideas
type: idea
status: draft
lineage: tech-writer-feature-extraction
created: "2026-08-24T11:44:32+10:00"
priority: normal
labels:
    - tech-writer
    - features
    - agent
    - lifecycle
    - enhancement
---

# Tech-Writer Feature Extraction and Maintenance from Ideas

The tech-writer agent currently produces documentation from ideas, but lacks the ability to identify and manage feature records as part of that process. This idea extends the tech-writer's responsibilities to include extracting new features from idea artifacts and updating existing feature records when ideas introduce changes or enhancements to already-catalogued functionality.

When the tech-writer processes an idea, it should determine whether the idea introduces a net-new feature (requiring a new `type: feature` artifact under `lifecycle/features/`) or constitutes an update to an existing feature (requiring the existing artifact to be amended or superseded). This distinction requires the agent to cross-reference the existing feature catalog before writing, applying consistent naming and lineage conventions.

The outcome is a continuously maintained feature catalog that stays in sync with the idea and requirements pipeline, giving product owners and reviewers an accurate, up-to-date picture of what the product does — derived directly from the lifecycle artifacts rather than maintained as a separate manual exercise.
