---
title: Single-Service Cloud SaaS
type: architecture
status: draft
lineage: arch-single-service-saas
labels:
    - architecture
    - catalog
    - collaborative
    - low-complexity
related_to:
    - architecture/tech-stacks/go-vue.md
    - architecture/tech-stacks/ts-react-nest.md
    - architecture/tech-stacks/python-fastapi.md
    - architecture/tech-stacks/php-symfony-postgres.md
    - architecture/tech-stacks/python-mongodb.md
summary: One cloud-hosted web application serving many tenants — the classic SaaS shape, without the cost of full microservices.
---

# Single-Service Cloud SaaS

**Focus:** cloud-hosted multi-tenant web app · managed infrastructure · one service, not many.

## Definition
A single web application deployed to the cloud (a PaaS, a container service, or a VM) that serves many customers over the internet. It is the pragmatic middle ground between a [Local Web](local-web.md) app and full [Microservices](cloud-native-microservices.md): centrally hosted and internet-facing, but still **one deployable service** (often a modular monolith inside).

## Data strategy
A managed cloud database (e.g. RDS/Cloud SQL Postgres) with **multi-tenancy** — shared schema with a tenant key, schema-per-tenant, or database-per-tenant depending on isolation needs. Object storage (S3/GCS) for blobs; a managed cache (Redis) for sessions/hot data.

## Scaling
**Horizontal** by running more stateless instances behind a managed load balancer/autoscaler, with the database scaled vertically + read replicas. No orchestration platform required to start.

## Best fit
Most B2B and B2C web products: dashboards, productivity tools, vertical SaaS. The default when you need internet reach and many customers but not yet independent per-domain scaling.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | No |
| Collaboration / shared state | Yes (multi-tenant) |
| Scale | Medium → high |
| Complexity to build | Low–moderate |
| Team skill required | Moderate (cloud basics) |

## Pros
- Internet-reachable, always-on, centrally updated.
- Multi-tenant economics — one deploy serves all customers.
- Much simpler than microservices; managed cloud services do the heavy lifting.

## Cons
- Requires cloud ops (deploys, monitoring, backups, security).
- Whole app scales together; noisy-neighbour tenant isolation needs care.
- Cloud cost and vendor lock-in to manage.

## Suitable tech stacks
- [Go + Vue](../tech-stacks/go-vue.md) — cheap to run, small footprint per instance.
- [TypeScript React + NestJS](../tech-stacks/ts-react-nest.md) — fast product iteration, huge ecosystem.
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — data- and AI-centric SaaS.
- [PHP + Symfony / PostgreSQL](../tech-stacks/php-symfony-postgres.md) — productive multi-tenant web on cheap, ubiquitous hosting.
- [Python + MongoDB](../tech-stacks/python-mongodb.md) — document-per-tenant models scale simply.

## Related architectures
Internally almost always a [[architecture/architectures/modular-monolith|Modular Monolith]]; scales into [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]] when domains need independent scaling; can shed infrastructure entirely toward [[architecture/architectures/serverless-faas|Serverless / FaaS]].
