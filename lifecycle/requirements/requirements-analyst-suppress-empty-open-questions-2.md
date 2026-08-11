---
title: 'Requirements Analyst: Suppress Empty Open Questions Section'
type: requirement
status: approved
lineage: requirements-analyst-suppress-empty-open-questions
created: "2026-07-07T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/requirements-analyst-suppress-empty-open-questions.md
labels:
    - agent
    - defect-fix
    - workflow
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

## Problem

The requirements-analyst agent routinely emits a `## Open Questions` section in
generated requirement artifacts, because the section is listed among the
*standard body sections* in the agent's prompt template
([lifecycle/config.yaml](../config.yaml)) **and** the "If you get stuck"
instruction reuses `## Open Questions` as the block signal. When there are no
genuine questions, the agent tends to fill the section with a placeholder such as
`None`, `N/A`, or a bare `- none` bullet.

Auto-block detection keys off this section. A *truly empty* section is already
non-blocking, and as of 2026-07-05 the detector (`artifact.HasOpenQuestions` in
[internal/artifact/artifact.go](../../internal/artifact/artifact.go)) also treats
a recognised "no questions" sentinel as empty. However, the agent-side behaviour
still produces avoidable placeholder sections, which:

- rely entirely on the detector's sentinel list staying in sync with whatever
  phrasing the model invents (a single unrecognised placeholder re-introduces the
  wrong-block bug), and
- add noise to artifacts, implying unresolved questions where none exist.

The residual work is to make the agent emit an Open Questions section **only when
there are real open questions**, so that a correctly-authored requirement never
carries a placeholder that could trigger — or appear to trigger — the auto-block
gate.

## Goals / Non-goals

### Goals

- Update the requirements-analyst (and, for consistency, planning-analyst) prompt
  template so the agent omits the `## Open Questions` section entirely when there
  are no open questions.
- Preserve the existing "If you get stuck" escalation path: when the agent has
  genuine blocking questions it still writes a `## Open Questions` section (and
  sets `status: blocked` with a `product-owner` assignee) exactly as today.
- Keep the hardened `HasOpenQuestions` sentinel detection as a defense-in-depth
  backstop so that any stray placeholder emitted by a non-compliant model does not
  auto-block.
- Ensure a requirement artifact produced with no open questions ends in a
  non-blocking status (e.g. `draft`) and is not auto-transitioned to `blocked`.

### Non-goals

- Changing the auto-block / auto-unblock state-machine coupling (delivered under
  [[agent-questions-trigger-blocked-status]]); this requirement only affects what
  the agent writes.
- Re-implementing or removing the sentinel handling already added to
  `HasOpenQuestions` on 2026-07-05 — it stays as the backstop.
- Introducing a `questions:` frontmatter field or any trigger mechanism other than
  the body-based `## Open Questions` section.
- Changing detection for artifact types other than those authored by the analyst
  agents (the detector already covers all types uniformly).

## Detailed Requirements

### Functional

1. **Conditional section emission.** The requirements-analyst prompt template must
   instruct the agent to include a `## Open Questions` section **only** when it has
   one or more genuine, unresolved questions. When there are none, the section —
   and any placeholder content — must be omitted from the artifact body.

2. **No placeholder sentinels.** The prompt must explicitly forbid emitting
   placeholder content (`None`, `N/A`, `n/a`, `nil`, `TBD`, `no open questions`, a
   bare `-` bullet, or similar) under an `## Open Questions` heading.

3. **Escalation path unchanged.** When the agent is genuinely blocked, it must
   still write each blocking question under a `## Open Questions` heading and set
   frontmatter `status: blocked` with an `assignees` entry of
   `role: product-owner, who: agent`, matching the current "If you get stuck"
   behaviour.

4. **Standard-sections list corrected.** `Open Questions` must be removed from (or
   clearly marked conditional in) the "Body sections (use ## headings)" list in the
   requirements-analyst prompt, so the agent is no longer pushed to emit it
   routinely. The section must appear only via the conditional / escalation path.

5. **Detector backstop retained.** `artifact.HasOpenQuestions` must continue to
   treat a section whose only content is a recognised sentinel as empty
   (non-blocking), so a non-compliant model output still does not auto-block. A
   section containing at least one real question must still return `true`.

6. **Consistency across analyst agents.** The same conditional-emission guidance
   should be applied to any other agent prompt (e.g. planning-analyst) that lists
   `Open Questions` as a standard body section, so behaviour is uniform.

### Non-functional

1. **No workflow regression.** Existing auto-block / auto-unblock behaviour and the
   workflow state machine must remain unchanged; only the agent's authored output
   changes.

2. **Backward compatibility.** Previously-authored artifacts that already contain a
   placeholder `## Open Questions` section must not regress: the detector's
   sentinel handling continues to keep them non-blocking without any migration.

3. **Prompt clarity.** The prompt change must be unambiguous enough that the model
   reliably complies (positive instruction: "omit the section when empty" rather
   than only a negative constraint), reducing reliance on the detector backstop.

## Acceptance Criteria

- [ ] A requirement generated by the requirements-analyst with no genuine open
      questions contains **no** `## Open Questions` heading anywhere in its body.
- [ ] Such an artifact is indexed with a non-blocking status (`draft`) and is
      **not** auto-transitioned to `blocked`.
- [ ] When the agent has genuine blocking questions, it still emits a
      `## Open Questions` section listing them and sets `status: blocked` with a
      `product-owner` assignee (escalation path preserved). See
      [[agent-questions-trigger-blocked-status]].
- [ ] The requirements-analyst prompt in `lifecycle/config.yaml` no longer lists
      `Open Questions` as an unconditional standard body section.
- [ ] `HasOpenQuestions` still returns `false` for a section whose only content is
      a sentinel (`none`, `n/a`, `na`, `nil`, `tbd`, `no open questions`,
      `no questions`, bare `-` bullet) and `true` for a section with a real
      question — existing unit tests
      (`TestHasOpenQuestions_SentinelIsNotBlocking`,
      `...RealQuestionAlongsideSentinelBlocks`,
      `...QuestionContainingSentinelWordBlocks`) remain green.
- [ ] The same conditional-emission guidance is applied to the planning-analyst
      prompt (and any other agent that listed `Open Questions` as a standard
      section).
- [ ] No change to the auto-block/auto-unblock state-machine coupling; existing
      autoblock tests remain green.
- [ ] Related GUI work for surfacing open questions is unaffected. See
      [[open-questions-gui]].

## Resolved Questions

_None — the fix scope is well defined; the primary detector hardening is already
implemented and this requirement covers the remaining prompt cleanup._
