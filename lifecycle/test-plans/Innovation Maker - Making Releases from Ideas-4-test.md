---
title: Test Plan — kaos-control v1
type: plan-test
status: done
lineage: innovation-maker
parent: requirements/Innovation Maker - Making Releases from Ideas-1.md
labels:
    - testing
    - qa
    - playwright
    - v1
---

> Target implementer: QA agent (Sonnet). Produces an automated test suite spanning unit, integration, and end-to-end layers, runnable locally and in CI. Acceptance criteria reference the milestone bullets in the backend and frontend plans. All section numbers in the form §N.N refer to the parent requirements document unless stated.

## 1. Scope

### In scope
- Unit tests for backend Go packages and frontend Vue components / composables.
- Integration tests that exercise the real artifact pipeline (SQLite, fsnotify, go-git) without mocks.
- End-to-end tests that drive the running Go binary with the built frontend via Playwright.
- Performance smoke tests for indexing and graph rendering.
- Accessibility smoke tests for the frontend.
- Security smoke tests for the auth and sandbox layers.
- CI wiring (GitHub Actions) and local runner ergonomics.

### Out of scope (v1)
- Full penetration testing (manual security review only).
- Load/stress testing at production scale (a smoke test establishes the performance floor; beyond that is post-v1).
- Cross-browser matrix beyond Chromium, Firefox, WebKit (covered by Playwright defaults).
- Agent-model quality evaluations (distinct from kaos-control's correctness).

## 2. Test Taxonomy

| Layer | Runner | Target | Typical runtime per test |
|---|---|---|---|
| Unit — backend | `go test` + `testify` | pure functions, parsing, validators | < 10 ms |
| Unit — frontend | `vitest` + `@vue/test-utils` | composables, Pinia stores, presentational components | < 50 ms |
| Integration — backend | `go test -tags=integration` | handlers + SQLite + temp git repo + fsnotify | < 2 s |
| E2E | Playwright (TypeScript) | full stack: Go binary + Chromium | < 30 s |
| Performance smoke | Playwright + Go benchmarks | indexing throughput, graph render frames | < 60 s |
| A11y smoke | `@axe-core/playwright` | key pages | < 10 s |
| Security smoke | Playwright + Go | auth/session/CSRF/path traversal | < 10 s |

**Why Playwright**: spec §7.3 ("QA testing includes headless browser robotic testing") leaves the framework to the user; Playwright is chosen here for (a) built-in Go binary process management via fixtures, (b) three-browser parity out of the box, (c) strong API testing for mixed UI + API scenarios, (d) trace viewer for debugging flaky runs, (e) single language (TypeScript) for e2e matching the frontend.

## 3. Directory Layout

```
/
├── internal/**/_test.go                     # Go unit tests (co-located)
├── internal/**/_integration_test.go         # Go integration tests (build tag `integration`)
├── web/src/**/*.spec.ts                     # frontend unit tests (co-located)
├── tests/
│   ├── e2e/                                 # Playwright tests
│   │   ├── playwright.config.ts
│   │   ├── fixtures/
│   │   │   ├── app.ts                       # boots the Go binary, returns base URL
│   │   │   ├── project.ts                   # creates a fresh project dir + registers
│   │   │   ├── user.ts                      # bootstraps admin + worker users
│   │   │   └── data/                        # seed markdown artifacts
│   │   ├── auth.spec.ts
│   │   ├── projects.spec.ts
│   │   ├── artifacts-read.spec.ts
│   │   ├── artifacts-write.spec.ts
│   │   ├── graph.spec.ts
│   │   ├── workflow.spec.ts
│   │   ├── agents.spec.ts
│   │   ├── realtime.spec.ts
│   │   ├── external-edit.spec.ts
│   │   ├── rename.spec.ts
│   │   ├── parse-errors.spec.ts
│   │   ├── a11y.spec.ts
│   │   └── security.spec.ts
│   └── perf/
│       ├── indexing-throughput.go_test.go   # Go benchmark
│       └── graph-render.spec.ts             # Playwright perf check
└── .github/workflows/ci.yml
```

## 4. Dependencies

### Backend test tooling
- `github.com/stretchr/testify/require` + `.../assert`.
- `github.com/go-git/go-git/v5` used to seed fixture repos (no shell).
- Stdlib `testing`, `testing/fstest`, `net/http/httptest`.
- `gopkg.in/yaml.v3` for crafting fixture frontmatter.
- No mocks for SQLite or filesystem — use `t.TempDir()`.

### Frontend test tooling
- `vitest`, `@vue/test-utils`, `jsdom`, `happy-dom` (faster for simple components).
- `msw` for API mocking at the fetch boundary when a composable can't be tested against a real backend.

### E2E tooling
- `@playwright/test` with TypeScript.
- `@axe-core/playwright` for accessibility checks.
- `p-limit` for parallelism control in perf scenarios.
- A fake LLM binary (`tests/e2e/fixtures/fake-claude.sh`) that the agent-runner driver points at for deterministic agent runs.

## 5. Unit Tests — Backend

Target: **≥ 80% statement coverage** across `internal/` packages, with the bar relaxed to 70% for thin wrapper packages (`internal/http`, `internal/git`).

### Must-have test subjects
- **`internal/artifact`**
  - `ParseFilename` handles slug/index/stage extraction and rejects invalid slugs (uppercase, underscores, too long, too short).
  - `ParseArtifact` round-trips frontmatter + body; preserves unknown frontmatter keys verbatim.
  - Wiki-link extractor handles `[[path]]`, `[[path|label]]`, escaped `\[[not a link]]`.
  - Required-field validation surfaces every missing field with a stable error code.
- **`internal/workflow`**
  - Transition matrix matches §6.2 of spec exactly (property-based table test).
  - `GateReady` returns correct missing-plans list.
  - Rejection creates a child artifact filename one index higher than the current max in the lineage.
- **`internal/git`**
  - Branch-template evaluator covers all supported placeholders.
  - Commit-message template emits exactly the format in §8.3 (byte-for-byte).
- **`internal/lock`**
  - Reaper releases locks older than the timeout; does not release fresh locks.
  - Concurrent `Acquire` calls from two goroutines return exactly one success.
- **`internal/config`**
  - Missing required fields → typed error.
  - Unknown `stage` names in `required_plans` rejected at load time.
- **`internal/agent`**
  - Scope enforcement rejects writes outside `allowed_write_paths`.
  - Prompt template renderer handles all placeholders and leaves unknown ones literal.

## 6. Unit Tests — Frontend

Target: **≥ 70% line coverage** across `web/src/`, with Pinia stores and composables expected to reach ≥ 85%.

### Must-have test subjects
- **`useWebSocket`**: reconnect with backoff; exponential cap; heartbeat cadence; dispatch into the event bus.
- **`useLock`**: acquires lock, sends heartbeat, releases on unmount; handles denial.
- **`useExternalChange`**: fires prompt only for changes not originated by our save.
- **`useGraphData`**: derives `{nodes, edges}` from store data; applies filters correctly; stable across event floods.
- **`stores/auth`**: login/logout flips state; `fetchMe` handles 401 by clearing state.
- **`stores/artifacts`**: cache invalidation on events.
- **Components**:
  - `MarkdownPreview` resolves wiki-links to in-app routes.
  - `FrontmatterPanel` preserves unknown keys through edit round-trip.
  - `ArtifactModal` action bar hides actions the user lacks the role for (mocked role store).

## 7. Integration Tests — Backend

Tag: `integration`. Each test:
1. Creates a `t.TempDir()` project root.
2. Runs `git init` (via go-git) and writes a `lifecycle/config.yaml`.
3. Spins up the full HTTP server against that dir on a random port (`httptest.NewServer`).
4. Drives it through the real REST/WebSocket API and asserts on the **filesystem** and **git log**, not just responses.

### Must-have scenarios
- **Full scan**: given a pre-seeded `lifecycle/` with N artifacts, startup indexes them and `GET /graph` returns the expected node/edge counts.
- **Create → commit → index**: POST an artifact; assert (a) file exists on disk, (b) ticket branch was created, (c) exactly one commit with the templated message, (d) SQLite has the row, (e) WebSocket broadcasted `file.changed` + `artifact.indexed` + `git.committed`.
- **External edit**: drop a file into `lifecycle/` directly; assert fsnotify picks it up within 1 s, index updated, websocket fired.
- **Rename with link rewrite**: create `a-2.md` linking to `b`, then rename `b → b-renamed`; assert `a-2.md` now links to `b-renamed` and the rewrite is a single atomic commit.
- **Transition with role gate**: logged-in user without the role gets 403; with the role succeeds; `status` field updated in place; git log shows the status change commit.
- **Required-plans gate**: `planning → in-development` fails with a readable error when a required plan is missing, succeeds when all approved.
- **Lock contention**: two parallel requests to acquire the lineage lock result in exactly one success and one `ErrLocked`.
- **Reaper**: simulate a crashed editor by inserting a stale lock row; advance the test clock; reaper releases it.
- **Schema migration**: open an index DB with `schema_version=0`; server rebuilds from disk; final state matches a fresh run.

## 8. End-to-End Tests — Playwright

### App lifecycle fixture
`tests/e2e/fixtures/app.ts` builds the Go binary once, then per-test:
1. Creates a temp projects dir and app config.
2. Starts `./kaos-control --config <temp>/config.yaml` on a random port.
3. Waits for `/api/health`.
4. Registers a project pointing at a seeded fixture repo.
5. Tears down on teardown (kill process, remove temp dirs).

### Fake agent fixture
`fake-claude.sh` echoes a scripted markdown artifact to the target path, waits a configurable number of seconds, then exits with a configurable code. Lets us test success, failure, kill, and partial-commit paths deterministically without calling a real LLM.

### Scenarios (traceable to milestone acceptance criteria)

#### Auth & Projects (BE M4, FE M1)
- Login with correct creds succeeds; wrong creds show error; session persists across reloads; logout clears.
- Admin CRUD on projects via UI; non-admin sees read-only project list.

#### Artifacts — Read (BE M2, FE M2)
- Pointed at a seeded repo with the real shape of this project, every artifact appears in the list view and the editor renders its preview. Wiki-link click navigates.

#### Artifacts — Write (BE M3, FE M4)
- Creating an artifact via UI results in a file on disk, a commit on a new ticket branch, and live graph update.
- Optimistic concurrency: simulate a stale `expected_sha` and assert the API returns 409 and the UI shows a reload prompt.
- Slug rename updates inbound links and commits atomically.

#### Graph (BE M2, FE M3)
- 3D graph renders expected nodes/edges; filters by type/status/label shrink the set; node click opens the modal with correct action bar.
- 2D graph (Cytoscape) renders the same dataset and responds to the same filters.

#### Workflow (BE M4, FE M5)
- With the matrix role mapping from the fixture, transitions succeed only for authorised users.
- Rejection flow captures feedback and produces a child artifact.

#### Agents (BE M5, FE M5)
- Trigger `fake-claude` via UI → run visible in status chip → completes → produced artifact appears on graph.
- Kill button terminates the run; UI reflects `killed`; partial artifact committed with `partial:` prefix.
- Double-run against the same lineage is refused with a clear message.
- Concurrency cap: trigger `max+1` runs across different lineages; the last queues or is refused per configured behaviour (assert exact semantics match backend plan §11).

#### Realtime (BE M3/M5, FE M4/M5)
- Two browser contexts open the same project: user A edits, user B sees live `file.changed` + `artifact.indexed` events; graph updates in B without reload.
- Lock banner appears in B while A is editing.

#### External Edit (BE M3, FE M4)
- User A has the editor open; a shell writes to the same file on disk; A's editor shows the reload prompt; Reload reflects disk content; Keep editing preserves unsaved state.

#### Parse errors (BE M2/M6, FE M6)
- Drop a malformed artifact on disk; header badge shows the error count; clicking navigates to `ParseErrorsView`.

## 9. Performance Smoke

- **Indexing throughput**: `go test -bench=BenchmarkIndex` seeds 5 000 artifacts and asserts full-scan completes < 10 s on CI runner; incremental re-parse of 10 files < 1 s.
- **Graph render**: Playwright perf spec loads a 1 000-node fixture and measures first-meaningful-render (< 2 s) and interaction latency on filter change (< 250 ms).
- Perf regressions fail CI when they exceed a 20% slowdown vs a committed `tests/perf/baselines.json`. Baselines updated by hand (never automatically) via a dedicated PR.

## 10. Accessibility Smoke

- `tests/e2e/a11y.spec.ts` runs `@axe-core/playwright` on: login, project picker, graph view, artifact editor, agents view, project config.
- Fails on any **serious** or **critical** violation. Allowed to have moderate/minor issues but they're logged.
- Keyboard navigation test: from login to creating a new artifact, no mouse input; assert focus ring visible and tab order sane.
- Dark mode renders all rules at AA contrast.

## 11. Security Smoke

- **Auth**: session cookie is `HttpOnly; Secure; SameSite=Lax`; expired session redirects to login.
- **CSRF**: a POST without the double-submit token is rejected with 403.
- **Path traversal**: API attempts with `../`, absolute paths, symlink traps all return 400/403 — no filesystem leak.
- **Scope enforcement**: a mock agent run tries to write outside `allowed_write_paths`; commit is refused and run marked failed.
- **Login throttling**: five failed logins within a minute return 429 (if implemented in backend; else file an issue and skip).

## 12. Fixtures & Seed Data

- `tests/e2e/fixtures/data/seed-basic/`: a minimal lifecycle with one idea → requirement → backend-plan lineage. Used by read/graph/editor tests.
- `tests/e2e/fixtures/data/seed-multi-user/`: adds a second user with `developer` role, and required_plans configured so the workflow tests can exercise the gate.
- `tests/e2e/fixtures/data/seed-malformed/`: includes a broken frontmatter file, a missing required field, a symlink escape attempt.
- `tests/perf/fixtures/seed-5k/`: generated on first run from a small script; cached across CI runs.

## 13. Milestones — aligned with BE/FE plans

Each test-plan milestone unblocks by the corresponding BE + FE milestones completing.

### T1 — Unit foundation (≈ 2 days) — alongside BE M1 + FE M1
- Backend unit tests for `internal/artifact`, `internal/config`, `internal/workflow`.
- Frontend unit tests for `stores/auth`, `api/client`.
- **Acceptance**: `make test-unit` green; coverage reporting wired.

### T2 — Integration + Read e2e (≈ 3 days) — alongside BE M2/M3 + FE M2
- Integration tests for indexing, write path, rename, fsnotify.
- E2E auth + projects + artifacts-read.
- Playwright app fixture (builds binary, boots server).
- **Acceptance**: `make test-integration` and `make test-e2e-read` green locally and in CI.

### T3 — Write + Graph e2e (≈ 2 days) — alongside BE M3 + FE M3/M4
- E2E artifacts-write, graph, rename.
- External-edit test with a shell-side writer.
- **Acceptance**: full write path covered end-to-end.

### T4 — Workflow + Agent e2e (≈ 3 days) — alongside BE M4/M5 + FE M5
- Integration tests for transitions, required-plans gate, lock contention.
- E2E workflow + agents (success, kill, crash, partial, concurrency cap).
- Fake-claude fixture in place.
- **Acceptance**: every BE M5 acceptance bullet has at least one passing e2e scenario.

### T5 — Realtime + External edit (≈ 2 days) — alongside FE M4/M5
- E2E realtime (two-context sync), lock banner, external-edit.
- **Acceptance**: no race flakes over 50 consecutive runs.

### T6 — A11y + Security + Perf (≈ 2 days) — alongside FE M6 + BE M6
- A11y suite passing; security smoke; perf baselines committed.
- CI matrix (Chromium + Firefox + WebKit) passes.
- **Acceptance**: full `make test` green; CI < 15 min wall clock on a standard runner.

**Total**: ≈ 14 working days.

## 14. CI

- GitHub Actions workflow `.github/workflows/ci.yml`:
  - Job `lint`: `go vet`, `staticcheck`, `gofmt -l`, `eslint`, `vue-tsc`.
  - Job `test-backend`: unit + integration.
  - Job `test-frontend`: unit (vitest).
  - Job `test-e2e`: builds binary + frontend, runs Playwright against Chromium; Firefox/WebKit run nightly to keep PR time short.
  - Job `release` (tag only): goreleaser + Docker build.
- Flaky tests: auto-retry once; if still flaky, fail and require a ticket.
- Artefacts on failure: Playwright traces + screenshots + server log uploaded.

## 15. Coverage & Quality Gates

- Backend: fail CI if total coverage drops > 2 percentage points vs previous main.
- Frontend: same.
- E2E: no coverage metric; instead, require that every BE/FE milestone acceptance bullet maps to at least one named e2e test (enforced by a metadata convention — tests tagged `@accept:<milestone>`).

## 16. Flake Policy

- A test that fails intermittently is quarantined (moved to `tests/quarantine/`) within 24 hours of first observation.
- A quarantined test must be fixed or deleted within one week; bot posts a reminder.
- No skipping tests to land a PR. Quarantine or fix only.

## 17. Coordination

- **Backend plan**: acceptance bullets in BE §18 are the contractual scenarios; any change must update this test plan in the same commit series.
- **Frontend plan**: same for FE §15.
- API-contract changes land in the backend plan first, then this plan's scenarios are updated before implementation.

## 18. Risks

| Risk | Mitigation |
|---|---|
| Playwright flakiness on CI | Stable app fixture boot-wait; disable animations in tests; retry once. |
| Indexing perf test drift with machine speed | Baselines stored per-runner-class; tolerate ±20%; manual re-baseline via PR. |
| Fake-claude fixture diverges from real binary | Keep the fake's contract minimal (writes a file then exits); add one smoke test that runs the **real** `claude` binary on a dev-only tag (`-tags=realagent`) to catch divergence. |
| Vitest ↔ Vue reactivity subtleties | Prefer composable-level tests over component-tree tests; isolate CodeMirror from reactivity. |

## 19. Open Questions

- **Login throttling** (§11): the backend plan doesn't explicitly spec this; confirm before writing the test.
- **`claude` binary in CI**: unlikely to be available and out of scope to install; real-binary tests run only locally under `-tags=realagent`.
- **Perf baseline stability**: CI runners vary; if we see more than 10% false-positive failures in the first month, swap to trend-based (compare to last 20 runs' median) rather than absolute thresholds.

## 20. References

- Parent spec: [[requirements/Innovation Maker - Making Releases from Ideas-1]]
- Backend plan (sibling, scenario source): [[backend-plans/Innovation Maker - Making Releases from Ideas-2-be]]
- Frontend plan (sibling, scenario source): [[frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe]]
