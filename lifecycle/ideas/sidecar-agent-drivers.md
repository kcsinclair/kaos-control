---
title: Sidecar Agent Drivers (LangChain, Aider, Goose, …)
type: idea
status: blocked
lineage: sidecar-agent-drivers
priority: medium
labels:
    - agent
    - architecture
assignees:
    - role: product-owner
      who: agent
---

# Sidecar Agent Drivers (LangChain, Aider, Goose, …)

## Context

Two related questions came up while debugging Ollama agents:

1. *Could Kaos Control use LangGraph?* — Yes, technically, but
   embedding it directly is a poor fit: LangGraph is Python, Kaos
   Control is a single Go binary, and a surprising amount of
   LangGraph's value (state machine, persistence, streaming,
   multi-agent orchestration) is already in
   [internal/workflow/](../../internal/workflow/),
   [internal/index/](../../internal/index/), the WebSocket hub, and
   the queue/lineage lock manager.

2. *Are there existing systems that could be installed in parallel?* —
   Yes: Aider, Goose, OpenHands, Cline/Continue, CrewAI, AutoGen,
   smolagents, eino. Most are local-first, most support Ollama, and
   none of them want to own lifecycle management — which is exactly
   the gap Kaos Control fills. The relationship is naturally
   complementary, not competitive.

Both threads converge on the same pattern: **let Kaos Control invoke
external agent runtimes as subprocesses, the same way it already
invokes the `claude` CLI**. The `Driver` interface in
[internal/agent/agent.go](../../internal/agent/agent.go) was
deliberately designed for this — `ClaudeCodeDriver` shells out to a
separately-installed `claude` binary that handles its own tool
execution. A LangChain driver (or Aider, Goose, …) would have the
exact same shape.

## Proposal — sidecar drivers

Architecture:

```
Kaos Control (Go, single binary)
        │
        │ spawn subprocess
        ▼
~/.kaos-control/venv/bin/python  -m  kaos_langchain_runner  \
    --prompt-file <path> --config <json> --project-root <path>
        │
        │ stdout: NDJSON events  (existing ProgressEvent format)
        ▼
LangChain / LangGraph
        │ tool calls: write_file, read_file, run_bash, …
        ▼
real disk writes inside <project-root>, respecting AllowedPaths
```

### Go side — small

- **New driver** at `internal/agent/langchain.go` implementing the
  existing `Driver` interface. ~150 lines, almost a copy of
  `ClaudeCodeDriver`: build args, spawn, read stdout line-by-line as
  `ProgressEvent`s, tee to log, wait for exit. The driver does not
  know or care that the subprocess is Python.
- **Driver registration** in
  [agent.go:345-347](../../internal/agent/agent.go#L345-L347):
  ```go
  "langchain": &LangChainDriver{PythonPath: pythonPath, RunnerPath: runnerPath},
  ```
- **Config plumbing** — `lifecycle/config.yaml` agents pick driver
  `langchain`, plus optional fields like
  `langchain_model: "ollama/qwen2.5-coder:32b"`.
- **One-time bootstrap** — `kaos-control init` (or a new
  `kaos-control sidecar install`) creates a venv at
  `~/.kaos-control/venv` and pip-installs the runner's dependencies.

### Python side — ~200 lines

```python
# kaos_langchain_runner.py
# - parse args (prompt file, agent config JSON, project root)
# - build LangGraph state graph with read_file / write_file / run_bash tools
# - apply sandbox: write_file rejects paths outside AllowedPaths
# - run with ChatOllama or ChatAnthropic etc.
# - stream NDJSON events to stdout matching ProgressEvent shape:
#     {"type":"tool_call","name":"write_file","path":"…"}
#     {"type":"tool_result","name":"write_file","ok":true}
#     {"type":"completed","summary":"…"}
```

The runner ships in the Go binary's `embed.FS` (same way `web/dist`
already is) and gets extracted to `~/.kaos-control/sidecars/` on first
use — so the user story stays "single binary install, then `python -m
venv` happens on first agent run".

## Why this is attractive

- **Zero overlap with existing code.** The `Driver` interface is
  already designed for exactly this. Workflow / queue / lineage / git
  commit all keep working unchanged.
- **Opt-in.** Users who don't want Python never touch it. The
  `claude` and `ollama` drivers stay first-class.
- **Pluggable model layer.** LangChain's `ChatOllama` /
  `ChatAnthropic` / `ChatOpenAI` mean one driver covers many backends.
- **Tested tool loop.** Retries, tool-call parsing, structured
  output, checkpointing — all inherited.
- **Cheap to experiment with.** The same sidecar pattern works for
  Aider, Goose, OpenHands, or any other CLI agent — they're all just
  "spawn subprocess, read NDJSON". Adding a second sidecar driver
  costs another 150 lines of Go plus a thin wrapper script.

## Candidates worth wrapping

| Tool       | Lang   | Distribution | Why bother? |
|------------|--------|--------------|-------------|
| LangChain / LangGraph | Python | pip | Most flexible; supports tool calls, multi-agent, checkpointing, many model backends. |
| Aider      | Python | pip | Mature, opinionated coding agent; makes commits itself; ChatOllama out of the box. |
| Goose      | Rust   | single binary | Closest in spirit to Kaos Control's distribution model; pluggable extensions. |
| OpenHands  | Python | Docker | Heavyweight but autonomous SWE workflow with browser + shell. Research-grade benchmark. |
| eino       | Go     | library | Only option that keeps the single-binary story intact if you ever want to *embed* (not sidecar) an agent framework. |

## Caveats

- **Sandbox is on the sidecar.** LangChain's tools will write
  anywhere by default. The Python runner has to call back to Kaos
  Control's allowlist — either by enforcing `AllowedPaths` inside
  Python, or by asking Kaos Control "may I write this?" over a stdin
  protocol (cleaner but fiddlier).
- **NDJSON contract.** Want a small versioned protocol between Go
  and Python so changes to one don't silently break the other.
- **Venv hygiene.** Python venvs rot. Need `kaos-control sidecar
  update` and pinned versions in a `requirements.txt` shipped via
  `embed.FS`.
- **Discovery / installation experience.** First-run pip install is
  slow and can fail (network, Python version, system libs). Need
  good error messages.

## Smallest viable proof-of-concept

1. Add `LangChainDriver` to the codebase — but for the first commit,
   point it at a `kaos_langchain_runner.py` that is literally just
   `print('hello'); sys.exit(0)`. Goal: prove the spawn + stream +
   log + exit flow works end-to-end without any LangChain involved.
2. Then iterate on the Python side — add tool definitions, sandbox
   enforcement, model selection — without touching the Go driver
   again.

Step 1 is maybe an hour of work and tells us whether the pattern
feels right before we invest in the Python runner properly.

## Open Questions

- Sandbox enforcement: in-Python (fast, but trust boundary inside the
  subprocess) or stdin-protocol callback to Go (safe, but more
  plumbing)?
- Should the sidecar venv be shared across projects, or per-project?
  Shared is simpler; per-project avoids dependency conflicts if two
  projects pin different LangChain versions.
- How does the existing precheck/timeout/rate-limit machinery in
  `supervise()` translate? The Python sidecar can emit `system/init`
  events to satisfy the existing Claude-shaped precheck, or we can
  add a new precheck contract.
- Auto-commit on success: does the Go side still commit the working
  tree as it does for Claude/Ollama runs, or does the sidecar
  participate in the commit (e.g. it might want to commit per
  tool-call iteration)?
- Distribution: embed `kaos_langchain_runner.py` via `embed.FS`, or
  publish to PyPI as `kaos-control-langchain-runner` so it updates
  independently of the Go binary?
