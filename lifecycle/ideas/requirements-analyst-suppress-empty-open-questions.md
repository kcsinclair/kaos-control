---
title: 'Requirements Analyst: Suppress Empty Open Questions Section'
type: idea
status: done
lineage: requirements-analyst-suppress-empty-open-questions
created: "2026-05-10T10:25:37+10:00"
priority: normal
labels:
    - agent
    - defect-fix
    - workflow
release: KC-Release4
---

# Requirements Analyst: Suppress Empty Open Questions Section

The requirements analyst agent currently includes an "open questions" section in generated requirement artifacts even when there are no questions to list. This causes the auto-blocking feature to trigger incorrectly, preventing tickets from progressing when they should be unblocked.

The agent should be updated to omit the open questions section entirely when there are no open questions, rather than emitting a placeholder such as "no questions" or an empty list. Alternatively, a sentinel value like "none" could be used if the section must always be present, but the blocking logic must be updated to treat this as a non-blocking state.

The fix likely requires changes to the requirements analyst prompt template and/or the workflow blocking logic that inspects the open questions field, ensuring that an absent or explicitly-empty section does not trigger the auto-block gate.

## Update — 2026-07-05: fix belongs in the auto-block detector (sentinel handling)

Checked against current code. The picture is narrower than first written:

- **A *truly empty* section is already non-blocking.** `artifact.HasOpenQuestions`
  ([internal/artifact/artifact.go](../../internal/artifact/artifact.go)) returns
  `false` when the `## Open Questions` heading has only whitespace, or is
  immediately followed by another `## ` heading. Locked in by
  `TestHasOpenQuestions_HeadingWithOnlyWhitespace` and
  `...HeadingFollowedImmediatelyByNextHeading`. So "empty → don't block" is done.

- **The residual bug is placeholder / sentinel content.** When the agent emits
  `## Open Questions` with a sentinel like `None`, `N/A`, or `- none`, that text
  is *non-empty* → `HasOpenQuestions` returns `true` → the artefact wrongly
  auto-blocks.

- **Why it happens:** the requirements-analyst prompt
  ([lifecycle/config.yaml](../config.yaml)) lists `Open Questions` as a *standard
  body section* (under "Body sections (use ## headings)") **and** the "If you get
  stuck" instruction uses `## Open Questions` as the block signal — so the agent
  is pushed to emit the section routinely, and any placeholder in it blocks.

### Recommended fix — harden the detector (defense in depth)

1. **Detector (primary — the auto-block improvement):** extend
   `HasOpenQuestions` so a section whose only content is a recognised
   "no questions" **sentinel** (`none`, `n/a`, `na`, `nil`, `no open questions`,
   `no questions`, a bare `-` bullet) is treated as empty → non-blocking. Models
   don't reliably comply with prompts, so the detector is the durable backstop.
   Real questions still block.

2. **Prompt (optional, for consistency):** either drop `Open Questions` from the
   standard body-sections list so it only appears via the "If you get stuck"
   path, or instruct the agent to write `None` when there are none (which the
   hardened detector then treats as non-blocking).

Net: keep the truly-empty handling; add sentinel handling; the placeholder case
that this defect is really about stops blocking.

### Implemented — 2026-07-05

Fix (1), the detector hardening, is **done**: `HasOpenQuestions` now treats a
section whose only content is a sentinel (`none`, `n/a`, `na`, `nil`,
`no open questions`, `no questions`, `tbd`) as empty → non-blocking, with unit
tests (`TestHasOpenQuestions_SentinelIsNotBlocking`,
`...RealQuestionAlongsideSentinelBlocks`, `...QuestionContainingSentinelWordBlocks`)
and the existing autoblock tests still green. Fix (2), the prompt cleanup,
remains optional — the detector now absorbs the placeholder case regardless.
