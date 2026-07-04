---
title: Go + Vue (High-Performance Lean Stack)
type: tech-stack
status: draft
lineage: stack-go-vue
labels:
    - tech-stack
    - catalog
    - backend
    - frontend
    - go
    - vue
summary: Go backend (goroutine concurrency) + reactive Vue SPA. Lean, high-throughput, low-latency. kaos-control's own stack.
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
