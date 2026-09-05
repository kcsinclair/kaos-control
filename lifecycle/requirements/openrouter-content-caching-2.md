---
title: OpenRouter Content Caching for Google and Anthropic Models
type: requirement
status: blocked
lineage: openrouter-content-caching
created: "2026-09-04T20:15:00+10:00"
priority: normal
parent: lifecycle/ideas/openrouter-content-caching.md
labels:
    - requirement
    - cache
    - provider
    - openrouter
    - cost
    - performance
release: KC-Release6
assignees:
    - role: analyst
      who: agent
    - role: product-owner
      who: agent
---

# OpenRouter Content Caching for Google and Anthropic Models

Parent: [[openrouter-content-caching]] (idea). Related:
[[open-provider-support]], [[provider-model-for-agents]],
[[openrouter-token-metrics-extraction]], [[agent-usage-analytics-report]],
[[agent-run-summary-panel]].

## Problem

When using `driver: openai-compatible` with OpenRouter, multi-turn agent runs
re-send large context on every turn (system instructions, project specifications,
prior conversation turns, and tool outputs). While frontier models from Anthropic
(Claude) and Google (Gemini) support prompt and context caching natively,
OpenRouter requests sent by kaos-control do not enable or tune OpenRouter's
content-caching features.

This leads to:
1. **Unnecessary Token Spend & Latency:** Without caching enabled or extended,
   identical prompts and large system contexts are repeatedly processed at full
   input price and higher latency.
2. **Ephemeral Default TTL:** By default, cache entries on supported providers
   expire quickly (e.g. 5 minutes). In iterative development or when agents pause
   waiting for queued jobs or multi-turn execution, cache entries expire before
   subsequent requests can reuse them.
3. **Lack of Invalidation & Control:** Callers and operators currently have no
   mechanism to explicitly enable caching, configure custom TTLs (e.g., 2 or 4
   hours, or up to 24 hours), clear caches on demand, or configure defaults per
   provider or per agent.

OpenRouter supports prompt content caching via custom headers and request options
such as `X-OpenRouter-Cache: true`, `X-OpenRouter-Cache-TTL: <seconds|duration>`,
and cache-invalidation headers (`X-OpenRouter-Cache-clear: true`). kaos-control
needs first-class configuration and runtime support for these caching capabilities
in its `openai-compatible` driver, provider configuration, and per-run options.

## Goals / Non-goals

### Goals

- Allow content caching to be configured at the **Provider** level (in app config
  `providers`), at the **Agent** level (in `lifecycle/config.yaml`), or
  configured via request headers (`X-OpenRouter-Cache`, `X-OpenRouter-Cache-TTL`,
  `X-OpenRouter-Cache-clear`).
- Support configurable Cache TTL ranging from 1 second up to 24 hours (with
  sensible durations such as 2h or 4h beyond the provider's default 5 minutes).
- Support explicit cache clearing / invalidation via header or parameter
  (`X-OpenRouter-Cache-clear: true`).
- Ensure caching headers are **only applied when the provider is OpenRouter** (or
  configured to accept them), avoiding invalid header errors on local or
  strict non-OpenRouter providers.
- Maintain seamless integration with [[openrouter-token-metrics-extraction]]: ensure
  cached prompt tokens (`cached_tokens`, `cache_read_input_tokens`, and
  `cache_write_tokens`) are accurately parsed, attributed, and reported in the
  run summary and usage analytics.
- Provide clear documentation in `docs/` and schema validation in
  `internal/config`.

### Non-goals

- Implementing an internal local disk or in-memory LLM response cache within
  kaos-control itself (caching is delegated entirely to OpenRouter and its
  upstream model hosts).
- Supporting caching headers on non-OpenRouter endpoints that do not recognize
  OpenRouter-specific headers (e.g., direct Ollama, llama-server, vLLM).
- Automatic semantic prompt deduplication or cache-key rewriting.
- Enforcing hard financial budgets based on cache hit ratios.

## Detailed Requirements

### Architecture-Breaking Requirements

Reviewed against `lifecycle/architecture/architecture-summary.md`, the promoted
architecture ([[modular-monolith]]), the tech stack ([[go-vue]]), the ADRs, and
`standards/`. **No architecture-breaking requirement is introduced, and no new
ADR is required.** Evaluation against standing constraints:

1. **Single self-contained binary (no cgo, no external services).** Caching is
   handled over HTTP headers with the existing OpenRouter provider using Go's
   standard `net/http` package. No local Redis, Memcached, SQLite cache schema, or
   cgo libraries are required. → **Satisfied.**
2. **Local filesystem is the source of truth; index is a rebuildable cache
   ([[index-is-a-cache]], [[adr-0003-pure-go-sqlite-index]]).** Configuration is
   persisted in standard YAML configs (`lifecycle/config.yaml` and app-level
   `config.yaml`). No state database becomes authoritative. → **Satisfied.**
3. **Agents execute arbitrary tools; tool calls must be mediated
   ([[adr-0006-mediated-agent-driver-permission-model]],
   [[filesystem-sandboxing]]).** Header construction and caching configuration
   reside in driver and provider management; agent execution sandboxing and
   permission checks remain untouched. → **Satisfied.**
4. **Direct-served, no trusted proxy hop
   ([[adr-0001-no-header-based-client-ip-trust]]).** The `X-OpenRouter-Cache*`
   headers are outbound headers sent *from* kaos-control *to* OpenRouter, not
   untrusted inbound headers used for client authentication or IP trust. →
   **Satisfied / not applicable.**
5. **Cost minimisation at launch.** Content caching directly serves this goal by
   lowering token consumption costs for repetitive multi-turn agent interactions
   on expensive models (e.g. Anthropic Claude 3.5 Sonnet, Google Gemini 1.5/2.5 Pro). →
   **Satisfied.**
6. **Secrets hygiene ([[secrets-handling]]).** Caching flags and TTLs do not
   contain secrets. Standard API keys remain masked and protected under
   existing rules. → **Satisfied.**

**Conflicts flagged against Architecture Summary: none.**

### Functional Requirements

#### FR-1: Provider and Agent Configuration for Caching

- `config.ProviderConfig` (`internal/config/config.go`) MUST support optional
  caching settings:
  ```yaml
  cache_enabled: true         # boolean, default false
  cache_ttl: "2h"             # string duration (e.g., "300s", "2h", "4h") or integer seconds
  ```
- `config.AgentConfig` (`lifecycle/config.yaml`) MUST support optional per-agent
  overrides:
  ```yaml
  cache_enabled: true         # optional boolean pointer
  cache_ttl: "4h"             # optional duration or seconds string
  ```
- When an agent runs, configuration precedence MUST be:
  1. Per-request override / caller parameter (if specified in execution payload),
  2. Per-agent config (`AgentConfig.CacheEnabled`, `AgentConfig.CacheTTL`),
  3. Provider config (`ProviderConfig.CacheEnabled`, `ProviderConfig.CacheTTL`),
  4. Default: caching disabled (`false`), default OpenRouter TTL (5 minutes).

#### FR-2: Outbound Header Construction for OpenRouter Requests

- When caching is enabled for an `openai-compatible` run and the provider is
  OpenRouter (matching `isOpenRouterProvider(prov)`: `base_url` contains
  `openrouter.ai` or provider name is `"openrouter"`):
  - The driver MUST attach the HTTP header:
    `X-OpenRouter-Cache: true`
  - If a custom TTL is configured, the driver MUST attach:
    `X-OpenRouter-Cache-TTL: <seconds>` (formatted as integer seconds, e.g. `7200` for 2h, or acceptable OpenRouter duration string)
  - If cache clearing is requested for the run/turn, the driver MUST attach:
    `X-OpenRouter-Cache-clear: true`
- If caching is disabled, `X-OpenRouter-Cache: false` or omission of the header
  MUST be respected.
- When the provider is **not** OpenRouter, these headers MUST NOT be injected
  automatically, unless explicitly specified in the provider's `extra_headers`.

#### FR-3: TTL Range Validation and Normalization

- Cache TTL MUST be validated within the bounds:
  - Minimum: 1 second
  - Maximum: 86400 seconds (24 hours)
- Parsing MUST accept standard Go duration strings (e.g. `30s`, `5m`, `2h`, `4h`,
  `24h`) as well as raw numeric seconds (e.g. `7200`, `14400`).
- Any TTL outside 1s–24h MUST be rejected during configuration validation with a
  descriptive error message.

#### FR-4: Cache Invalidation / Clear Trigger

- The system MUST support triggering a cache clear on demand:
  - At the agent run launch API (`POST /api/p/:project/agents/runs`) via an optional
    parameter `clear_cache: true`.
  - When active, the initial request of that run MUST include
    `X-OpenRouter-Cache-clear: true`.
  - Subsequent multi-turn requests within the same run MUST NOT retain the clear
    header, allowing subsequent turns to build and benefit from the new cache.

#### FR-5: Integration with Token Metrics and Cost Reporting

- OpenRouter response stream chunks provide usage in `prompt_tokens_details`:
  - `cached_tokens`: Tokens served from cache.
  - `cache_write_tokens`: Tokens written to cache.
- The `openai-compatible` driver MUST continue mapping `cached_tokens` to
  `RunResultUsage.CacheReadInputTokens` and, where available from OpenRouter,
  `cache_write_tokens` to `RunResultUsage.CacheCreationInputTokens`.
- Run logs MUST record the caching status at start:
  `# openrouter_caching: enabled=true ttl=<seconds> clear=<bool>`
- In `RunSummaryCard.vue` and analytics reports ([[agent-usage-analytics-report]]),
  cache reads and savings MUST reflect the caching performance achieved with
  these headers.

#### FR-6: Provider Form and UI Settings

- In `web/src/components/provider/ProviderForm.vue` (and `ProviderSettingsView.vue`),
  when configuring a provider with OpenRouter or OpenAI-compatible driver:
  - Add optional toggle: "Enable Content Caching" (`cache_enabled`).
  - Add optional input: "Cache TTL" (`cache_ttl`), placeholder "e.g. 2h, 4h, 300s".
- Display a hint explaining that extended TTLs (up to 24 hours) apply to supported
  upstream providers such as Google Gemini and Anthropic Claude on OpenRouter.

### Non-Functional Requirements

#### NFR-1: Provider Compatibility & Safety

- Local providers (Ollama, llama.cpp, LocalAI) and direct OpenAI/Mistral endpoints
  MUST NOT receive unexpected `X-OpenRouter-*` headers unless explicitly configured
  by the user in `extra_headers`.
- Any error response from OpenRouter regarding caching parameters (such as an
  unsupported model or TTL syntax) MUST be clearly captured and reported in the
  run log without crashing the daemon.

#### NFR-2: Observability & Logging

- Outbound caching headers MUST be logged to the run's session log for
  diagnostics (with secrets masked per NFR-2 of
  [[openrouter-token-metrics-extraction]]).
- The run summary MUST clearly show cache read/write token counts so operators
  can verify caching effectiveness.

#### NFR-3: Performance

- Evaluating caching headers and formatting TTL MUST have negligible CPU and
  memory overhead (< 1ms per turn request).

## Acceptance Criteria

- [ ] `ProviderConfig` supports `cache_enabled: bool` and `cache_ttl: string|int` in app config. ([[provider-model-for-agents]])
- [ ] `AgentConfig` in `lifecycle/config.yaml` supports `cache_enabled` and `cache_ttl` per agent.
- [ ] When caching is enabled on an OpenRouter provider, outgoing `/v1/chat/completions` requests include `X-OpenRouter-Cache: true`. ([[open-provider-support]])
- [ ] Configured TTL values (e.g. `2h`, `4h`, `7200`) are translated into valid `X-OpenRouter-Cache-TTL` header values within the allowed range (1s to 24h).
- [ ] Out-of-bounds TTL values (< 1s or > 24h) fail config validation with a clear error.
- [ ] Triggering a run with `clear_cache: true` sets `X-OpenRouter-Cache-clear: true` on the initial request and clears the flag for subsequent turns.
- [ ] Non-OpenRouter providers do not receive `X-OpenRouter-*` headers by default.
- [ ] Token usage metrics accurately record `cached_tokens` and `cache_write_tokens` when caching is active. ([[openrouter-token-metrics-extraction]])
- [ ] Run summary card displays cache hit ratio and cached token metrics for runs utilizing OpenRouter content caching. ([[agent-run-summary-panel]])
- [ ] The Web UI provider form allows enabling caching and setting TTL for OpenRouter providers.
- [ ] All unit and integration tests pass (`go test ./... -short` and frontend test suites).

## Open Questions

1. **Upstream Model Support & Graceful Degradation:** When `X-OpenRouter-Cache: true`
   is sent for models that do not natively support caching (e.g. older open-weight
   models or models without prompt caching support), does OpenRouter silently ignore
   the header or return a 400 Bad Request? If an error is returned, should the
   driver implement an automatic retry fallback similar to FR-2 in
   [[openrouter-token-metrics-extraction]]?
2. **Dynamic Invalidation vs. Natural TTL:** Is an automatic cache clear desirable
   when a prompt template or agent definition in `lifecycle/config.yaml` is updated,
   or should cache invalidation remain strictly manual / TTL-based?
