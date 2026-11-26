---
title: Per-Agent Sampling Parameters (temperature, top_p, max_tokens, …)
type: idea
status: draft
lineage: agent-sampling-parameters
created: "2026-11-26T10:00:00+10:00"
priority: normal
labels:
    - feature
    - open-provider-support
    - agents
---

# Per-Agent Sampling Parameters (temperature, top_p, max_tokens, …)

## Origin

Community contribution from Hugo Burton, GitHub PR #25 ("Add llama.cpp driver for local model support"). Most of that PR is now covered on `kc-dev`: llama.cpp is reached through the generic `openai-compatible` driver and `providers`. The part kc-dev does not have is the **per-agent `max_tokens` and `temperature` settings**. We want to merge that idea into kc-dev, credit Hugo as co-author, and add other common sampling parameters while we're at it.

## Problem

Agent model calls currently send no sampling parameters. Every agent gets the provider's or model's defaults, so there is no way to:

- make an analyst or reviewer agent more deterministic (low temperature) or a brainstorming agent more creative;
- follow model-specific settings that model authors recommend (for example Qwen/Gemma recommend particular `temperature`/`top_p`/`top_k` values for coding work);
- limit output length for small local models, or raise it where a server's default is too low;
- get repeatable runs while testing (a fixed `seed`).

## Proposal

Add an optional `sampling` block on each agent in `lifecycle/config.yaml`, with optional per-provider defaults in the app config.

```yaml
# lifecycle/config.yaml
agents:
  - name: backend-developer-local
    roles: [backend-developer]
    driver: openai-compatible
    provider: local-llama
    model: qwen3-coder-30b
    sampling:
      temperature: 0.2
      top_p: 0.9
      top_k: 40          # non-OpenAI extension; llama.cpp, vLLM, OpenRouter
      min_p: 0.05        # non-OpenAI extension
      max_tokens: 8192
      seed: 42
      presence_penalty: 0.0
      frequency_penalty: 0.0
      repeat_penalty: 1.1  # llama.cpp extension
      stop: ["</done>"]
```

```yaml
# ~/.kaos-control/config.yaml: optional provider-level defaults
providers:
  - name: local-llama
    base_url: http://localhost:8080
    driver: openai-compatible
    sampling:
      temperature: 0.6
```

### Parameters in scope

| Parameter | Type / range | OpenAI-compatible | Gemini API (`generationConfig`) |
|---|---|---|---|
| `temperature` | float, 0–2 | `temperature` | `temperature` |
| `top_p` | float, 0–1 | `top_p` | `topP` |
| `top_k` | int ≥ 0 | `top_k` (extension) | `topK` |
| `min_p` | float, 0–1 | `min_p` (extension) | n/a |
| `max_tokens` | int > 0 | `max_tokens` | `maxOutputTokens` |
| `seed` | int | `seed` | `seed` |
| `presence_penalty` | float, −2–2 | `presence_penalty` | `presencePenalty` |
| `frequency_penalty` | float, −2–2 | `frequency_penalty` | `frequencyPenalty` |
| `repeat_penalty` | float > 0 | `repeat_penalty` (llama.cpp) | n/a |
| `stop` | list of strings | `stop` | `stopSequences` |

Possible extension: an `extra_body` map merged into the request as-is, for provider-specific knobs (for example OpenRouter `provider` routing or `reasoning`) without a code change each time. Requirements should decide whether this is in scope.

## Backwards compatibility (hard requirement)

- **If a parameter is not set, it is not sent.** It is left out of the request body completely, so behaviour stays exactly as it is today, the provider's or model's default.
- No default values in code: no fallback to `temperature: 0.7` and no `max_tokens: 4096` cap. (PR #25 had both. The 4096 cap would cut off long outputs from file-writing agents.)
- An explicit `temperature: 0` must be sent as 0, meaning greedy/deterministic, not treated as "unset". Implementation should use pointer/nullable fields (`*float64`, `*int`) so "unset" and "zero" are different.
- Existing config files with no `sampling` block must load and run unchanged.

## Precedence

1. Agent `sampling` value, if set.
2. Provider `sampling` default, if set.
3. Not sent (the model's default).

The values actually used should be recorded in the run log header (and in the run row / usage report if cheap to do), so a run can be reproduced and compared.

## Driver applicability

- **`openai-compatible`**: all parameters. Integration point is the request body built in `internal/agent/openai_compatible.go` (`body := map[string]any{...}`), and the same values should apply to recovery/retry requests. Preflight/warmup calls that deliberately use `max_tokens: 1` are not affected.
- **`gemini` (API)**: map to `generationConfig` as in the table above.
- **CLI drivers** (`claude-code-cli`, `claude-mediated`, `claude-env`, `codex-cli`, `gemini-cli`): the CLIs don't expose these settings. Config validation should **warn** (not fail) when `sampling` is set on such an agent, and the parameters are ignored.
- **`ideachat`**: already has `MaxTokens`. Requirements should decide whether idea-chat follows the same block.

## Validation

- Range-check values at config load, with errors naming the agent and the field (same style as the existing `max_tool_iterations` validation).
- If a provider rejects a parameter with HTTP 400 (some servers reject unknown fields), the error should name the parameter. Consider the same "drop the field and retry once" approach already used for `stream_options` / usage parameters.

## UI (nice to have)

- Show the configured sampling values on the agent details / settings view (read-only to start).
- Show the values used in the run detail header.

## Out of scope

- Changing models per run, or tuning sampling per prompt or per role inside a single agent.
- Any change to the separate llama.cpp driver from PR #25. It is superseded by `openai-compatible` + `providers`.

## Acceptance sketch

- Agents with no `sampling` block send byte-identical request bodies to today's (covered by a test).
- `temperature: 0` is sent as `0`.
- Agent values override provider defaults; provider defaults apply when the agent leaves a field unset.
- Values out of range are rejected at config load with a clear message.
- CLI-driver agents with `sampling` produce a warning and run as before.
- Hugo Burton credited (`Co-Authored-By`) on the implementing commit; PR #25 closed with a link to it.
