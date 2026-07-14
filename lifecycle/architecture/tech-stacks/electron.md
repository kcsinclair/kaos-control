---
title: Electron (Node + Chromium)
type: tech-stack
status: draft
lineage: stack-electron
labels:
    - tech-stack
    - catalog
    - desktop
    - typescript
summary: Desktop apps bundling Chromium + Node.js — the industry standard (VS Code, Slack); richest ecosystem, heaviest footprint.
---

# Electron (Node + Chromium)

**Focus:** feature completeness and developer velocity.

## Overview
Bundles a complete Chromium browser and a Node.js runtime with every application — the industry standard for desktop-web hybrids like **VS Code** and **Slack**. Any web frontend works, and the full Node ecosystem is available on the backend.

## Technical implication
The widest range of ready-made components and the smoothest web-developer on-ramp, at the cost of significant **RAM and binary-size bloat** (a Chromium copy per app).

## Profile
| Trait | Value |
| --- | --- |
| Layer | Cross-platform desktop |
| Languages | JS/TS (Node), web (frontend) |
| Binary / RAM | Massive |
| Learning curve | Low (web) |
| Best use case | Feature-rich, complex desktop apps |

## Suits these architectures
- [Standalone Desktop](../architectures/standalone-desktop.md) — when ecosystem breadth and web-dev velocity outweigh footprint.
