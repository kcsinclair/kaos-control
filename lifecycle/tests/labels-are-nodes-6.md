---
title: "Integration Tests — Labels as Graph Nodes with Priority Visualisation"
type: test
status: draft
lineage: labels-are-nodes
parent: lifecycle/test-plans/labels-are-nodes-5-test.md
---

# Integration Tests — Labels as Graph Nodes with Priority Visualisation

Integration test suite for the [[labels-are-nodes]] feature. Tests are tagged `integration` and run with:

```sh
go test -tags=integration ./tests/integration/...
```

## Test Files

| File | Milestones |
|---|---|
| `tests/integration/priority_patch_test.go` | Milestone 1 — Priority PATCH endpoint |
| `tests/integration/graph_labels_test.go` | Milestone 2 — Graph labels normalisation |
| `tests/integration/graph_priority_test.go` | Milestone 3 — Priority in graph nodes |
| `tests/integration/artifact_update_test.go` | Milestone 4 — PUT artifact priority validation |
| `tests/integration/graph_node_types_test.go` | Milestone 5 — Node type coverage |
| `tests/integration/graph_performance_test.go` | Milestone 6 — Performance baseline |
| `tests/integration/priority_roundtrip_test.go` | Milestone 7 — End-to-end round-trip |

Supporting helpers added to `tests/integration/helpers_test.go`:
`makeArtifactWithPriority`, `findNodeByID`, `graphNodeLabels`, `graphResponseForProject`,
`createArtifactViaAPI`, `artifactFrontmatterJSON`, `roundTripJSON`, `buildCookieHeader`.

## Scenarios Covered

### Milestone 1 — Priority PATCH Endpoint (`priority_patch_test.go`)

- `TestPriorityPatchHappyPath` — PATCH `priority: normal → high`; response and subsequent GET both reflect new value.
- `TestPriorityPatchUnset` — PATCH with `priority: ""`; verifies the `priority:` line is absent from the file on disk.
- `TestPriorityPatchInvalidValue` — PATCH with `priority: "critical"`; expects 400 with `error.code = "bad_request"`.
- `TestPriorityPatchNonExistent` — PATCH a path that does not exist; expects 404.
- `TestPriorityPatchFrontmatterPreservation` — After PATCH, all other frontmatter fields (title, type, status, lineage, labels) are unchanged.
- `TestPriorityPatchBodyPreservation` — After PATCH, the markdown body returned by GET is byte-identical to before.
- `TestPriorityPatchWebSocketEvent` — A connected WebSocket client receives an `artifact.indexed` event with the correct path after a successful PATCH.

### Milestone 2 — Graph Labels Normalisation (`graph_labels_test.go`)

- `TestGraphLabelsNormalisedWhenAbsent` — Artifact with no `labels` field; graph node has `labels: []`, not null or absent.
- `TestGraphLabelsPresent` — Artifact with `labels: [auth, backend]`; graph node carries exactly those values.
- `TestGraphLabelsMixedSet` — Four artifacts (some labelled, some not); every graph node has a non-null labels array.

### Milestone 3 — Priority in Graph Nodes (`graph_priority_test.go`)

- `TestGraphPriorityPresent` — Artifact with `priority: high`; graph node carries `priority: "high"`.
- `TestGraphPriorityAbsent` — Artifact with no priority field; graph node has `priority: ""` or the field omitted — no error.
- `TestGraphPriorityAfterPatch` — PATCH `priority: low → medium`; subsequent graph query reflects `"medium"`.

### Milestone 4 — PUT Artifact Priority Validation (`artifact_update_test.go`)

- `TestPutArtifactValidPriority` — PUT with `priority: "normal"`; succeeds and response reflects the value.
- `TestPutArtifactInvalidPriority` — PUT with `priority: "urgent"`; expects 400 with error message listing allowed values.
- `TestPutArtifactEmptyPriority` — PUT with `priority: ""`; succeeds (unset is valid).
- `TestPutArtifactNoPriorityField` — PUT without a `priority` key at all; succeeds.

### Milestone 5 — Node Type Coverage (`graph_node_types_test.go`)

- `TestAllSpecTypesInGraph` — One artifact of each of the 12 spec types (`idea`, `ticket`, `epic`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `test`, `prototype`, `release`, `sprint`, `defect`); all 12 nodes appear in the graph with the correct type field.
- `TestTypeFieldAccuracy` — Graph node `type` matches the `type` written in frontmatter for every spec-defined type.

### Milestone 6 — Performance Baseline (`graph_performance_test.go`)

- `TestGraphPerformance500Artifacts` — 500 artifacts spread across 5 stages, with 0–3 labels each drawn from a pool of 50 distinct labels; GET /graph responds in under 2 seconds and returns all 500 nodes with non-null labels.
- `TestGraphLabelDensity` — 200 artifacts with 3–5 labels each from a 50-label pool; GET /graph responds in under 2 seconds and every node has the expected label count.

### Milestone 7 — End-to-End Round-Trip (`priority_roundtrip_test.go`)

- `TestPriorityFullRoundTrip` — PATCH `low → high`; verifies the updated priority in the graph API response, in the raw file on disk, and confirms no other frontmatter fields changed.
- `TestPriorityMultipleRapidUpdates` — Five sequential PATCHes (`low → normal → medium → high → low`); final state is `low` in both the API and on disk.
- `TestPriorityPatchConcurrentReads` — Eight concurrent GET requests while a PATCH is in flight; all GETs return 200 with parseable JSON containing an `artifact` key, confirming no data corruption under concurrent access.
