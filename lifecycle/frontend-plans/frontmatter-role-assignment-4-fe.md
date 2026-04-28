---
title: "Frontend Plan: Assignee Editing in Frontmatter Panel"
type: plan-frontend
status: in-development
lineage: frontmatter-role-assignment
parent: lifecycle/requirements/frontmatter-role-assignment-2.md
---

# Frontend Plan: Assignee Editing in Frontmatter Panel

This plan adds interactive assignee add/edit/remove controls to the frontmatter editor, populated from the project's configured roles via the new `GET /roles` endpoint (see [[frontmatter-role-assignment]]).

## Milestone 1 — Roles API client and Pinia store

### Description

Create the API client function and a lightweight Pinia store to fetch and cache the project's roles and users list. The store must fetch once on first access and serve from cache on subsequent reads (NFR-1).

### Files to change

- `web/src/api/config.ts` — Add `getRoles(project: string): Promise<{ roles: string[]; users: { email: string; roles: string[] }[] }>`. This calls `GET /api/p/{project}/roles` using the existing `api.get<T>` wrapper.
- `web/src/stores/projectConfig.ts` — New file. Create `useProjectConfigStore` with:
  - State: `roles: string[]`, `users: { email: string; roles: string[] }[]`, `loaded: boolean`.
  - Action: `fetchRoles(project: string)` — calls `getRoles()` if `!loaded`, sets state, sets `loaded = true`.
  - Getter: `availableWhoOptions` — returns a deduplicated list combining all user emails plus the literal `"agent"` string, for use in the `who` picker.

### Acceptance criteria

- [ ] `useProjectConfigStore().fetchRoles(project)` fetches from the API on first call.
- [ ] Subsequent calls to `fetchRoles()` return cached data without a network request.
- [ ] `availableWhoOptions` returns `["agent", ...emails]` with `"agent"` always first.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 2 — Assignee editor component

### Description

Build a reusable `AssigneeEditor.vue` component that renders the list of assignees with add/edit/remove controls. This component is used inside `FrontmatterEditor.vue` in edit mode.

### Files to change

- `web/src/components/artifact/AssigneeEditor.vue` — New file. Component contract:
  - **Props:** `modelValue: ArtifactAssignee[]`, `roles: string[]`, `whoOptions: string[]`.
  - **Emits:** `update:modelValue` (standard v-model pattern).
  - **Template:**
    - For each assignee in the array, render a row containing:
      - A `<select>` dropdown for `role`, populated from the `roles` prop. Must have `aria-label="Role"`.
      - A combo-box (text `<input>` with a `<datalist>`) for `who`, with suggestions from `whoOptions` prop. Must have `aria-label="Assignee"`.
      - A remove button (X icon from lucide-vue-next) that splices the entry.
    - An "Add assignee" button below the rows that pushes `{ role: '', who: '' }` to the array.
  - **Validation:** The component itself does not block saves; validation is handled by the parent (Milestone 3).

### Acceptance criteria

- [ ] Clicking "Add assignee" appends a new empty row.
- [ ] The role dropdown lists exactly the values from the `roles` prop.
- [ ] The `who` input offers suggestions from `whoOptions` including `"agent"`.
- [ ] Clicking the remove button on a row removes that assignee from the array.
- [ ] Changes emit `update:modelValue` so the parent's v-model stays in sync.
- [ ] Role dropdown and who input have `aria-label` attributes (NFR-2).
- [ ] Keyboard navigation works: Tab moves between fields, Enter on "Add assignee" adds a row.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 3 — Integrate into FrontmatterEditor and add validation

### Description

Wire `AssigneeEditor` into the existing `FrontmatterEditor.vue` so that assignees are editable during artifact editing. Add client-side validation that prevents saving when any assignee row has an empty role or empty who (FR-3).

### Files to change

- `web/src/components/artifact/FrontmatterEditor.vue` — 
  1. Import `AssigneeEditor` and `useProjectConfigStore`.
  2. In the component's `onMounted` (or setup), call `projectConfigStore.fetchRoles(project)`.
  3. Add an "Assignees" section after the existing fields (labels, depends_on, blocks area). Render `<AssigneeEditor v-model="frontmatter.assignees" :roles="projectConfigStore.roles" :who-options="projectConfigStore.availableWhoOptions" />`.
  4. In the save/validation logic, check each assignee: if `role` is empty or `who` is empty, prevent save and display a validation error message below the offending row (or as a banner).

### Acceptance criteria

- [ ] The FrontmatterEditor shows an "Assignees" section with the `AssigneeEditor` component when in edit mode.
- [ ] Roles dropdown is populated from the project config (fetched once and cached).
- [ ] Saving with an empty role or empty who shows a validation error and does not submit.
- [ ] Saving with valid assignees persists the `assignees` array to the artifact's YAML frontmatter via the existing `PUT /artifacts/*` flow.
- [ ] Removing all assignees and saving results in the `assignees` field being removed or set to `[]` in the YAML.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

## Milestone 4 — Read-only display preservation

### Description

Ensure `FrontmatterPanel.vue` (the read-only view) continues to render assignees as styled text, unchanged from current behaviour (FR-5).

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue` — No functional changes expected. Verify that the existing assignee rendering (lines 43-51) still works correctly after the editor changes. If the `assignees` array shape has changed at all (it should not have), update the display accordingly.

### Acceptance criteria

- [ ] In read-only mode (artifact detail view, not editing), assignees render as `role: who` styled text.
- [ ] The read-only display does not show edit controls (no dropdowns, no add/remove buttons).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
