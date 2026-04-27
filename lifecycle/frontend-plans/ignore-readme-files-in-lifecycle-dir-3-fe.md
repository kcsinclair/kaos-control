---
title: "Frontend plan: no UI changes required for ignore-pattern support"
type: plan-frontend
status: draft
lineage: ignore-readme-files-in-lifecycle-dir
parent: lifecycle/defects/ignore-readme-files-in-lifecycle-dir.md
---

# Frontend plan: no UI changes required for ignore-pattern support

This defect is entirely backend-scoped. The fix adds configurable ignore patterns to the lifecycle indexer so that files like `README.md` are never indexed. Because ignored files will simply not appear in API responses, the frontend requires **no code changes** to support this fix.

## Milestone 1 — Verify frontend behaviour with ignored files absent from API

**Description:** Confirm that the existing artifact list, graph, and lineage views render correctly when previously-indexed `README.md` files are no longer returned by the API. This is a manual verification milestone — no code changes are expected.

**Files to change:**
- (none)

**Acceptance criteria:**
- [ ] The artifact list view (`/artifacts`) does not show `README.md` entries after the backend fix is deployed.
- [ ] The graph view renders without errors (no dangling nodes referencing removed README artifacts).
- [ ] The lineage detail view for any lineage that previously contained a README artifact loads without errors.
- [ ] No console errors or warnings appear related to missing artifacts.

---

## Cross-references

- [[ignore-readme-files-in-lifecycle-dir]] backend plan (index 2): all functional changes live in the backend; this plan depends on those changes being complete.
- [[ignore-readme-files-in-lifecycle-dir]] test plan (index 4): integration tests will validate end-to-end API behaviour, which implicitly covers frontend data correctness.
