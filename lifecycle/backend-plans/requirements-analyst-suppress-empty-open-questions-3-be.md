---
title: 'Backend Plan: Suppress Empty Open Questions Section'
type: plan-backend
status: in-development
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/requirements/requirements-analyst-suppress-empty-open-questions-2.md
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
