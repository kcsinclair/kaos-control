---
title: Simple PHP (Server-Rendered, No Framework)
type: tech-stack
status: draft
lineage: stack-php-simple
labels:
    - tech-stack
    - catalog
    - backend
    - php
    - low-complexity
summary: Plain PHP server-rendered pages with no framework — the classic small dynamic website. Runs on any shared host; optional SQLite or flat files for data.
---

# Simple PHP (Server-Rendered, No Framework)

**Focus:** the simplest *dynamic* website. Server-rendered pages, no framework, deployable to the cheapest ubiquitous hosting there is.

## Overview
Plain PHP scripts that render HTML on the server, one file per page (or a tiny front controller), with no framework. This is the classic small-business / brochure-plus-a-form site: contact forms, a small admin page, a handful of database-backed listings. It trades the structure and safety rails of a framework for near-zero setup and the broadest hosting footprint of any stack — virtually every shared host runs PHP out of the box. When the site grows real domain logic, graduate to the structured option, [PHP + Symfony / PostgreSQL](php-symfony-postgres.md).

## Communication layer
**Server-rendered HTTP** — PHP builds each page per request behind Apache/nginx (or `php -S` in development). Progressive enhancement with a little client-side JS as needed.

## Data persistence
Whatever's simplest for the job: **SQLite** (a single file, zero setup), **flat files** (JSON/CSV), or **MySQL/MariaDB/PostgreSQL** on shared hosting via PDO. Use PDO prepared statements throughout.

## Quality tooling
Keep it honest without heavy scaffolding: **`php -l`** lint, **PHP_CodeSniffer** for style, optional **PHPStan** for static analysis, and Playwright/axe end-to-end + accessibility checks against the rendered pages.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Full-stack web (server-rendered) |
| Languages | PHP, HTML, a little JS |
| Footprint | Low; runs on any shared PHP host |
| Learning curve | Low |
| Best use case | Small dynamic sites: forms, small admin, DB-backed listings |

## Suits these architectures
- [Local Web Application](../architectures/local-web.md) — a small server-rendered app on a LAN box.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — a simple hosted site at the low-complexity end.

## When to reach for something else
- Purely content/read-only, no server logic → a [Static Website](../architectures/static-site.md) ([Hugo](hugo.md) or [hand-authored](static-html-js.md)) is cheaper and safer.
- Real domain complexity, teams, long-lived app → [PHP + Symfony / PostgreSQL](php-symfony-postgres.md) adds the structure and safety rails.

## Stack profile

See [[agent-directives-generation]]. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: php -S localhost:8000 -t public/   # or -t . if pages live at the root
  repo_layout:
    - {path: public/, note: web root (entry PHP pages, assets)}
    - {path: src/,    note: shared PHP (helpers, DB access via PDO)}
    - {path: tests/,  note: PHPUnit tests}
  roles:
    backend-developer:
      write_paths: [public, src]
      build: composer install --no-interaction   # if using composer; else n/a
      lint: php -l public/index.php               # syntax lint; add phpcs if configured
      test: vendor/bin/phpunit
    frontend-developer:
      required: false          # server-rendered PHP — markup lives with the pages (backend role)
    test-developer:
      write_paths: [tests]
      test: vendor/bin/phpunit
```
