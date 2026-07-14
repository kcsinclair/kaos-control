---
title: Modular Monolith
type: architecture
status: draft
lineage: arch-modular-monolith
labels:
    - architecture
    - catalog
    - collaborative
    - low-complexity
    - low-cost-start
related_to:
    - architecture/tech-stacks/go-vue.md
    - architecture/tech-stacks/ts-react-nest.md
    - architecture/tech-stacks/python-fastapi.md
    - architecture/tech-stacks/java-spring-angular.md
    - architecture/tech-stacks/php-symfony-postgres.md
    - architecture/tech-stacks/python-mongodb.md
summary: A single deployable application organised into well-bounded internal modules — the pragmatic default most projects should start with.
---

# Modular Monolith

**Focus:** one deployable unit · strong internal module boundaries · microservices discipline without the distributed cost.

## Definition
A single application, built and deployed as one unit, but internally partitioned into well-defined modules with explicit interfaces (by domain: billing, users, reporting…). It captures most of the *design* benefits of microservices — clear boundaries, separation of concerns — without the network, orchestration, and consistency costs of distributing them. **This is where the majority of projects should start.**

## Data strategy
A single shared database, but with **module-owned schemas/tables** and no cross-module reach-around — modules talk through their public interfaces, so boundaries can later be extracted into services if needed.

## Scaling
**Vertical first**, then **horizontal by running multiple instances** of the whole app behind a load balancer. Genuinely hot modules can be extracted into their own service later (the monolith is the on-ramp to microservices).

## Best fit
Startups and new products, internal business apps, and any team that wants clean boundaries and fast iteration without a platform team. The safe default when scale requirements are still unknown.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | No (server app) |
| Collaboration / shared state | Yes |
| Scale | Low → high (defer the split) |
| Complexity to build | Low |
| Team skill required | Low–moderate |

## Pros
- One codebase, one deploy — simple to build, test, run, and debug.
- Strong module boundaries preserve future optionality (extract to services later).
- Lowest operational cost; no distributed-systems tax up front.

## Cons
- Whole app scales/deploys together (no per-module scaling).
- Boundary discipline is a team habit, not enforced by the network — erosion is easy.
- Very large monoliths can slow build/test cycles.

## Suitable tech stacks
- [Go + Vue](../tech-stacks/go-vue.md) — lean single binary, ideal monolith host.
- [TypeScript React + NestJS](../tech-stacks/ts-react-nest.md) — NestJS modules map directly onto bounded contexts.
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — fast to stand up, great for data-centric apps.
- [Java + Spring Boot / Angular](../tech-stacks/java-spring-angular.md) — enterprise modular monolith with strong DI.
- [PHP + Symfony / PostgreSQL](../tech-stacks/php-symfony-postgres.md) — bundles map cleanly onto modules; ubiquitous hosting.
- [Python + MongoDB](../tech-stacks/python-mongodb.md) — flexible document store for evolving data.

## Related architectures
The natural predecessor to [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]] (extract modules when scale demands). Deployed for one team on a LAN it *is* a [[architecture/architectures/local-web|Local Web Application]]; hosted for many tenants it becomes a [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]].
