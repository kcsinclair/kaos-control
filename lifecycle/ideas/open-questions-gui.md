---
title: GUI View for Open Questions
type: idea
status: draft
lineage: open-questions-gui
created: "2026-05-19T15:23:33+10:00"
priority: normal
labels:
    - feature
    - frontend
    - usability
    - workflow
    - artifacts
---

# GUI View for Open Questions

Artifacts across the lifecycle often contain open questions embedded in their markdown body — things that need answers before work can proceed. Currently these are invisible unless you open each artifact individually, making it easy to miss blockers or lose track of what still needs resolution.

This idea is to surface all open questions in a dedicated GUI view: a list or board that aggregates questions from across all artifacts, shows which artifact each question belongs to, and lets users answer or dismiss them in place. Questions could be detected by a convention such as lines starting with `?` or a specific frontmatter field, or by a lightweight markdown parsing heuristic.

The view would make the current state of uncertainty in a project immediately visible, helping the team prioritise clarification work and unblock downstream lifecycle stages.
