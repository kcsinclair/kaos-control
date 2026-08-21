---
title: Flutter (Dart, Canvas-Rendered)
type: tech-stack
status: approved
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

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: flutter run
  repo_layout:
    - {path: lib/,               note: Dart application code (widgets, state, services)}
    - {path: test/,              note: Dart unit/widget tests}
    - {path: integration_test/,  note: integration tests}
  roles:
    backend-developer:
      required: false          # single Dart codebase — no separate backend
    frontend-developer:
      write_paths: [lib]
      build: flutter build apk        # or ios / web / <target>
      lint: flutter analyze
      test: flutter test
    test-developer:
      write_paths: [test, integration_test]
      test: flutter test integration_test
```
