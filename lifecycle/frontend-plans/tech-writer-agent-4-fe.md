---
title: "Tech Writer Agent — Frontend Plan"
type: plan-frontend
status: in-development
lineage: tech-writer-agent
parent: lifecycle/requirements/tech-writer-agent-2.md
---

# Tech Writer Agent — Frontend Plan

Frontend implementation for the tech-writer agent feature. Covers the "Request docs" button, "New Docs" entry, BrainDumpModal extension, agent-routing updates, and graph node styling.

Cross-references: [[tech-writer-agent]] backend plan for API changes and config; test plan for E2E coverage.

---

## Milestone 1 — Extend BrainDumpModal to support `doc` type

### Description

The existing `BrainDumpModal` handles `idea` and `defect` artifact types. Extend it to accept `doc` as a third variant. The modal must support two flows:

1. **Source-linked** (FR1): pre-populated with `source_lineage` and `source_path` from the triggering artifact. The user provides a brief describing what documentation is needed.
2. **Standalone** (FR2): no source artifact. The user supplies a slug and the brief.

### Files to change

- `web/src/components/idea/BrainDumpModal.vue`:
  - Extend the `artifactType` prop type from `'idea' | 'defect'` to `'idea' | 'defect' | 'doc'`.
  - Add a `sourceLineage` and `sourcePath` optional prop (used when triggered from an artifact view).
  - Update `headerLabel` computed: add `'doc'` → `'New Docs'`.
  - Update `placeholderText` computed: add `'doc'` → `'Describe what documentation is needed…'`.
  - When `artifactType === 'doc'`, pass `type: "doc"`, `source_lineage`, and `source_path` to the store's `generate()` call.
- `web/src/stores/brainDump.ts`:
  - Update the `generate()` action to accept and pass through `source_lineage` and `source_path` fields to the API.
  - Update `acceptProposal()` to use `stage: "docs"` when `artifactType === "doc"`.
  - Handle the response frontmatter shape for doc artifacts (lineage inherited from source or PO-supplied).

### Acceptance criteria

- [ ] `BrainDumpModal` renders correctly with `artifactType="doc"`, showing "New Docs" header and doc-specific placeholder.
- [ ] When `sourceLineage` and `sourcePath` props are provided, they are included in the generate API call.
- [ ] The accepted proposal creates an artifact in `lifecycle/docs/` with correct frontmatter.
- [ ] Existing `idea` and `defect` flows are unaffected.

---

## Milestone 2 — "Request docs" button on artifact view (FR1)

### Description

Add a **Request docs** button to the artifact editor view. The button is visible only when the artifact's status is `done`. Clicking it opens the `BrainDumpModal` in `doc` mode, pre-populated with the current artifact's lineage slug and path as `sourceLineage` and `sourcePath`.

### Files to change

- `web/src/views/project/ArtifactEditorView.vue`:
  - Import a docs icon from `lucide-vue-next` (e.g. `FileText` or `BookOpen`).
  - Add a `showDocsModal` ref.
  - Add a **Request docs** button in the toolbar/header area, conditionally rendered when `artifact.status === 'done'`.
  - On click: set `showDocsModal = true`.
  - Render `BrainDumpModal` with `artifactType="doc"`, `:sourceLineage="artifact.lineage"`, `:sourcePath="artifact.path"`.
  - Wire `@created` to navigate to the new docs artifact.

### Acceptance criteria

- [ ] Button is visible only when `artifact.status === 'done'`.
- [ ] Button is hidden for all other statuses (`draft`, `approved`, `in-development`, etc.).
- [ ] Clicking the button opens BrainDumpModal in doc mode with source lineage and path pre-filled.
- [ ] Submitting the modal creates a `doc` artifact in `lifecycle/docs/<slug>-N-doc.md` with correct lineage and parent pointing to the source artifact.
- [ ] After creation, the user is navigated to the new doc artifact.

---

## Milestone 3 — "New Docs" button on Dashboard and Artifact List (FR2)

### Description

Add a **New Docs** button alongside the existing **New Idea** and **New Defect** buttons on the Dashboard and Artifact List views. Clicking it opens the BrainDumpModal in `doc` mode without a source artifact (standalone docs flow).

### Files to change

- `web/src/views/project/DashboardView.vue`:
  - Import a docs icon from `lucide-vue-next` (e.g. `BookOpen`).
  - Extend `brainDumpType` ref type to include `'doc'`.
  - Add a **New Docs** button in `header-actions` alongside the existing buttons, calling `openBrainDump('doc', ...)`.
  - Update `openBrainDump` function signature to accept `'idea' | 'defect' | 'doc'`.
- `web/src/views/project/ArtifactListView.vue`:
  - Same changes: add **New Docs** button, extend type, wire to BrainDumpModal.

### Acceptance criteria

- [ ] Dashboard shows three buttons: **New Defect**, **New Idea**, **New Docs**.
- [ ] Artifact List shows three buttons: **New Defect**, **New Idea**, **New Docs**.
- [ ] Clicking **New Docs** opens BrainDumpModal with `artifactType="doc"` and no source lineage.
- [ ] Submitting creates an originating doc artifact at `lifecycle/docs/<slug>.md` with no parent and no index suffix.
- [ ] Existing **New Idea** and **New Defect** flows are unaffected.

---

## Milestone 4 — Agent routing: `doc` → `tech-writer`

### Description

Update the `typeToAgent` mapping so that `doc` artifacts route to the `tech-writer` agent. This ensures the **Queue Work** button on `doc` artifacts in `approved` status correctly queues a tech-writer run.

### Files to change

- `web/src/composables/useAgentForArtifact.ts`:
  - Add `doc: 'tech-writer'` to the `typeToAgent` record.

### Acceptance criteria

- [ ] `agentForArtifact({ frontmatter: { type: 'doc' } }, agents)` returns `'tech-writer'`.
- [ ] A `doc` artifact in `approved` status shows the **Queue Work** button.
- [ ] Clicking **Queue Work** on an approved `doc` enqueues a `tech-writer` agent run.
- [ ] Existing agent routing for all other types is unchanged.

---

## Milestone 5 — Graph node styling for `doc` type (NFR1)

### Description

Add a colour and label entry for the `doc` artifact type in both the 3D force-graph and 2D Cytoscape graph views so `doc` nodes render distinctly.

### Files to change

- `web/src/components/graph/` — locate the type-to-colour mapping (likely a `const` or computed in the 3D and 2D graph components). Add an entry for `doc` with a distinct colour (suggested: a teal or blue-green to differentiate from existing types).
- `web/src/components/graph/` — if the graph legend is generated from the type list, ensure `doc` appears automatically. If hardcoded, add it.

### Acceptance criteria

- [ ] `doc` nodes in the 3D graph render with a distinct colour and are labelled "doc".
- [ ] `doc` nodes in the 2D Cytoscape graph render with a distinct colour.
- [ ] Parent/lineage edges from `doc` nodes to their source artifacts render correctly.
- [ ] The graph legend (if present) includes the `doc` type.
- [ ] No changes required beyond adding the colour/label entry — the indexer already handles unknown types by reading frontmatter.

---

## Milestone 6 — Dashboard tracked types (optional)

### Description

If the product owner wants `doc` artifacts to appear on the dashboard status counts and Kanban board, add `doc` to the `dashboard.tracked_types` list in `lifecycle/config.yaml`. This is a config-only change and is optional — evaluate whether docs should be tracked alongside requirements, ideas, and defects.

### Files to change

- `lifecycle/config.yaml` — optionally add `doc` to `dashboard.tracked_types`.

### Acceptance criteria

- [ ] If added: `doc` artifacts appear in the dashboard's status counts and Kanban columns.
- [ ] If not added: `doc` artifacts are visible only in the Artifacts list and graph views, not the dashboard. This is acceptable for v1.
- [ ] No regressions to existing dashboard behaviour.
