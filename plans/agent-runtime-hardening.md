# Agent Runtime: Model Selection, Timeout, Block-on-Questions, Observability

## Context

The user ran their first real agent and surfaced four gaps:

1. **Model selection** is configured but ignored. `AgentConfig.Model` exists in [config.go:185](internal/config/config.go#L185) but the driver at [agent.go:97-100](internal/agent/agent.go#L97-L100) hard-codes `claude --dangerously-skip-permissions -p <prompt>` with no `--model` flag. Every agent uses whatever the user's CLI defaults to.

2. **Timeout is hardcoded at 10 minutes** ([agent.go:310](internal/agent/agent.go#L310)) and produces a `killed` status indistinguishable from a user-initiated Kill. Real analyst/developer work routinely exceeds 10 min, so partial-work runs are common and there's no signal that the cause was a timeout.

3. **No way for an agent to block on a missing input.** `claude -p` is non-interactive — the agent must commit to one path even when ambiguous. The simple fix is a prompt convention: when stuck, the agent writes questions to the artifact, sets `status: blocked`, assigns to `product-owner`, and stops. Currently `blocked` isn't in `KnownStatuses` ([artifact.go:28-33](internal/artifact/artifact.go#L28-L33)) and there are no transitions for it.

4. **Live observability is thin.** The UI already shows live progress lines via the `agent.progress` WS event ([AgentsRunsView.vue:88-91](web/src/views/project/AgentsRunsView.vue), [stores/agents.ts:61-64](web/src/stores/agents.ts)), but `claude -p` without `--output-format stream-json` only emits the final assistant text — so the panel stays empty until the run ends. No persistent log file exists either.

This change ships all four together because they're tightly coupled (driver invocation, status vocabulary, supervisor logic, prompts).

## Scope

### 1. Model selection — wire `AgentConfig.Model` to the driver

- Add `Model string` to the `Run` struct in [internal/agent/agent.go](internal/agent/agent.go) (around line 32).
- Populate it from the agent config in `Manager.StartRun` (around line 280).
- In `ClaudeCodeDriver.Start` ([agent.go:97](internal/agent/agent.go#L97)), append `--model <name>` to args when `run.Model != ""`. Claude Code accepts the aliases `opus`, `sonnet`, `haiku`.
- Set sensible defaults in [lifecycle/config.yaml](lifecycle/config.yaml):
  - `analyst-requirements`, `analyst-planner`: `model: opus` (deeper reasoning for plans/requirements)
  - `backend-developer`, `frontend-developer`, `test-developer`, `qa`: `model: sonnet` (faster execution)

### 2. Per-agent timeout — `timeout_minutes`, default 0 = disabled

- Add `TimeoutMinutes int \`yaml:"timeout_minutes,omitempty"\`` to `AgentConfig` in [internal/config/config.go](internal/config/config.go).
- In `Manager.StartRun`, choose context shape:
  ```go
  if ag.TimeoutMinutes > 0 {
      runCtx, cancel = context.WithTimeout(context.Background(),
          time.Duration(ag.TimeoutMinutes) * time.Minute)
  } else {
      runCtx, cancel = context.WithCancel(context.Background())
  }
  ```
- In `supervise` ([agent.go:368-377](internal/agent/agent.go#L368-L377)), distinguish kill cause via `errors.Is(ctx.Err(), context.DeadlineExceeded)`:
  - `DeadlineExceeded` → status `killed-timeout`
  - `Canceled` (which is what `Manager.Kill` triggers via `ar.cancel()`) → status `killed`
  - Other error → `failed`
- Add `"killed-timeout"` to the index `agent_runs.status` allowed values (it's just a `TEXT` column — no schema change needed; document in code comment).
- Update the UI status chip in [AgentsRunsView.vue](web/src/views/project/AgentsRunsView.vue) to add a colour for `killed-timeout` (amber, distinct from `killed` red).
- Update [lifecycle/config.yaml](lifecycle/config.yaml) per-agent: leave `timeout_minutes: 0` for analyst (long thinking is fine) and developer roles (build steps take time); keep something modest like `timeout_minutes: 30` on `qa` if desired (or leave 0 everywhere by default).

### 3. Block-on-questions — `blocked` status + prompt convention

- Add `"blocked": true` to `KnownStatuses` in [artifact.go:28-33](internal/artifact/artifact.go#L28-L33).
- Update the spec §4.2 status vocabulary to include `blocked`.
- Update [internal/workflow/workflow.go](internal/workflow/workflow.go) `defaultRules`:
  - `* → blocked`: allowed for all agent-bearing roles (`analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`) — this is how an agent self-blocks.
  - `blocked → draft`: allowed for `product-owner`, `analyst` — re-opens the artifact after questions are answered, so the agent can re-run from scratch.
- Update spec §6.2 transition matrix table accordingly.
- **Allowed write paths**: each developer agent gets its input plan dir added to `allowed_write_paths` so it can append `## Open Questions` and update frontmatter on the plan when stuck:
  - `backend-developer` adds `lifecycle/backend-plans`
  - `frontend-developer` adds `lifecycle/frontend-plans`
  - `test-developer` adds `lifecycle/test-plans` (already has `lifecycle/tests`)
  - Analyst agents already write the artifact in question.
  - QA already writes to `lifecycle/defects`.
- **Prompt template addition** — append a uniform "stuck" instruction to every agent's `prompt_templates.<role>` in [lifecycle/config.yaml](lifecycle/config.yaml):

  > **If you cannot proceed** (input is ambiguous, contradictory, or missing critical detail), do NOT guess. Append a `## Open Questions` section to the artifact at `{target_path}` listing each blocking question explicitly. Update its frontmatter:
  > ```yaml
  > status: blocked
  > assignees:
  >   - role: product-owner
  >     who: agent
  > ```
  > Then stop — do not attempt partial work elsewhere.

### 4. Observability — stream-json + per-run log file

- Switch driver invocation to `claude --dangerously-skip-permissions -p --output-format stream-json --verbose <prompt>`. (`stream-json` requires `--verbose` per Claude Code CLI.)
- Each stdout line is now a JSON event (assistant deltas, tool uses, results). The driver parses each line into a structured payload:
  ```json
  {"type": "assistant", "content": "..."} | {"type": "tool_use", "name": "Edit", "input": {...}} | {"type": "result", ...}
  ```
  Forward the parsed event as the `agent.progress` payload (currently a raw `line` string — change to `{event: <parsed>, raw: <line>}` so the UI keeps backward compat). The store and UI ([stores/agents.ts:61-64](web/src/stores/agents.ts), [AgentsRunsView.vue:88-91](web/src/views/project/AgentsRunsView.vue)) currently render the `line` field as a `<pre>`; extend to format event types nicely (assistant text inline, tool calls as `▸ Edit path/to/file`, results as a summary). Keep raw fallback so unparseable lines still show.
- **Per-run log file**: in `ClaudeCodeDriver.Start`, open `~/.kaos-control/data/<project>/runs/<run_id>.log` and tee both stdout and stderr to it. The `data` dir path is already known to `project.Project`; add a method or pass it via the `Run` struct.
- Add an HTTP endpoint `GET /api/p/:project/agents/runs/:run_id/log` that streams the file contents (or returns 404 if no log exists). Wire into [internal/http/server.go](internal/http/server.go) and the existing `agents.go` handlers.
- Add a "View log" link in the AgentsRunsView expanded row alongside the existing "Stderr tail" / "Artifacts produced" sections.

## Files to modify

| File | Change |
|---|---|
| `internal/agent/agent.go` | Run struct gets `Model`, `TimeoutMinutes`, `LogPath`. Driver appends `--model`, switches to stream-json, tees to log file. Manager picks context type. Supervise distinguishes timeout vs kill. |
| `internal/config/config.go` | Add `TimeoutMinutes int` to `AgentConfig`. |
| `internal/artifact/artifact.go` | Add `"blocked"` to `KnownStatuses`. |
| `internal/workflow/workflow.go` | Add `* → blocked` and `blocked → draft` rules. |
| `internal/http/agents.go` | New `handleGetAgentRunLog` for streaming. |
| `internal/http/server.go` | Wire the new route. |
| `internal/project/project.go` | Expose data dir if not already (for `runs/<id>.log` path). |
| `lifecycle/config.yaml` | `model:` and `timeout_minutes:` per agent; extend `allowed_write_paths` for developer agents to include their input plan stage; append "if stuck" stanza to every `prompt_templates.<role>`. |
| `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md` | Spec §4.2 (add `blocked`), §6.2 (transition matrix update), §7 (note on stream-json + log file). |
| `web/src/stores/agents.ts` | Update `agent.progress` payload shape to handle `{event, raw}`; same 200-line cap. |
| `web/src/views/project/AgentsRunsView.vue` | Status chip colour for `killed-timeout`; pretty-print event types; "View log" link. |
| `web/src/api/agents.ts` | Add `getRunLog(project, runId)` returning text. |

## Verification

1. **Build**: `go build ./...` and `go vet ./...` clean.
2. **Frontend**: `pnpm exec vue-tsc --noEmit` clean; `pnpm build` succeeds.
3. **Model wiring**: start the server, run `analyst-requirements` against an idea, check the run's process args (visible in stderr_tail or log file) include `--model opus`.
4. **No timeout**: configure `timeout_minutes: 0`, start a long-running agent, confirm the supervisor never auto-kills (run continues until the subprocess exits naturally or the user clicks Kill).
5. **Timeout fires**: configure `timeout_minutes: 1`, start an agent likely to take >1 min, confirm status flips to `killed-timeout` (not `killed`) in the UI and the chip is amber.
6. **User-kill**: start a run, click Kill in the UI, confirm status becomes `killed` (not `killed-timeout`).
7. **Block flow**: hand the analyst-requirements agent a deliberately vague idea (e.g. just a title with no body). Expect: agent appends `## Open Questions` to the idea, sets `status: blocked`, assigns `product-owner`. Run finishes `done` (the agent stopped cleanly, not failed). The blocked artifact appears in the artifact list with the new status. As `product-owner`, transition `blocked → draft` and re-run the agent — confirm the transition is allowed.
8. **Live progress**: during a run, expand the row in AgentsRunsView, confirm assistant text + tool-use lines stream in (not just final output). 
9. **Log file**: `cat ~/.kaos-control/data/kaos-control/runs/<run-id>.log` shows full stdout+stderr. Click "View log" in the UI and the same content renders.

## Out of scope

- Mid-run interactive Q&A (would need stdin pipe + structured `[[ASK]]` markers; see prior conversation).
- Anthropic SDK / MCP drivers (already out of v1 per spec §7.2).
- Auto-rerun when a `blocked → draft` transition happens (manual rerun for now).
- Migration of any existing `killed` rows in the index — historical runs stay as-is.
