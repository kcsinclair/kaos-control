---
title: "Backend Plan: Remove Non-Functional Hyperlinks from Artifact Breadcrumb"
type: plan-backend
status: draft
lineage: artifact-breadcrumb-remove-broken-links
parent: lifecycle/requirements/artifact-breadcrumb-remove-broken-links-2.md
created: "2026-05-06"
assignees:
    - role: backend-developer
      who: agent
---

## Summary

No backend changes are required for this feature. The requirement (§ Non-goals) explicitly states: "Any backend changes — this is a frontend-only fix."

The breadcrumb component (`LineageBreadcrumb.vue`) is entirely client-side. It computes path segments from the `path` prop and uses Vue Router for navigation. The existing REST API (`GET /artifacts/*`, `PUT /artifacts/*`) and WebSocket events (`artifact.indexed`, `file.changed`) are unaffected.

## Milestone 1 — Confirm No Backend Impact

### Description

Verify that no API route, handler, or index query needs to change for the frontend breadcrumb fix described in [[artifact-breadcrumb-remove-broken-links]].

### Files to review (read-only, no changes)

- `internal/http/` — router and artifact handlers; no breadcrumb-related logic exists here.
- `internal/artifact/` — parser and type vocab; breadcrumb is not rendered server-side.

### Acceptance Criteria

- [ ] Confirmed: no Go source files require modification.
- [ ] The frontend plan ([[artifact-breadcrumb-remove-broken-links]]-4-fe) and test plan ([[artifact-breadcrumb-remove-broken-links]]-5-test) do not depend on any backend changes.
