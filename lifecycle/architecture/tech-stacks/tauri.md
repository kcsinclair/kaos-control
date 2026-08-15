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

## Stack profile

See [[agent-directives-generation]]. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: cargo tauri dev
  repo_layout:
    - {path: src-tauri/, note: Rust core (commands, app lifecycle)}
    - {path: src/,       note: web frontend (HTML/TS + framework)}
    - {path: tests/,     note: tests}
  roles:
    backend-developer:
      write_paths: [src-tauri]
      build: cargo build --manifest-path src-tauri/Cargo.toml
      lint: cargo clippy --manifest-path src-tauri/Cargo.toml
      test: cargo test --manifest-path src-tauri/Cargo.toml
    frontend-developer:
      write_paths: [src]
      build: pnpm build
      lint: pnpm lint
      test: pnpm test
    test-developer:
      write_paths: [tests]
      test: cargo test --manifest-path src-tauri/Cargo.toml
```
