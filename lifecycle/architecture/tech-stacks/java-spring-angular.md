---
title: Java + Spring Boot / Angular (Enterprise Heavyweight)
type: tech-stack
status: approved
lineage: stack-java-spring-angular
labels:
    - tech-stack
    - catalog
    - backend
    - frontend
    - java
    - angular
summary: Robust, opinionated Java/Spring Boot backend + Angular frontend. Built for high-security, mission-critical enterprise systems.
---

# Java + Spring Boot / Angular (Enterprise Heavyweight)

**Focus:** enterprise robustness. Long-term stability for high-security, mission-critical systems with complex logic.

## Overview
A robust, highly opinionated stack built for longevity. **Spring Boot** manages complex dependency injection, transaction management, and a vast mature ecosystem; **Angular** provides a batteries-included, strongly-typed frontend framework. Favoured where governance, auditability, and long support horizons matter.

## Communication layer
Primarily **REST**, with legacy **SOAP** where enterprise integration demands it; strong Kafka/JMS messaging support.

## Data persistence
Enterprise RDBMS (Oracle, SQL Server, PostgreSQL) via JPA/Hibernate; strong transactional guarantees.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Full-stack web (backend + frontend) |
| Languages | Java, TypeScript (Angular) |
| Footprint | Large |
| Learning curve | High |
| Best use case | Banking, insurance, mission-critical enterprise |

## Suits these architectures
- [Modular Monolith](../architectures/modular-monolith.md) — enterprise monolith with strong DI/boundaries.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — regulated multi-tenant products.
- [Cloud-Native Microservices](../architectures/cloud-native-microservices.md) — Spring Cloud service ecosystem.
- [Event-Driven / Streaming](../architectures/event-driven-streaming.md) — mature Kafka/JMS integration.

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: ./mvnw spring-boot:run            # backend; run Angular with: cd frontend && npm start
  repo_layout:
    - {path: src/main/java/,      note: Spring Boot application code}
    - {path: src/main/resources/, note: config, templates, static}
    - {path: frontend/,           note: Angular app}
    - {path: src/test/java/,      note: JUnit tests}
  roles:
    backend-developer:
      write_paths: [src/main/java, src/main/resources]
      build: ./mvnw compile
      lint: ./mvnw spotless:check
      test: ./mvnw test
    frontend-developer:
      write_paths: [frontend/src]
      build: cd frontend && npm run build
      lint: cd frontend && npm run lint
      test: cd frontend && npm test
    test-developer:
      write_paths: [src/test/java]
      test: ./mvnw verify
```
