---
title: "Test Plan: Labels as Graph Nodes with Priority Visualisation"
type: plan-test
status: draft
lineage: labels-are-nodes
parent: requirements/labels-are-nodes-2.md
---

# Test Plan: Labels as Graph Nodes with Priority Visualisation

This plan covers integration tests for [[labels-are-nodes]]. Tests exercise the backend API endpoints added/modified in the [[labels-are-nodes]] backend plan and verify end-to-end behaviour of the graph data pipeline that feeds the frontend from the [[labels-are-nodes]] frontend plan.

## Milestone 1: Priority PATCH Endpoint Tests

### Description

Test the new `PATCH /api/p/:project/artifacts/:path/priority` endpoint for valid updates, validation failures, and side effects (re-index, WebSocket events).

### Files to Change

- `tests/priority_patch_test.go` — **new file**: integration tests for priority PATCH

### Test Cases

1. **Happy path**: Create an artifact with `priority: normal`, PATCH to `high`, verify the response contains `priority: "high"` and a subsequent GET returns the updated value.
2. **Unset priority**: PATCH with `priority: ""` to clear priority, verify it is removed from frontmatter on disk.
3. **Invalid priority**: PATCH with `priority: "critical"`, expect `400 bad_request`.
4. **Non-existent artifact**: PATCH a path that doesn't exist, expect `404`.
5. **Frontmatter preservation**: After PATCH, verify all other frontmatter fields (title, type, status, labels, lineage) are unchanged.
6. **Body preservation**: After PATCH, verify the markdown body is byte-identical to before.
7. **WebSocket event**: Connect a WebSocket client before the PATCH, verify an `artifact.indexed` event is received after a successful PATCH.

### Acceptance Criteria

- [ ] All 7 test cases pass.
- [ ] Tests create their own test artifacts in a temporary project directory (no dependency on existing lifecycle files).
- [ ] Tests clean up after themselves.

## Milestone 2: Graph API — Labels Array Normalisation

### Description

Verify that the graph API always returns `labels` as an array (never `null`) on every node, regardless of whether the artifact has labels in its frontmatter.

### Files to Change

- `tests/graph_labels_test.go` — **new file**: integration tests for graph label normalisation

### Test Cases

1. **Artifact without labels**: Create an artifact with no `labels` field. Call `GET /graph`, verify the node's `labels` field is `[]` (empty array), not `null` or absent.
2. **Artifact with labels**: Create an artifact with `labels: [auth, backend]`. Call `GET /graph`, verify `labels` is `["auth", "backend"]`.
3. **Mixed set**: Create several artifacts, some with labels, some without. Verify all nodes have a non-null `labels` array.

### Acceptance Criteria

- [ ] All 3 test cases pass.
- [ ] No node in the graph response has `labels: null`.

## Milestone 3: Graph API — Priority on Nodes

### Description

Verify the graph API correctly returns `priority` on nodes, including after a PATCH update.

### Files to Change

- `tests/graph_priority_test.go` — **new file**: integration tests for priority in graph nodes

### Test Cases

1. **Priority present**: Create an artifact with `priority: high`. Call `GET /graph`, verify the node has `priority: "high"`.
2. **Priority absent**: Create an artifact without priority. Call `GET /graph`, verify `priority` is `""` or omitted (not an error).
3. **Priority after PATCH**: Create artifact with `priority: low`, PATCH to `medium`, call `GET /graph`, verify node now has `priority: "medium"`.

### Acceptance Criteria

- [ ] All 3 test cases pass.
- [ ] Priority changes via PATCH are reflected in subsequent graph queries.

## Milestone 4: Full PUT Artifact — Priority Validation

### Description

Verify that the existing `PUT /artifacts/:path` endpoint now rejects invalid priority values.

### Files to Change

- `tests/artifact_update_test.go` — extend existing file or **new file**: tests for priority validation on PUT

### Test Cases

1. **Valid priority on PUT**: PUT with `priority: "normal"`, expect success.
2. **Invalid priority on PUT**: PUT with `priority: "urgent"`, expect `400`.
3. **Empty priority on PUT**: PUT with `priority: ""`, expect success (unset).
4. **No priority field on PUT**: PUT without a priority field, expect success.

### Acceptance Criteria

- [ ] All 4 test cases pass.
- [ ] Validation error message lists the allowed priority values.

## Milestone 5: Node Colour Coverage Verification

### Description

Verify that every spec-defined artifact type receives a non-fallback colour from the backend graph data, by creating one artifact of each type and checking the graph response.

### Files to Change

- `tests/graph_node_types_test.go` — **new file**: integration test ensuring all types are represented

### Test Cases

1. **All spec types present**: Create one artifact for each of the 12 spec types (`idea`, `ticket`, `epic`, `plan-backend`, `plan-frontend`, `plan-dev`, `plan-test`, `test`, `prototype`, `release`, `sprint`, `defect`). Call `GET /graph`, verify all 12 nodes appear with their correct `type` field.
2. **Type field accuracy**: For each created artifact, verify the graph node's `type` matches the frontmatter `type`.

### Acceptance Criteria

- [ ] All 12 spec-defined types appear in the graph response.
- [ ] No type field is empty or mismatched.
- [ ] Test artifacts are created in a temp directory and cleaned up.

## Milestone 6: Performance Baseline

### Description

Establish a performance baseline to verify that the graph API can handle the expected scale (NFR-1: 500 artifacts, 50 labels) without significant latency degradation.

### Files to Change

- `tests/graph_performance_test.go` — **new file**: performance/load test for graph endpoint

### Test Cases

1. **Scale test**: Generate 500 artifacts with randomised labels (50 distinct labels across the set). Call `GET /graph` and assert the response time is under 2 seconds.
2. **Label density**: Generate 200 artifacts where each has 3-5 labels. Verify the graph response contains the correct number of label entries per node and responds within 2 seconds.

### Acceptance Criteria

- [ ] Graph API responds in under 2 seconds for 500 artifacts with 50 labels.
- [ ] All generated artifacts appear in the response with correct label arrays.
- [ ] Test artifacts are created in a temp directory and cleaned up.

## Milestone 7: End-to-End Priority Update Round-Trip

### Description

Test the full priority update flow: PATCH priority → verify graph node reflects new priority → verify artifact file on disk has correct frontmatter.

### Files to Change

- `tests/priority_roundtrip_test.go` — **new file**: end-to-end round-trip test

### Test Cases

1. **Full round-trip**: Create artifact with `priority: low`. PATCH to `high`. GET the graph, verify node priority is `high`. Read the file from disk, verify frontmatter `priority: high`. Verify no other frontmatter fields changed.
2. **Multiple rapid updates**: PATCH priority 5 times in quick succession (`low` → `normal` → `medium` → `high` → `low`). Verify final state is `low` in both API and on disk.
3. **Concurrent reads**: While a PATCH is in flight, issue a GET for the artifact. Verify no error or corrupt data is returned (file-level atomicity check).

### Acceptance Criteria

- [ ] All 3 test cases pass.
- [ ] The round-trip (PATCH → GET graph) consistently shows the updated priority.
- [ ] Disk state matches API state after every update.
