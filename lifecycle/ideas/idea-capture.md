---
title: Show original idea and enhanced idea
type: idea
status: draft
lineage: idea-capture
priority: normal
labels:
    - feature
    - artifacts
    - ux
    - editor
---

## Raw Idea

## Raw Idea
When an idea is captured, display the raw idea along with the enhanced idea, save them into the markdown with the headings Raw Idea and Idea

## Idea

When a user captures a new idea, the system should preserve and display both the original raw brain-dump and the AI-enhanced version side by side. This ensures no context or nuance from the original input is lost during the enhancement process.

The markdown artifact for each idea should include two clearly delineated sections using the headings `## Raw Idea` and `## Idea`. The `Raw Idea` section contains the verbatim user input, while the `Idea` section contains the structured, enhanced version produced by the capture assistant.

This change improves transparency and traceability, allowing contributors to understand the intent behind an idea even if the enhanced version interpreted it differently. It also supports future tooling that may want to diff or audit how ideas evolve from capture through planning.
