---
title: "Backend Plan — Inline Priority Display and Editing"
type: plan-backend
status: approved
lineage: artefact-priority-inline-edit
parent: lifecycle/requirements/artefact-priority-inline-edit-2.md
---

# Backend Plan — Inline Priority Display and Editing

## Overview

The backend already supports the full priority PATCH workflow required by this feature. The `PATCH /api/p/:project/artifacts/:path/priority` endpoint (`internal/http/write.go:428-496`) reads the artifact, updates `Frontmatter.Priority`, writes the file, re-indexes, and broadcasts an `artifact.indexed` WebSocket event. The `Priority` field is a plain `string` (`internal/artifact/artifact.go:65`), accepting any value — no closed vocabulary is enforced server-side, which aligns with the requirement's non-goal.

This plan confirms correctness of the existing endpoint and addresses one edge case: ensuring the priority field is included in all artifact detail API responses so the frontend never needs an extra fetch.

---

## Milestone 1 — Verify priority field in artifact detail response

### Description

Confirm that `GET /api/p/:project/artifacts/:path` returns `priority` in the JSON payload (via the `Frontmatter` struct's `json:"priority,omitempty"` tag). When priority is unset (empty string), the field is omitted — the frontend must treat a missing field as `"normal"` (per resolved question #1 in the requirement).

### Files to inspect (no changes expected)

- `internal/artifact/artifact.go` — `Frontmatter` struct, `Priority` field (line ~65)
- `internal/http/read.go` — `handleGetArtifact` response serialisation

### Acceptance criteria

- [ ] `GET /api/p/:project/artifacts/:path` response includes `"priority": "<value>"` when set.
- [ ] When priority is empty/unset, the field is omitted (`omitempty`). Frontend handles this as `"normal"`.
- [ ] No additional API call is required by the frontend to obtain priority.

---

## Milestone 2 — Verify PATCH priority endpoint behaviour

### Description

Confirm the existing `handlePatchPriority` handler (`internal/http/write.go:428-496`) meets all requirement constraints: accepts any string, writes to disk, re-indexes, and broadcasts the WebSocket event.

### Files to inspect (no changes expected)

- `internal/http/write.go:428-496` — `handlePatchPriority`
- `internal/http/server.go:147-154` — route registration (wildcard suffix dispatch)

### Acceptance criteria

- [ ] `PATCH .../priority` with `{"priority":"high"}` updates the file's YAML frontmatter and returns `200` with the re-indexed artifact row.
- [ ] Unknown priority strings (e.g. `"critical"`) are accepted and persisted without error.
- [ ] After a successful PATCH, an `artifact.indexed` WebSocket event with `"action":"updated"` is broadcast.
- [ ] Concurrent PATCH calls do not corrupt the file (sequential file read-write is acceptable for single-user scenarios; file-level locking is out of scope).

---

## Milestone 3 — Verify lock/permission gating on PATCH

### Description

Confirm that the PATCH priority endpoint respects the existing lineage lock and authentication middleware so that locked or unauthorised requests are rejected with appropriate HTTP status codes.

### Files to inspect (no changes expected)

- `internal/http/write.go` — lock check within `handlePatchPriority` (if present) or middleware chain
- `internal/http/server.go` — middleware stack applied to the `/artifacts/*` route group
- `internal/lock/` — lock manager API

### Acceptance criteria

- [ ] A PATCH request on a locked artifact (locked by another user) returns `409 Conflict` or `423 Locked`.
- [ ] An unauthenticated request returns `401`.
- [ ] The [[artefact-priority-inline-edit]] frontend plan can rely on HTTP status codes to determine read-only state.

---

## Summary

No code changes are anticipated. This plan serves as a verification checklist for the [[artefact-priority-inline-edit]] frontend and test plans. If any milestone's acceptance criteria fail during verification, a follow-up implementation milestone should be added.
