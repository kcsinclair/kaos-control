---
title: Wails (Go + Native Webview)
type: tech-stack
status: draft
lineage: stack-wails
labels:
    - tech-stack
    - catalog
    - desktop
    - go
summary: Desktop apps with a Go backend and the OS-native webview — Electron-like DX, far lower overhead, no Rust learning curve.
---

# Wails (Go + Native Webview)

**Focus:** efficiency and developer velocity.

## Overview
Like Tauri, Wails uses the OS-native webview rather than bundling Chromium — but the backend is **Go**, and the frontend is any web framework (Vue, React…). It offers the power of Go with the familiarity of modern web development.

## Technical implication
Excellent resource efficiency and very low overhead compared to Electron, **without the steep Rust learning curve** of Tauri — a strong middle ground.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Cross-platform desktop |
| Languages | Go (backend), web (frontend) |
| Binary / RAM | Low |
| Learning curve | Low (Go) |
| Best use case | Fast, efficient desktop apps |

## Suits these architectures
- [Standalone Desktop](../architectures/standalone-desktop.md) — efficient local apps; natural fit for Go teams (and a Go+Vue web app can share frontend code).
