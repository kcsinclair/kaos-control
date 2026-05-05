---
title: 'Backend Plan: Lineage Status Checker API'
type: plan-backend
status: in-development
lineage: status-checker-button
parent: lifecycle/requirements/status-checker-button-2.md
---

## Overview

Implement the staleness detection engine and REST endpoint (`GET /api/p/{project}/status-check`) that powers the lineage status checker feature. The backend provides the algorithm, query layer, and advance action orchestration.

---

## Milestone 1: Staleness Detection Engine

### Description

Create a new package `internal/statuscheck` containing the staleness detection algorithm. Given a set of artifacts sharing a lineage, walk from leaves to root and identify artifacts whose status lags behind all of their actively-progressing children.

### Files to Change

- `internal/statuscheck/statuscheck.go` (new) — core algorithm
- `internal/statuscheck/statuscheck_test.go` (new) — unit tests

### Acceptance Criteria

- Define the canonical status order: `draft → clarifying → planning → in-development → in-qa → approved → done`.
- Terminal statuses (`rejected`, `abandoned`, `blocked`) are excluded from comparison.
- An artifact is stale if **every** non-terminal direct child has a status strictly later in the order than the parent's current status.
- The suggested target status is the **furthest valid status** that the parent can advance to (i.e. the minimum status among all non-terminal children).
- A single-artifact lineage (no children) never reports staleness.
- The algorithm operates on `[]index.ArtifactRow` (no DB access itself — pure logic).

---

## Milestone 2: Index Query Helper

### Description

Add a method to `internal/index` that efficiently fetches all artifacts grouped by lineage, or for a single lineage slug, using the SQLite index.

### Files to Change

- `internal/index/index.go` — add `ListByLineage(slug string) ([]*ArtifactRow, error)` and `ListAllGroupedByLineage() (map[string][]*ArtifactRow, error)` methods.

### Acceptance Criteria

- `ListByLineage("")` returns all artifacts grouped by lineage (project-wide check).
- `ListByLineage("foo")` returns only artifacts with `lineage = "foo"`.
- Queries use indexed columns; no full-table scans for the single-lineage case.
- Returns only columns needed by the staleness algorithm (path, lineage, status, type).

---

## Milestone 3: REST Endpoint

### Description

Expose `GET /api/p/{project}/status-check?lineage={slug}` (single lineage) and `GET /api/p/{project}/status-check` (all lineages). Return the JSON response schema defined in the requirement.

### Files to Change

- `internal/http/status_check.go` (new) — handler implementation
- `internal/http/server.go` — register route under the project router

### Acceptance Criteria

- Route registered as `r.Get("/status-check", s.handleStatusCheck)` inside the per-project route group.
- The `lineage` query param is optional; when omitted, check all lineages.
- For each stale artifact, determine `can_advance` by calling `p.Workflow.CanTransition(current, suggested, userRoles)`.
- When `can_advance` is false, include `blocked_reason` (e.g. `"requires role: approver"`).
- Response matches the JSON schema in the requirement (fields: `path`, `lineage`, `current_status`, `suggested_status`, `reason`, `children`, `can_advance`, `blocked_reason`).
- The endpoint requires authentication (user must be logged in to determine roles).
- Project-wide check completes within 500 ms for 1 000 indexed artifacts.

---

## Milestone 4: Batch Advance Endpoint

### Description

Expose `POST /api/p/{project}/status-check/advance` to apply one or more suggested transitions. Process sequentially so each transition sees the updated state of previously fixed artifacts.

### Files to Change

- `internal/http/status_check.go` — add `handleStatusCheckAdvance`
- `internal/http/server.go` — register `r.Post("/status-check/advance", s.handleStatusCheckAdvance)`

### Acceptance Criteria

- Request body: `{"paths": ["lifecycle/ideas/foo.md", ...]}` — list of artifact paths to advance.
- Each path is re-evaluated against the staleness algorithm at execution time (not trusting client-side suggested status).
- Transitions are applied sequentially in the order provided.
- Each successful transition updates frontmatter on disk, re-indexes, and broadcasts `artifact.indexed` via WebSocket (reusing existing transition logic from [[status-checker-button]] `handleTransitionArtifact`).
- Transitions that fail permission checks are skipped and reported in the response with `blocked_reason`.
- Response: `{"results": [{"path": "...", "advanced_to": "planning", "ok": true}, {"path": "...", "ok": false, "error": "requires role: approver"}]}`.
- Idempotent: running advance on an already-current artifact is a no-op (reported as `ok: true` with no change).

---

## Milestone 5: Integration with Existing Transition Infrastructure

### Description

Ensure the advance action reuses the existing disk-write + re-index + WebSocket broadcast path from `handleTransitionArtifact` rather than duplicating logic.

### Files to Change

- `internal/http/transition.go` — extract a reusable `applyTransition(p *project.Project, relPath, toStatus string, user *auth.User) error` helper.
- `internal/http/status_check.go` — call the extracted helper.

### Acceptance Criteria

- No duplication of frontmatter-update or re-index logic.
- `artifact.indexed` WebSocket event fires for each status change made by the checker.
- Existing transition endpoint behaviour is unchanged (no regression).
