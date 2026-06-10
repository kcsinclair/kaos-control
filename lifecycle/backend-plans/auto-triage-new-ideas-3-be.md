---
title: "Auto-Triage Raw Ideas — Backend Plan"
type: plan-backend
status: done
lineage: auto-triage-new-ideas
parent: lifecycle/requirements/auto-triage-new-ideas-2.md
assignees:
    - role: backend-developer
      who: agent
---

# Auto-Triage Raw Ideas — Backend Plan

Implements a new `internal/triage` subsystem that watches for `status: raw` / `type: idea` artifacts under `lifecycle/ideas/`, runs them through the existing `ideachat.Generate` (`idea-generate` prompt, inline driver), rewrites the body into `## Raw Idea` + `## Idea` sections, mutates frontmatter through the workflow engine, and records each attempt as an agent run. Triggers are: the file watcher, a startup re-scan, and a new authenticated REST endpoint.

Cross-references: [[auto-triage-new-ideas-4-fe]] (UI surface that calls the endpoint and shows runs in agent history), [[auto-triage-new-ideas-5-test]] (integration tests).

---

## Milestone 1 — Triage agent in `lifecycle/config.yaml`

### Description

Declare the triage agent in the project config so it is loadable by the existing agent manager and visible via `GET /api/agents`. The agent reuses the existing `idea-generate` prompt template (already on `idea-capture`); no new template body is needed in this milestone. The runner does not invoke this entry as a normal CLI-mediated agent — instead, the new `internal/triage` package (Milestone 4) looks the entry up by name and uses its `model`, `allowed_write_paths`, `git_identity`, and prompt template to drive a single-shot `ideachat.Generate` call. Declaring it under `agents:` is what makes it appear in `/api/agents` and lets per-run records attribute to a known agent name.

### Files to change

- `lifecycle/config.yaml` — append a new entry to `agents:`:
  - `name: idea-triage`
  - `role: [product-owner]` (resolved Q: agent transitions `raw → draft` as `product-owner`; the state machine already permits this).
  - `driver: inline`
  - `model: claude-sonnet-4-6` (matches `idea-capture`).
  - `active_status: draft` (the status the agent produces; consistent with `requirements-analyst` → `clarifying`, etc.).
  - `source_types: [idea]`.
  - `allowed_write_paths: [lifecycle/ideas]` only.
  - `git_identity:` `name: Idea Triage Agent`, `email: idea-triage@kaos-control.local`.
  - `prompt_templates.idea-generate:` reuses the existing `idea-generate` template verbatim (copy from `idea-capture` so the triage entry is self-contained; the runtime resolver looks up by agent name + template key).

### Acceptance criteria

- [ ] `make build` succeeds with the new agent entry.
- [ ] `GET /api/p/:project/agents` returns a record named `idea-triage` with `role: ["product-owner"]`, `driver: "inline"`, `source_types: ["idea"]`, and `allowed_write_paths: ["lifecycle/ideas"]`.
- [ ] No other agent entries in `lifecycle/config.yaml` are modified.
- [ ] The existing `idea-capture` agent and its `idea-generate` template are unchanged.

---

## Milestone 2 — `internal/triage` package skeleton, types, and dedup state

### Description

Create the new package that owns triage orchestration. This milestone introduces the public surface (constructor, `Trigger`, `Stop`) and the in-memory deduplication state, but does not yet call the LLM or mutate files.

### Files to change

- `internal/triage/triage.go` (new) — defines:
  - `type Manager struct` — holds `*project.Project` dependencies passed in as narrow interfaces (idx, hub, agent manager, workflow engine, lock manager, project root, sandbox, agent registry lookup). Avoid a direct `*project.Project` field to keep the import graph acyclic (project already imports agent and workflow).
  - `Options` struct: `MaxConcurrent int` (default 2 — resolved Q), `AgentName string` (default `"idea-triage"`).
  - `func New(deps Deps, opts Options) *Manager`.
  - `func (m *Manager) Trigger(ctx, relPath string, trigger TriggerSource) (runID string, err error)` — public entry point. `TriggerSource` is an enum: `TriggerWatcher`, `TriggerStartup`, `TriggerAPI`.
  - `func (m *Manager) Stop(ctx)` — waits for in-flight runs.
  - Private: `inFlight map[string]string` (path → runID) guarded by a `sync.Mutex`; `sem chan struct{}` of size `MaxConcurrent` for the concurrency cap.
- `internal/triage/triage_test.go` (new, stub) — covers happy-path call into `Trigger` with a stubbed dependency that records invocations (no LLM call yet; eligibility check returns no-op).

### Acceptance criteria

- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `Trigger` called twice for the same `relPath` while a synthetic "run" is in flight starts only one inner invocation (the second returns the in-flight run ID; verified by the stub test).
- [ ] `Trigger` called for `MaxConcurrent+1` distinct paths blocks the (N+1)th until one slot frees, without panic.
- [ ] `Stop` returns only after every in-flight goroutine has exited.

---

## Milestone 3 — Eligibility filter and lock acquisition

### Description

Implement the FR-2 eligibility check and the lineage-lock acquisition that gates body mutation. Eligibility is evaluated inside `Trigger` before consuming a concurrency slot, so ineligible paths are cheap.

### Files to change

- `internal/triage/eligibility.go` (new):
  - `func eligible(ctx, idx index.Reader, relPath string) (ok bool, reason string, err error)`.
  - Checks: path matches `lifecycle/ideas/*.md` (exact directory, no nesting), the indexed row has `type == "idea"` and `status == "raw"`. Returns a stable `reason` string (`"not_in_ideas_dir"`, `"wrong_type"`, `"wrong_status"`, `"not_indexed"`) so the API handler (Milestone 8) can surface it.
- `internal/triage/triage.go`:
  - Inside `Trigger`: call `eligible`. If `ok == false`, return `ErrIneligible{Reason: reason}` (a typed error the API handler unwraps to a 409 body).
  - On eligible, acquire a lineage write-lock via `m.deps.Locks.Acquire(lineage, "idea-triage", ttl)` before any file mutation. Release on completion (success or failure) via deferred call. If `Acquire` fails (already locked), return `ErrLocked` and do not start a run.

### Acceptance criteria

- [ ] A path outside `lifecycle/ideas/` returns `ErrIneligible{Reason: "not_in_ideas_dir"}`.
- [ ] A `type: defect` file under `lifecycle/ideas/` (test fixture) returns `ErrIneligible{Reason: "wrong_type"}`.
- [ ] A `status: draft` idea returns `ErrIneligible{Reason: "wrong_status"}`.
- [ ] A path the index has no row for returns `ErrIneligible{Reason: "not_indexed"}`.
- [ ] An eligible path with the lineage already locked returns `ErrLocked`; no agent run is created.
- [ ] On every eligible path, the lock is released even when the downstream call panics (deferred release covers the panic case in unit tests).

---

## Milestone 4 — Run execution: LLM call, body rewrite, frontmatter mutation

### Description

The core of the subsystem. Look up the `idea-triage` agent config, build the inline-driver `ModelConfig`, call `ideachat.Generate` with the artifact body as input, then rewrite the file as specified by FR-3 and FR-4.

### Files to change

- `internal/triage/run.go` (new):
  - `func (m *Manager) execute(ctx, runID, relPath string) (err error)` — the inner job started by `Trigger`.
  - Read the artifact bytes from disk, parse via `artifact.Parse`, capture the original body.
  - Resolve the `idea-triage` agent entry from `m.deps.AgentRegistry.Get("idea-triage")`. From it, build an `ideachat.ModelConfig` (same construction path used by `resolveIdeaCaptureConfig` in `internal/http/idea_generate.go` — refactor that helper into `internal/ideachat/` so both call sites share it).
  - Build `ideachat.GenerateOptions{Input: rawBody, ArtifactType: "idea", ExistingLabels: labels, ExistingSlugs: slugs, ModelCfg: modelCfg}`.
  - Call `ideachat.Generate`; on error or empty `result.Body`, return the error to be recorded as a failed run (FR-10). Do not mutate the file.
  - Body rewrite (FR-3):
    - Split the existing body into `(h1, rest)`. `h1` is the leading `# …` line if present, else empty.
    - Strip the agent result body's leading H1 to avoid duplication.
    - If the existing body already contains `## Raw Idea` and `## Idea`, replace only the `## Idea` block (re-run case). Otherwise produce `h1 + "\n\n## Raw Idea\n\n" + originalRest + "\n\n## Idea\n\n" + stripped`.
  - Frontmatter mutation (FR-4):
    - `status: draft` is *not* written here directly; status is applied via the workflow engine in Milestone 5.
    - Merge labels: existing first, then agent-proposed labels, deduped, dropping any not in `idx.Labels()` vocabulary.
    - Set `priority: "normal"` (resolved Q — `normal` as the default) only when frontmatter has no `priority` key.
    - All other keys (`lineage`, `parent`, `release`, `created`, `assignees`, custom) are preserved verbatim.
  - Re-serialise via `artifact.Marshal` (or whatever the project's existing write helper is — confirm in `internal/http/write.go`). Write to disk through `sandbox.WritePath` so the `allowed_write_paths: [lifecycle/ideas]` policy is honoured (NFR-4).

### Acceptance criteria

- [ ] On success, the file body contains `## Raw Idea` followed by the verbatim pre-triage body and `## Idea` followed by the agent-generated content.
- [ ] The H1 title line at the top of the file is preserved unchanged.
- [ ] On re-run (file already contains both `## Raw Idea` and `## Idea`), the `## Raw Idea` block is byte-identical to the previous run and only `## Idea` differs.
- [ ] `priority` is set to `normal` when absent; an existing `priority: high` (or anything else) is preserved.
- [ ] Labels proposed by the agent that are not in `idx.Labels()` are silently dropped from the merged list.
- [ ] An empty agent body or `action != "propose"` causes the function to return an error and leaves the file byte-identical to its pre-call contents (no partial write).
- [ ] A write attempt outside `lifecycle/ideas/` (verified by passing a crafted path through the sandbox boundary) is denied by policy.

---

## Milestone 5 — Workflow transition + agent run record

### Description

After a successful write, drive the `raw → draft` status change through the workflow engine using the `product-owner` role (resolved Q), so existing audit logging and WS broadcasts fire. Independently of success/failure, record the attempt as an agent run row so it appears in the artifact's run history panel (FR-9 / NFR-2).

### Files to change

- `internal/triage/run.go`:
  - On successful write, call `m.deps.Workflow.CanTransition("raw", "draft", []string{"product-owner"}, "idea")` to guard; if it returns false, fail loudly (configuration drift). Then apply the transition via the same path used by `internal/http/transition.go`'s `applyTransition` helper. Extract the shared logic into a new `internal/workflow.Apply(...)` helper or reuse the existing one — do not duplicate state-machine logic in `triage/`.
  - Re-index the artifact synchronously via `idx.IndexFile(absPath)` so `artifact.indexed` and `file.changed` fire on the existing hub (FR-5).
- `internal/triage/runrecord.go` (new):
  - `func (m *Manager) recordRunStart(relPath, runID, trigger string) error` — inserts a row into the existing `agent_runs` SQLite table using the existing `index.AgentRunStore` (or whichever package owns it; locate via `grep "agent_runs"` in `internal/index/`).
  - `func (m *Manager) recordRunComplete(runID string, status string, durationMs int64, stdout, stderr string)`.
  - `agent_name` field is `"idea-triage"`, `target_path` is the artifact rel path, `status` is `success` or `failed`, `started_at` and `ended_at` are RFC3339 timestamps.

### Acceptance criteria

- [ ] On successful triage, the file's frontmatter shows `status: draft` and the existing `artifact.indexed` WS event is observed by a connected client (verified in the test plan).
- [ ] If `CanTransition` returns false, the run is recorded as `failed` with stderr `"workflow_denied"` and the file is rolled back (re-written to its pre-mutation bytes) so the artifact does not end up with `status: raw` and a triaged body.
- [ ] Every triage attempt — success or failure — produces exactly one row in `agent_runs` with `agent_name = "idea-triage"`.
- [ ] `target_path` on the row equals the artifact's project-relative path.

---

## Milestone 6 — fsnotify trigger wiring

### Description

Hook the watcher so that when a file under `lifecycle/ideas/` is created or modified and the (re-)indexed row is `raw` + `idea`, the triage manager is called. Reuse the existing 150 ms debounce; do not add a second one (NFR-3).

### Files to change

- `internal/watcher/watcher.go`:
  - Add an optional callback field: `triageFn func(relPath string)`.
  - `func (w *Watcher) SetTriageCallback(fn func(relPath string))`.
  - In `handleChange`, after the existing `idx.IndexFile` succeeds: read the freshly indexed row via `idx.Get(relPath)`. If `row.Type == "idea"` and `row.Status == "raw"`, invoke `w.triageFn(relPath)` in a goroutine (the manager itself handles concurrency and dedup).
- `internal/project/project.go`:
  - In `Open`, construct the `triage.Manager` with the project's deps.
  - Register it on the watcher via `w.SetTriageCallback(func(p string) { _, _ = mgr.Trigger(ctx, p, triage.TriggerWatcher) })`.
  - Add `mgr.Stop(ctx)` to `Close`.

### Acceptance criteria

- [ ] Creating `lifecycle/ideas/<slug>.md` with `type: idea` / `status: raw` triggers a triage run within ~1 s of the watcher event (verified by integration test in [[auto-triage-new-ideas-5-test]]).
- [ ] Modifying a file whose status is already `draft` does NOT call `triageFn` (eligibility short-circuits earlier, but the watcher should still avoid spamming `Trigger` for non-`raw` artifacts — verified by counting runs in `agent_runs`).
- [ ] Two rapid modifications inside the 150 ms debounce window produce at most one `Trigger` call (existing watcher debounce + manager dedup combine).

---

## Milestone 7 — Startup re-scan

### Description

After the initial full index scan completes in `project.Open`, enumerate every `lifecycle/ideas/*.md` with `status: raw` / `type: idea` and enqueue each through the triage manager (FR-1 bullet 2).

### Files to change

- `internal/project/project.go`:
  - After `idx.Scan(...)` returns and after the watcher has started, call a new helper `triage.RescanRaw(ctx, mgr, idx)`.
- `internal/triage/rescan.go` (new):
  - Query `idx.List(index.Filter{Type: "idea", Status: "raw", Unlimited: true})`.
  - For each row, call `mgr.Trigger(ctx, row.Path, TriggerStartup)`. Errors are logged at warn level and otherwise swallowed.

### Acceptance criteria

- [ ] On startup with an existing `raw` idea on disk, an `agent_runs` row appears within ~5 s of `project.Open` returning.
- [ ] On startup with no `raw` ideas, `RescanRaw` is a no-op and emits no warnings.
- [ ] If three `raw` ideas are present at startup and `MaxConcurrent = 2`, the third triage starts only after one of the first two finishes.

---

## Milestone 8 — REST endpoint `POST /api/p/:project/ideas/{slug}/triage`

### Description

Add the on-demand trigger endpoint specified by FR-8. Authentication and authorisation match the existing chi middleware patterns in `internal/http/`.

### Files to change

- `internal/http/triage.go` (new):
  - `func (s *Server) handleTriageIdea(w http.ResponseWriter, r *http.Request)`.
  - Resolve project from context; resolve user from context. 401 if missing user.
  - Read `slug` via `chi.URLParam(r, "slug")`. Look up the artifact via `idx.List(index.Filter{Lineage: slug, Type: "idea", Unlimited: true})` and pick the row whose path starts with `lifecycle/ideas/`. 404 if none.
  - Role check: `p.Cfg.RolesFor(user.Email)` must contain `product-owner`, `analyst`, or `reviewer`. Otherwise 403.
  - Call `mgr.Trigger(r.Context(), row.Path, triage.TriggerAPI)`.
  - Map errors: `ErrIneligible` → 409 with body `{"error":"not_eligible","reason":"<reason>"}`; `ErrLocked` → 409 with `{"error":"locked"}`; `ErrBusy` → 503; other → 500.
  - On success → 202 with `{"run_id":"<id>"}` (the inner run executes asynchronously).
- `internal/http/server.go`:
  - Register the route inside the existing project sub-router behind `requireAuth`: `r.Post("/ideas/{slug}/triage", s.handleTriageIdea)`.

### Acceptance criteria

- [ ] Unauthenticated → 401.
- [ ] Authenticated user without `product-owner` / `analyst` / `reviewer` → 403.
- [ ] Slug that resolves to no `lifecycle/ideas/<…>.md` row → 404.
- [ ] Slug resolving to a `draft` (or any non-`raw`) idea → 409 with `reason: "wrong_status"`.
- [ ] Successful call → 202 with a non-empty `run_id`; the run appears in `agent_runs` shortly after.
- [ ] An in-flight run for the same slug → 202 returning the in-flight `run_id` (idempotent coalescing per FR-7).

---

## Milestone 9 — Observability and failure handling polish

### Description

Final pass to satisfy NFR-2 and FR-10 observability requirements end-to-end.

### Files to change

- `internal/triage/triage.go`, `internal/triage/run.go` — emit structured logs:
  - `slog.Info("triage started", "path", relPath, "lineage", lineage, "run_id", runID, "trigger", triggerSource)`.
  - `slog.Info("triage completed", "path", relPath, "lineage", lineage, "run_id", runID, "duration_ms", ms)`.
  - `slog.Warn("triage failed", "path", relPath, "lineage", lineage, "run_id", runID, "reason", err.Error())`.
- `internal/triage/run.go` — guarantee: any non-nil error from `execute` results in (a) the file having been rolled back to pre-call bytes, (b) an `agent_runs` row with `status = "failed"` and `stderr` containing the error string, (c) a single warn-level log line, and (d) **no** automatic retry from within the manager (FR-10).

### Acceptance criteria

- [ ] A forced failure (e.g. stubbed LLM returning malformed JSON) leaves the artifact byte-identical to its pre-call state and records exactly one failed `agent_runs` row.
- [ ] Log lines for start / complete / fail contain `path`, `lineage`, and (for complete) `duration_ms`.
- [ ] Triggering the same path again immediately after a failure starts a brand-new run only when the trigger source allows (subsequent file modification, restart, or explicit API call) — there is no internal retry loop.

---

## Notes on what this plan does NOT do

- Triage for `raw` `defect` artifacts is out of scope per the requirement's non-goals; the eligibility filter explicitly rejects non-`idea` types.
- No changes to `KnownStatuses`, the workflow rule table, or other agents' prompts/scopes.
- No new queue, scheduler, or distributed coordination — single-process semaphore + lineage lock only.
- Promotion past `draft` (clarifying / approved / planning) remains the operator's / analyst's job; cross-referenced consumer [[analyst-agent-sees-draft-ideas]] is unaffected.
