---
title: 'Frontmatter Editor: Priority Dropdown and Status Dropdown'
type: requirement
status: blocked
lineage: frontmatter-editor-dropdowns
priority: normal
parent: ideas/frontmatter-editor-dropdowns.md
labels:
    - enhancement
    - frontend
    - usability
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Frontmatter Editor: Priority Dropdown and Status Dropdown

## Problem

The frontmatter editor (`web/src/components/artifact/FrontmatterEditor.vue`) has two gaps:

1. **No priority field.** The `ArtifactFrontmatter` type includes an optional `priority` field and the backend stores it, but the editor never renders it. Users cannot set or change priority through the UI.

2. **Status is free-text.** Status is rendered as a plain `<input type="text">`, so users can enter any string — including values outside the spec vocabulary (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked`). Invalid statuses silently pass through and may break workflow transitions or confuse filtering.

Both issues slow editing and increase the risk of malformed frontmatter.

## Goals / Non-goals

### Goals

- Add a **priority** dropdown to the frontmatter editor populated with the valid priority values.
- Replace the status **text input** with a **dropdown** (`<select>`) populated with the full status vocabulary.
- Preserve the existing update-on-change behaviour (`update:modelValue` emit pattern).
- Apply consistent visual styling so both dropdowns match the existing `.fm-input` design tokens.

### Non-goals

- Extending the backend API or Go parser — both fields already exist in the data model.
- Validating status server-side (out of scope; the server already accepts any string). Server-side validation of the status vocabulary is a separate concern.
- Converting the `type` field to an editable dropdown — type remains read-only per current design.
- Adding dropdown behaviour to any other frontmatter fields (labels, release, sprint, depends_on, blocks).

## Detailed Requirements

### FR-1 Status dropdown

| Aspect | Detail |
|---|---|
| **Element** | Replace the `<input type="text">` for Status with a `<select>` element. |
| **Options** | Hard-coded list: `draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`, `blocked`. Order must match the list above (lifecycle progression). |
| **Selected value** | Bound to `modelValue.status`. |
| **Emit** | On change, call `update('status', value)` — same pattern as current text input. |
| **Styling** | Apply the `.fm-input` class so the select inherits the same dimensions, border, background, and focus ring as text inputs. Add `appearance: none` with a custom caret if needed to match the design language. |
| **Edge case — unknown value** | If `modelValue.status` contains a value not in the list (e.g. a legacy or hand-edited file), render it as an additional disabled option at the top so it is visible but not re-selectable once changed. |

### FR-2 Priority dropdown

| Aspect | Detail |
|---|---|
| **Element** | Add a new `<select>` element for Priority, placed immediately after the Status field. |
| **Options** | `normal`, `high`. Include an empty/placeholder option (`""`) labelled "— none —" so the field can be unset. |
| **Selected value** | Bound to `modelValue.priority ?? ''`. |
| **Emit** | On change, call `update('priority', value \|\| undefined)` — emit `undefined` when the placeholder is selected so the key is omitted from frontmatter. |
| **Styling** | Same `.fm-input` class treatment as FR-1. |

### FR-3 Visual consistency

- Both `<select>` elements must use the same height, font, border-radius, and colour variables as the existing `.fm-input` text fields.
- Focus state must show `border-color: var(--color-accent)` to match the existing `.fm-input:focus` rule.
- Add a shared `.fm-select` rule (or extend `.fm-input`) if `<select>` needs additional styles (e.g. `appearance`, padding-right for caret).

### NFR-1 No new dependencies

The implementation must use native `<select>` HTML elements. Do not introduce a third-party dropdown or combobox library.

### NFR-2 Accessibility

- Each `<select>` must be wrapped in a `<label>` with a visible `.fm-label` span, consistent with the existing fields.
- Keyboard navigation (Tab, Arrow keys, Enter) must work natively — no custom JS key handlers required for `<select>`.

## Acceptance Criteria

- [ ] The Status field in the frontmatter editor renders as a `<select>` dropdown with exactly 10 options matching the spec vocabulary.
- [ ] Selecting a status value emits `update:modelValue` with the chosen status.
- [ ] If an artifact has a status value not in the vocabulary, it is displayed as a disabled option and the dropdown still functions.
- [ ] A Priority dropdown appears after the Status field with options "— none —", "normal", "high".
- [ ] Selecting "— none —" for priority emits `undefined` (omits the key from frontmatter).
- [ ] Selecting "normal" or "high" emits the corresponding string value.
- [ ] Both dropdowns visually match the height, font, border, and focus styling of existing text inputs.
- [ ] No new npm dependencies are introduced.
- [ ] Existing fields (title, type, lineage, labels, release, sprint, depends_on, blocks) are unchanged and still function.
- [ ] The component passes TypeScript type-checking (`pnpm vue-tsc --noEmit`).

## Open Questions

- Should the priority vocabulary be extended beyond `normal` and `high`? (e.g. `low`, `critical`.) The current codebase only uses `normal` and `high`; the spec does not define the priority vocabulary. Confirm with product owner.
- Should the status dropdown disable options that are invalid transitions from the current status (e.g. cannot jump from `draft` to `done`)? This would couple the editor to workflow rules and is likely a separate feature, but worth confirming scope.
