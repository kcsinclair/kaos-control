---
title: TypeScript React + Node/NestJS (Ecosystem Leader)
type: tech-stack
status: draft
lineage: stack-ts-react-nest
labels:
    - tech-stack
    - catalog
    - backend
    - frontend
    - typescript
    - react
summary: TypeScript end-to-end — React UI + structured NestJS backend on Node. Maximum ecosystem and developer velocity.
---

# TypeScript React + Node/NestJS (Ecosystem Leader)

**Focus:** developer velocity and ecosystem reach. Rapid feature iteration with one language across the stack.

## Overview
Leverages the massive NPM ecosystem. **NestJS** provides a highly structured, Angular-inspired backend on top of Node.js, while **React** offers unparalleled UI component modularity. One language (TypeScript) from database to browser lowers context-switching cost.

## Communication layer
**GraphQL** (via Apollo/Mercurius) for complex data graphs, or **REST** for standard CRUD.

## Data persistence
Any — commonly PostgreSQL or MySQL via an ORM (Prisma, TypeORM), MongoDB for document data, Redis for caching.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Full-stack web (backend + frontend) |
| Languages | TypeScript (front + back) |
| Footprint | Moderate |
| Learning curve | Low |
| Best use case | Standard SaaS, fast-moving product teams |

## Suits these architectures
- [Modular Monolith](../architectures/modular-monolith.md) — NestJS modules map to bounded contexts.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — fast iteration, huge ecosystem.
- [Cloud-Native Microservices](../architectures/cloud-native-microservices.md) — per-service Nest backends.
- [Serverless / FaaS](../architectures/serverless-faas.md) — first-class Node function support.

## Stack profile

See [[agent-directives-generation]]. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: pnpm dev                          # runs the api + web apps (or each app's dev script)
  repo_layout:
    - {path: apps/api/,  note: NestJS backend}
    - {path: apps/web/,  note: React frontend}
    - {path: packages/,  note: shared TypeScript packages}
    - {path: tests/,     note: e2e / integration tests}
  roles:
    backend-developer:
      write_paths: [apps/api, packages]
      build: pnpm --filter api build
      lint: pnpm --filter api lint
      test: pnpm --filter api test
    frontend-developer:
      write_paths: [apps/web, packages]
      build: pnpm --filter web build
      lint: pnpm --filter web lint
      test: pnpm --filter web test
    test-developer:
      write_paths: [tests]
      test: pnpm test:e2e
```
