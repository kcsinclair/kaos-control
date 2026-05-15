---
title: 'Backend Plan: Artefact Relationship Labels and Clickable Links'
type: plan-backend
status: in-development
lineage: artefact-relationship-labels-and-links
parent: lifecycle/requirements/artefact-relationship-labels-and-links-2.md
---

# Backend Plan: Artefact Relationship Labels and Clickable Links

## Overview

This feature is predominantly frontend work — the existing `GraphEdge` response already carries `source`, `target`, and `kind` fields, which is sufficient for directional label mapping and in-app navigation. The backend plan is deliberately minimal: introduce named constants for edge kinds so the codebase has a single source of truth, and confirm that no API changes are required.

Related: [[artefact-relationship-labels-and-links]]

## Milestone 1 — Define edge kind constants

### Description

Edge kinds (`parent`, `depends_on`, `blocks`, `related_to`, `members`, `wiki`) are currently scattered as string literals across `internal/index/index.go`, `internal/artifact/artifact.go`, and `internal/http/releases.go`. Extract them into exported constants in the artifact package so that both Go code and future validation logic reference a single authoritative list. This also makes it trivial for the frontend plan ([[artefact-relationship-labels-and-links]]) to stay in sync.

### Files to change

- `internal/artifact/artifact.go` — Add a block of exported `EdgeKind*` string constants:

  ```go
  // Edge kinds used in GraphEdge.Kind
  const (
      EdgeKindParent    = "parent"
      EdgeKindDependsOn = "depends_on"
      EdgeKindBlocks    = "blocks"
      EdgeKindRelatedTo = "related_to"
      EdgeKindMembers   = "members"
      EdgeKindWiki      = "wiki"
      EdgeKindAssigned  = "assigned"
      EdgeKindTimeline  = "timeline"
  )
  ```

- `internal/index/index.go` — Replace string literals for edge kinds in `buildEdges()` / link insertion with the new constants (e.g. `artifact.EdgeKindParent` instead of `"parent"`).

- `internal/http/releases.go` — Replace `"assigned"` and `"timeline"` literals with `artifact.EdgeKindAssigned` and `artifact.EdgeKindTimeline`.

### Acceptance criteria

- [ ] All edge kind string literals in `internal/index/` and `internal/http/releases.go` are replaced by the constants from `artifact.go`.
- [ ] `go build ./...` compiles without errors.
- [ ] `go vet ./...` and `staticcheck ./...` report no new findings.
- [ ] No changes to the `GraphEdge` JSON response shape — existing API consumers (frontend, tests) are unaffected.
- [ ] Existing unit and integration tests pass (`make test-unit`).

## Milestone 2 — Verify no API changes needed

### Description

Confirm that the graph API endpoint (`GET /api/p/:project/graph`) already returns all data the frontend needs for directional label mapping and SPA navigation:

1. `source` and `target` fields on every `GraphEdge` encode direction.
2. `kind` matches one of the constants defined in Milestone 1.
3. The node list includes `id` (file path) which the frontend can use to construct an in-app route.

No new fields, endpoints, or query parameters are introduced.

### Files to change

- None. This milestone is a verification gate.

### Acceptance criteria

- [ ] Manual or scripted `curl` of `GET /api/p/{project}/graph` confirms that edges include `source`, `target`, and `kind` with the expected values.
- [ ] No new API endpoints or response fields have been added.
- [ ] The requirement's NFR-2 ("No additional API calls") is satisfied.
