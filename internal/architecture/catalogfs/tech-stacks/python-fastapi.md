---
title: Python + FastAPI (Intelligence & Prototyping Stack)
type: tech-stack
status: draft
lineage: stack-python-fastapi
labels:
    - tech-stack
    - catalog
    - backend
    - python
    - ai-ml
summary: Async Python via FastAPI + a React/Vue frontend. The default for AI/ML-integrated apps and rapid data-driven prototyping.
---

# Python + FastAPI (Intelligence & Prototyping Stack)

**Focus:** AI/ML integration and rapid data-driven prototyping.

## Overview
Combines the industry-standard data-science and AI libraries (PyTorch, TensorFlow, scikit-learn, pandas) with modern **asynchronous Python** via FastAPI. Pairs with a React or Vue frontend. Fastest path from a model or dataset to a working API.

## Communication layer
**RESTful APIs** documented automatically via **OpenAPI/Swagger** (FastAPI generates the schema and interactive docs from type hints).

## Data persistence
PostgreSQL (via SQLAlchemy) for relational data, plus vector databases (pgvector, Pinecone) and object storage for ML/AI workloads.

## Profile
| Trait | Value |
| --- | --- |
| Layer | Backend (+ React/Vue frontend) |
| Languages | Python (back), TS/JS (front) |
| Footprint | High / variable (ML deps) |
| Learning curve | Very low |
| Best use case | AI/ML apps, data science, rapid prototypes |

## Suits these architectures
- [Modular Monolith](../architectures/modular-monolith.md) — quick to stand up, clear modules.
- [Single-Service Cloud SaaS](../architectures/single-service-saas.md) — data/AI-centric products.
- [Cloud-Native Microservices](../architectures/cloud-native-microservices.md) — ML services alongside others.
- [Event-Driven / Streaming](../architectures/event-driven-streaming.md) — analytics/ML stream consumers.
- [Edge / Distributed Hybrid](../architectures/edge-hybrid.md) — cloud-side aggregation & inference.
- [Serverless / FaaS](../architectures/serverless-faas.md) — Python functions for data workloads.

## Stack profile

kaos-control reads this profile to configure the coding agents for this stack (repo layout, per-role write paths, and build/lint/test/run commands); it also doubles as a plain-language primer on how the stack is laid out and run. `write_paths` are stack source roots only; the generator adds the constant lifecycle paths. Roles that do not apply are marked `required: false`.

```yaml
stack_profile:
  run: uvicorn app.main:app --reload
  repo_layout:
    - {path: app/,         note: FastAPI application (routers, models, services)}
    - {path: app/main.py,  note: ASGI entrypoint}
    - {path: tests/,       note: pytest tests}
  roles:
    backend-developer:
      write_paths: [app]
      build: pip install -r requirements.txt
      lint: ruff check app
      test: pytest
    frontend-developer:
      required: false          # FastAPI is a backend/API — add a frontend stack if you need a UI
    test-developer:
      write_paths: [tests]
      test: pytest
```
