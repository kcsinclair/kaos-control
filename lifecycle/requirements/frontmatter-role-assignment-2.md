---
title: Frontmatter Role-Based Assignment Control
type: requirement
status: draft
lineage: frontmatter-role-assignment
created: "2026-04-28"
priority: normal
parent: lifecycle/ideas/frontmatter-role-assignment.md
labels:
    - feature
    - frontend
    - workflow
---

# Frontmatter Role-Based Assignment Control

## Problem

The frontmatter panel (`FrontmatterPanel.vue`) currently renders the `assignees` field as read-only text — role and who pairs are displayed but cannot be added, changed, or removed through the UI. Users and agents must hand-edit YAML frontmatter to assign work, which is error-prone and allows values that don't match the project's configured roles.

There is no UI path to populate the `assignees` field on a new or existing artifact, and no validation that a chosen role actually exists in `lifecycle/config.yaml`.

## Goals / Non-goals

### Goals

1. Provide an interactive assignment control in the frontmatter panel that lets users add, edit, and remove `assignees` entries.
2. Populate the role picker from the project's configured `roles` list so only valid roles can be selected.
3. Support both `agent` and human-user values for the `who` field.
4. Persist changes through the existing artifact save flow (`PUT /artifacts/*`).

### Non-goals

- Changing the `assignees` YAML schema (the existing `role`/`who` pair structure is kept).
- Adding notification or routing logic when assignments change — that is a separate concern.
- Restricting which users can assign which roles (no RBAC on the assignment action itself).
- Bulk assignment across multiple artifacts.

## Detailed Requirements

### Functional

**FR-1 Role picker control**
The frontmatter panel must include an "Assignees" section with controls to:
- Add a new assignee row.
- Select a role from a dropdown populated with the project's `roles` array (sourced from `config.yaml` via the backend API).
- Enter a `who` value — either the literal string `agent` or a user identifier (email). A dropdown or combo-box should offer known users from the project's `users` list as suggestions, plus an explicit `agent` option.
- Remove an existing assignee row.

**FR-2 Backend roles endpoint**
The backend must expose the project's role list to the frontend. Either:
- An explicit `GET /api/projects/:id/roles` endpoint returning `string[]`, or
- Include `roles` in the existing project-config response if one already exists.

The frontend must fetch this list at mount time (or from a Pinia store if project config is already cached) and use it to populate the role dropdown.

**FR-3 Validation**
- The role field must be constrained to values present in the project's `roles` list. Free-text entry of roles is not permitted.
- The `who` field must be non-empty.
- The UI must prevent saving an assignee row with an empty role or empty who.

**FR-4 Persistence**
When the user saves the artifact, the `assignees` array in the frontmatter YAML must reflect the current state of the assignment control. No additional API changes are needed beyond FR-2 — the existing `PUT /artifacts/*` handler already round-trips frontmatter fields including `assignees`.

**FR-5 Display consistency**
When the panel is in read-only mode (e.g. viewing without edit permissions), assignees continue to render as they do today — role and who as styled text.

### Non-functional

**NFR-1 Performance**
Fetching the roles list must not add a blocking request to the artifact-load critical path. Roles should be cached in the project store after first fetch.

**NFR-2 Accessibility**
The role dropdown and who input must be keyboard-navigable and have appropriate `aria-label` attributes.

## Acceptance Criteria

- [ ] The frontmatter panel shows an "Assignees" section with an "Add assignee" button when in edit mode.
- [ ] Clicking "Add assignee" inserts a new row with a role dropdown and a who input.
- [ ] The role dropdown lists exactly the roles from the project's `config.yaml` `roles` array.
- [ ] The who input suggests known user emails from `config.yaml` `users` and offers an `agent` option.
- [ ] Selecting a role and who, then saving the artifact, persists the `assignees` array in the file's YAML frontmatter.
- [ ] Removing an assignee row and saving removes that entry from the YAML.
- [ ] Attempting to save with an empty role or empty who is prevented (validation error shown).
- [ ] In read-only mode, the assignees render as styled text (current behaviour preserved).
- [ ] The roles list is fetched once and cached; subsequent renders do not re-fetch.
- [ ] `go vet` and `vue-tsc --noEmit` pass after all changes.
- [ ] Related: [[frontmatter-role-assignment]]

## Open Questions

1. Should the `who` field for human users accept free-text emails, or should it be restricted to emails present in the project's `users` list? (Free-text is more flexible for new team members; restricted is safer.)

> Who is not required just the role, when a human logs in as product-owner they should look for work assisgned to that role.

2. Should there be a visual distinction between agent-assigned and human-assigned rows (e.g. an icon or badge)?

> Not required as the role could be human or agent depending on the team.
