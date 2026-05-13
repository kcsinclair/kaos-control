---
title: "Tech Writer Agent — Test Plan"
type: plan-test
status: draft
lineage: tech-writer-agent
parent: lifecycle/requirements/tech-writer-agent-2.md
---

# Tech Writer Agent — Test Plan

Integration and E2E test plan for the tech-writer agent feature. Covers artifact type registration, workflow transitions, API endpoints, UI triggers, and agent routing.

Cross-references: [[tech-writer-agent]] backend plan for API contracts; frontend plan for UI behaviour.

---

## Milestone 1 — Unit tests: `doc` type and workflow transitions

### Description

Verify that the `doc` artifact type is recognised and that the workflow engine enforces the correct medium-length transition chain for doc artifacts while leaving existing transitions intact.

### Files to change

- `tests/workflow_doc_test.go` (new file, or extend existing workflow tests if present).

### Test cases

1. **Type recognition** — `artifact.KnownTypes["doc"]` is `true`.
2. **Happy path: doc transitions** — assert the full chain is permitted:
   - `draft → approved` by `product-owner` for type `doc` ✓
   - `approved → in-development` by `tech-writer` for type `doc` ✓
   - `in-development → in-qa` by `tech-writer` for type `doc` ✓
   - `in-qa → done` by `qa` for type `doc` ✓
3. **Defect loop** — `in-qa → in-development` by `qa` for type `doc` ✓.
4. **Blocked transitions for doc** — doc artifacts must NOT use the standard feature flow:
   - `draft → clarifying` by `analyst` for type `doc` ✗
   - `clarifying → planning` by `analyst` for type `doc` ✗
   - `planning → in-development` by `approver` for type `doc` ✗
5. **No regression** — the standard feature transitions still work for `requirement`, `plan-backend`, `test`, etc.:
   - `draft → clarifying` by `analyst` for type `requirement` ✓
   - `planning → in-development` by `approver` for type `plan-backend` ✓
6. **Product-owner override** — product-owner can perform any transition on `doc` (existing superuser behaviour).

### Acceptance criteria

- [ ] All 6 test case groups pass.
- [ ] No existing workflow tests break.

---

## Milestone 2 — Unit tests: `required_plans` gate exclusion

### Description

Verify that the `required_plans` gate does not block `doc` artifacts from transitioning `approved → in-development`.

### Files to change

- `tests/workflow_doc_test.go` (same file as milestone 1, or a dedicated section).

### Test cases

1. **Doc bypasses gate** — create a `doc` artifact in a lineage that has no plans. Transition `approved → in-development` succeeds.
2. **Requirement still gated** — create a `requirement` artifact in a lineage with no approved plans. Transition `planning → in-development` is blocked.

### Acceptance criteria

- [ ] Doc artifact transitions are independent of plan existence.
- [ ] Requirement gating is unchanged.

---

## Milestone 3 — API tests: doc generation endpoint

### Description

Test the `POST /api/p/:project/ideas/generate` endpoint with `type: "doc"`, both with and without source-lineage context.

### Files to change

- `tests/api_doc_generate_test.go` (new file).

### Test cases

1. **Standalone doc generation** — `POST` with `{"input": "Document the installation process for new users", "type": "doc"}`. Assert response includes `target_dir: "lifecycle/docs"`, a valid slug, and doc-shaped frontmatter (`type: doc`, `status: draft`).
2. **Source-linked doc generation** — `POST` with `{"input": "Document the login feature", "type": "doc", "source_lineage": "login", "source_path": "lifecycle/requirements/login-2.md"}`. Assert response frontmatter includes `lineage: "login"` and `parent: "lifecycle/requirements/login-2.md"`.
3. **Input too short** — `POST` with `{"input": "docs", "type": "doc"}`. Assert 400 with `input_too_short` error.
4. **Existing types unaffected** — `POST` with `type: "idea"` and `type: "defect"` still produce correct results.

### Acceptance criteria

- [ ] All 4 test cases pass.
- [ ] Response shapes match the contract defined in the backend plan.

---

## Milestone 4 — API tests: doc artifact creation

### Description

Test `POST /api/p/:project/artifacts` for creating doc artifacts in `lifecycle/docs/`.

### Files to change

- `tests/api_doc_create_test.go` (new file).

### Test cases

1. **Originating doc** — create with `stage: "docs"`, `slug: "install-guide"`, empty lineage. Assert file is written to `lifecycle/docs/install-guide.md` (no index suffix), frontmatter has no `parent`.
2. **Source-linked doc** — create with `stage: "docs"`, `slug: "login"`, `lineage: "login"`, `parent: "lifecycle/requirements/login-2.md"`. Assert filename includes the next lineage index and `-doc` suffix (e.g. `login-7-doc.md`).
3. **Indexer picks up doc** — after creation, `GET /api/p/:project/artifacts/lifecycle/docs/<file>` returns the doc with correct type and lineage.
4. **Git commit** — after creation, a git commit exists with the new file.

### Acceptance criteria

- [ ] Both originating and source-linked docs are created with correct filenames and frontmatter.
- [ ] The indexer indexes doc artifacts without error.

---

## Milestone 5 — API tests: doc workflow transitions

### Description

Test the full transition chain for a doc artifact via the status-transition API endpoint.

### Files to change

- `tests/api_doc_transition_test.go` (new file).

### Test cases

1. **Full happy path** — create a `doc` artifact in `draft`, then transition through `approved → in-development → in-qa → done`. Assert each transition succeeds and the artifact's status is updated.
2. **Assignees on in-qa** — after transitioning `in-development → in-qa`, assert the artifact's `assignees` is `[{ role: qa, who: agent }]`.
3. **Invalid transition blocked** — attempt `draft → clarifying` on a `doc` artifact. Assert 403/422 rejection.
4. **Defect loop** — transition `in-qa → in-development` (QA sends back). Assert success.

### Acceptance criteria

- [ ] The medium-length pipeline is enforced end-to-end via the API.
- [ ] Invalid transitions are rejected.

---

## Milestone 6 — E2E tests: "Request docs" button (FR1)

### Description

End-to-end browser test verifying the "Request docs" button on an artifact view.

### Files to change

- `tests/e2e/doc_request_test.ts` (or `.spec.ts`, matching existing E2E test conventions).

### Test cases

1. **Button visibility** — navigate to an artifact with `status: done`. Assert "Request docs" button is visible.
2. **Button hidden** — navigate to an artifact with `status: approved`. Assert "Request docs" button is NOT visible.
3. **Flow** — click "Request docs" on a `done` artifact. Fill in a brief. Submit. Assert:
   - A new doc artifact is created in `lifecycle/docs/`.
   - The artifact's `lineage` matches the source.
   - The artifact's `parent` points to the source artifact.
   - The user is navigated to the new doc artifact view.

### Acceptance criteria

- [ ] Button conditional visibility works.
- [ ] End-to-end creation flow produces a correctly-formed doc artifact.

---

## Milestone 7 — E2E tests: "New Docs" button (FR2)

### Description

End-to-end browser test verifying the "New Docs" button on the Dashboard and Artifact List.

### Files to change

- `tests/e2e/doc_new_test.ts` (or `.spec.ts`).

### Test cases

1. **Dashboard button present** — navigate to Dashboard. Assert "New Docs" button is visible alongside "New Idea" and "New Defect".
2. **Artifact List button present** — navigate to Artifact List. Assert "New Docs" button is visible.
3. **Standalone flow** — click "New Docs" on Dashboard. Fill in a slug and brief. Submit. Assert:
   - A new doc artifact is created at `lifecycle/docs/<slug>.md` (no index suffix).
   - Frontmatter has `type: doc`, `status: draft`, no `parent`.
   - The user is navigated to the new doc artifact view.

### Acceptance criteria

- [ ] Buttons are present on both Dashboard and Artifact List.
- [ ] Standalone doc creation produces an originating artifact with correct structure.

---

## Milestone 8 — E2E tests: agent routing and Queue Work (NFR3)

### Description

Verify that a `doc` artifact in `approved` status shows the Queue Work button and that it routes to the `tech-writer` agent.

### Files to change

- `tests/e2e/doc_queue_test.ts` (or `.spec.ts`).

### Test cases

1. **Queue Work visible** — create a `doc` artifact, transition to `approved`. Assert Queue Work button is visible.
2. **Agent routing** — click Queue Work. Assert the enqueue request targets the `tech-writer` agent.
3. **Ready count** — verify the agents API ready-count endpoint includes the doc artifact for the `tech-writer` agent.

### Acceptance criteria

- [ ] Queue Work button is shown for approved doc artifacts.
- [ ] The correct agent (`tech-writer`) is targeted.

---

## Milestone 9 — E2E tests: graph rendering (NFR1)

### Description

Verify that `doc` nodes appear in the graph views with correct styling and edges.

### Files to change

- `tests/e2e/doc_graph_test.ts` (or `.spec.ts`).

### Test cases

1. **Node exists** — create a `doc` artifact linked to an existing lineage. Open the graph view. Assert a node labelled with the doc's title exists.
2. **Edge exists** — assert an edge connects the doc node to its parent artifact.
3. **Colour distinct** — assert the doc node uses a different colour from `idea`, `requirement`, and plan nodes.

### Acceptance criteria

- [ ] Doc nodes render in both 2D and 3D graph views.
- [ ] Parent/lineage edges are drawn correctly.

---

## Milestone 10 — Regression suite

### Description

Run the full existing test suite to confirm no regressions from the tech-writer additions.

### Files to change

- No new files — execute existing test suites.

### Test cases

1. **Go unit tests** — `make test-unit` passes.
2. **Frontend type check** — `pnpm exec vue-tsc --noEmit` passes.
3. **Frontend build** — `pnpm build` passes.
4. **Existing E2E tests** — all pre-existing E2E tests pass without modification.

### Acceptance criteria

- [ ] All existing tests pass.
- [ ] No new warnings or type errors introduced.
