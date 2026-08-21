---
title: Python + MongoDB (Document-Oriented)
type: tech-stack
status: approved
lineage: stack-python-mongodb
labels:
    - tech-stack
    - catalog
    - backend
    - python
    - nosql
summary: Python backend over a MongoDB document store — flexible schemas, fast iteration, natural fit for evolving, semi-structured data.
---

# Python + MongoDB (Document-Oriented)

**Focus:** flexible, schema-light data modelling and rapid iteration.

## Overview
A Python backend (FastAPI, Django, or Flask) over **MongoDB**, a document database that stores JSON-like BSON documents. Where a relational stack forces the schema up front, MongoDB lets the data model evolve with the product — ideal for semi-structured, nested, or fast-changing data. Pairs the productivity and library depth of Python with a store that matches how the objects are actually shaped.

## Communication layer
**REST APIs** (FastAPI/Django REST Framework, OpenAPI-documented); a React/Vue frontend or any client consumes them.

## Data persistence
**MongoDB** via an ODM (Beanie/Motor for async, MongoEngine, or PyMongo) — flexible documents, secondary indexes, aggregation pipelines, and horizontal scale via sharding. Managed options (MongoDB Atlas) remove ops overhead.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Backend (+ React/Vue or any client) |
| Languages | Python (back), TS/JS (front) |
| Footprint | Moderate |
| Learning curve | Low |
| Best use case | Evolving/semi-structured data, content, catalogs, rapid prototypes |

## When to prefer this over [Python + FastAPI](python-fastapi.md) (relational)
Choose MongoDB when documents are naturally nested and the schema is still moving; choose the relational FastAPI stack when you need strict schemas, complex joins, or hard transactional integrity.

## Suits these architectures
- [Modular Monolith](../architectures/modular-monolith.md) — quick to stand up with a flexible store.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — document-per-tenant models scale simply.
- [Serverless / FaaS](../architectures/serverless-faas.md) — stateless functions over a managed document store (Atlas).

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: uvicorn app.main:app --reload     # or your app's entrypoint
  repo_layout:
    - {path: app/,   note: application code (routers/handlers, models, repositories)}
    - {path: tests/, note: pytest tests}
  roles:
    backend-developer:
      write_paths: [app]
      build: pip install -r requirements.txt
      lint: ruff check app
      test: pytest
    frontend-developer:
      required: false          # document-data backend — add a frontend stack if you need a UI
    test-developer:
      write_paths: [tests]
      test: pytest
```
