---
title: "Frontend Plan: Frontmatter Created Date"
type: plan-frontend
status: done
lineage: frontmatter-created-date
parent: ideas/frontmatter-created-date.md
labels:
    - artefacts
    - frontend
    - feature
---

# Frontend Plan: Frontmatter Created Date

Display the `created` and `mtime` (modified) dates in the artifact detail view with human-friendly formatting and tooltip hover for full date-time. Depends on [[frontmatter-created-date-2-be]] exposing both fields in API responses.

## Milestone 1 — Update TypeScript types to include `created`

**Description**: Add the `created` field to `ArtifactRow` and `ArtifactFrontmatter` interfaces so the frontend can consume the new API field.

**Files to change**:
- `web/src/types/api.ts` — add `created?: string` to `ArtifactFrontmatter`; add `created: string` to `ArtifactRow`

**Acceptance criteria**:
- `ArtifactRow.created` is typed as `string` (ISO 8601 or empty)
- No TypeScript compilation errors after the change
- Existing code that references `ArtifactRow` is unaffected (field is additive)

## Milestone 2 — Display created and modified dates in FrontmatterPanel

**Description**: Add "Created" and "Modified" rows to the `FrontmatterPanel.vue` details sidebar. Each shows a short human-readable date (e.g. "27 Apr 2026") with a tooltip on hover that displays the full date and time with timezone (e.g. "27 Apr 2026, 10:00:00 AM AEST").

**Files to change**:
- `web/src/components/artifact/FrontmatterPanel.vue` — add a `Created` row that renders `artifact.created` (or `artifact.frontmatter.created`) using a date formatter; update the existing `Modified` row to use the same tooltip pattern; add a `formatDate` and `formatDateTimeFull` helper (or a shared composable)

**Acceptance criteria**:
- "Created" row appears in the details panel when `created` is a non-empty string
- "Modified" row continues to appear and now matches the same formatting as "Created"
- Hovering over either date shows a tooltip with full date-time including timezone
- If `created` is empty or zero-value (legacy artifacts), the row is hidden or shows "Unknown"
- The date formatting respects the user's browser locale for month names and ordering

## Milestone 3 — Show dates in the artifact list view

**Description**: Add a "Created" column (or secondary line) to the artifact list table in `ArtifactListView.vue` so users can sort and scan by creation date at a glance.

**Files to change**:
- `web/src/views/project/ArtifactListView.vue` — add a column or metadata line showing the created date; use the same short format and tooltip pattern from Milestone 2

**Acceptance criteria**:
- Each artifact row in the list view shows its created date in short format
- Tooltip on hover shows the full date-time
- Artifacts without a created date show "—" or are gracefully blank
- The list remains readable and not overcrowded (dates should be compact)

## Milestone 4 — Extract shared date formatting composable

**Description**: Create a small composable (e.g. `useFormatDate`) to avoid duplicating date formatting logic between the panel and list view. This composable returns a short-format function and a full-format function.

**Files to change**:
- `web/src/composables/useFormatDate.ts` — new file; export `formatShortDate(iso: string): string` (e.g. "27 Apr 2026") and `formatFullDateTime(iso: string): string` (e.g. "27 Apr 2026, 10:00:00 AM AEST"); handle empty/invalid input gracefully
- `web/src/components/artifact/FrontmatterPanel.vue` — refactor to use the composable
- `web/src/views/project/ArtifactListView.vue` — refactor to use the composable

**Acceptance criteria**:
- Both components use the shared composable for date formatting
- `formatShortDate("")` returns "—" or a sensible fallback
- `formatFullDateTime` includes timezone offset or name
- No duplicate date formatting code across components
