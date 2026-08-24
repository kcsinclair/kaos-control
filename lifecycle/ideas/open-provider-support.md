---
title: Open Provider Support
type: idea
status: approved
lineage: open-provider-support
created: "2026-08-24T18:36:36+10:00"
priority: normal
labels:
    - provider
    - driver
    - agent-runner
    - ai-ml
    - architecture
    - config
    - feature
    - high-complexity
    - open-provider-support
release: KC-Release6
---

# Open Provider Support

**Epic.** Make kaos-control provider-agnostic so agent runs are not locked to a
single upstream API — covering local inference (Ollama, llama.cpp), gateways
(OpenRouter), and direct cloud APIs (Anthropic, OpenAI).

## The real gap

The gap is **not** "add more providers" — it is that local and third-party
models can *talk* but cannot *act*:

- the native **`ollama`** driver is single-shot `/api/chat` — no tool use, no
  file edits, no multi-turn loop;
- **`claude-env`** does give the full agentic loop, but only by shelling out to
  the `claude` binary speaking the Anthropic wire protocol.

So the expensive, shared deliverable is a **first-party OpenAI-compatible driver
with a tool-calling agent loop**. A provider registry without that loop only
buys more ways to run a chatbot. Every remaining target — OpenAI, OpenRouter,
llama.cpp, Ollama, Groq, Together, Azure — speaks the same
`/v1/chat/completions` wire format, so one driver plus configuration covers them
all.

## Settled decisions

Captured here pending an ADR under `lifecycle/architecture/decisions/`:

1. **Shape.** Not a provider/driver/model three-tuple. The **driver moves onto
   the provider**, so a Provider is `{name, base_url, api_key, driver,
   extra_headers}` and an Agent is a clean `{provider, model}` pair.
2. **The native `ollama` driver is removed outright** — no deprecation shim, no
   back-compat for `driver: ollama`. Ollama exposes an OpenAI-compatible
   endpoint, so the new driver reaches it with no special-casing. Zero agents
   reference `driver: ollama` today, so nothing breaks functionally.

   **Sequencing matters, though:** the removal is not standalone dead-code
   cleanup. The whole Ollama surface goes with it —
   `internal/agent/ollama.go`, the `/ollama/instances` API
   (`internal/http/ollama.go`), `OllamaInstances` in app config, and the
   frontend (`OllamaSettingsView.vue`, `components/ollama/`,
   `stores/ollamaInstances.ts`, `api/ollama.ts`) — roughly 9 backend and 5
   frontend files. That surface is *in use*: an instance is registered today
   (`Loki` → `http://leia.packsin.com:11434`). So removal lands **with** the
   provider replacement in workstream 1, where `ollama_instances` becomes
   `providers` and the existing registration carries over as a provider record
   (same base URL, `driver: openai-compatible`). Removing it before the
   replacement exists would leave no local-model path and drop a live config.
3. **Sidecar agent drivers are deferred, not rejected.**
   [[sidecar-agent-drivers]] (external LangChain/Aider/Goose runners emitting
   ProgressEvent NDJSON) stays a future extensibility path for third-party agent
   frameworks; the first-party Go driver is the primary approach. The ADR should
   record this as deferred.

## Verification targets (live, confirmed)

Two local servers are already running and were probed during this analysis —
both expose the **same** OpenAI-compatible `/v1/models` surface, which is the
empirical proof that one driver covers both:

| Endpoint | Server | Evidence |
|---|---|---|
| `leia.packsin.com:11434` | **Ollama** (registered as `Loki`) | serves `/v1/models` *and* native `/api/tags`; models incl. `qwen3-coder:30b`, `gemma3:12b`, `glm-5.2:cloud` |
| `leia.packsin.com:7442` | **llama.cpp** | `/v1/models` → `owned_by: "llamacpp"`; `llama-server` launched with `--jinja` (tool-call templating), model `Dolphin3.0-Llama3.1-8B-Q4_K_M`, lazy-loaded |

That Ollama answers `/v1/models` is exactly why the native `ollama` driver can
be removed: the openai-compatible driver reaches the same server and the same
models. And `--jinja` on the llama.cpp side is the template support the
tool-calling loop depends on, so agent-mode can be verified against a real local
server rather than in theory.

## Workstreams

This lineage is the parent for three requirements:

1. **Provider abstraction + OpenAI-compatible agent driver** — the Provider
   entity (config, API, UI) plus the tool-calling driver. Absorbs the ideas
   below. The tool-loop specification already exists in detail as
   `requirements/llama-cpp-driver-2.md` and is being generalised rather than
   rewritten.
2. **Provider failover** ([[switch-provider]]) — depends on 1. Cheaper than it
   looks: rate-limit/529 detection already ships
   ([[rate-limit-event-detection]], `extractRateLimitText` →
   `queue.rate_limit`), so this only needs to *act* on an existing signal.
3. **Local-model operability** ([[ollama-support-improvements]]) — prompt
   templates tuned to local models, model availability checks, error surfacing.
   Partly dissolves into 1 once the shared driver owns request logging.

## Absorbed ideas (deduplicated)

All three are the same wire protocol; they collapse into workstream 1 as
configuration plus one driver:

- [[openai-api-integration]] — the openai-compatible driver itself.
- [[openrouter-llm-integration]] — config preset + `extra_headers`
  (`HTTP-Referer`, `X-Title`). No new code beyond the header field.
- [[llama-cpp-driver]] — the tool-call round-trip and local-server target; its
  requirement is the most developed spec of the hard part.

## Related (not absorbed)

- [[provider-model-for-agents]] — the architecture spine for workstream 1.
- [[switch-provider]] — workstream 2.
- [[ollama-support-improvements]] — workstream 3.
- [[sidecar-agent-drivers]] — deferred alternative (see decision 3).
- [[agent-cost-basis-and-savings]] — becomes materially more useful once cost
  varies per provider/model.
- Shipped predecessors, for context: [[ollama-agent-support]],
  [[ollama-claude-code-driver]]. [[ollama-agents-need-execution-layer]]
  (abandoned) was the original framing of the can-talk-but-cannot-act gap.
