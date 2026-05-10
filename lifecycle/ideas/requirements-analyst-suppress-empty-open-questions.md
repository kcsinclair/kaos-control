---
title: 'Requirements Analyst: Suppress Empty Open Questions Section'
type: idea
status: draft
lineage: requirements-analyst-suppress-empty-open-questions
created: "2026-05-10T10:25:37+10:00"
priority: normal
labels:
    - agent
    - defect-fix
    - workflow
---

# Requirements Analyst: Suppress Empty Open Questions Section

The requirements analyst agent currently includes an "open questions" section in generated requirement artifacts even when there are no questions to list. This causes the auto-blocking feature to trigger incorrectly, preventing tickets from progressing when they should be unblocked.

The agent should be updated to omit the open questions section entirely when there are no open questions, rather than emitting a placeholder such as "no questions" or an empty list. Alternatively, a sentinel value like "none" could be used if the section must always be present, but the blocking logic must be updated to treat this as a non-blocking state.

The fix likely requires changes to the requirements analyst prompt template and/or the workflow blocking logic that inspects the open questions field, ensuring that an absent or explicitly-empty section does not trigger the auto-block gate.
