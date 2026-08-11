---
title: Local Web-based Application
type: architecture
status: draft
lineage: arch-local-web
labels:
    - architecture
    - catalog
    - collaborative
    - low-complexity
related_to:
    - architecture/tech-stacks/go-vue.md
    - architecture/tech-stacks/python-fastapi.md
    - architecture/tech-stacks/ts-react-nest.md
    - architecture/tech-stacks/php-symfony-postgres.md
    - architecture/tech-stacks/php-simple.md
summary: Thin browser clients talking to one centralised server on a LAN; single source of truth, shared state, easy updates.
---

# Local Web-based Application

**Focus:** client-server on a LAN · centralised database/server · shared state within a local network.

## Definition
A multi-tier architecture where thin clients (web browsers) interact with a centralised server on the same Local Area Network. Logic is hosted on one node that serves many clients over HTTP/HTTPS. *(This is the architecture kaos-control itself uses: a single Go binary serving an embedded SPA.)*

## Data strategy
Centralised database (e.g. PostgreSQL, SQLite) on the local server — a single source of truth where all users see and modify the same state in near real-time.

## Scaling
**Vertical** (upgrade the server), plus **limited horizontal** scaling via LAN load balancing — constrained by the physical local network.

## Best fit
Internal enterprise tools: inventory management, Laboratory Information Management Systems (LIMS), office resource scheduling.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | Partial (LAN only) |
| Collaboration / shared state | Yes |
| Scale | Team / department |
| Complexity to build | Low–moderate |
| Team skill required | Moderate |

## Pros
- Centralised data and easy updates (one server to deploy).
- Seamless collaboration / shared state.
- Reduced client-side hardware requirements.

## Cons
- Single point of failure (server downtime hits all users).
- Depends on local network stability.
- Wider security exposure if the LAN is breached.

## Suitable tech stacks
- [Go + Vue](../tech-stacks/go-vue.md) — lean single-binary server + reactive SPA (kaos-control's own stack).
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — fast to build, great for data-driven internal tools.
- [TypeScript React + NestJS](../tech-stacks/ts-react-nest.md) — one language end-to-end, large ecosystem.
- [PHP + Symfony / PostgreSQL](../tech-stacks/php-symfony-postgres.md) — classic central business app with a relational source of truth.
- [Simple PHP](../tech-stacks/php-simple.md) — no-framework server-rendered site for a small LAN tool.

## Related architectures
Drop the server and it becomes a [[architecture/architectures/standalone-desktop|Standalone Desktop]] app; move it to the cloud for many tenants and it becomes a [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]]. Internally it is usually a [[architecture/architectures/modular-monolith|Modular Monolith]].
