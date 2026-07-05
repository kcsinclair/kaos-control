---
title: GUI Flow for Resolving Open Questions on Blocked Artefacts
type: requirement
status: blocked
lineage: open-questions-gui
created: "2026-07-05T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/open-questions-gui.md
labels:
    - frontend
    - workflow
    - usability
    - artefacts
    - onboarding
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

## Problem

When an agent cannot proceed on ambiguous or incomplete input it appends a `## Open Questions` list to the artefact. The existing auto-block logic ([[agent-questions-trigger-blocked-status]], `internal/index/autoblock.go`) detects that section and transitions the artefact to `blocked` with a `product-owner` assignee. To unblock it, a human must today perform a fragile manual ritual entirely in the raw markdown editor:

1. Read each question in the `## Open Questions` list.
2. Hand-type an answer under each question, prefixing every line with `> ` (blockquote) with correct blank-line spacing.
3. Rename the heading `## Open Questions` → `## Resolved Questions` — miss this and the auto-block silently re-blocks the artefact.
4. Rely on the reviewer to verify and approve so work continues.

Every step is manual and error-prone: inconsistent answer formatting, and — most damagingly — forgetting the heading rename, which leaves the artefact silently stuck. New users in particular struggle to discover that action is even required and to complete the ritual correctly. This is a usability and discoverability gap, not a state-machine gap: the backend transition half is already built and must not be re-implemented.

## Goals / Non-goals

### Goals

- **Discoverability.** Make it obvious, from a global indicator and from the artefact itself, that an artefact is `blocked` awaiting a human's answers.
- **Guided resolution.** Provide a step-through UI (one question per panel) that captures an answer for each open question without the user hand-editing markdown.
- **House-format write-back.** Write each answer into the artefact immediately after its question, using the configured answer format (default: blockquote), consistent with the current manual convention.
- **Safe completion.** Rename `## Open Questions` → `## Resolved Questions` only when every question is answered, letting the existing auto-unblock (→ `draft`) fire. Never set status directly from the GUI.
- **Partial progress.** Allow answers to be saved and resumed across sessions without unblocking prematurely.
- **Correct routing.** After resolution, offer the appropriate deliberate follow-up based on who raised the questions (approve-and-continue vs. requeue the raising developer role), never automatically.
- **Permissions.** Allow the `product-owner` assignee or any authorised role on the artefact to resolve.

### Non-goals

- Re-implementing or changing the auto-block / auto-unblock state transitions in `internal/index/autoblock.go` — the GUI only edits artefact body content.
- Setting artefact `status` directly from the GUI (status changes remain a side effect of the body edit through the existing indexer).
- Skipping or automating the reviewer approval gate — approval remains a deliberate human action.
- Changing the agent instructions that produce the `## Open Questions` list (top-level list items under an `## Open Questions` heading remain the contract).
- Building a general-purpose rich-text markdown editor; this is a focused question/answer flow.
- Bulk resolution across multiple artefacts in one action.

## Detailed Requirements

### Functional

1. **Global awaiting-answers indicator.**
   - The menu bar displays a count/badge of artefacts currently `blocked` with a non-empty `## Open Questions` section (e.g. "N awaiting your answers"), consistent with the existing agent/queue indicators ([[agents-indicator-in-menu-bar]]).
   - The badge is hidden (or shows no count) when zero artefacts await answers.
   - The count updates live in response to the existing `artifact.indexed` / status WebSocket events, without page refresh.
   - Activating the indicator navigates the user to the set of awaiting artefacts (e.g. a filtered list).

2. **Per-artefact call-to-action.**
   - When an open artefact is `blocked` and has a non-empty `## Open Questions` section, its detail/top-bar shows a clear banner: *"This artefact is blocked awaiting your answers"* with a **Resolve Questions** action.
   - The **Resolve Questions** button appears whenever the artefact body contains an `## Open Questions` list, regardless of how the block was raised.

3. **Guided resolution modal.**
   - Parses the `## Open Questions` section, treating each top-level list item as one question.
   - Presents one question per panel: question text (rendered markdown), an answer text field, Back / Next navigation, and a progress indicator (e.g. "2 of 5").
   - Pre-populates the answer field with any previously saved answer for that question so partially-answered sessions resume in place.
   - The **Complete/Finish** action is enabled only when every question has a non-empty answer.

4. **Answer write-back format.**
   - Answers are written into the artefact body with each question immediately followed by its answer.
   - The answer rendering format is configurable; the default is a blockquote (`> …`) with blank-line spacing matching the existing house convention.
   - The default format must reproduce, byte-for-byte in spirit, what a correct manual edit produces today (so downstream tooling and diffs stay consistent).

5. **Partial save / resume.**
   - The user may save progress with some questions unanswered; the artefact stays `blocked` (heading remains `## Open Questions`) until all are answered.
   - Saved partial answers persist across page reloads and sessions and are visible when the modal is re-opened.

6. **Completion and unblock.**
   - When all questions are answered and the user completes, the system writes all answers and renames the heading `## Open Questions` → `## Resolved Questions` in a single write.
   - The GUI does not set `status`; the existing indexer auto-unblock transitions the artefact to `draft` as a result of the heading no longer matching `## Open Questions`.

7. **Routing after resolution.**
   - The GUI determines the originating role from the artefact's type/assignees.
   - **Requirements case (default):** after resolution the artefact returns to `draft` and awaits the reviewer's deliberate approval, which hands off to the planning-analyst. The GUI offers an "approve & continue" affordance but does not auto-approve.
   - **Developer-raised case:** after resolution the GUI offers to requeue the originating developer role so that agent continues with the answers now in context.
   - No follow-up (approval or requeue) is triggered automatically; each requires an explicit user action.

8. **Permissions.**
   - The `product-owner` assignee and any other authorised role on the artefact may open and complete the resolution flow.
   - Users without an authorised role on the artefact do not see the resolve action (or see it disabled) and cannot write answers.

9. **Re-opening (discouraged but possible).**
   - An item under `## Resolved Questions` can be edited or re-opened. Re-opening a question (moving it back under `## Open Questions` such that the section is non-empty) allows the artefact to re-block via the existing auto-block path.

### Non-functional

1. **No status writes from the client.** All status changes are a side effect of the body edit via the existing indexer/auto-block logic; the resolve flow issues only artefact-body writes through the standard artefact write API (`PUT /artifacts/*`).
2. **Live consistency.** The badge and per-artefact banner reflect the true blocked/awaiting state within one indexer cycle of any change (write-back, external edit, or manual edit), driven by existing WebSocket events.
3. **Idempotent, loss-safe write-back.** Writing answers must not corrupt or drop unrelated body content (frontmatter, other sections, existing prose) and must be safe to repeat (re-saving the same answers yields an equivalent body).
4. **Configurable format isolation.** The answer format is read from configuration; changing it affects only newly written answers and does not require code changes.
5. **Accessibility & discoverability.** The banner and badge are visible without hover, and the modal is keyboard-navigable (Back/Next/complete reachable and focus-managed).
6. **Graceful parsing.** If the `## Open Questions` section is malformed or empty, the resolve action is not offered (or clearly reports nothing to resolve) rather than erroring.

## Acceptance Criteria

- [ ] A `blocked` artefact with a non-empty `## Open Questions` section is counted in a menu-bar "awaiting answers" badge that updates live and hides at zero. Related: [[agents-indicator-in-menu-bar]].
- [ ] Opening such an artefact shows a "blocked awaiting your answers" banner with a **Resolve Questions** action.
- [ ] The **Resolve Questions** button is shown whenever the body contains an `## Open Questions` list and hidden otherwise.
- [ ] The modal renders exactly one panel per top-level list item under `## Open Questions`, with question text, answer field, Back/Next, and an "X of N" progress indicator.
- [ ] Answers can be saved with some questions still unanswered; the artefact remains `blocked` and the heading stays `## Open Questions`.
- [ ] Re-opening the modal on a partially-answered artefact pre-populates previously entered answers.
- [ ] Completion is disabled until every question has a non-empty answer.
- [ ] On completion, each question in the body is immediately followed by its answer in the configured format (default blockquote), and the heading is renamed `## Open Questions` → `## Resolved Questions` in a single write.
- [ ] After the rename, the existing auto-unblock transitions the artefact to `draft` without the GUI issuing any status change. Related: [[agent-questions-trigger-blocked-status]].
- [ ] The GUI never writes the `status` field directly; verified by inspecting the write payload (body-only).
- [ ] Changing the configured answer format changes the rendering of newly written answers without code changes; default matches the current manual house convention.
- [ ] For a requirements-lineage artefact, resolution routes to `draft` and offers a deliberate "approve & continue" affordance (no auto-approval); reviewer approval hands off to the planning-analyst.
- [ ] For a developer-raised artefact, resolution offers to requeue the originating developer role; the requeue is explicit, not automatic.
- [ ] The `product-owner` assignee and any other authorised role can complete the flow; an unauthorised user cannot write answers.
- [ ] An item under `## Resolved Questions` can be edited/re-opened; re-opening restores a non-empty `## Open Questions` section that re-blocks via existing logic.
- [ ] Write-back preserves all unrelated body content and frontmatter; re-saving the same answers is idempotent.
- [ ] The badge, banner, and completion routing are exercised by an end-to-end/integration test (write questions → block → resolve via UI flow → `draft`).

## Open Questions

1. **Answer-format configuration scope.** Is the answer format an app-level config key (`~/.kaos-control/config.yaml`), a per-project key (`lifecycle/config.yaml`), or both with project overriding app? The default (blockquote) is agreed; only the location/precedence is unspecified.
2. **"Authorised role" definition.** The idea says the product-owner *or any authorised role* may resolve. What is the precise authorisation rule — any role listed in the artefact `assignees`, any role permitted to write that artefact type, or a specific configured set? This affects who sees the Resolve action.
3. **Where the badge navigates.** Should the menu-bar count deep-link to a dedicated filtered "awaiting answers" list, reuse an existing list/kanban filter, or open the single artefact when N == 1?
4. **Partial-answer persistence medium.** Should partial answers persist by writing them into the artefact body incrementally (kept under `## Open Questions`, hence still blocked) or held in client/server-side draft state separate from the artefact until completion? This determines what an external reader sees mid-resolution.
5. **Developer-role recovery.** For the developer-raised case, exactly which field identifies the originating role to requeue — the artefact `type`, the current `assignees`, or run history — when multiple roles have touched the artefact?
