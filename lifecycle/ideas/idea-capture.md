---
title: Show original idea and enhanced idea
type: idea
status: draft
lineage: idea-capture
priority: normal
labels:
    - feature
    - frontend
    - artifacts
    - ux
---

## Raw Idea

## Raw Idea
When an idea is captured, display the raw idea along with the enhanced idea, save them into the markdown with the headings Raw Idea and Idea

## Idea

When an idea is captured through the UI, the capture form or confirmation view should display both the original raw brain-dump entered by the user and the AI-enhanced version side by side, so the user can see exactly what was transformed.

The saved markdown file should include both versions under distinct headings: `## Raw Idea` containing the verbatim input, and `## Idea` containing the enhanced, structured content. This preserves provenance and allows future readers (and agents) to understand the original intent alongside the polished version.

This change touches the idea capture flow on both the frontend (display) and the artifact writing logic on the backend (markdown structure), ensuring the raw content is never silently discarded after enhancement.
