---
title: GUI View for Resolving Open Questions
type: idea
status: planning
lineage: open-questions-gui
created: "2026-05-19T15:23:33+10:00"
priority: high
labels:
    - feature
    - frontend
    - usability
    - workflow
    - artifacts
    - onboarding
release: KC-Release4
---

# GUI View for Resolving Open Questions

## Essence

When an agent can't proceed on ambiguous or incomplete input, it stops and
appends a `## Open Questions` list to the artefact, sets `status: blocked`, and
assigns the product-owner (per the "If you get stuck" instruction in
[config.yaml](../config.yaml)). A human then has to answer those questions
before work can continue.

Today that answering is a **fiddly manual ritual**, blocked artefacts are visible 
in the list or kanban view as well as other views and usually seen so they can be managed.
New users have struggled a bit with the resolving questions step, and need help.
This idea makes resolving open questions **smooth and discoverable**: highlight that
answers are needed, walk the user through each question, write the answers back
in the house format, unblock the artefact, and route it to the right next role —
replacing the hand-editing entirely.

## The manual ritual to remove (what I do today)

1. Open the artefact and read the `## Open Questions` list.
2. Under each question, hand-type the answer, prefixing every line with `> `
   (blockquote) so it's visibly a response, with blank-line spacing.
3. Rename the heading `## Open Questions` → `## Resolved Questions` — **otherwise
   the automation re-blocks it** (see below).
4. It returns to `draft`; the reviewer verifies and approves; it moves on.

Every step is manual and easy to get wrong (miss the rename → it silently
re-blocks; inconsistent formatting).

## Already built — do not re-implement the status change

The status machine is automatic ([internal/index/autoblock.go](../../internal/index/autoblock.go)):

- Artefact gains a non-empty `## Open Questions` section → auto-transitions to
  `blocked` (+ product-owner assignee).
- That section removed or **renamed** → auto-unblocks back to `draft`.

So the GUI must **not** set status itself. It writes the answers and renames the
heading to `## Resolved Questions`; the existing auto-unblock then lands the
artefact at `draft`. (This is exactly the manual rename I do today — automated.)

## Proposed flow

1. **Discoverability (the new-user piece).** Surface artefacts that are
   `blocked` on open questions so a human knows action is needed:
   - a count/badge in the menu bar (e.g. "N awaiting your answers"),
   - a clear call-to-action on the artefact itself: *"This artefact is blocked
     awaiting your answers"* → **Resolve Questions**.
2. **Resolve Questions button** on the artefact's top bar, shown when it has an
   `## Open Questions` list.
3. **Guided modal, one question per panel** — the question text, an answer
   field, Back / Next, and a progress indicator (e.g. "2 of 5"). Questions are
   always a list under the heading (guaranteed by the agent instructions), so
   each top-level list item = one panel.
4. **Write-back** — render **each question immediately followed by its answer**
   in the artefact. The answer format is **configurable** (the default is a
   blockquote `> …` with spacing, matching the house convention). Answers may be
   **saved partially and resumed** later.
5. **Complete only when all answered** — the heading is renamed
   `## Open Questions` → `## Resolved Questions` **only once every question has an
   answer**; until then the artefact stays `blocked`. That rename triggers the
   existing **auto-unblock → `draft`**. Resolving does **not** skip the review
   gate — the reviewer still verifies and **approves deliberately** before the
   artefact proceeds.

## Routing after resolution

Where the artefact goes next depends on **who raised the questions**:

- **Requirements (the ~99% case)** — raised by the requirements-analyst,
  assigned product-owner. Resolve → `draft` → reviewer approves → picked up by
  the **planning-analyst** (the next role). The approval is the deliberate hand-off.
- **Developer-raised questions** — answer, then **requeue the originating
  developer role** so that agent continues with the answers now in context.

The originating role is recoverable from the artefact's type/assignees; after
answers are written the GUI should offer the appropriate follow-up (approve &
continue, or requeue to the raising role) — still gated on the deliberate human
approval, never automatic.

## Design decisions

- **Partial answering** — permitted. Answers can be saved and resumed; the
  artefact stays `blocked` until *all* questions are answered (only a fully
  answered section is renamed and thereby unblocked).
- **Answer format** — configurable (blockquote-under-each-question is the
  default house style).
- **Permissions** — the product-owner assignee **or any authorised role** on
  the artefact may resolve.
- **Re-opening** — discouraged but possible: an item in `## Resolved Questions`
  can be edited / re-opened.

## Related

- [[agent-questions-trigger-blocked-status]] — the already-done backend half
  (questions → `blocked`); this idea is its human-facing counterpart.
- [[agents-indicator-in-menu-bar]] — the discoverability badge could live
  alongside the existing agent/queue indicators.
