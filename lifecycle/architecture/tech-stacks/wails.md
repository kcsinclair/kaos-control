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

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: wails dev
  repo_layout:
    - {path: main.go,       note: Wails app entrypoint (root)}
    - {path: app.go,        note: bound Go methods exposed to the frontend (root)}
    - {path: internal/,     note: backend Go packages}
    - {path: frontend/src/, note: web frontend source}
  roles:
    backend-developer:
      write_paths: [internal]          # plus the root main.go / app.go
      build: go build ./...
      lint: go vet ./...
      test: go test ./...
    frontend-developer:
      write_paths: [frontend/src]
      build: cd frontend && pnpm build
      lint: cd frontend && pnpm lint
      test: cd frontend && pnpm test
    test-developer:
      write_paths: [tests]
      test: go test ./...
```
