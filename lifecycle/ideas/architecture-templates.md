---
title: Architecture Templates for Project Bootstrapping
type: idea
status: draft
lineage: architecture-templates
created: "2026-05-15T13:10:31+10:00"
priority: normal
labels:
    - architecture
    - feature
    - onboarding
release: KC-Release3
---

# Architecture Templates for Project Bootstrapping

Provide a library of curated architecture templates that define the underlying tech stack and high-level architecture decisions for common project patterns. Templates remove the burden of foundational decision-making from individual contributors, ensuring teams start with proven, consistent foundations.

Initial templates should include: GoLang Web App with Embedded SQLite, PHP with Symfony and PostgreSQL, and Python with MongoDB. Each template captures the canonical choices for language, framework, database, and key architectural patterns so that new projects inherit these decisions automatically.

Templates should be selectable at project creation time and stored as versioned artifacts within the lifecycle system. They should be extensible so teams can fork and customise a base template without losing traceability to the original.
