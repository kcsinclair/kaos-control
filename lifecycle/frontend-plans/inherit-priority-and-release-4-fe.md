---
title: "Frontend Plan — Inherit Priority and Release Through Lineage"
type: plan-frontend
status: approved
lineage: inherit-priority-and-release
parent: lifecycle/requirements/inherit-priority-and-release-2.md
---

# Frontend Plan — Inherit Priority and Release Through Lineage

## Overview

Inheritance is implemented entirely server-side at creation time (see the
[[inherit-priority-and-release]] backend plan). The frontend's job is therefore
narrow and mostly **subtractive**:

1. **Don't defeat inheritance.** Any create flow that posts to
   `POST /api/p/:project/artifacts` with a `parent` must **omit** `priority` and
   `release` (or send them empty) when the user hasn't explicitly chosen one — an
   explicit value always wins server-side (FR-4), so a frontend that pads the
   request with a default `priority: "normal"` would silently override the
   inherited value.
2. **No inheritance indicator.** Goal 5 / FR-10 asked for an "inherited vs
   overridden" indicator in the editor, but the requirement's **Resolved
   Question 4 answers "No visual difference is required."** This indicator is
   therefore **out of scope** — do not build a badge, hint, tooltip, or
   reset-to-inherited affordance.
3. **Reuse existing controls.** The inline priority/release edit controls
   ([[artefact-priority-inline-edit]], [[inline-release-display-edit]]) and the
   list columns ([[artefacts-list-release-priority-columns]]) already render and
   override these fields; they need no change beyond verifying they correctly
   display values that arrived via inheritance.

The net frontend change is small: an audit of create-request payloads plus
verification that inherited values display correctly. No new components.

---

## Milestone 1 — Audit create-artifact payloads to not override inheritance

### Description

Find every frontend path that issues `POST .../artifacts` with a `parent` and
ensure it does not send a non-empty `priority`/`release` the user did not choose.
Where a create form defaults `priority` to `"normal"`, change it to omit the
field (send empty / leave it out of the JSON body) when the user leaves it at the
default, so the server inherits from the parent. The agent/idea-generate preview
already receives inherited values from the backend (backend Milestone 4); the
persist step must forward them verbatim rather than re-defaulting.

### Files to change

- `web/src/stores/artifacts.ts` — the create action that builds the `POST .../artifacts` body; ensure `priority`/`release` are only included when explicitly set.
- Any create dialogs/modals that assemble frontmatter for a child (e.g. new-from-parent / "create child" flows) — audit and, where present, stop padding `priority: "normal"`.
- `web/src/stores/testing.ts` and other stores that call the create endpoint — audit for the same defaulting.

### Acceptance criteria

- [ ] Creating a child artifact through the UI without choosing a priority sends a request with no (or empty) `priority`, and the persisted child inherits the parent's value.
- [ ] Creating a child without choosing a release sends no (or empty) `release`, and the child inherits the parent's value.
- [ ] When the user explicitly picks a `priority`/`release` in a create form, that value is sent and preserved (FR-4).
- [ ] No create path hard-codes `priority: "normal"` for a parented artifact.

---

## Milestone 2 — Verify inherited values render in existing views

### Description

Verify (no new UI) that values arriving via inheritance render identically to
explicitly-set values in: the inline priority control, the inline release control,
and the artifacts list `release`/`priority` columns. Since inheritance is just a
normal frontmatter value to the client, this should already work — this milestone
guards against any client-side assumption that `priority` is always present or
always `"normal"`.

### Files to change

- None expected. Inspect `web/src/components/.../*` priority and release renderers and the artifacts list columns referenced by [[artefacts-list-release-priority-columns]].

### Acceptance criteria

- [ ] A child whose `priority` was inherited shows that priority in the inline control and list column (not a hard-coded `normal`).
- [ ] A child whose `release` was inherited shows that release in the inline control and list column.
- [ ] Editing an inherited value via the inline controls updates only that artifact (relies on backend FR-9); the UI reflects the new value after the `artifact.indexed` WS event.

---

## Milestone 3 — Confirm no inheritance indicator is added (descope guard)

### Description

Explicitly record that FR-10 / Goal 5 are descoped per Resolved Question 4 ("No
visual difference is required"). No "inherited"/"overridden" badge, tooltip, or
reset affordance is to be implemented. This milestone exists so a later reviewer
does not re-introduce the indicator believing FR-10 is unmet.

### Files to change

- None.

### Acceptance criteria

- [ ] The editor shows priority/release with no inherited-vs-overridden indicator.
- [ ] No new component, store flag, or parent-comparison logic is added for indicator display.
- [ ] `pnpm exec vue-tsc --noEmit` passes with no new errors.

---

## Cross-links

- Backend (where inheritance actually happens): [[inherit-priority-and-release]] backend plan.
- Reused inline controls: [[artefact-priority-inline-edit]], [[inline-release-display-edit]].
- List columns displaying the inherited metadata: [[artefacts-list-release-priority-columns]].
- Requirement / idea lineage: [[inherit-priority-and-release]].
