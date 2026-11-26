---
title: Go + Vue (High-Performance Lean Stack)
type: tech-stack
status: approved
lineage: stack-go-vue
labels:
    - tech-stack
    - backend
    - frontend
    - go
    - vue
summary: Go backend (goroutine concurrency) + reactive Vue SPA. Lean, high-throughput, low-latency. kaos-control's own stack.
parent: lifecycle/architecture/tech-stacks/go-vue.md
created: 2026-08-21T10:46:34+10:00
---

# Go + Vue (High-Performance Lean Stack)

**Focus:** performance and leanness. High-concurrency, low-latency web apps and services.

## Overview
Go's extremely efficient concurrency model (goroutines) on the backend, paired with a reactive, lightweight Vue.js frontend. Optimised for high throughput and low latency, small memory footprint, and simple single-binary deployment. *(This is kaos-control's stack: a single Go binary with an embedded Vue SPA.)*

## Communication layer
Primarily **RESTful APIs**, or **gRPC** when internal microservices are involved (gRPC brings HTTP/2 + Protocol Buffers performance).

## Data persistence
Relational databases like **PostgreSQL** for ACID compliance, embedded **SQLite** for single-binary apps, or a **Redis** KV store for caching/sessions.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Full-stack web (backend + frontend) |
| Languages | Go, TypeScript/JS (Vue) |
| Footprint | Low (server-side); tiny single binary |
| Learning curve | Moderate |
| Best use case | Lean high-concurrency web apps & services |

## Suits these architectures
- [Local Web Application](../architectures/local-web.md) — single-binary LAN server.
- [Modular Monolith](../architectures/modular-monolith.md) — lean, fast monolith host.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — cheap to run per instance.
- [Edge / Distributed Hybrid](../architectures/edge-hybrid.md) — small efficient edge agent.
- [Serverless / FaaS](../architectures/serverless-faas.md) — tiny cold-start on Lambda.

## Stack profile

Machine-readable profile consumed by kaos-control to tune `AGENTS.md` (repo layout) and the
`config.yaml` developer-agent prompts (write paths + build/test commands).
`write_paths` are the **stack source roots only** — the generator adds the
constant lifecycle paths (`lifecycle/<stage>-plans`, `architecture/decisions`).

```yaml
stack_profile:
  run: go run ./cmd/<app>            # dev server; build the SPA first (cd web && pnpm build)
  repo_layout:
    - {path: internal/, note: Go packages — backend logic}
    - {path: cmd/,      note: binary entry points}
    - {path: web/src/,  note: Vue 3 + TypeScript SPA source}
    - {path: web/dist/, note: built SPA, embedded into the binary}
    - {path: tests/,    note: integration + e2e tests}
  roles:
    backend-developer:
      write_paths: [internal, cmd]
      build: go build ./...
      lint: go vet ./...
      test: go test ./... -short
    frontend-developer:
      write_paths: [web/src]
      build: cd web && pnpm build
      lint: cd web && pnpm run lint && pnpm exec vue-tsc --noEmit
      test: cd web && pnpm test
    test-developer:
      write_paths: [tests, web/src]
      test: go test -tags integration ./tests/...
```
