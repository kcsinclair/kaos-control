---
title: "Backend: Auto-transition to blocked when Open Questions are added"
type: plan-backend
status: done
lineage: artefact-status-blocked-on-questions
parent: lifecycle/defects/artefact-status-blocked-on-questions.md
---

# Backend: Auto-transition to blocked when Open Questions are added

When an artifact is saved (via `PUT /artifacts/*`) and the body contains a non-empty `## Open Questions` section, the backend must automatically set `status: blocked` and add a `product-owner` assignee — rather than leaving the artifact in its current status with no assignee.

## Milestone 1 — Detect Open Questions in artifact body

### Description

Add a helper function that inspects the markdown body for a `## Open Questions` heading followed by at least one non-whitespace line of content. This determines whether the artifact should be considered "blocked on questions".

### Files to change

- `internal/artifact/artifact.go` — add `HasOpenQuestions(body string) bool` function.

### Acceptance criteria

- `HasOpenQuestions` returns `true` when the body contains `## Open Questions` followed by at least one non-blank line before the next `##` heading or end-of-file.
- `HasOpenQuestions` returns `false` when:
  - There is no `## Open Questions` heading.
  - The heading exists but the section body is empty or whitespace-only.
- The function is case-sensitive on the heading text (must be exactly `## Open Questions`).

## Milestone 2 — Auto-set status and assignees on artifact update

### Description

In `handleUpdateArtifact` (`internal/http/write.go`), after validation and before writing to disk, call `HasOpenQuestions` on the incoming body. If it returns `true`:

1. Override `frontmatter.Status` to `"blocked"`.
2. Ensure `frontmatter.Assignees` contains at least one entry with `Role: "product-owner"` and `Who: "agent"`. If such an entry already exists, do not duplicate it.

If `HasOpenQuestions` returns `false` and the current status is `"blocked"`, do **not** automatically unblock — that remains a manual product-owner action via the transition endpoint (per the existing workflow rule `blocked → draft`).

### Files to change

- `internal/http/write.go` — add logic in `handleUpdateArtifact` between the validation block and the `buildMarkdown` call (~line 239).

### Acceptance criteria

- Saving an artifact whose body has a populated `## Open Questions` section results in `status: blocked` in the written file, regardless of the status sent in the request payload.
- The written file contains `assignees` with at least `- role: product-owner\n  who: agent`.
- Existing assignees are preserved; the product-owner entry is appended only if not already present.
- Saving an artifact without open questions does NOT alter the requested status (no auto-block, no auto-unblock).
- `go build ./...` and `go vet ./...` pass.

## Milestone 3 — Include blocked-reason in hub event

### Description

When the auto-block triggers, the `artifact.indexed` WebSocket event should include an additional field `blocked_reason: "open-questions"` so the frontend (see [[artefact-status-blocked-on-questions]]) can display context-appropriate messaging without re-parsing the body.

### Files to change

- `internal/http/write.go` — extend the hub broadcast payload for the auto-block case.
- `internal/hub/hub.go` — no structural changes needed if `Payload` is already `map[string]any`; verify.

### Acceptance criteria

- When the auto-block fires, the WebSocket event `artifact.indexed` includes `"blocked_reason": "open-questions"` in its payload.
- When no auto-block occurs, the field is absent (not sent as empty string).
- No changes to existing hub message shape for non-blocked updates.
