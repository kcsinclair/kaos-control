# Artifact Types & Graph Node Types

## Artifact Document Types

These are the valid values for the `type` field in artifact frontmatter, defined in `internal/artifact/artifact.go`.

| Type | Stage Directory | Description |
|---|---|---|
| `idea` | `lifecycle/ideas/` | Originating ideas |
| `requirement` | `lifecycle/requirements/` | Requirements |
| `plan-backend` | `lifecycle/backend-plans/` | Backend implementation plans |
| `plan-frontend` | `lifecycle/frontend-plans/` | Frontend implementation plans |
| `plan-test` | `lifecycle/test-plans/` | Test plans |
| `test` | `lifecycle/tests/` | Test artifacts |
| `prototype` | `lifecycle/prototypes/` | Prototypes |
| `defect` | `lifecycle/defects/` | Defects raised by QA |

## Graph Node Types & Colours

Node colours are defined in `web/src/components/graph/graphConstants.ts`.

| Type | Colour | Hex |
|---|---|---|
| `idea` | Amber | `#f59e0b` |
| `requirement` | Blue | `#3b82f6` |
| `plan-backend` | Violet | `#8b5cf6` |
| `plan-frontend` | Light violet | `#a78bfa` |
| `plan-test` | Lavender | `#c084fc` |
| `test` | Cyan | `#06b6d4` |
| `prototype` | Teal | `#14b8a6` |
| `defect` | Rose | `#f43f5e` |
| `label` | Purple | `#a855f7` |

> `label` is a synthetic node type generated client-side when **Show label nodes** is enabled in the graph filter panel. These nodes are not stored as artifacts on disk.

## Priority Ring Colours

Nodes with a `priority` frontmatter field display a coloured ring in both the 2D and 3D graph views. The ring is grey (`#6b7280`) when the artifact's status is `done`.

| Priority | Colour | Hex |
|---|---|---|
| `high` | Red | `#ef4444` |
| `medium` | Orange | `#f97316` |
| `normal` | Green | `#22c55e` |
| `low` | Blue | `#3b82f6` |
