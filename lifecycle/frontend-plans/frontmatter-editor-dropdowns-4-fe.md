---
title: 'Frontend Plan: Frontmatter Editor Dropdowns'
type: plan-frontend
status: draft
lineage: frontmatter-editor-dropdowns
parent: requirements/frontmatter-editor-dropdowns-2.md
---

# Frontend Plan: Frontmatter Editor Dropdowns

Replace the status text input with a `<select>` dropdown and add a new priority `<select>` dropdown in `FrontmatterEditor.vue`. No new dependencies. Native HTML `<select>` only.

## Milestone 1: Status Dropdown

### Description

Replace the `<input type="text">` for the Status field with a `<select>` element populated with the spec's status vocabulary in lifecycle-progression order.

Handle the edge case where `modelValue.status` contains a value not in the vocabulary: render it as a disabled `<option>` at the top of the list so it remains visible but cannot be re-selected once changed.

### Files to change

- `web/src/components/artifact/FrontmatterEditor.vue` — template and script

### Acceptance criteria

- [ ] The Status field renders as a `<select>` element, not a text input.
- [ ] The dropdown contains exactly 10 options in this order: `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked`.
- [ ] The selected value is bound to `modelValue.status`.
- [ ] On change, `update('status', value)` is called, emitting `update:modelValue` with the new status.
- [ ] If `modelValue.status` is not in the vocabulary, it appears as a disabled option at the top and the dropdown still functions.
- [ ] The `<select>` is wrapped in a `<label>` with a `.fm-label` span reading "Status".

## Milestone 2: Priority Dropdown

### Description

Add a new `<select>` element for the Priority field, placed immediately after the Status field. Options: an empty placeholder labelled "— none —", plus `normal` and `high`. Selecting the placeholder emits `undefined` so the key is omitted from frontmatter.

### Files to change

- `web/src/components/artifact/FrontmatterEditor.vue` — template and script

### Acceptance criteria

- [ ] A Priority `<select>` appears immediately after the Status field.
- [ ] Options are: `""` (labelled "— none —"), `normal`, `high`.
- [ ] The selected value is bound to `modelValue.priority ?? ''`.
- [ ] Selecting "— none —" calls `update('priority', undefined)`.
- [ ] Selecting "normal" or "high" calls `update('priority', value)`.
- [ ] The `<select>` is wrapped in a `<label>` with a `.fm-label` span reading "Priority".

## Milestone 3: Styling Consistency

### Description

Ensure both `<select>` elements visually match the existing `.fm-input` text fields. Add a `.fm-select` class (or extend `.fm-input`) that applies `appearance: none`, a custom caret via `background-image`, and appropriate `padding-right` to accommodate it.

### Files to change

- `web/src/components/artifact/FrontmatterEditor.vue` — `<style scoped>` block and template class attributes

### Acceptance criteria

- [ ] Both `<select>` elements have the same height, font-size, font-family, border, border-radius, and background as `.fm-input` text fields.
- [ ] Focus state shows `border-color: var(--color-accent)` matching `.fm-input:focus`.
- [ ] `appearance: none` is set so the native browser chrome is hidden.
- [ ] A CSS caret indicator is visible on the right side of each dropdown.
- [ ] No new npm dependencies are introduced.
- [ ] Existing fields (title, type, lineage, labels, release, sprint, depends_on, blocks) are visually unchanged and still function.

## Milestone 4: Type-Check Verification

### Description

Run `pnpm exec vue-tsc --noEmit` and `pnpm build` to confirm the component passes TypeScript type-checking and the production build succeeds.

### Files to change

None (verification step). Fix any type errors surfaced in `FrontmatterEditor.vue` if needed.

### Acceptance criteria

- [ ] `pnpm exec vue-tsc --noEmit` exits 0 with no errors in `FrontmatterEditor.vue`.
- [ ] `pnpm build` exits 0.
- [ ] The `ArtifactFrontmatter` type in `web/src/types/api.ts` already has `priority?: string` — no changes needed there.

## Cross-references

- Backend plan [[frontmatter-editor-dropdowns]] confirms no API changes are needed — the frontend reads/writes the same fields.
- Test plan [[frontmatter-editor-dropdowns]] covers integration tests for round-tripping status and priority values through the API.
