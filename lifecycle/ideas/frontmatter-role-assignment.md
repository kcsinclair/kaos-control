---
title: Frontmatter Editor Role-Based Assignment
type: idea
status: done
lineage: frontmatter-role-assignment
created: "2026-04-28T10:27:09+10:00"
priority: normal
labels:
    - feature
    - frontend
    - workflow
release: KC-OG-Sprint
---

# Frontmatter Editor Role-Based Assignment

The frontmatter editor should support assigning an artifact to a specific role — either an agent role (e.g. `backend-developer`, `qa`) or a human role (e.g. `product-owner`, `reviewer`) — rather than requiring the user to type a raw string.

The assignment control should present the available roles drawn from the project's configured roles (as defined in `lifecycle/config.yaml`), so the value is always valid and consistent with the project's role vocabulary.

This makes it easy to route work to the right person or agent at any lifecycle stage, and ensures the `assigned_to` frontmatter field stays in sync with the roles the system actually understands.
