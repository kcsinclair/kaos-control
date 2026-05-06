---
title: "Graph: Show Tests Toggle — Backend Plan"
type: plan-backend
status: approved
lineage: graph-show-tests-toggle
parent: lifecycle/requirements/graph-show-tests-toggle-2.md
---

## Overview

No backend changes are required for this feature. The requirement explicitly scopes filtering to the client side: "No changes to the graph data API or backend indexing are in scope; filtering is purely client-side."

The existing graph API already returns `type` on every `GraphNode`, and `test`-type artifacts are already indexed and served correctly. The frontend store will handle all filtering logic.

---

## Milestone 1: Confirm backend readiness (verification only)

**Description:** Verify that the graph API already includes `type` on every node and that `test`-type artifacts are returned in the node list without any changes.

**Files to change:** None.

**Acceptance criteria:**

- [ ] `GET /api/projects/:slug/graph` returns nodes where `type` includes `"test"` when test artifacts exist in the lifecycle directory.
- [ ] The `type` field is present on every node in the response payload.
- [ ] No backend code changes are committed as part of this plan.

---

## Dependencies

- This plan has no implementation work. The [[graph-show-tests-toggle]] frontend plan (`-4-fe`) contains all implementation milestones.
- The [[graph-show-tests-toggle]] test plan (`-5-test`) should verify the API contract as part of its integration tests.
