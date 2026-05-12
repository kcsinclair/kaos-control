---
title: Manual Idea Capture Without AI Assistance
type: idea
status: draft
lineage: manual-idea-capture
created: "2026-05-12T16:44:23+10:00"
priority: normal
labels:
    - feature
    - frontend
    - usability
    - v1
---

# Manual Idea Capture Without AI Assistance

Allow users to create a new idea artifact directly in the UI without invoking any AI assistance. The user should be able to type a raw note or freeform description and save it immediately as a draft idea in the lifecycle system.

This is useful for quickly capturing thoughts before they are lost, without needing to wait for an AI prompt cycle or network round-trip to an agent. The captured note becomes a standard idea artifact with the appropriate frontmatter (title, type, status, lineage) filled in by the system based on user input.

The feature should integrate naturally alongside the existing AI-assisted flow — both paths lead to the same artifact format, but the manual path is instant and always available even if agent services are unavailable or the user simply prefers direct control.
