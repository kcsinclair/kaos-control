---
created: "2026-05-16T20:41:53+10:00"
title: "Tech Writer Agent — Test Suite"
type: test
status: done
lineage: tech-writer-agent
parent: lifecycle/test-plans/tech-writer-agent-5-test.md
---

# Tech Writer Agent — Test Suite

Integration and E2E tests covering the tech-writer agent feature as specified in the test plan at `lifecycle/test-plans/tech-writer-agent-5-test.md`.

---

## Scenarios Covered

### Milestones 1 & 2 — Workflow (Go integration tests)

File: `tests/integration/workflow_doc_test.go`

- **TestDocType_KnownTypes** — `artifact.KnownTypes["doc"]` is registered.
- **TestDocWorkflow_HappyPathTransitions** — full doc pipeline permitted: `draft → approved` (product-owner), `approved → in-development` (tech-writer), `in-development → in-qa` (tech-writer), `in-qa → done` (qa).
- **TestDocWorkflow_DefectLoop** — `in-qa → in-development` by qa is allowed.
- **TestDocWorkflow_BlockedTransitions** — standard feature flow (`draft → clarifying`, `clarifying → planning`, `planning → in-development`) is blocked for `doc`.
- **TestDocWorkflow_NoRegression** — existing transitions (`requirement`, `plan-backend`, `test`) are unaffected.
- **TestDocWorkflow_ProductOwnerOverride** — product-owner bypasses all doc transition rules.
- **TestDocGate_DocBypasses** — `approved → in-development` succeeds for doc with no plans present.
- **TestDocGate_RequirementStillGated** — `planning → in-development` is blocked for requirement when plans are absent.

### Milestone 3 — Generate endpoint (Go integration tests)

File: `tests/integration/api_doc_generate_test.go`

- **TestDocGenerate_StandaloneDoc** — `POST /ideas/generate` with `type=doc` returns `target_dir: lifecycle/docs`, valid slug, `type: doc`, `status: draft`.
- **TestDocGenerate_SourceLinkedDoc** — with `source_lineage`/`source_path` the frontmatter carries matching `lineage` and `parent`.
- **TestDocGenerate_InputTooShort** — very short input returns 400 with an error field.
- **TestDocGenerate_IdeaTypeUnaffected** — `type=idea` still returns `target_dir: lifecycle/ideas`.
- **TestDocGenerate_DefectTypeUnaffected** — `type=defect` still returns `target_dir: lifecycle/defects`.

### Milestone 4 — Artifact creation (Go integration tests)

File: `tests/integration/api_doc_create_test.go`

- **TestDocCreate_OriginatingDoc** — `POST /artifacts` with `stage: docs`, no parent, writes `lifecycle/docs/<slug>.md` with no index suffix and no `parent` field.
- **TestDocCreate_SourceLinkedDoc** — with lineage + parent, the filename carries a monotonic index and `-doc` suffix (e.g. `login-7-doc.md`).
- **TestDocCreate_IndexerPicksUp** — after creation, `GET /artifacts/<path>` returns the doc with correct type and lineage.
- **TestDocCreate_GitCommit** — creation produces a git commit with message prefix `create(docs): <path>`.

### Milestone 5 — Transition API (Go integration tests)

File: `tests/integration/api_doc_transition_test.go`

- **TestDocTransition_FullHappyPath** — full pipeline via HTTP: `draft → approved → in-development → in-qa → done`.
- **TestDocTransition_AssigneesOnInQA** — `in-development → in-qa` sets `assignees: [{role: qa, who: agent}]`.
- **TestDocTransition_InvalidTransitionBlocked** — `draft → clarifying` on a doc is rejected with 403.
- **TestDocTransition_DefectLoop** — qa can send a doc back from `in-qa → in-development`.

### Milestone 6 — "Request docs" button (E2E)

File: `tests/e2e/flows/06-doc-request.spec.ts`

- **TC1** — "Request docs" button is visible on a `done` artifact.
- **TC2** — "Request docs" button is hidden on a non-done artifact.
- **TC3** — clicking "Request docs", filling the brief, and submitting creates a doc under `lifecycle/docs/` and navigates to the new artifact view.

### Milestone 7 — "New Docs" button (E2E)

File: `tests/e2e/flows/07-doc-new.spec.ts`

- **TC1** — "New Docs" button is present on the Dashboard.
- **TC2** — "New Docs" button is present on the Artifact List.
- **TC3** — standalone creation flow writes `lifecycle/docs/<slug>.md` (no index suffix), navigates to the new doc, and shows `status: draft`.

### Milestone 8 — Queue Work / agent routing (E2E)

File: `tests/e2e/flows/08-doc-queue.spec.ts`

- **TC1** — Queue Work button is visible on an `approved` doc artifact.
- **TC2** — clicking Queue Work sends `POST /api/queue` with `agent: "tech-writer"`.
- **TC3** — the `/api/p/:project/agents` endpoint reports `tech-writer` with `ready_count >= 1` when an approved doc exists.

### Milestone 9 — Graph rendering (E2E)

File: `tests/e2e/flows/09-doc-graph.spec.ts`

- **TC1** — a doc node appears in the 2D map (Cytoscape) view.
- **TC2** — an edge connects the doc node to its parent artifact.
- **TC3** — the doc node uses a colour distinct from `idea` and `requirement` nodes.

---

## Fixture Changes

The following fixtures were added or modified to support the E2E suites:

| File | Change |
|------|--------|
| `tests/e2e/fixtures/lifecycle/config.yaml` | Added `docs` stage, `tech-writer` role, `techwriter@kaos-e2e.local` user, stub `tech-writer` agent |
| `tests/e2e/fixtures/lifecycle/requirements/smoke-req-done.md` | New — `done` requirement for Milestone 6 tests |
| `tests/e2e/fixtures/lifecycle/docs/smoke-doc-approved.md` | New — `approved` doc for Milestone 8 tests |
| `tests/e2e/fixtures/lifecycle/docs/smoke-doc-linked.md` | New — `draft` doc linked to `smoke-req-01` for Milestone 9 tests |
