---
title: Standalone Desktop Application
type: architecture
status: draft
lineage: arch-standalone-desktop
labels:
    - architecture
    - catalog
    - offline-capable
    - low-complexity
related_to:
    - architecture/tech-stacks/tauri.md
    - architecture/tech-stacks/wails.md
    - architecture/tech-stacks/electron.md
    - architecture/tech-stacks/flutter.md
summary: Self-contained app running entirely on the local machine; zero network dependency, data tied to the installation.
---

# Standalone Desktop Application

**Focus:** local processing/storage · zero network dependency · portability vs. data silos.

## Definition
A self-contained software installation that runs directly on a local operating system (Windows, macOS, Linux). All computation and logic run on the host machine's CPU and memory without a requirement for persistent internet connectivity.

## Data strategy
Local persistence — flat files (JSON, XML), an embedded relational DB (SQLite), or a proprietary file format on the user's disk. Data is tied strictly to the specific installation instance.

## Scaling
**Vertical only** — bounded by the host machine's hardware (RAM, CPU, disk). There is no horizontal scaling for a single instance.

## Best fit
High-performance specialised tools: professional video editors (e.g. DaVinci Resolve), CAD software, offline productivity tools — where privacy and local performance are paramount.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | Yes (fully) |
| Collaboration / shared state | No (data silos) |
| Scale | Single user |
| Complexity to build | Low |
| Team skill required | Low–moderate |

## Pros
- Zero network latency; works entirely offline.
- High data privacy (nothing leaves the machine).
- Predictable local resource usage.

## Cons
- Data silos — hard to sync or share.
- Manual deployment/update cycles.
- Hardware-dependent performance; collaboration is difficult to add.

## Suitable tech stacks
- [Tauri](../tech-stacks/tauri.md) — tiny binary, native webview, Rust backend.
- [Wails](../tech-stacks/wails.md) — Go backend + web frontend, low overhead.
- [Electron](../tech-stacks/electron.md) — widest ecosystem, heavier footprint.
- [Flutter Desktop](../tech-stacks/flutter.md) — canvas-rendered, high-FPS UI.

## Related architectures
Add a server and it becomes a [[architecture/architectures/local-web|Local Web Application]]. For a phone-first equivalent see [[architecture/architectures/mobile-native|Mobile-Native]].
