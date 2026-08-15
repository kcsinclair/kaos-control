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

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only;
the generator adds the constant lifecycle paths.

```yaml
stack_profile:
  run: symfony serve                 # or: php -S localhost:8000 -t public/
  repo_layout:
    - {path: src/,        note: Symfony app code (Controllers, Entities, Services)}
    - {path: templates/,  note: Twig templates}
    - {path: public/,     note: web root (front controller, built assets)}
    - {path: migrations/, note: Doctrine migrations}
    - {path: tests/,      note: PHPUnit tests}
  roles:
    backend-developer:
      write_paths: [src, config, migrations]
      build: composer install --no-interaction
      lint: vendor/bin/phpstan analyse src
      test: php bin/phpunit
    frontend-developer:
      write_paths: [templates, assets, public]
      build: php bin/console asset-map:compile
      lint: php bin/console lint:twig templates
      test: php bin/phpunit
    test-developer:
      write_paths: [tests]
      test: php bin/phpunit
```
