---
title: 'Backend Plan: Run All Tests and Auto-file Defects'
type: plan-backend
status: draft
lineage: test-everything
parent: lifecycle/requirements/test-everything-2.md
---

## Overview

Implement the backend infrastructure for the test-everything feature: a new `test-runner` agent configuration, a test execution orchestrator that runs Go/Vitest/Playwright suites and captures structured JSON output, parsers for each output format, artifact mapping logic, deduplication, defect creation with correct lineage and role routing, and a `test-all.yaml` DevOps pipeline. The agent operates without a `target_path` and writes only to `lifecycle/defects/`.

Related: [[test-everything]], [[devops-pipelines]], [[agent-task-scheduler]]

## Milestone 1 — Test Output Parsers

### Description

Create an `internal/testrunner/` package with parsers for each supported test framework's JSON output. Each parser normalises failures into a common `TestFailure` struct.

### Files to change

- `internal/testrunner/types.go` (new) — Define shared types:
  ```go
  type TestFailure struct {
      Suite       string // "go", "vitest", "playwright"
      Package     string // Go package or Vitest file
      TestName    string // fully qualified test name
      File        string // source file path
      Line        int    // line number, 0 if unknown
      ErrorMsg    string // assertion/error text
      Output      string // full test output for context
      Elapsed     float64
  }
  type SuiteResult struct {
      Suite    string
      Total    int
      Passed   int
      Failed   int
      Skipped  int
      Elapsed  float64
      Failures []TestFailure
      RawError string // non-empty if suite failed to produce JSON (NF4)
  }
  ```
- `internal/testrunner/parse_go.go` (new) — `ParseGoJSON(r io.Reader) (*SuiteResult, error)` reads the newline-delimited JSON stream from `go test -json`. Extracts `Test`, `Package`, `Action`, `Output`, `Elapsed`. Accumulates output lines per test. Marks a test as failed when `Action == "fail"` and `Test != ""`. If input is not valid JSON, returns a `SuiteResult` with `RawError` set (NF4).
- `internal/testrunner/parse_vitest.go` (new) — `ParseVitestJSON(r io.Reader) (*SuiteResult, error)` parses the Vitest JSON reporter format. Iterates `testResults[].assertionResults[]`, extracts `ancestorTitles`, `title`, `status`, `failureMessages`, `location.line`. Builds `TestName` from ancestors + title.
- `internal/testrunner/parse_playwright.go` (new) — `ParsePlaywrightJSON(r io.Reader) (*SuiteResult, error)` parses `suites[].specs[].tests[].results[]`. Extracts `title`, `status`, `error.message`, `location`.
- `internal/testrunner/parse_go_test.go` (new) — Unit tests for Go parser with fixture data.
- `internal/testrunner/parse_vitest_test.go` (new) — Unit tests for Vitest parser.
- `internal/testrunner/parse_playwright_test.go` (new) — Unit tests for Playwright parser.

### Acceptance criteria

- [ ] Go parser correctly extracts failures from `go test -json` output including package, test name, file, and error output.
- [ ] Go parser handles compilation errors (non-JSON output) by returning `SuiteResult.RawError` (NF4).
- [ ] Vitest parser extracts failures with ancestor titles, location, and failure messages.
- [ ] Playwright parser extracts failures from nested suite/spec/test structure.
- [ ] All parsers correctly count total, passed, failed, and skipped tests.
- [ ] `go test ./internal/testrunner/ -short` passes.

## Milestone 2 — Test Suite Executor

### Description

Build the orchestrator that runs each test suite sequentially, captures structured JSON output and stderr, and handles suite-level failures gracefully.

### Files to change

- `internal/testrunner/executor.go` (new) — `Executor` struct with:
  - `RunAll(ctx context.Context, projectDir string) ([]SuiteResult, error)` — runs suites in order: Go, Vitest, Playwright (if `tests/e2e/` exists). Each suite runs via `exec.CommandContext`; stdout is piped to the appropriate parser; stderr is captured separately. Exit codes are recorded but do not halt subsequent suites. Wall-clock time is tracked per suite.
  - `runGoTests(ctx, dir)`, `runVitestTests(ctx, dir)`, `runPlaywrightTests(ctx, dir)` — internal methods for each suite. Vitest runs from `tests/web/`, Playwright from `tests/e2e/`.
- `internal/testrunner/executor_test.go` (new) — Unit tests with mock commands (use `exec.Command` with test helper pattern).

### Acceptance criteria

- [ ] All configured suites run to completion even if earlier suites fail.
- [ ] Both JSON output and stderr are captured for each suite.
- [ ] If a suite fails to produce valid JSON (compilation error), the result has `RawError` populated and no parsed failures.
- [ ] Playwright suite is skipped if `tests/e2e/` does not exist.
- [ ] Wall-clock elapsed time is recorded per suite.

## Milestone 3 — Artifact Mapping

### Description

Implement the logic that maps each `TestFailure` to a `lifecycle/tests/*.md` artifact using the three-tier lookup described in F4.

### Files to change

- `internal/testrunner/mapper.go` (new) — `ArtifactMapper` struct initialised with a reference to the index:
  - `MapFailure(f TestFailure) (matchedArtifact *artifact.Artifact, err error)` — attempts lookup in order:
    1. **Filename match**: queries the index for test artifacts where `source_file` frontmatter matches the test file's basename, or where the artifact slug is derived from the test filename.
    2. **Label match**: queries for test artifacts whose `labels` contain the test file's parent directory or Go package name.
    3. **Lineage match**: queries for test artifacts whose `lineage` matches a slug derived from the test file path.
  - Returns `nil` artifact if no match (caller uses `tests-orphaned` lineage).
- `internal/testrunner/mapper.go` — `DetectOrphans(results []SuiteResult, testArtifacts []*artifact.Artifact) []string` — finds tests that have no corresponding `lifecycle/tests/*.md` artifact (coverage gap signal per resolved question).
- `internal/testrunner/mapper_test.go` (new) — Unit tests for each lookup tier and the orphan detector.

### Acceptance criteria

- [ ] Filename-based matching correctly maps `artifact_store_test.go` to a test artifact with matching `source_file` or slug.
- [ ] Label-based matching finds a test artifact when labels include the Go package name.
- [ ] Lineage-based matching works as a fallback.
- [ ] Unmatched failures return `nil` for the artifact, indicating orphaned status.
- [ ] Orphan detector identifies tests with no corresponding test artifact.

## Milestone 4 — Deduplication Engine

### Description

Implement deduplication logic (F5) that checks existing open defects before creating new ones, and groups failures from the same run that share a root assertion.

### Files to change

- `internal/testrunner/dedup.go` (new) — `Deduplicator` struct with index access:
  - `FindDuplicate(f TestFailure, lineage string) (*artifact.Artifact, error)` — queries the index for open (non-`done`, non-`abandoned`) defects in the given lineage matching by:
    1. Same test identifier (package + test name for Go; file + test title for Vitest/Playwright).
    2. Same assertion location (file + line).
    3. Similar error message (first 100 characters match after stripping timestamps and pointer addresses via regex).
  - `GroupByAssertion(failures []TestFailure) [][]TestFailure` — groups failures sharing the same file:line into clusters; each cluster produces one defect with all test names as witnesses.
- `internal/testrunner/dedup.go` — `NormaliseError(msg string) string` — strips variable content (timestamps matching `\d{4}[-/]\d{2}[-/]\d{2}`, hex pointers `0x[0-9a-f]+`, UUIDs) and returns the first 100 characters.
- `internal/testrunner/dedup_test.go` (new) — Tests for duplicate detection, assertion grouping, and error normalisation.

### Acceptance criteria

- [ ] An existing open defect with the same test identifier is found as a duplicate.
- [ ] An existing open defect at the same file:line is found as a duplicate.
- [ ] Error messages differing only in timestamps or pointer addresses are considered similar.
- [ ] Five failures at the same file:line are grouped into one cluster.
- [ ] Closed (`done`/`abandoned`) defects are not considered duplicates.
- [ ] `NormaliseError` strips timestamps, hex pointers, and UUIDs, then truncates to 100 chars.

## Milestone 5 — Defect Filing

### Description

Create defect artifacts in `lifecycle/defects/` with correct frontmatter, lineage, role routing (F7), witness entries, and reproduction commands. Append witness entries to existing defects when duplicates are found.

### Files to change

- `internal/testrunner/defect.go` (new) — `DefectFiler` struct:
  - `FileDefect(failures []TestFailure, matched *artifact.Artifact, projectDir string) (string, error)` — constructs the defect artifact content per F6 requirements. Determines the next lineage index by scanning existing files. Writes the file via `atomicWrite` and re-indexes via `idx.IndexFile`. Returns the created file path.
  - `AppendWitness(defectPath string, f TestFailure) error` — appends a witness entry to an existing defect's body.
  - `routeRole(f TestFailure, matched *artifact.Artifact) string` — implements F7 routing: label-based first, then path-based (`internal/` → `backend-developer`, `tests/web/` or `tests/e2e/` → `frontend-developer`), default `backend-developer`.
  - `buildTitle(f TestFailure) string` — derives a concise title from test name and error message.
  - `buildReproduction(f TestFailure) string` — generates the exact command to re-run just that test.
- `internal/testrunner/defect_test.go` (new) — Tests for defect creation, witness appending, role routing, and title generation.

### Acceptance criteria

- [ ] Created defect has correct frontmatter: `type: defect`, `status: draft`, correct `lineage`, `parent`, `labels` (including `auto-filed`), `assignees`, and `release`.
- [ ] Defect body includes failing test name, location, error message, suite name, and reproduction command.
- [ ] Grouped failures produce one defect with all witnesses listed.
- [ ] Witness append adds to existing defect body without corrupting frontmatter.
- [ ] Role routing: `internal/` failures → `backend-developer`, `tests/web/` failures → `frontend-developer`.
- [ ] Orphaned failures use `tests-orphaned` lineage and note the missing test artifact.
- [ ] The `tests-orphaned` lineage idea artifact is auto-created if it does not exist.
- [ ] Lineage index is correctly incremented (no reuse).

## Milestone 6 — Agent Orchestrator & Configuration

### Description

Wire the parsers, executor, mapper, deduplicator, and defect filer together into a coherent agent flow. Configure the `test-runner` agent in `lifecycle/config.yaml`.

### Files to change

- `internal/testrunner/agent.go` (new) — `Run(ctx context.Context, projectDir string, idx *index.Index, hub *hub.Hub) (*RunSummary, error)`:
  1. Check if source files changed since last successful run (for scheduled mode — query git or file modification times).
  2. Call `Executor.RunAll()` to run all suites.
  3. For each suite's failures, call `ArtifactMapper.MapFailure()`.
  4. Call `Deduplicator.GroupByAssertion()` to cluster same-location failures.
  5. For each cluster, call `Deduplicator.FindDuplicate()` — if found, `AppendWitness()`; if not, `DefectFiler.FileDefect()`.
  6. Call `DetectOrphans()` and include coverage gaps in the summary.
  7. Assemble and return `RunSummary` (totals per suite, defects created, duplicates found, orphans, wall-clock time).
  8. Broadcast summary via hub as an agent run output event.
- `internal/testrunner/agent.go` — `RunSummary` struct with fields for total/passed/failed/skipped per suite, defects created, duplicates found, orphaned failures, coverage gaps, and elapsed duration.
- `lifecycle/config.yaml` — Add `test-runner` agent entry:
  ```yaml
  test-runner:
    roles: [qa]
    driver: claude-code-cli
    model: sonnet
    active_status: ""
    source_types: []
    timeout_minutes: 30
    allowed_write_paths:
      - lifecycle/defects
    prompt: |
      Run all test suites, parse failures, and file defect artifacts.
  ```
- `internal/agent/agent.go` — Ensure `Manager.StartRun` with empty `targetPath` works for the `test-runner` agent (already supported per exploration, but verify no regressions).

### Acceptance criteria

- [ ] `test-runner` agent can be invoked without a `target_path`.
- [ ] Full flow executes: run suites → parse → map → deduplicate → file defects → summary.
- [ ] Run summary includes correct totals for all suites.
- [ ] Coverage gaps (tests without corresponding test artifacts) are reported in the summary.
- [ ] Scheduled runs skip execution if no source files changed since last successful run.
- [ ] Agent overhead (parsing + mapping + dedup + filing) completes within 10 seconds (NF1).
- [ ] Running the agent twice on identical failures does not create duplicate defects (NF2).

## Milestone 7 — DevOps Pipeline Definition

### Description

Create the `test-all.yaml` DevOps pipeline that invokes the `test-runner` agent, making the full test battery triggerable from the DevOps UI.

### Files to change

- `lifecycle/devops/test-all.yaml` (new) — Pipeline definition:
  ```yaml
  name: Run All Tests
  type: test
  steps:
    - name: Run test-runner agent
      description: Execute all test suites and file defects for failures
      command: |
        # Invoke the test-runner agent via the kaos-control API
        curl -s -X POST http://localhost:${KC_PORT}/api/p/${KC_PROJECT}/agents/test-runner/run \
          -H "Content-Type: application/json" \
          -H "Cookie: session=${KC_SESSION}" \
          -d '{}'
      timeout: 1800
  ```

### Acceptance criteria

- [ ] `test-all.yaml` is discovered by the DevOps pipeline discovery.
- [ ] The pipeline can be triggered from the DevOps UI ([[devops-pipelines]]).
- [ ] Pipeline status reflects the test-runner agent's outcome.
- [ ] Pipeline timeout (30 minutes) accommodates large test suites.
