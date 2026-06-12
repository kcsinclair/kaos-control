---
title: Triage API returns 404 (not_found) for non-idea artifact instead of 409 (wrong_type)
type: defect
status: approved
lineage: triage-wrong-type-returns-404
created: "2026-06-12T00:00:00+10:00"
labels:
    - defect
release: KC-Release3
assignees:
    - role: backend-developer
      who: agent
---

# Triage API returns 404 (not_found) for non-idea artifact instead of 409 (wrong_type)

## Reproduction Steps

1. Seed `lifecycle/ideas/raw-defect.md` with `type: defect`, `status: raw`.
2. POST `/api/p/testproject/ideas/raw-defect/triage`.
3. Observe the response.

## Expected Behaviour

HTTP 409 with `reason: wrong_type` — the artifact exists but is not an idea.

## Actual Behaviour

HTTP 404 with `code: not_found` and `message: "no idea artifact found for slug raw-defect"`.

The triage API handler looks up artifacts filtered to `type=idea`. When the slug exists but belongs to a different type (e.g. `defect`), the lookup returns nothing and the handler issues a 404 rather than detecting the type mismatch and returning 409.

## Failing Test

- `TestTriageAPI_WrongType` (`triage_api_test.go:105`)

## Fix

The triage handler should first look up the artifact by slug regardless of type, then:
- If not found at all → 404
- If found but type ≠ idea → 409 `wrong_type`
- If found, type = idea, status ≠ raw → 409 `wrong_status`
- If found, type = idea, status = raw → proceed with triage
