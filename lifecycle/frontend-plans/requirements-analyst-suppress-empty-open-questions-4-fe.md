---
title: 'Frontend Plan: Suppress Empty Open Questions Section (No-Regression)'
type: plan-frontend
status: approved
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/requirements/requirements-analyst-suppress-empty-open-questions-2.md
---

## Overview

This requirement is a backend/config change to how the analyst agents author
artifacts. It has **no frontend feature work**: the requirement's Non-goals and
Acceptance Criteria state that GUI work for surfacing open questions is unaffected
and is delivered separately under [[open-questions-gui]]. The artifact renderer
already handles bodies with or without an `## Open Questions` heading — omitting the
section is just "one fewer heading to render", which needs no code path.

Accordingly this plan is a **no-regression verification plan**, not an
implementation plan. It exists so the lineage's required `plan-frontend` gate is
satisfied honestly and so the frontend behaviour is explicitly confirmed rather than
assumed. See [[requirements-analyst-suppress-empty-open-questions]] backend plan
`-3-be` for where the actual change lives.

## Milestone 1 — Confirm renderer handles a missing Open Questions section

**Description.** Verify that an artifact whose body has no `## Open Questions`
heading renders correctly in the editor/preview and graph detail views — no empty
placeholder heading, no console error, no layout artifact where the section used to
be. This is the new common case for correctly-authored requirements.

**Files to change.**
- None expected. Relevant existing components (artifact detail / markdown preview
  under `web/src/`) already render arbitrary markdown bodies; confirm no component
  assumes an Open Questions section is present.

**Acceptance criteria.**
- [ ] An artifact body with no `## Open Questions` heading renders with no empty
      section, no injected placeholder, and no console warning/error.
- [ ] The status/blocked badge for such an artifact reflects its authored
      non-blocking status (e.g. `draft`) — consistent with the backend not
      auto-blocking it.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass with no changes to
      `web/src/**` (confirming no frontend change was required).

## Milestone 2 — Confirm the blocked/open-questions UI still works for real questions

**Description.** Verify the escalation path is unchanged from the UI's perspective:
when an artifact genuinely has an `## Open Questions` section and `status: blocked`,
the existing blocked-questions surfacing (badge/indicator and any open-questions
display governed by [[open-questions-gui]]) still behaves exactly as before. Guards
against an accidental coupling where "usually no section" assumptions break the
blocked case.

**Files to change.**
- None expected.

**Acceptance criteria.**
- [ ] A `blocked` artifact with a populated `## Open Questions` section still shows
      the blocked indicator and renders the questions as today.
- [ ] Existing web tests covering blocked/open-questions behaviour
      (e.g. `tests/web/artifact-blocked-questions.test.ts`) remain green with no
      `web/src/**` changes — see [[requirements-analyst-suppress-empty-open-questions]]
      test plan `-5-test`.
