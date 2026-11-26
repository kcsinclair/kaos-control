---
title: "openai-compatible driver sets no max_tokens, so one turn can run until the timeout"
type: defect
status: draft
lineage: openai-driver-unbounded-generation
created: "2026-08-27T16:20:00+10:00"
priority: high
labels:
    - defect
    - agent
    - openai-compatible
    - local-models
    - reliability
---

## Summary

The `openai-compatible` driver never sends `max_tokens` on a chat completion
request. `max_tokens` appears only in the preflight probe
([internal/agent/openai_preflight.go:177](../../internal/agent/openai_preflight.go#L177)),
where it is set to `1`. The real request at
[internal/agent/openai_compatible.go](../../internal/agent/openai_compatible.go)
has no cap, so a model that fails to terminate generates until the run-level
timeout kills it.

`max_tool_iterations` does not help — it bounds the number of *turns*, not the
length of any single turn.

## Evidence — run `da3b06f2f3b1ce63`

qa agent, `gemma-4-26B-A4B-it-UD-Q8_K_XL` on `leia-llamacpp`.
`killed-timeout`, exit -1, 30m wall.

| Turn | Action | Elapsed |
|------|--------|---------|
| 1 | `bash make test-unit` | 33s |
| 2 | `bash make test-integration` | 10m 23s |
| 3–7 | 5 × `read_file` / `list_dir` | 7s total |
| 8 | reasoning only — **38,189 log lines, zero tool calls** | **18m 55s** → wall |

Turn 8 opened with `reasoning_content` deltas reading:

> The integration tests failed with many "server did not become ready" errors.
> Specifically: `auth_middleware_test.go` …

and then never emitted a tool call, never produced a message, and never stopped.
It ran for 18m55s and produced a 10.2 MB run log — roughly 8 MB of it a single
turn's SSE token deltas.

The model was reasoning about the 33 readiness failures caused by the
config-defaults bug fixed in commit `dfac2de7`, but the trigger is not
the point: any input that induces a non-terminating generation has the same
effect, and there is nothing in the driver to stop it.

## Impact

- A single runaway turn consumes the entire agent timeout budget, so the run
  fails with `killed-timeout` and produces nothing — no defect files, no partial
  result, despite 7 successful turns of real work beforehand.
- The run log grows without bound. 10.2 MB here; it is written to disk and read
  back in full by the frontend when the run-detail modal parses turns.
- Local models are more prone to this than hosted ones, which makes it a direct
  obstacle to the "use local models more" goal.

## Proposed fix

Not obvious enough to apply without a decision, because a naive cap truncates a
legitimate long `write_file` mid-defect. Options:

1. **Per-agent `max_tokens`** (config field alongside `max_tool_iterations` and
   `timeout_minutes`; plumbing mirrors `MaxToolIterations` — 6 touchpoints:
   `config.go` ×3, `agent.go` ×2, `openai_compatible.go`, `http/agents.go`).
   On `finish_reason: "length"`, fail the turn with a named reason rather than
   silently truncating, so the operator sees why.
2. **Per-turn wall-clock budget** — cap any single turn at a fraction of
   `timeout_minutes`, leaving budget for the agent to recover and write output.
3. **Both** — token cap for cost, turn budget for time.

A new `FailureReason` (e.g. `generation_runaway`) with remediation naming the
turn and the token count would make this self-diagnosing; the current
`killed-timeout` gives no hint that one turn consumed 63% of the budget.

## Related

- Commit `dfac2de7` — the config-defaults bug that triggered this particular run.
- [[agent-run-wall-clock-exceeds-reported-duration]] — also concerns run timing
  accounting.
