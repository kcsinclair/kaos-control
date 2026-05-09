---
title: Investigate Integration with MemPalace
type: idea
status: approved
lineage: mempalace-integration
created: "2026-05-10T09:19:22+10:00"
priority: normal
labels:
    - integration
    - architecture
    - backend
    - agent
release: KC-TokenEfficiency
---

# Investigate Integration with MemPalace

Investigate integrating MemPalace as a persistent project memory layer for kaos-control. The goal is to give agents and the lifecycle tool a structured, queryable memory store so that context from previous runs, decisions, and artifacts can be retrieved efficiently rather than re-read from disk or re-processed on every invocation.

A key motivation is token efficiency: by offloading long-context recall to MemPalace, agent prompts can stay focused and lean, reducing cost and latency. This may also improve agent coherence across multi-step lifecycle phases by providing a shared memory substrate that survives individual agent runs.

The investigation should assess MemPalace's API and embedding model compatibility, evaluate how project artifacts and agent outputs would be indexed, and propose an architecture for how the kaos-control backend would read and write to MemPalace. A prototype or proof-of-concept scoped to a single agent role (e.g. requirements-analyst) would help validate the approach before broader rollout.
