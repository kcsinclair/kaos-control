---
title: "Auto-Triage Raw Ideas — Test Plan"
type: plan-test
status: in-development
lineage: auto-triage-new-ideas
parent: lifecycle/requirements/auto-triage-new-ideas-2.md
assignees:
    - role: test-developer
      who: agent
---

# Auto-Triage Raw Ideas — Test Plan

Integration tests live in repo-root `tests/` and exercise the triage subsystem end-to-end against a real `testEnv` (admin-logged-in HTTP test server, per memory `project-kaos-control`). LLM calls are replaced with a deterministic fake `ideachat.CallLLM` so tests do not hit the network.

Cross-references: [[auto-triage-new-ideas-3-be]] (backend implementation under test), [[auto-triage-new-ideas-4-fe]] (frontend smoke pass; not retested here).

---

## Milestone 1 — LLM fake and shared fixtures

### Description

Provide a deterministic in-process replacement for the LLM call so tests are hermetic and fast. All subsequent milestones build on this fixture.

### Files to change

- `tests/triage/fixtures_test.go` (new — package `triage_test`):
  - `func installLLMFake(t *testing.T, scripted []string)` — swaps `ideachat.CallLLM` (or whatever package-level var the production code exposes) for a function that pops responses from `scripted` in order. Restores on `t.Cleanup`.
  - `func defaultProposeJSON(slug, title string, labels []string) string` — returns a valid `idea-generate` JSON response block matching the production prompt's contract.
  - `func writeRawIdea(t *testing.T, projectRoot, slug, title, body string) string` — writes a `lifecycle/ideas/<slug>.md` artifact with `status: raw`, `type: idea`, `lineage: <slug>` and returns its rel path.
  - `func readArtifact(t *testing.T, projectRoot, relPath string) (frontmatter map[string]any, body string)`.
- `tests/triage/env_test.go` (new):
  - `func newTriageEnv(t *testing.T) *testEnv` — thin wrapper around the existing project-wide `testEnv` helper that additionally installs the LLM fake by default. Returns the env so each test can override the scripted responses.

### Acceptance criteria

- [ ] `go test ./tests/triage/...` compiles.
- [ ] `installLLMFake` is symmetric: when the test finishes, the package-level var is restored to the production implementation (verified by a follow-up test reading the var directly).
- [ ] `defaultProposeJSON` produces JSON that `ideachat.parseAction` accepts as a `propose` action without error.

---

## Milestone 2 — Eligibility filter tests

### Description

Unit-level coverage of `internal/triage.eligible` driven directly (not through the API) — fast, deterministic, no LLM needed.

### Files to change

- `internal/triage/eligibility_test.go` (new — package `triage`):

### Test cases

1. **Path outside ideas dir** — `lifecycle/requirements/foo.md` with `type: idea`, `status: raw` → `ok=false`, `reason="not_in_ideas_dir"`.
2. **Nested under ideas** — `lifecycle/ideas/sub/foo.md` → `not_in_ideas_dir` (FR-2 says exact `lifecycle/ideas/*.md`).
3. **Wrong type** — `lifecycle/ideas/foo.md`, `type: defect`, `status: raw` → `wrong_type`.
4. **Wrong status: draft** — `type: idea`, `status: draft` → `wrong_status`.
5. **Wrong status: clarifying** → `wrong_status`.
6. **Eligible** — `type: idea`, `status: raw` → `ok=true`.
7. **Not indexed** — path the index has no row for → `not_indexed`.
8. **Case sensitivity** — `Status: Raw` (capital R) → `wrong_status` (status comparison is exact per FR-2).

### Acceptance criteria

- [ ] All 8 cases pass.
- [ ] No case takes longer than 50 ms.
- [ ] Tests use a stub index (`index.Reader` interface implemented by an in-memory map) — no SQLite needed.

---

## Milestone 3 — Body rewrite and frontmatter mutation tests

### Description

Direct unit tests of the FR-3 / FR-4 transformation logic in `internal/triage/run.go`, bypassing the LLM. Each test feeds in a synthetic `ideachat.GenerateResult` and asserts the resulting file bytes.

### Files to change

- `internal/triage/run_test.go` (new — package `triage`).

### Test cases

1. **Fresh triage** — original body is `"# Title\n\nbrain-dump text"`. After rewrite, body is `"# Title\n\n## Raw Idea\n\nbrain-dump text\n\n## Idea\n\n<agent body>"`.
2. **Re-run** — original already contains `## Raw Idea` and `## Idea`. After rewrite: `## Raw Idea` block is byte-identical; `## Idea` block is replaced.
3. **No H1** — original body has no leading H1. After rewrite, body starts with `## Raw Idea` (no synthesised heading) and original lines go below.
4. **Agent body has H1** — `result.Body` starts with `# Foo`. Resulting `## Idea` block does NOT contain the duplicate H1.
5. **Frontmatter: labels merge** — existing labels `[a, b]`, agent proposes `[b, c, d]`, vocabulary is `{a, b, c}`. Final: `[a, b, c]` (existing first, agent appended, dedup, vocabulary-filtered drops `d`).
6. **Frontmatter: priority absent → set normal** — existing frontmatter has no `priority`; agent proposes `high`. Final priority is `normal` (resolved Q: `normal` as default).
7. **Frontmatter: priority present → preserved** — existing `priority: high`; agent proposes `normal`. Final priority is `high`.
8. **Frontmatter: other keys preserved** — `release`, `parent`, `lineage`, `created`, `assignees`, custom `foo: bar` all survive unchanged.
9. **Empty agent body** — `result.Body == ""` → mutation function returns an error and does not produce a write.
10. **Title change in frontmatter** — agent proposes a new title; if applied (FR-3), the H1 line in the body matches the new frontmatter `title`.

### Acceptance criteria

- [ ] All 10 cases pass.
- [ ] Each test asserts exact byte content (or a normalised diff) — no "contains" checks for the body structure.
- [ ] Tests run in under 1 s total.

---

## Milestone 4 — Concurrency, dedup, and lock tests

### Description

Exercise FR-7 (single concurrent run per path, global cap of N, lineage-lock interaction).

### Files to change

- `internal/triage/concurrency_test.go` (new — package `triage`).

### Test cases

1. **Coalesce same path** — two `Trigger` calls for the same path while a synthetic "long-running" inner execute is blocked on a channel start exactly one inner execute and both calls return the same `run_id`.
2. **Cap honoured** — `MaxConcurrent = 2`, three distinct paths triggered simultaneously. Verify exactly 2 inner executes start; the third is blocked until one slot frees.
3. **Cap respects ordering loosely** — the third trigger eventually completes (no deadlock) — assert via a timeout-bounded wait.
4. **Locked lineage** — pre-acquire the lineage lock for `slug`, then `Trigger`. Expect `ErrLocked`, no inner execute, no `agent_runs` row.
5. **Lock released on failure** — inner execute returns an error → the lineage lock is released (verified by acquiring it again after the call returns, with no timeout).
6. **Lock released on panic** — inner execute panics → manager recovers, records run as failed, releases the lock.
7. **Stop drains in-flight** — `Stop(ctx)` blocks until all in-flight inner executes finish; verified by setting a `t.Cleanup(cancel)` and ensuring no goroutine leak via `goleak` (if not already a project dep, skip the goleak assertion and use a `sync.WaitGroup` counter instead).

### Acceptance criteria

- [ ] All 7 cases pass.
- [ ] No test relies on `time.Sleep` for synchronisation longer than 50 ms; use channels for handshake.
- [ ] No goroutine is leaked at the end of any test.

---

## Milestone 5 — Watcher → triage integration tests

### Description

Spin up a project with the real watcher + index + triage manager and verify the file-creation → triage flow.

### Files to change

- `tests/triage/watcher_test.go` (new — package `triage_test`).

### Test cases

1. **Create `raw` idea → triage runs** — `writeRawIdea(env, "alpha", ...)`. Within 2 s, `GET /api/p/<project>/artifacts/lifecycle/ideas/alpha.md` returns `status: draft` and body containing `## Raw Idea` and `## Idea`.
2. **Create `draft` idea → no triage** — write a file with `status: draft`. After 2 s, `agent_runs` for that target_path is empty.
3. **Create `raw` defect → no triage** — write `lifecycle/ideas/bug.md` with `type: defect`, `status: raw`. After 2 s, no `agent_runs` row.
4. **Modify a `draft` idea → no triage** — pre-create a `draft` idea, modify its body. After 2 s, no new `agent_runs` row.
5. **Two rapid writes within debounce → one run** — write the same `raw` file twice within 150 ms. After 2 s, exactly one `agent_runs` row.
6. **Re-run via status reset** — after step 1, manually set `status: raw` again (write the file with `status: raw` and the existing `## Raw Idea` / `## Idea` sections). Triage runs again; the resulting body's `## Raw Idea` block is byte-identical to the previous version, `## Idea` is replaced.

### Acceptance criteria

- [ ] All 6 cases pass.
- [ ] Each test isolates its project root via `t.TempDir()`.
- [ ] LLM fake produces deterministic content; no test asserts the exact agent-generated text, only its placement.

---

## Milestone 6 — Startup re-scan tests

### Description

Exercise FR-1 bullet 2 (server-startup enqueue of pre-existing `raw` ideas).

### Files to change

- `tests/triage/startup_test.go` (new — package `triage_test`).

### Test cases

1. **Single raw on startup** — write `lifecycle/ideas/foo.md` with `status: raw` BEFORE calling `project.Open`. After `Open` returns, within 5 s the artifact is `draft`.
2. **No raw on startup** — empty `lifecycle/ideas/`. `Open` completes; no `agent_runs` rows are inserted (verified by querying the table).
3. **Multiple raws on startup with cap** — write 3 `raw` ideas, set `MaxConcurrent = 2`. All three are eventually triaged; observation: at any single moment during the run, no more than 2 are `in flight` (verified by reading `agent_runs.status` while sleeping briefly in a polling loop).
4. **Re-startup with already-`draft` artifacts** — write 2 `draft` ideas (simulating a previously triaged project). `Open` completes; no new `agent_runs` rows are inserted.

### Acceptance criteria

- [ ] All 4 cases pass.
- [ ] Tests respect `t.TempDir()` isolation; no shared state across tests.
- [ ] Total milestone runtime under 15 s.

---

## Milestone 7 — REST endpoint tests

### Description

Cover the FR-8 endpoint behaviours via real HTTP requests against `testEnv`.

### Files to change

- `tests/triage/api_test.go` (new — package `triage_test`).

### Test cases

1. **Unauthenticated → 401** — call `POST /api/p/<project>/ideas/foo/triage` with no session cookie.
2. **Authenticated but wrong role → 403** — sign in as a synthetic `backend-developer`-only user; expect 403.
3. **Unknown slug → 404** — call against `nope`. Body contains `not_found`.
4. **Already draft → 409 with `wrong_status`** — write a `draft` idea, call endpoint, expect 409 and `reason: "wrong_status"`.
5. **Wrong type under ideas → 409 with `wrong_type`** — write a `type: defect`, `status: raw` file under `lifecycle/ideas/`, expect 409 and `reason: "wrong_type"`.
6. **Success → 202 with run_id** — write a `raw` idea, call endpoint, expect 202 and a non-empty `run_id`. Poll `GET /api/p/<project>/runs` (or whatever endpoint lists agent runs) and verify the run completes with `status: success` within 5 s.
7. **In-flight coalesce** — install an LLM fake that blocks on a channel. Call the endpoint twice in quick succession. Both calls return 202 with the SAME `run_id`. Verify only one `agent_runs` row exists.
8. **Locked lineage → 409 with `locked`** — pre-acquire the lineage lock via `p.Locks.Acquire`. Call endpoint, expect 409 with `error: "locked"`.

### Acceptance criteria

- [ ] All 8 cases pass.
- [ ] Each case uses the existing `testEnv` URL helpers (per memory `project-kaos-control`, URL helpers return full URLs for `http.Get`).
- [ ] No test takes longer than 10 s.

---

## Milestone 8 — Failure and observability tests

### Description

Exercise FR-10 (failure → artifact untouched, run failed, warn log) and NFR-2 (log line content).

### Files to change

- `tests/triage/failure_test.go` (new — package `triage_test`).

### Test cases

1. **Malformed JSON from LLM** — fake returns `"not json"`. Triage fails; the artifact bytes are byte-identical to pre-call; `agent_runs` row has `status: failed` and `stderr` contains the parse error string; the artifact frontmatter still has `status: raw`.
2. **`action: clarify` rejected** — fake returns a valid JSON envelope but with `"action": "clarify"`. Run is failed (FR-10 lists "action other than `propose`" as invalid).
3. **Empty body** — fake returns `"action": "propose"` with empty `"body"`. Run is failed; artifact untouched.
4. **Sandbox violation** — invoke the run with a forged target path outside `lifecycle/ideas/` (test-only constructor entry point) to confirm the sandbox / write-policy denial path is hit; run is failed with stderr matching `denied`. (If the public `Trigger` does not allow this — by virtue of eligibility — exercise the policy layer separately via a unit test against `internal/agent/policy.go`.)
5. **No internal retry** — after a failure for path X, no second `agent_runs` row appears unless a new trigger source fires (verified by waiting 3 s and asserting count is exactly 1).
6. **Log line contents** — capture the `slog` output (via a test-injected `slog.Handler`) and assert one warn line containing `path=lifecycle/ideas/<slug>.md`, `lineage=<slug>`, and `reason=<...>`.

### Acceptance criteria

- [ ] All 6 cases pass.
- [ ] The sandbox-violation case (4) explicitly verifies no files exist outside `lifecycle/ideas/` after the call.
- [ ] Log assertions match exact field names (`path`, `lineage`, `reason`, `duration_ms`) so future log-format changes break the test loudly.

---

## Milestone 9 — Test artifact in `lifecycle/tests/`

### Description

Per the `test-developer` agent's responsibilities, write a companion artifact summarising what was built.

### Files to change

- `lifecycle/tests/auto-triage-new-ideas-<n>-test.md` (new artifact — index assigned at write time per CLAUDE.md lineage rules):
  - Frontmatter:
    - `title: Auto-Triage Raw Ideas — Integration Test Suite`
    - `type: test`
    - `status: draft`
    - `lineage: auto-triage-new-ideas`
    - `parent: lifecycle/test-plans/auto-triage-new-ideas-5-test.md`
  - Body summarises scenarios from milestones 2–8 and points to the specific files under `internal/triage/` and `tests/triage/`.

### Acceptance criteria

- [ ] The artifact passes index validation on next watcher tick (no parse error in `ParseErrorsView`).
- [ ] Its body lists at least one test file per milestone.

---

## Notes on what this plan does NOT cover

- No end-to-end browser automation (Playwright/Cypress) — frontend coverage is the smoke pass in [[auto-triage-new-ideas-4-fe]] Milestone 5.
- No load or stress testing of the concurrency cap beyond N+1 (single host, single process).
- No tests for the `idea-generate` prompt's content quality — that is the LLM's responsibility and is exercised by existing `idea-capture` tests.
