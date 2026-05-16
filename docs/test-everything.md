# Test Everything — Run All Tests and Auto-file Defects

The **test-everything** feature adds a `test-runner` agent that executes all configured test suites in one action, parses structured output from each suite, and automatically creates defect artifacts in `lifecycle/defects/` for every genuine failure. Duplicate failures across runs are deduplicated; failures with no matching test artifact are filed under a synthetic `tests-orphaned` lineage.

---

## Why it exists

Before this feature, a failing test suite required a developer to:

1. Read raw test output.
2. Identify each discrete failure.
3. Cross-reference it with the correct lifecycle test artifact.
4. Manually create a defect, set the lineage, choose a role, and write a reproduction command.

This triage was slow and inconsistent. Failures got missed; defects were filed in wrong lineages; routing to the correct developer role depended on tribal knowledge. The problem was most acute before a release, when every failure must be tracked to resolution.

`test-runner` automates the triage entirely. The DevOps pipeline runs tests; the agent turns failures into actionable defect artifacts.

---

## How it works

```
trigger (manual / pipeline / scheduled)
    │
    ▼
Executor.RunAll()
    ├── go test -json ./...
    ├── pnpm test -- --reporter=json     (tests/web/)
    └── npx playwright test --reporter=json  (tests/e2e/, if present)
    │
    ▼
Parsers  →  []SuiteResult  →  []TestFailure
    │
    ▼
ArtifactMapper.MapFailure()
    ├── filename match  (source_file frontmatter / slug)
    ├── label match     (package / directory name)
    └── lineage match   (derived slug)
    │
    ▼
Deduplicator
    ├── GroupByAssertion()   (same file:line → one cluster)
    └── FindDuplicate()      (already-open defect for this failure?)
    │
    ├── duplicate found → AppendWitness() to existing defect
    └── new failure    → DefectFiler.FileDefect()
    │
    ▼
RunSummary  →  broadcast via WebSocket hub
```

All suites run to completion regardless of earlier failures. A suite that fails to produce valid JSON (e.g. a compilation error) produces a single suite-level defect and does not block subsequent suites.

---

## Triggering the agent

### Manual — agent launcher panel

Open **Agents** in the sidebar, select `test-runner`, and click **Run**. Because `test-runner` has no `source_types`, the target artifact picker is hidden and the Run button is enabled immediately. An informational message confirms that no target is required.

### DevOps pipeline — `test-all.yaml`

The pipeline is visible in the **DevOps** view under the **Test** group:

```
Name: Run ALL Tests
Type: test
Steps:
  1. Lint                  — go vet + staticcheck
  2. Go unit tests         — make test-unit
  3. Frontend tests        — cd tests/web && pnpm test
  4. Go integration tests  — make test-integration
  5. E2E smoke tests       — make test-e2e
```

Trigger it from the DevOps UI or via the API:

```sh
curl -s -X POST http://localhost:${KC_PORT}/api/p/${KC_PROJECT}/devops/pipelines/all-tests/run \
  -H "Cookie: session=${KC_SESSION}"
```

### Scheduled

The agent can be registered with the task scheduler to run daily or before a release. When running on a schedule, it skips execution if no source files have changed since the last successful run (checked via git modification times).

---

## Agent configuration

The `test-runner` agent is defined in `lifecycle/config.yaml`:

```yaml
- name: test-runner
  role:
    - qa
  driver: claude-code-cli
  model: sonnet
  active_status: ""        # no status required on a target artifact
  source_types: []         # target-less: no artifact picker shown
  timeout_minutes: 30
  allowed_write_paths:
    - lifecycle/defects
  git_identity:
    name: Test Runner Agent
    email: test-runner@kaos-control.local
  prompt_templates:
    qa: |
      Run all test suites, parse failures, and file defect artifacts.
```

Key points:

- `source_types: []` — the agent is invocable without a `target_path`.
- `allowed_write_paths` is scoped to `lifecycle/defects` — the agent cannot modify test code or other lifecycle artifacts.
- The 30-minute timeout accommodates large test suites.

---

## Test output parsers

All parsers live in `internal/testrunner/` and normalise failures into a common `TestFailure` struct:

```go
type TestFailure struct {
    Suite    string  // "go", "vitest", "playwright"
    Package  string  // Go package path or Vitest file path
    TestName string  // fully qualified test name
    File     string  // source file path
    Line     int     // line number (0 if unknown)
    ErrorMsg string  // assertion / error text
    Output   string  // full test output for diagnostic context
    Elapsed  float64
}
```

### Go — `go test -json`

`ParseGoJSON` reads the newline-delimited JSON stream emitted by `go test -json`. Each JSON object has an `Action` field (`pass`, `fail`, `skip`, `output`, `run`). A test is considered failed when `Action == "fail"` and `Test != ""`.

```json
{"Action":"fail","Package":"github.com/you/kaos/internal/http","Test":"TestArtifactGet","Elapsed":0.041}
```

Subtests are supported — `TestFoo/subcase` is parsed with the full slash-separated name.

If the input is not valid JSON (e.g. a compilation error), the parser sets `SuiteResult.RawError` with the raw text and returns zero failures. The orchestrator then files a single suite-level defect.

### Vitest — `--reporter=json`

`ParseVitestJSON` parses the `testResults` array. Each entry contains `assertionResults[]` with:

- `ancestorTitles` — the `describe` block names
- `title` — the test name
- `status` — `"passed"` | `"failed"` | `"pending"`
- `failureMessages` — array of error strings
- `location.line` — source line

The full test name is built by joining `ancestorTitles` with ` > ` and appending `title`.

Run Vitest with JSON reporting:

```sh
cd tests/web && pnpm test -- --reporter=json
```

### Playwright — `--reporter=json`

`ParsePlaywrightJSON` walks the `suites[] → specs[] → tests[] → results[]` tree. Each result provides `title`, `status`, `error.message`, and `location`.

Run Playwright with JSON reporting:

```sh
cd tests/e2e && npx playwright test --reporter=json
```

The Playwright suite is skipped if `tests/e2e/` does not exist.

---

## Artifact mapping

For each `TestFailure`, the `ArtifactMapper` attempts to find a corresponding `lifecycle/tests/*.md` artifact using three lookup tiers in priority order:

| Tier | What is checked |
|------|-----------------|
| 1. Filename match | Test artifact's `source_file` frontmatter matches the test file's basename (e.g. `auth_test.go`), **or** the artifact's filename-derived slug matches |
| 2. Label match | Test artifact's `labels` contain the test file's parent directory or Go package name |
| 3. Lineage match | Test artifact's `lineage` matches a slug derived from the test file path |

If no match is found across all three tiers, the failure is **orphaned** and filed under the `tests-orphaned` lineage (auto-created if absent).

### Coverage gap detection

After mapping, the `DetectOrphans` function identifies tests that ran but have **no** corresponding `lifecycle/tests/*.md` artifact at all — not just unmatched failures, but entire test files that are undocumented. These appear in the `RunSummary.CoverageGaps` list, visible in the agent run history panel.

---

## Deduplication

Before creating a new defect artifact, the deduplicator checks for existing **open** (non-`done`, non-`abandoned`) defects in the same lineage that match on any of:

| Check | Criterion |
|-------|-----------|
| Test identifier | Same package + test name (Go) or file + test title (Vitest/Playwright) |
| Assertion location | Same file path + line number |
| Error message similarity | First 100 characters match after normalisation |

**Error normalisation** (`NormaliseError`) strips variable content before comparing:

- Timestamps: `2026-05-15`, `2026/05/15`
- Hex pointer addresses: `0x1a2b3c`
- UUIDs: `550e8400-e29b-41d4-a716-446655440000`

Then truncates to 100 characters.

**Within a single run**, failures that share the same `file:line` assertion location are grouped into one cluster by `GroupByAssertion`. Each cluster produces a single defect listing all test names as witnesses.

If a duplicate defect is found, `AppendWitness` adds an entry to the existing defect's body without modifying its frontmatter.

---

## Defect creation

### File location and naming

Defects are created at `lifecycle/defects/<lineage>-<next-index>.md` following the standard lineage convention. The next index is determined by scanning existing files in the lineage — indices are monotonic and never reused.

### Frontmatter

```yaml
title: "TestArtifactGet: want 200, got 404"
type: defect
status: draft
lineage: http-api-tests
parent: lifecycle/tests/http-api-tests-2-test.md
labels:
  - http
  - backend
  - auto-filed
assignees:
  - role: backend-developer
    who: agent
release: KC-Release2
```

Fields are populated as follows:

| Field | Source |
|-------|--------|
| `title` | Derived from test name + first 80 chars of error message |
| `lineage` | From matched test artifact, or `tests-orphaned` |
| `parent` | Path to matched test artifact, or originating idea if orphaned |
| `labels` | Inherited from test artifact, plus `auto-filed` |
| `assignees.role` | Determined by role routing (see below) |
| `assignees.who` | Always `agent` (auto-filed) |
| `release` | Inherited from test artifact or current release context |

### Body structure

```markdown
## Failure

**Suite:** Go
**Test:** `github.com/you/kaos/internal/http.TestArtifactGet`
**Location:** `internal/http/artifact_handler_test.go:47`

```
want status 200, got 404
Response body: {"error":"not found"}
```

## Reproduction

```sh
go test -run TestArtifactGet ./internal/http/
```

## Witnesses

- `TestArtifactGet/with_trailing_slash` — same assertion at line 47
```

### Role routing

The `assignees.role` is set by the following rules in priority order:

1. **Label-based**: if the matched test artifact has label `backend` → `backend-developer`; label `frontend` → `frontend-developer`.
2. **Path-based**:
   - Test file under `internal/` → `backend-developer`
   - Test file under `tests/web/` or `tests/e2e/` → `frontend-developer`
3. **Default**: `backend-developer` when ambiguous.

---

## Run summary

After all suites complete, the orchestrator produces a `RunSummary`:

```go
type RunSummary struct {
    Suites []struct {
        Name    string
        Total   int
        Passed  int
        Failed  int
        Skipped int
        Elapsed float64
    }
    DefectsCreated   int
    DuplicatesFound  int
    OrphanedFailures int
    CoverageGaps     []string // test files with no lifecycle/tests/*.md artifact
    Elapsed          time.Duration
}
```

This is broadcast as an agent run output event over the WebSocket hub and rendered in the **Agent Run History** panel.

### Run history panel display

The `TestRunSummaryCard` component shows:

- A per-suite table: suite name, total / passed / failed / skipped counts, duration.
- A summary line: "X defects created, Y duplicates found, Z orphaned failures."
- A collapsible **Coverage Gaps** section listing undocumented test files.
- Wall-clock duration of the full run.

Defects-created and orphaned-failures counts are highlighted with a warning style when non-zero.

---

## Frontend UI changes

### Agent launcher — target-less invocation

When `test-runner` is selected in the agent launcher:

- The target artifact picker is hidden (because `source_types` is empty).
- An informational message appears: *"This agent runs all test suites — no target artifact required."*
- The **Run** button is enabled without any target selection.

Agents that do require a target (e.g. `qa`) are unaffected — the picker still appears for them.

### Auto-filed badge

Defects created by the `test-runner` agent carry the `auto-filed` label. In the artifact list view, these defects display a small bot icon badge with the tooltip *"Auto-filed by test-runner agent."* The badge appears only on defect-type artifacts; other artifact types are unaffected.

---

## Internal package — `internal/testrunner/`

| File | Purpose |
|------|---------|
| `types.go` | `TestFailure`, `SuiteResult`, `RunSummary` struct definitions |
| `executor.go` | `Executor.RunAll()` — spawns each test suite, pipes output to parsers, records elapsed time |
| `parse_go.go` | `ParseGoJSON(r io.Reader)` — Go `test -json` stream parser |
| `parse_vitest.go` | `ParseVitestJSON(r io.Reader)` — Vitest JSON reporter parser |
| `parse_playwright.go` | `ParsePlaywrightJSON(r io.Reader)` — Playwright JSON reporter parser |
| `mapper.go` | `ArtifactMapper.MapFailure()`, `DetectOrphans()` |
| `dedup.go` | `Deduplicator.FindDuplicate()`, `GroupByAssertion()`, `NormaliseError()` |
| `defect.go` | `DefectFiler.FileDefect()`, `AppendWitness()`, `routeRole()`, `buildTitle()`, `buildReproduction()` |
| `agent.go` | `Run()` — top-level orchestrator; wires all components together and emits the summary |

Unit tests live alongside each source file (`*_test.go`). Integration tests live in `tests/integration/` under the `integration` build tag.

---

## Non-goals

- **Does not fix failures automatically** — the agent files defects; developers fix them.
- **Does not replace the `qa` agent's per-artifact flow** — `qa` verifies a specific test artifact; `test-runner` runs everything.
- **Does not support test frameworks other than Go testing, Vitest, and Playwright** in this version.
- **Does not run in external CI/CD** — it orchestrates runs inside kaos-control's agent/pipeline system. For GitHub Actions integration, see the [End-to-End Smoke Tests](end-to-end-smoke-tests.md) documentation.

---

## Limitations and known constraints

- **NF1 — Performance overhead**: parsing, mapping, deduplication, and filing must add less than 10 seconds beyond actual test execution time.
- **NF2 — Idempotency**: running the agent twice against identical failures produces no new defects.
- **NF3 — Isolation**: the agent reads test output and writes only to `lifecycle/defects/`. Test code and test artifacts are never modified.
- **NF4 — Graceful suite failure**: a compilation error or other suite-level failure produces one suite-level defect and does not halt the run.
- **NF5 — Compatibility**: Go 1.25+ `go test -json` format and Vitest 3.x `--reporter=json` format are required.

---

## Related documentation

- [End-to-End Smoke Tests](end-to-end-smoke-tests.md) — the Playwright smoke suite that `test-runner` can exercise.
- `lifecycle/requirements/test-everything-2.md` — authoritative requirements and acceptance criteria.
- `lifecycle/config.yaml` — full agent configuration.
- `lifecycle/devops/all-tests.yaml` — the DevOps pipeline definition.
