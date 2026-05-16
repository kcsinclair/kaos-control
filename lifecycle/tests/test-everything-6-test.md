---
title: 'Tests: Run All Tests and Auto-file Defects'
type: test
status: draft
lineage: test-everything
parent: lifecycle/test-plans/test-everything-5-test.md
---

Integration and frontend tests covering the test-everything feature: test output parsers, artifact mapping, deduplication, defect creation and routing, the `test-runner` agent orchestrator, the `test-all.yaml` DevOps pipeline, and frontend UI elements.

## Milestones Covered

### Milestones 1–3 (Parser, Mapper, Dedup Unit Tests)

Already implemented in `internal/testrunner/` as package-level unit tests (white-box):

- `internal/testrunner/parse_go_test.go` — Go `test -json` parser: pass/fail/skip, subtests, compile errors, empty input, parallel interleaving.
- `internal/testrunner/parse_playwright_test.go` — Playwright JSON parser: nested suites, failures, skipped tests, malformed input.
- `internal/testrunner/parse_vitest_test.go` — Vitest JSON parser: ancestor titles, failure extraction, bad JSON.
- `internal/testrunner/mapper_test.go` — ArtifactMapper: slug match, label match, lineage match, no-match, orphan detection, slug/path helpers.
- `internal/testrunner/dedup_test.go` — Deduplicator: by test label, by location label, closed defects excluded, assertion grouping, NormaliseError.
- `internal/testrunner/defect_test.go` — DefectFiler: frontmatter fields, orphaned lineage, grouped failures, AppendWitness, route role, buildTitle, buildReproduction, lineage index increment.
- `internal/testrunner/executor_test.go` — Executor: skips absent dirs, continues after suite failure, records elapsed, non-JSON RawError.

### Milestone 1 Fixture Files

- `tests/fixtures/testrunner/go_mixed_results.json` — Go NDJSON: passes, failure, skip, subtest.
- `tests/fixtures/testrunner/go_compile_error.txt` — Non-JSON compilation error.
- `tests/fixtures/testrunner/vitest_results.json` — Vitest JSON with mixed results.
- `tests/fixtures/testrunner/playwright_results.json` — Playwright JSON with nested failures.

### Milestone 2 Fixture Artifacts

- `tests/fixtures/testrunner/test_artifacts/http-api-tests-2-test.md` — HTTP API test artifact (labels: `http`, `backend`).
- `tests/fixtures/testrunner/test_artifacts/auth-tests-2-test.md` — Auth test artifact (`auth`, `backend`, `source_file: auth_test.go`).

### Milestone 4 — Defect Filing Integration Tests

`tests/integration/testrunner_defect_test.go` — Integration tests using a real SQLite index and temp filesystem:

- `TestDefectFiler_BackendLabelRouting` — artifact with `backend` label routes to `backend-developer`.
- `TestDefectFiler_FrontendPathRouting` — Vitest failure in `tests/web/` routes to `frontend-developer`.
- `TestDefectFiler_OrphanedCreatesIdeaArtifact` — orphaned failure auto-creates `lifecycle/ideas/tests-orphaned.md`.
- `TestDeduplicator_FindsExistingOpenDefect` — full round-trip: file defect → dedup finds it open.
- `TestDefectFiler_LineageIndicesMonotonic` — two defects for same lineage get different indices.
- `TestDefectFiler_WitnessAppendPreservesFrontmatter` — AppendWitness does not corrupt YAML.

### Milestone 5 — Agent Orchestrator Integration Tests

`tests/integration/testrunner_agent_test.go` — Uses `tests/fixtures/testrunner/project/` (minimal Go module with `TestWidgetFails`) to invoke `testrunner.Run()` end-to-end:

- `TestRun_FullFlow` — one failing test produces one defect.
- `TestRun_Idempotency` — second run finds duplicates, creates no new defects.
- `TestRun_SuiteLevelError` — project with syntax error runs without panic.
- `TestRun_CoverageGaps` — test artifact with no failure appears in `CoverageGaps`.
- `TestRun_OverheadUnderTenSeconds` — fixture completes under 60s total wall time.

Fixture project: `tests/fixtures/testrunner/project/` (`testrunner-fixture/project` Go module, `widget/widget_test.go` with `TestWidgetFails`, lifecycle test artifact for `widget-tests`).

### Milestone 6 — DevOps Pipeline Integration Tests

`tests/integration/devops_test_all_test.go` — Tests `lifecycle/devops/all-tests.yaml` via the DevOps HTTP API:

- `TestDevopsTestAll_Discoverable` — pipeline appears in `GET /devops/pipelines`.
- `TestDevopsTestAll_Metadata` — type `test`, 5 steps.
- `TestDevopsTestAll_Triggerable` — `POST .../run` returns 16-char run_id.
- `TestDevopsTestAll_RunHistory` — completed run appears in `GET /devops/runs/{run_id}`.
- `TestDevopsTestAll_RequiresAuth` — unauthenticated requests rejected.

### Milestone 7 — Frontend UI Tests

`tests/web/agent-launcher-targetless.test.ts` — `AgentLaunchModal` target-less behaviour:

- `source_types: []` hides artifact picker and shows info message.
- Run button enabled without target.
- `listArtifacts` not called for target-less agent.
- Agent with non-empty `source_types` still shows picker area.

`tests/web/agent-run-summary.test.ts` — `TestRunSummaryCard` rendering:

- Per-suite rows (Go, Vitest, Playwright).
- Defect / duplicate / orphan counts and plural/singular forms.
- Warning style on defects created and orphaned failures.
- Duration formatted as ms / s / m:s.
- Collapsible coverage gaps: hidden initially, visible after click, closable.
- Empty suites state message.

`tests/web/auto-filed-badge.test.ts` — Auto-filed badge in `ArtifactListView`:

- Defect with `auto-filed` label shows `.auto-filed-badge`.
- Badge has `title` and `aria-label` set to `Auto-filed by test-runner agent`.
- Defect without `auto-filed` label has no badge.
- Non-defect artifact with `auto-filed` label has no badge.
- Multiple auto-filed defects each get a badge; unmarked defect does not.
