---
title: Run all tests and auto-file defects for failures — Requirements
type: requirement
status: done
lineage: test-everything
priority: high
parent: lifecycle/ideas/test-everything.md
labels:
    - test
    - testing
    - qa
    - workflow
    - artefacts
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Run all tests and auto-file defects for failures — Requirements

## Problem

The project has multiple test suites — Go unit/integration tests (`make test-unit`), frontend Vitest component tests (`tests/web/`), and upcoming E2E smoke tests (`tests/e2e/`). When a suite fails, the developer must manually read the output, identify each failure, cross-reference it with the relevant lifecycle artifact, and decide who should fix it. This manual triage is error-prone and slow: failures get missed, defects are filed inconsistently, and routing to the correct developer role depends on tribal knowledge.

There is no automated mechanism that:
1. Runs the full test battery on demand or on a schedule.
2. Parses structured test output to extract individual failures.
3. Maps each failure to its corresponding `lifecycle/tests/*.md` artifact.
4. Creates well-formed defect artifacts with correct lineage, routing, and deduplication.

The gap is most painful before a release, when the full suite must pass and every failure must be tracked to resolution.

## Goals / Non-goals

### Goals

- Provide a single action (agent invocation or DevOps pipeline trigger) that runs all test suites and produces a defect artifact for every genuine failure.
- Parse structured test output from both Go (`go test -json`) and Vitest (`--reporter=json`) to extract per-test pass/fail with file and line information.
- Map test failures to existing `lifecycle/tests/*.md` artifacts so defects are filed under the correct lineage.
- Deduplicate failures: multiple assertions failing at the same file/line or with the same error message produce one defect with all witnesses listed.
- Route each defect to the correct developer role based on test artifact labels or test file path conventions.
- Support both on-demand invocation (manual trigger, no `target_path`) and scheduled invocation (daily or pre-release).

### Non-goals

- Replacing or modifying the existing `qa` agent's per-artifact test verification flow.
- Fixing test failures automatically — the agent files defects, humans (or other agents) fix them.
- Providing a UI dashboard for test results beyond the existing artifact list and defect views.
- Supporting test frameworks other than Go's `testing` package and Vitest in the initial implementation.
- Running tests in CI/CD — this feature orchestrates test runs within kaos-control's own agent/pipeline system, not in an external CI provider.

## Detailed Requirements

### Functional

#### F1 — Agent configuration

- A new agent entry `test-runner` (or extend the existing `qa` agent with a `mode: full-suite` parameter) must be defined in `lifecycle/config.yaml`.
- The agent is invocable without a `target_path`. When no target is provided, it runs the full test battery.
- The agent's `allowed_write_paths` must include `lifecycle/defects/` for creating defect artifacts.
- The agent must be assignable to the `qa` role.

#### F2 — Test execution

- The agent must execute the following test suites in sequence:
  1. **Go tests**: `go test -json ./...` from the project root.
  2. **Frontend tests**: `pnpm test -- --reporter=json` from `tests/web/`.
  3. **E2E tests** (if present): `npx playwright test --reporter=json` from `tests/e2e/`.
- Each suite's exit code is captured but does not halt execution of subsequent suites — all suites run to completion.
- The agent captures both structured JSON output and raw stderr for diagnostic context.

#### F3 — Output parsing

- **Go JSON output**: parse the stream of JSON objects emitted by `go test -json`. Extract `Test`, `Package`, `Action` (pass/fail/skip), `Output`, and `Elapsed` fields. A test is considered failed when `Action` is `"fail"` and `Test` is non-empty.
- **Vitest JSON output**: parse the `testResults` array. Each entry contains `name`, `status`, `assertionResults[]` with `ancestorTitles`, `title`, `status`, `failureMessages`, and `location` (file/line).
- **Playwright JSON output**: parse the `suites[]` → `specs[]` → `tests[]` → `results[]` structure. Extract `title`, `status`, `error.message`, and `location`.
- Skipped tests are recorded but do not produce defects.

#### F4 — Artifact mapping

- For each failed test, the agent must attempt to find a corresponding `lifecycle/tests/*.md` artifact using the following lookup order:
  1. **Filename match**: the test file's basename (e.g., `artifact_store_test.go`) maps to a test artifact with a matching `source_file` frontmatter field or a filename-derived slug.
  2. **Label match**: the test artifact's `labels` contain a label matching the test file's parent directory or package name.
  3. **Lineage match**: the test artifact's `lineage` field matches the test file's conventional slug.
- If no matching artifact is found, the failure is filed under a synthetic `tests-orphaned` lineage. The defect's body must note that no matching test artifact was found.

#### F5 — Deduplication

- Before creating a defect, the agent must check for existing open (non-`done`, non-`abandoned`) defects in the same lineage with:
  - The same failing test identifier (package + test name for Go, file + test title for Vitest/Playwright), OR
  - The same assertion location (file path + line number), OR
  - A substantially similar error message (first 200 characters match after stripping variable content like timestamps and pointer addresses).
- If a duplicate is found, the agent appends a witness entry to the existing defect's body rather than creating a new artifact.
- If multiple failures in the same run share the same root assertion (same file/line), they are grouped into a single defect with all failing test names listed as witnesses.

#### F6 — Defect creation

- Each defect artifact must be created at `lifecycle/defects/<lineage>-<next-index>.md` following the lineage convention.
- Defect frontmatter must include:
  - `title`: concise description derived from the test name and error message.
  - `type: defect`
  - `status: draft`
  - `lineage`: inherited from the matched test artifact, or `tests-orphaned`.
  - `parent`: path to the matched test artifact, or the originating idea if orphaned.
  - `labels`: inherited from the test artifact, plus `auto-filed`.
  - `assignees`: role determined by F7 routing.
  - `release`: inherited from the test artifact or the current release context.
- Defect body must include:
  - The failing test name and location (file:line).
  - The error message or assertion failure text.
  - The suite that produced the failure (Go / Vitest / Playwright).
  - All witness entries if multiple failures were deduplicated.
  - A `## Reproduction` section with the exact command to re-run just that test.

#### F7 — Role routing

- The defect's `assignees.role` is determined by:
  1. The matched test artifact's labels: `backend` → `backend-developer`, `frontend` → `frontend-developer`.
  2. The test file's path: `tests/web/` or `tests/e2e/` → `frontend-developer`; Go test files under `internal/` → `backend-developer`.
  3. If ambiguous, default to `backend-developer`.
- The `assignees.who` field is set to `agent` to indicate the defect was auto-filed.

#### F8 — Run summary

- After processing all suites, the agent must produce a summary artifact or log entry containing:
  - Total tests run, passed, failed, skipped per suite.
  - Number of defects created vs. duplicates found.
  - Number of orphaned failures (no matching test artifact).
  - Wall-clock duration of the full run.
- This summary is emitted as an agent run output visible in the agent run history panel.

#### F9 — Trigger modes

- **Manual**: the agent can be triggered from the agent launcher panel without a `target_path`.
- **Pipeline**: a new DevOps pipeline `test-all.yaml` (or a step in an existing pipeline) invokes the agent.
- **Scheduled**: the agent can be scheduled to run daily or before a release via the existing agent task scheduler (see [[agent-task-scheduler]]).

### Non-functional

- **NF1 — Performance.** The agent's overhead (parsing, mapping, deduplication, defect creation) must add less than 10 seconds beyond the actual test execution time.
- **NF2 — Idempotency.** Running the agent twice against the same set of failures must not create duplicate defects (enforced by F5).
- **NF3 — Isolation.** The agent must not modify any test code or test artifacts. It only reads test output and writes to `lifecycle/defects/`.
- **NF4 — Graceful failure.** If a test suite fails to produce valid JSON output (e.g., compilation error), the agent logs the raw output, files a single defect noting the suite-level failure, and continues to the next suite.
- **NF5 — Compatibility.** Must work with Go 1.25+ `go test -json` output format and Vitest 3.x `--reporter=json` format.

## Acceptance Criteria

- [ ] Agent can be invoked without a `target_path` and runs all configured test suites to completion.
- [ ] Go test failures are parsed from `go test -json` output and produce correctly formed defect artifacts in `lifecycle/defects/`.
- [ ] Vitest failures are parsed from `--reporter=json` output and produce correctly formed defect artifacts.
- [ ] Each defect artifact has correct lineage, parent, labels, and role assignment per F4/F6/F7.
- [ ] Five failures at the same assertion location produce exactly one defect with five witnesses listed (F5 deduplication).
- [ ] Running the agent twice on identical failures does not create duplicate defects.
- [ ] Failures with no matching `lifecycle/tests/*.md` artifact are filed under `tests-orphaned` lineage.
- [ ] Defect role routing matches: `internal/` test failures → `backend-developer`, `tests/web/` failures → `frontend-developer`.
- [ ] Agent run summary shows correct totals for tests run, passed, failed, skipped, defects created, and duplicates found.
- [ ] `test-all.yaml` pipeline invokes the agent and the pipeline status reflects the test outcome.
- [ ] Agent completes parsing and defect filing within 10 seconds after test suites finish (NF1).
- [ ] Agent handles a suite that fails to compile (no JSON output) by filing a single suite-level defect and continuing (NF4).
- [ ] Related artifacts: [[end-to-end-smoke-tests]], [[agent-task-scheduler]], [[test-artifact-management]].

## Resolved Questions

- Should the agent also detect and report **new tests that have no corresponding `lifecycle/tests/*.md` artifact** as a separate finding (not a defect, but a coverage gap signal)?

> Yes

- What is the threshold for "substantially similar error message" in deduplication (F5)? Is first-200-characters sufficient, or should a more sophisticated similarity metric be used?

> 100 characters will be good for now.

- Should the `tests-orphaned` lineage be auto-created if it does not exist, or should it be a pre-seeded artifact?

> auto-created.

- When running on a schedule (daily), should the agent skip the run if no source files have changed since the last successful run?

> yes.
