---
title: Ollama Agents Need an Execution Layer
type: idea
status: blocked
lineage: ollama-agents-need-execution-layer
priority: high
labels:
    - agent
    - ollama
assignees:
    - role: product-owner
      who: agent
---

# Ollama Agents Need an Execution Layer

## Problem

Ollama-driven agents run successfully and produce plausible output, but
the output is *text only*. Nothing is written to disk, no tests run, no
commits happen. The agent appears to "work" — the run completes, the WS
stream looks healthy, the on-disk log shows a clean `done_reason=stop`
— but the artifacts the agent was asked to create never materialise.

Observed example (run `08ec885f91a22f24`): a QA agent was given a test
artifact and asked to "run the relevant integration tests in tests/ and
for EACH failing test, create one defect artifact in lifecycle/defects/
with frontmatter ...". The agent dutifully *typed out* a complete
defect markdown file in its response (including hallucinated test
output and a fake stack trace at `src/scheduler/core.ts:142:12` — a
file that doesn't exist), and the run terminated successfully. No file
was created in `lifecycle/defects/`.

## Why

Claude Code is a CLI with built-in tools (`Write`, `Edit`, `Read`,
`Bash`). When the model emits a tool-call event in its stream-json
output, the `claude` binary itself executes the I/O. Our
`OllamaDriver` just tees that stream to disk and the WebSocket hub —
Claude does the actual file writes inside its own process.

Ollama's `/api/generate` and `/api/chat` are pure text-completion APIs.
The model returns a string. There is no execution layer — no one
writes files, runs tests, or commits. The "agent" is just a chat with
a system prompt. This is correct LLM behaviour for a plain completion
endpoint, not a bug in the driver.

## Options

Three plausible paths, in increasing order of effort and capability:

### Option 1 — Response post-processor (cheap, brittle)

Teach `OllamaDriver` (or a thin wrapper after it) to scan the completed
response for fenced code blocks with file headers and actually write
them to disk. Example output shape the system prompt would enforce:

````
### lifecycle/defects/scheduler-deadlock.md
```markdown
---
title: …
type: defect
…
---

Body…
```
````

Implementation: ~100 lines of Go and one regex. Works surprisingly
well with strong system-prompt instructions ("one fenced block per
file, header line `### <relative-path>` immediately before the block,
no commentary outside blocks"). Path validation must go through the
existing sandbox/allowlist enforcement (`internal/sandbox`).

**Pros:** quick to ship; doesn't depend on the Ollama tool-calling
protocol; works with any model.
**Cons:** brittle to model output drift; one-shot only (model can't
read existing files or run tests); no iterative behaviour.

### Option 2 — Tool-call loop in the driver (real work, real agent)

Ollama supports OpenAI-style `tools` / `tool_calls` on models that were
fine-tuned for it (qwen2.5-coder, llama3.1+, mistral, etc.). The
driver becomes a mini-runtime:

  1. Pass `tools: [{name:"write_file", …}, {name:"read_file", …},
     {name:"run_tests", …}, …]` in the request.
  2. Parse `tool_calls` from the response.
  3. Execute each call against the sandbox.
  4. Send the result back as a tool-role message.
  5. Loop until the model returns without tool calls (i.e. done).

This is what Claude Code does internally. Effort: maybe a week of
focused work — plus careful sandbox enforcement, error handling for
malformed tool calls, and a safety budget on tool-call iterations.

**Pros:** any tool-capable Ollama model becomes a real agent; matches
the mental model users already have from Claude.
**Cons:** much more code; tied to one specific tool-call protocol;
quality varies by model; harder to reason about safety bounds.

### Option 3 — Delegate to an existing harness

Instead of writing our own tool loop, shell out to a project that has
already solved this (Aider, openai-agents-python, etc.) and let it
manage the model interaction:

```
aider --model ollama/qwen2.5-coder:32b --message "<prompt>" <files…>
```

**Pros:** cheaper to integrate; inherits a maintained tool-call /
sandbox model.
**Cons:** new dependency and runtime; their conventions may clash with
ours (commit format, sandbox rules, frontmatter handling); harder to
embed in the single-binary distribution.

## Recommendation

Start with **Option 1**. It unblocks Ollama agents for the workflows
where the model just needs to produce structured markdown artifacts
(defects, plans, test specs, ideas) — which is most of our agent
roster. Pair it with strict system-prompt instructions on output
shape. Once we've measured how often models follow the format, decide
whether the gap to **Option 2** is worth closing for the few agent
roles that need iterative tool use (qa is the obvious candidate
because it needs to actually run tests).

## Open Questions

- Should the post-processor reject responses that contain *any* text
  outside fenced file blocks (strict mode), or accept them and just
  log the extra prose (lenient mode)?
- What sandbox enforcement applies — the existing
  `AgentConfig.AllowedWritePaths`, or something tighter?
- Should we still commit when no files were actually produced (i.e.
  the agent only emitted prose)? Currently every successful run
  commits; an Ollama run that wrote nothing would commit an empty
  diff.
- Does the same machinery need to work for the run summary / fail
  surfacing on the UI, or is the existing `agent.progress` →
  `agent.finished` flow sufficient?
