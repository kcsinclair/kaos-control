---
title: PHP + Symfony / PostgreSQL
type: tech-stack
status: draft
lineage: stack-php-symfony-postgres
labels:
    - tech-stack
    - catalog
    - backend
    - frontend
    - php
    - postgres
summary: Mature, batteries-included PHP web framework (Symfony) over PostgreSQL — structured, productive full-stack web apps with a huge hosting footprint.
---

# PHP + Symfony / PostgreSQL

**Focus:** structured, productive full-stack web development on the most widely-hosted runtime.

## Overview
**Symfony** is a mature, opinionated PHP framework with first-class dependency injection, a powerful ORM (**Doctrine**), a component ecosystem, and conventions that keep large codebases organised. It renders server-side with **Twig** and/or exposes JSON/REST APIs (API Platform) for an SPA frontend. Backed by **PostgreSQL** for ACID-compliant relational data. PHP's ubiquity means it runs almost anywhere and hires easily.

## Communication layer
Server-rendered **Twig** templates for classic web apps, and/or **REST APIs** (via API Platform, which also speaks GraphQL/JSON-LD) when paired with a JS frontend.

## Data persistence
**PostgreSQL** via Doctrine ORM (migrations, entities, DQL) — strong transactional guarantees and rich SQL. Redis for caching/sessions/queues.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Full-stack web (backend + server-rendered/API frontend) |
| Languages | PHP, Twig (+ optional TS/JS SPA) |
| Footprint | Moderate |
| Learning curve | Low–moderate |
| Best use case | Structured business web apps, CMS-like and CRUD-heavy products |

## Suits these architectures
- [Modular Monolith](../architectures/modular-monolith.md) — Symfony bundles map cleanly onto bounded modules.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — productive multi-tenant web products; runs on cheap, ubiquitous hosting.
- [Local Web Application](../architectures/local-web.md) — a central LAN business app with a relational source of truth.
