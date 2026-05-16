---
title: 2D map shows 'project not found' alert and no nodes render when doc artifacts present
type: defect
status: done
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# 2D map shows "project not found" alert and no nodes render when doc artifacts present

## Reproduction Steps

1. Start the E2E harness with the fixtures that include `lifecycle/docs/smoke-doc-linked.md` (a `doc` artifact linked to `lifecycle/requirements/smoke-req-01.md`).
2. Log in and navigate to `/p/testproject/map`.
3. Wait up to 15 s for a `<canvas>` element to appear (Cytoscape 2D renderer).

## Expected Behaviour

- A `<canvas>` element is visible within 15 s.
- The Cytoscape instance (`window.__cy`) is initialised with at least one node.
- A `doc` node appears in the graph for `lifecycle/docs/smoke-doc-linked.md`.
- An edge connects the doc node to `lifecycle/requirements/smoke-req-01.md`.
- The doc node uses a distinct colour (teal-400 `#2dd4bf`) from idea (amber) and requirement (blue) nodes.

## Actual Behaviour

The filter/legend sidebar renders (including "Doc" in the node type legend), but:
- No `<canvas>` element appears within 15 s.
- An `alert` element displays `"project not found: testproject"`.
- `Filters 0 / 0 nodes` is shown, meaning the graph data fetch returned no nodes.

ARIA snapshot of main area at timeout:
```yaml
- main:
  - complementary:
    - text: Filters 0 / 0 nodes
    - ...
  - group "Map view mode":
    - button "3D"
    - button "2D"
  - alert: "project not found: testproject"
  - text: Nodes Idea Requirement Plan Backend Plan Frontend Plan Test Test Prototype Defect Doc ...
```

The "project not found" alert originates from a failed API call for graph data (`GET /api/p/testproject/graph` or equivalent). The frontend legend correctly lists "Doc" as a node type, suggesting the type is registered client-side but the data fetch fails.

## Logs / Output

```
Error: expect(locator).toBeVisible() failed
Locator: locator('canvas')
Expected: visible
Timeout: 15000ms
Error: element(s) not found
  at flows/09-doc-graph.spec.ts:23:42  (TC1)
  at flows/09-doc-graph.spec.ts:57:42  (TC2)
  at flows/09-doc-graph.spec.ts:103:42 (TC3)
```

All three TCs fail at the same canvas visibility check, meaning none of the doc-node-specific assertions (node existence, edge, colour) were reached.

**Failing tests:** `Flow 09 TC1`, `Flow 09 TC2`, `Flow 09 TC3` (`tests/e2e/flows/09-doc-graph.spec.ts`).
