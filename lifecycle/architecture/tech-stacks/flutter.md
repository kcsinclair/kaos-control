---
title: Flutter (Dart, Canvas-Rendered)
type: tech-stack
status: draft
lineage: stack-flutter
labels:
    - tech-stack
    - catalog
    - desktop
    - mobile
    - dart
summary: Dart UI toolkit that renders every pixel on its own canvas — one codebase for mobile, desktop, and web with high-FPS, consistent visuals.
---

# Flutter (Dart, Canvas-Rendered)

**Focus:** high-performance UI / canvas rendering, from one codebase across mobile + desktop + web.

## Overview
Rather than using web technologies or a system webview, **Flutter** uses its own rendering engine to draw every pixel on a canvas, written in **Dart**. A single codebase targets iOS, Android, desktop, and web with pixel-consistent results.

## Technical implication
Unparalleled **60/120 FPS** UI performance and highly consistent visuals across platforms — at the cost of a non-native rendering model and Dart as a less-common language.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Mobile + cross-platform desktop UI |
| Languages | Dart |
| Binary / RAM | Moderate |
| Learning curve | Moderate |
| Best use case | Graphics-heavy, animation-rich, multi-platform apps |

## Suits these architectures
- [Mobile-Native](../architectures/mobile-native.md) — one codebase for iOS + Android.
- [Standalone Desktop](../architectures/standalone-desktop.md) — high-FPS desktop UI.
- [Edge / Distributed Hybrid](../architectures/edge-hybrid.md) — UI on edge/handheld devices.
