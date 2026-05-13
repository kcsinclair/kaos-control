---
title: "Backend Plan — Dashboard New Idea & Defect Buttons"
type: plan-backend
status: done
lineage: dashboard-new-idea-defect-buttons
parent: lifecycle/requirements/dashboard-new-idea-defect-buttons-2.md
created: "2026-05-13"
---

# Backend Plan — Dashboard New Idea & Defect Buttons

## Summary

This feature requires **no backend changes**. The existing REST API endpoints for artifact creation (`POST /p/:project/artifacts`) and idea/defect generation (`POST /p/:project/ideas/generate`) are fully sufficient. The `BrainDumpModal` component and `brainDump` Pinia store already handle all API communication, and the dashboard buttons simply reuse that existing flow.

This plan exists to formally confirm the backend scope is nil and to document the contract the frontend depends on.

---

## Milestone 1: Confirm API Surface Is Unchanged

### Description
Verify that no backend routes, handlers, or data models need modification to support the new dashboard buttons. The frontend will invoke the same endpoints it already uses from the artifacts list page.

### Files to change
_None._

### Acceptance Criteria
- [ ] `POST /p/:project/ideas/generate` continues to accept `{ input: string, artifactType: "idea" | "defect" }` and return a proposal.
- [ ] `POST /p/:project/artifacts` continues to accept a proposal payload and return the created artifact path.
- [ ] No new routes, middleware, or handler logic is introduced.

---

## Cross-references

- [[dashboard-new-idea-defect-buttons]] (frontend plan): the frontend plan describes the Vue component changes that consume these existing endpoints.
- [[dashboard-new-idea-defect-buttons]] (test plan): integration tests will exercise the creation flow end-to-end via the dashboard entry point.
