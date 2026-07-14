---
title: Tauri (Rust + Native Webview)
type: tech-stack
status: draft
lineage: stack-tauri
labels:
    - tech-stack
    - catalog
    - desktop
    - rust
summary: Desktop apps with a Rust backend and the OS-native webview — tiny binaries (<10MB), minimal RAM.
---

# Tauri (Rust + Native Webview)

**Focus:** lightweight and speed.

## Overview
Unlike Electron, Tauri does not bundle Chromium — it uses the operating system's **native webview** (WebView2 on Windows, WebKit on macOS/Linux). The backend is written in **Rust** for memory safety and performance, while the UI is any web frontend.

## Technical implication
Produces extremely small executables (**<10MB**) with minimal RAM overhead — the leanest of the desktop shells.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Cross-platform desktop |
| Languages | Rust (backend), web (frontend) |
| Binary / RAM | Tiny |
| Learning curve | High (Rust) |
| Best use case | Lightweight utility desktop tools |

## Suits these architectures
- [Standalone Desktop](../architectures/standalone-desktop.md) — minimal-footprint local apps.
