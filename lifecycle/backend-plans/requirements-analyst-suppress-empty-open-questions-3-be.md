---
created: "2026-08-15T10:57:10+10:00"
title: 'Backend Plan: Suppress Empty Open Questions Section'
type: plan-backend
status: abandoned
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/requirements/requirements-analyst-suppress-empty-open-questions-2.md
assignees:
    - role: product-owner
      who: agent
---

## Overview

This is predominantly a **prompt/config change** plus a **verify-and-retain** pass
over the detector backstop. The behavioural fix lives in the agent prompt templates
in [lifecycle/config.yaml](../config.yaml); the Go detector
(`artifact.HasOpenQuestions`) and the auto-block coupling in `internal/index/` are
already implemented and must be preserved unchanged. No new Go behaviour is
introduced — the milestones below change configuration and add regression coverage
that locks the current backstop in place.

Test coverage for these milestones is specified in
[[requirements-analyst-suppress-empty-open-questions]] (test plan `-5-test`). The
frontend has no work; see the frontend plan for the no-regression verification.

## Milestone 1 — Make analyst prompts emit Open Questions conditionally

**Description.** Update the `requirements-analyst` and `planning-analyst` agent
prompt templates so the `## Open Questions` section is emitted **only** when there
are genuine, unresolved questions. Remove `Open Questions` from the unconditional
"Body sections (use ## headings)" list, and add a positive instruction to omit the
section entirely when empty plus an explicit prohibition on placeholder sentinels.
The existing "If you get stuck" escalation block stays verbatim — it remains the one
sanctioned path that writes an `## Open Questions` section and sets
`status: blocked`.

**Files to change.**
- `lifecycle/config.yaml`
  - `agents[].name == requirements-analyst` → `prompt_templates.analyst`:
    - Remove the `- Open Questions` line from the "Body sections" list (satisfies
      Functional §4).
    - Add a short instruction after the body-sections list, e.g.:
      "Include a `## Open Questions` section ONLY when you have one or more genuine,
      unresolved questions. When there are none, omit the section entirely — do NOT
      emit placeholder content such as `None`, `N/A`, `nil`, `TBD`, `no open
      questions`, or a bare `-` bullet under an Open Questions heading."
    - Leave the "── If you get stuck ──" block unchanged.
  - `agents[].name == planning-analyst` → `prompt_templates.analyst`: apply the same
    conditional-emission instruction. (The planning-analyst template does not list
    `Open Questions` as a standard body section today, so no removal is needed there
    — only the positive/negative guidance is added, satisfying Functional §6.)

**Acceptance criteria.**
- [ ] The `requirements-analyst` prompt no longer lists `Open Questions` as an
      unconditional standard body section.
- [ ] Both analyst prompts contain an explicit "omit when empty" instruction and an
      explicit list of forbidden placeholder sentinels.
- [ ] The "If you get stuck" escalation block is byte-for-byte unchanged in every
      agent template that had it.
- [ ] `internal/config` still parses `config.yaml` without error (the file remains
      valid YAML and all templates load) — covered by the test plan's config-load
      assertion.

## Milestone 2 — Confirm and lock the detector backstop

**Description.** No code change is expected here; the goal is to guarantee the
`HasOpenQuestions` sentinel backstop (added 2026-07-05) is retained exactly as-is so
that a non-compliant model emitting a placeholder still does not auto-block. Verify
the sentinel set and matching logic, and confirm the three named unit tests remain
green. Only if a review reveals a gap (e.g. a forbidden placeholder from Milestone 1
that is not yet a recognised sentinel) is a minimal addition to
`openQuestionSentinels` made.

**Files to change.**
- `internal/artifact/artifact.go` — expected **no change**. If, and only if, a
  placeholder listed in the Milestone-1 prompt is not already covered by
  `openQuestionSentinels` / `isOpenQuestionSentinel`, add that literal to the map.
  (Current map covers `none`, `n/a`, `na`, `nil`, `no open questions`,
  `no questions`, `tbd`; a bare `-` bullet is handled by list-marker stripping.)

**Acceptance criteria.**
- [ ] `artifact.HasOpenQuestions` still returns `false` for a section whose only
      content is any sentinel and `true` for a section containing a real question
      (Functional §5).
- [ ] `TestHasOpenQuestions_SentinelIsNotBlocking`,
      `TestHasOpenQuestions_RealQuestionAlongsideSentinelBlocks`, and
      `TestHasOpenQuestions_QuestionContainingSentinelWordBlocks` pass unmodified.
- [ ] Every placeholder string forbidden by the Milestone-1 prompt is recognised as
      a sentinel by `isOpenQuestionSentinel` (defense-in-depth: prompt and detector
      agree on the placeholder vocabulary).

## Milestone 3 — Confirm no auto-block / auto-unblock regression

**Description.** Verify that the auto-block state-machine coupling in
`internal/index/autoblock.go` and `internal/index/index.go` is untouched and behaves
as before: an artifact with no `## Open Questions` section (the new common case)
indexes to its authored non-blocking status and is never auto-transitioned to
`blocked`; an artifact with a real question still auto-blocks with a `product-owner`
assignee. This milestone is verification-only — no code change.

**Files to change.**
- None expected. `internal/index/autoblock.go` and `internal/index/index.go` remain
  unchanged.

**Acceptance criteria.**
- [ ] A freshly indexed requirement with **no** `## Open Questions` heading keeps
      status `draft` (Non-functional §1; Acceptance criteria bullet 2 of the
      requirement).
- [ ] A requirement whose only Open Questions content is a sentinel is **not**
      auto-blocked (Non-functional §2 backward compatibility).
- [ ] A requirement with a genuine question is still auto-blocked and gains a
      `role: product-owner, who: agent` assignee (escalation path preserved).
- [ ] Existing autoblock/awaiting-answers tests in `internal/index/` remain green.

## Discarded Questions

- **Milestone 1 targets a file outside the backend-developer's write scope.**
  Milestone 1's "Files to change" is entirely `lifecycle/config.yaml` (the
  `requirements-analyst` and `planning-analyst` `prompt_templates.analyst`
  strings). The backend-developer agent's `allowed_write_paths` in
  `lifecycle/config.yaml` itself is `internal`, `cmd`, `lifecycle/backend-plans`,
  `lifecycle/architecture/decisions` — it does not include the config file, and
  no other configured role (`requirements-analyst`, `planning-analyst`,
  `frontend-developer`, `test-developer`, `qa`) lists it either. This isn't a Go
  code change at all, so it doesn't fit the backend-developer role regardless of
  path permissions. Who should make this edit — should the config change be
  made directly by a human/product-owner outside the agent pipeline, or should
  a role's `allowed_write_paths` be extended to cover `lifecycle/config.yaml`?
  Milestones 2 and 3 cannot be meaningfully verified as "no regression" against
  the new prompt behaviour until Milestone 1 actually lands, so the whole plan
  is blocked on this rather than just skipping Milestone 1.

- **Milestone 2's own gap-check surfaces a real bug, not a hypothetical.** I
  instrumented `artifact.HasOpenQuestions` directly (temporary test, reverted)
  against a body containing a bare `-` bullet under `## Open Questions` and it
  returns `true` (treated as a genuine open question), not `false`. The plan's
  Milestone 2 description assumes list-marker stripping already handles this
  ("a bare `-` bullet is handled by list-marker stripping") but that's
  incorrect: `isOpenQuestionSentinel` strips the leading `-` and any
  surrounding punctuation down to `""`, and `""` is not a key in
  `openQuestionSentinels`, so the line falls through to "real question."  Since
  Milestone 1's prompt explicitly forbids agents from emitting a bare `-`
  bullet as a placeholder, and Milestone 2's acceptance criteria requires every
  forbidden placeholder to be a recognised sentinel, this is exactly the kind
  of gap Milestone 2 says to fix with "a minimal addition to
  `openQuestionSentinels`" — e.g. treating the post-strip empty string as a
  sentinel. Confirm this is the intended fix (add `"": true` to the map, or
  equivalent explicit empty-string handling in `isOpenQuestionSentinel`) so it
  can be implemented once Milestone 1 unblocks the rest of the plan.

## Abandoned — blocked on role write-scope, and the harm is already contained

**Status: abandoned 2026-09-03.** Not delivered. Reasoning recorded so it is not
rediscovered later.

### Why these stalled

Not a product question — a **role/write-scope mismatch**. The work these
artifacts describe cannot be performed by any configured agent:

| Artifact | Target file | Problem |
|---|---|---|
| `-3-be` M1 | `lifecycle/config.yaml` | No role has it in `allowed_write_paths` — not backend-developer, nor either analyst. It is also not a Go change, so it does not fit the backend-developer role regardless of permissions. |
| `-5-test` / `-6-test` M1–M2 | `internal/config/config_test.go`, `internal/artifact/artifact_test.go` | Outside test-developer's `allowed_write_paths` (`tests`, `web/src`, `lifecycle/tests`, `lifecycle/test-plans`, `lifecycle/architecture/decisions`). |

The discarded questions below asked who should make these edits. In practice the
answer is that prompt and config curation happens **by hand**, directly in
`lifecycle/config.yaml`, rather than being routed through an agent — so these
tickets were waiting on a workflow that does not exist.

### Why abandoning is safe

The Go half of this lineage shipped, and it is the half that mattered.
`artifact.HasOpenQuestions` (`internal/artifact/artifact.go:334`) ignores
placeholder content, treating these as "no real question":

`none`, `n/a`, `na`, `nil`, `no open questions`, `no questions`, `tbd`

It is wired into the auto-block transition (`internal/index/autoblock.go:34`)
and the index write path (`internal/index/index.go:623`), with both packages'
tests passing. So a hollow "Open Questions: None" does **not** wrongly force an
artifact to `blocked` — the actual damage this lineage set out to stop.

The prompt half was never done. Verified against `lifecycle/config.yaml` on
2026-09-03: `requirements-analyst` still lists Open Questions as an
unconditional body section, and neither analyst prompt carries an
"omit when empty" instruction or a placeholder prohibition. Analysts therefore
still emit the heading routinely and fill it with a sentinel — cosmetic noise,
not a blocked artifact, which is why this is dropped rather than finished.

### Residual risk if it needs reopening

The sentinel list is **finite and literal**. A model writing anything outside it
— "No outstanding questions.", "Nothing further at this stage." — does not
match, so `HasOpenQuestions` returns true and `applyOpenQuestionTransition`
auto-blocks the artifact. The prompt change was the belt to the detector's
braces; without it, protection depends on the model phrasing its non-answer in
one of seven exact ways.

If spurious `blocked` statuses appear on analyst output, look here first. The
fix is the M1 prompt edit described above, not extending the sentinel list.

### Why the heading below was renamed

`HasOpenQuestions` matches the `## Open Questions` heading exactly, and
`applyOpenQuestionTransition` auto-blocks any artifact carrying one. While the
heading read "Open Questions" this file could not be abandoned at all — the
status reverted to `blocked` within seconds of every edit. Renaming it to
`## Discarded Questions` releases that. Worth knowing if another lineage needs
closing the same way.
