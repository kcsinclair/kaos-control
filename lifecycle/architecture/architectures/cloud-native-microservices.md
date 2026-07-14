---
title: Cloud-Native Microservices
type: architecture
status: draft
lineage: arch-cloud-native-microservices
labels:
    - architecture
    - catalog
    - collaborative
    - high-scale
    - high-complexity
related_to:
    - architecture/tech-stacks/go-grpc-microservices.md
    - architecture/tech-stacks/ts-react-nest.md
    - architecture/tech-stacks/java-spring-angular.md
summary: Application built as many small autonomous services, each owning its data, scaled independently in containers.
---

# Cloud-Native Microservices

**Focus:** distributed systems · horizontal scaling via containers · decoupling through APIs.

## Definition
Structures an application as a collection of small, autonomous services modelled around business domains. Services communicate over lightweight protocols (REST, gRPC) and deploy in isolated containers (Docker), typically orchestrated by Kubernetes.

## Data strategy
**Database-per-service** to keep services loosely coupled — each owns its private store, avoiding hidden dependencies through shared schemas. Distributed transactions need patterns like the **Saga pattern** for consistency.

## Scaling
**Horizontal (primary)** — replicate individual services on demand. Scale only the Payment service without touching the User Profile service.

## Best fit
Large-scale SaaS platforms, e-commerce at scale (Amazon, Netflix), and complex enterprise applications with high feature volatility and many teams.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | No |
| Collaboration / shared state | Yes (many teams) |
| Scale | Very high |
| Complexity to build | High |
| Team skill required | High (DevOps + distributed systems) |

## Pros
- High fault isolation; independent deploy/scale per component.
- Technology heterogeneity (each service can use a different language).
- Extreme resilience.

## Cons
- Significant operational complexity.
- Added network latency between services.
- Hard to keep global data consistency; distributed debugging is difficult.

## Suitable tech stacks
- [Go + gRPC Microservices](../tech-stacks/go-grpc-microservices.md) — the high-performance service backbone.
- [TypeScript React + NestJS](../tech-stacks/ts-react-nest.md) — structured per-service backends + UI.
- [Java + Spring Boot / Angular](../tech-stacks/java-spring-angular.md) — enterprise-grade services and tooling.

## Related architectures
Usually the destination a [[architecture/architectures/modular-monolith|Modular Monolith]] evolves into once boundaries and scale justify it. Frequently combined with [[architecture/architectures/event-driven-streaming|Event-Driven / Streaming]]. For a smaller footprint start at [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]].
