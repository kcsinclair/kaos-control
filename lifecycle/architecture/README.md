# Architecture Catalog

A curated, selectable catalog of **high-level architectures** and the **tech
stacks** that suit them. This folder ships with kaos-control and is copied into
every new project at init, so the tool can help people start a new project:

> **pick an architecture → pick a compatible stack → the project is scaffolded
> with matching config, pipelines, and the standards/ADRs the agents follow.**

See the design in [architecture-templates](../ideas/architecture-templates.md)
and the onboarding UX in
[onboarding-architecture-selection](../ideas/onboarding-architecture-selection.md).

## Layout

```
architecture/
├── README.md                 this index
├── architectures/            one artifact per architecture  (type: architecture)
└── tech-stacks/              one artifact per stack          (type: tech-stack)
```

Each file is a first-class lifecycle artifact with frontmatter, so it is
indexed, graphable, filterable (by `labels`), and selectable. Architecture
files declare their compatible stacks via `related_to:` — those become the
compatibility edges in the graph and the filter for stack selection.

## Architectures

| Architecture | Focus | Typical scale |
| --- | --- | --- |
| [Modular Monolith](architectures/modular-monolith.md) | One deploy, strong internal boundaries — the pragmatic default | Low → high |
| [Local Web Application](architectures/local-web.md) | Central LAN server, shared state | Team |
| [Single-Service Cloud SaaS](architectures/single-service-saas.md) | One cloud web app, multi-tenant | Medium → high |
| [Standalone Desktop](architectures/standalone-desktop.md) | Fully local, offline | Single user |
| [Mobile-Native](architectures/mobile-native.md) | Phone-first client + backend sync | Backend high |
| [Cloud-Native Microservices](architectures/cloud-native-microservices.md) | Many autonomous services | Very high |
| [Event-Driven / Streaming](architectures/event-driven-streaming.md) | Async events, stream processing | Very high (throughput) |
| [Serverless / FaaS](architectures/serverless-faas.md) | Managed functions, scale-to-zero | Elastic |
| [Edge / Distributed Hybrid](architectures/edge-hybrid.md) | Edge + cloud split | High (many devices) |

## Tech stacks

| Stack | Layer | Strength |
| --- | --- | --- |
| [Go + Vue](tech-stacks/go-vue.md) | Full-stack web | Lean, high-concurrency (kaos-control's own) |
| [TypeScript React + NestJS](tech-stacks/ts-react-nest.md) | Full-stack web | Ecosystem & velocity |
| [Python + FastAPI](tech-stacks/python-fastapi.md) | Backend | AI/ML & rapid prototyping |
| [Java + Spring Boot / Angular](tech-stacks/java-spring-angular.md) | Full-stack web | Enterprise robustness |
| [Go + gRPC Microservices](tech-stacks/go-grpc-microservices.md) | Service backbone | High-perf inter-service |
| [Tauri](tech-stacks/tauri.md) | Desktop | Tiny footprint (Rust) |
| [Wails](tech-stacks/wails.md) | Desktop | Efficient (Go) |
| [Electron](tech-stacks/electron.md) | Desktop | Ecosystem breadth |
| [Flutter](tech-stacks/flutter.md) | Mobile + desktop | High-FPS UI, one codebase |

## Notes

- **Filenames** are clean slugs (no lineage `-N` index) — catalog entries are
  standalone reference artifacts, not steps in an idea→release lineage.
- **Not exhaustive.** Contributions welcome; keep one file per entry and follow
  the existing template so entries stay comparable and machine-readable.
