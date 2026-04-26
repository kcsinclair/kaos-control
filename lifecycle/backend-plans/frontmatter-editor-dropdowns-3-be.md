---
title: 'Backend Plan: Frontmatter Editor Dropdowns'
type: plan-backend
status: done
lineage: frontmatter-editor-dropdowns
parent: requirements/frontmatter-editor-dropdowns-2.md
---

# Backend Plan: Frontmatter Editor Dropdowns

This plan covers backend changes needed to support the frontmatter editor dropdown feature. Per the requirement, both `priority` and `status` fields already exist in the data model and the API accepts any string for both. **No backend changes are required.**

## Milestone 1: Confirm No Backend Changes Needed

### Description

Verify that the existing backend already supports the frontend changes described in [[frontmatter-editor-dropdowns]]:

1. The `ArtifactFrontmatter` Go struct already includes an optional `Priority` field.
2. The `PUT /artifacts/*` endpoint already reads and writes both `status` and `priority` from/to YAML frontmatter without validation.
3. The `POST /artifacts` endpoint similarly passes these fields through.
4. The SQLite index stores `status` and `priority` and the `/artifacts` list endpoint returns them.

### Files to change

None. This milestone is a verification-only step.

### Acceptance criteria

- [ ] Confirmed that `internal/artifact/` parses the `priority` field from frontmatter and includes it in the struct returned by the parser.
- [ ] Confirmed that `internal/http/` handlers for `PUT /artifacts/*` and `POST /artifacts` round-trip `priority` without dropping it.
- [ ] Confirmed that `internal/index/` stores `priority` in the SQLite cache and the list/get responses include it.
- [ ] No Go code changes are committed — this plan's output is verification only.

## Notes

The requirement explicitly lists as a non-goal: "Extending the backend API or Go parser — both fields already exist in the data model." and "Validating status server-side (out of scope; the server already accepts any string)."

If the [[frontmatter-editor-dropdowns]] frontend plan or test plan surface a need for backend changes (e.g. a vocabulary endpoint), this plan should be revisited and new milestones added.
