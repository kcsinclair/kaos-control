---
title: "Frontend Plan: Open-Questions Resolution GUI"
type: plan-frontend
status: done
lineage: open-questions-gui
parent: lifecycle/requirements/open-questions-gui-2.md
---

# Frontend Plan — Open-Questions Resolution GUI

Vue 3 + TypeScript UI for the guided open-questions resolution flow in
[[open-questions-gui]] (requirement `open-questions-gui-2`). This plan consumes
the read/compute endpoints added by the [[open-questions-gui]] backend plan
(`-3-be`) and persists **only artefact-body edits** through the existing
`updateArtifact` PUT — the GUI never writes `status` (NFR1). Verified by the
[[open-questions-gui]] test plan (`-5-test`).

Existing conventions to follow: the menu-bar queue badge in
`web/src/components/layout/AppHeader.vue` (see [[agents-indicator-in-menu-bar]]),
the artifacts store/api (`web/src/stores/artifacts.ts`,
`web/src/api/artifacts.ts`), the WS store, and `markdown-it` for rendering.

## Milestone 1 — Awaiting-answers store + live count

**Description.** Track the number of artefacts `blocked` with a non-empty
`## Open Questions` section, kept live off existing WebSocket events (FR1,
NFR2). No polling loop beyond an initial fetch.

**Files to change.**
- `web/src/api/artifacts.ts` — add `fetchAwaitingAnswersCount()` calling
  `GET …/artifacts?awaiting_answers=true&count_only=true`.
- `web/src/stores/artifacts.ts` (or a small dedicated store) — hold
  `awaitingAnswersCount`; fetch once on auth/project load; recompute on
  `artifact.indexed` / `status_changed` WS events (subscribe via the existing
  ws store) with light debounce.

**Acceptance criteria.**
- Count matches the number of blocked+open-questions artefacts on load.
- Creating/answering questions updates the count without a page refresh, within
  one indexer cycle.
- Count is `0` (not stale) once the last such artefact is resolved.

## Milestone 2 — Menu-bar badge

**Description.** Surface the count in the header next to the existing queue
badge; hide it at zero; activating it deep-links to the blocked list (Resolved
Q3, FR1).

**Files to change.**
- `web/src/components/layout/AppHeader.vue` — add an "awaiting answers" badge
  (reuse the queue-badge styling/tooltip idiom) bound to
  `awaitingAnswersCount`; render only when `> 0`; `router-link` to
  `/artifacts?status=blocked&awaiting=1`.

**Acceptance criteria.**
- Badge shows "N" with an accessible label (visible without hover, NFR5); shows
  a tooltip like "N awaiting your answers".
- Badge is absent/empty when the count is zero.
- Clicking navigates to the filtered artefacts list (Milestone 3 below).

## Milestone 3 — Blocked/awaiting list filter

**Description.** The artefacts list honours the query so the badge lands on the
current blocked items (Resolved Q3: "navigate to the artefact list using a
query to show current blocked items").

**Files to change.**
- The artifacts list view (`web/src/views/…` artifacts list) and its api call —
  read `status=blocked` (and optional `awaiting=1`) from the route query and
  pass through to the list request; show a heading/empty-state that makes the
  "awaiting your answers" intent clear.

**Acceptance criteria.**
- Visiting `/artifacts?status=blocked&awaiting=1` lists only blocked artefacts
  with open questions.
- The filter is reflected in existing list filter UI state (not a hidden mode).

## Milestone 4 — Per-artefact banner + Resolve action

**Description.** On the artefact detail/editor, show a clear call-to-action when
the body has a non-empty `## Open Questions` section (FR2), gated by permission
(Resolved Q2: product-owner for now, FR8).

**Files to change.**
- `web/src/components/artifact/…` (detail/top-bar component) + the artifact
  editor view — fetch `GET …/artifacts/*path/open-questions`; when
  `questions.length > 0`, render a banner *"This artefact is blocked awaiting
  your answers"* with a **Resolve Questions** button. The button appears
  whenever an `## Open Questions` list is present, regardless of how the block
  was raised (FR2).
- Permission gate: hide or disable the action for users without the
  `product-owner` role (use existing role/auth store); an unauthorised user
  cannot open the modal.

**Acceptance criteria.**
- Banner + button appear iff the body has a non-empty `## Open Questions`
  section; hidden when the section is absent, empty, or malformed (NFR6).
- Banner is visible without hover and keyboard-focusable (NFR5).
- A non-product-owner does not see (or cannot activate) the Resolve action.

## Milestone 5 — Guided resolution modal

**Description.** A focused step-through modal (not a general markdown editor,
Non-goal) that captures one answer per question and writes back via the backend
builder + existing PUT (FR3–FR6).

**Files to change.**
- `web/src/components/artifact/ResolveQuestionsModal.vue` (new) — one panel per
  question: markdown-rendered question text (`markdown-it`), an answer
  `textarea`, **Back / Next**, and an "X of N" progress indicator; pre-populate
  each answer from the endpoint's `answer` field (resume, FR3/FR5); **Finish**
  enabled only when every answer is non-empty (FR3).
- `web/src/api/artifacts.ts` — `previewOpenQuestions(path, answers, complete)`
  → `{body}`, then persist by calling the existing `updateArtifact(path,
  {frontmatter, body})` with the **unchanged frontmatter** and returned body
  (NFR1 — body-only; no status).
- A composable `web/src/composables/useOpenQuestions.ts` to hold answer state,
  save (partial) and complete actions.

**Save/complete semantics.**
- **Save (partial):** `previewOpenQuestions(path, answers, complete=false)` →
  PUT; heading stays `## Open Questions`, artefact remains `blocked` (FR5). Safe
  to invoke repeatedly (idempotent via backend builder).
- **Finish:** `previewOpenQuestions(path, answers, complete=true)` → PUT; the
  returned body has the heading renamed to `## Resolved Questions`; the existing
  indexer auto-unblocks to `draft` (FR6) — the GUI issues no status write.

**Acceptance criteria.**
- Exactly one panel per top-level question; Back/Next and "X of N" work;
  keyboard-navigable with managed focus (NFR5).
- Re-opening a partially answered artefact pre-fills saved answers.
- Finish is disabled until all answers are non-empty.
- The PUT payload contains only `frontmatter` (unchanged) + `body`; no `status`
  change is ever sent (asserted in the test plan).
- After Finish, the artefact transitions to `draft` via the existing WS event
  and the banner/badge clear.

## Milestone 6 — Post-resolution routing affordance

**Description.** After a successful Finish, offer the correct **deliberate**
follow-up based on who raised the questions (FR7, Resolved Q5). Nothing is
triggered automatically.

**Files to change.**
- `web/src/components/artifact/…` (post-completion prompt) — determine the case
  from the artefact `type`/`assignees` (Resolved Q5: creator role vs actioning
  role):
  - **Requirements case (default, `type: requirement`):** artefact is now
    `draft`; show an **"Approve & continue"** affordance that uses the existing
    transition/approve action (which hands off to the planning-analyst). No
    auto-approval.
  - **Developer-raised case (`plan-*` raised the questions):** show a
    **"Requeue &lt;role&gt;"** affordance that uses the existing queue/run API to
    re-queue the originating developer role so that agent resumes with the
    answers in context. No auto-requeue.

**Acceptance criteria.**
- Requirement-lineage artefact shows the approve-and-continue affordance and is
  not auto-approved; approval hands to the planning-analyst.
- Developer-raised artefact shows an explicit requeue affordance targeting the
  originating role; requeue is user-initiated.
- Neither follow-up fires without an explicit click.
