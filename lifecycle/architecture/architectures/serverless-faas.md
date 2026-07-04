---
title: Serverless / Functions-as-a-Service
type: architecture
status: draft
lineage: arch-serverless-faas
labels:
    - architecture
    - catalog
    - low-cost-start
    - medium-complexity
related_to:
    - architecture/tech-stacks/ts-react-nest.md
    - architecture/tech-stacks/python-fastapi.md
    - architecture/tech-stacks/go-vue.md
    - architecture/tech-stacks/python-mongodb.md
summary: Application composed of managed functions triggered on demand; no servers to run, pay-per-execution, scales to zero.
---

# Serverless / Functions-as-a-Service

**Focus:** managed compute · event/HTTP-triggered functions · pay-per-use · scale-to-zero.

## Definition
The application is composed of small, stateless **functions** (AWS Lambda, Cloud Functions, Azure Functions) triggered by HTTP requests, queue messages, schedules, or storage events. There are no long-running servers to provision or patch — the platform runs, scales, and bills per invocation. Usually paired with managed API gateways, queues, and databases.

## Data strategy
Managed, serverless-friendly stores: a serverless SQL/NoSQL database (DynamoDB, Aurora Serverless, Firestore), object storage, and managed queues/streams. Functions are stateless — all state lives in these external services.

## Scaling
**Automatic and elastic** — the platform spins up as many concurrent function instances as demand requires and scales to **zero** when idle. Watch for cold starts and per-account concurrency limits.

## Best fit
Spiky or unpredictable workloads, event processing, webhooks, cron/back-office jobs, glue between SaaS systems, and lean startups minimising fixed cost. Also a great **companion** tier to other architectures.

## Decision signals
| Signal | Value |
| --- | --- |
| Works offline | No |
| Collaboration / shared state | Yes (via managed stores) |
| Scale | Elastic (to zero, and up) |
| Complexity to build | Moderate (distributed, many managed parts) |
| Team skill required | Moderate (cloud-native patterns) |

## Pros
- No servers to manage; pay only for what runs.
- Effortless elastic scaling, including down to zero cost when idle.
- Fast to ship small units of functionality.

## Cons
- Cold-start latency; execution time/memory limits.
- Strong vendor lock-in; local testing and debugging are harder.
- Cost can surprise at sustained high volume vs. always-on servers.

## Suitable tech stacks
- [TypeScript React + NestJS](../tech-stacks/ts-react-nest.md) — Node has first-class FaaS support.
- [Python + FastAPI](../tech-stacks/python-fastapi.md) — Python functions for data/AI workloads.
- [Go + Vue](../tech-stacks/go-vue.md) — Go's tiny cold-start and fast startup suit Lambda well.
- [Python + MongoDB](../tech-stacks/python-mongodb.md) — stateless functions over a managed document store (Atlas).

## Related architectures
Often the compute tier of a [[architecture/architectures/single-service-saas|Single-Service Cloud SaaS]], and a common execution model for [[architecture/architectures/event-driven-streaming|Event-Driven / Streaming]] consumers and [[architecture/architectures/cloud-native-microservices|Cloud-Native Microservices]] where each service is a set of functions.
