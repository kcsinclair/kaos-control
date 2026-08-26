---
title: openai-compatible driver kills reasoning models — TTFT only counts delta.content
type: defect
status: draft
lineage: openai-driver-ttft-ignores-reasoning-content
created: "2026-08-25T14:30:00+10:00"
priority: high
labels:
    - defect
    - agent
    - agent-runner
    - driver
    - provider
    - local-models
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

# openai-compatible driver kills reasoning models — TTFT only counts `delta.content`

## Reproduction Steps

1. Configure an `openai-compatible` provider pointing at llama.cpp
   (`http://leia.packsin.com:7442`).
2. Use a **reasoning** model — e.g. `gemma-4-26B-A4B-it-qat-UD-Q4_K_XL`.
3. Start an agent run (observed: `7301b2c7d17b7a70`).
4. The run is killed at exactly the configured load timeout, reporting
   `model did not respond within the loading timeout`, **while the model is
   actively streaming**.

## Expected Behaviour

A model that is streaming *any* output — reasoning tokens, content tokens, or
tool-call deltas — is demonstrably responding. The load-timeout watchdog exists
to catch a model that never produces anything (still loading weights, or hung);
it must not fire against a model that is visibly generating.

## Actual Behaviour

The run is cancelled mid-stream. The final line of the run log is a chunk that
is unambiguously live generation:

```json
{"choices":[{"finish_reason":null,"index":0,
  "delta":{"reasoning_content":" can"}}],
  "model":"gemma-4-26B-A4B-it-qat-UD-Q4_K_XL", ...}
# error: model did not respond within the loading timeout after 5m0s
```

The model streamed **`delta.reasoning_content`** — not `delta.content` — for the
entire window, and was killed anyway.

## Root cause

Two defects in `internal/agent/openai_compatible.go`:

1. **`reasoning_content` is not parsed at all.** `openAIStreamChoice.Delta`
   declares only `role`, `content` and `tool_calls`. There is no
   `reasoning_content` field, so those chunks decode to an empty `Delta` and are
   invisible to the driver.
2. **TTFT is recorded from `delta.content` alone.** The sole `recordTTFT()` call
   site is:

   ```go
   for _, choice := range chunk.Choices {
       if choice.Delta.Content != "" {
           recordTTFT()
           turnContent.WriteString(choice.Delta.Content)
       }
       for _, tc := range choice.Delta.ToolCalls { ... }   // no recordTTFT
   }
   ```

   The load-timeout watchdog gates purely on `ttftRecorded`. For a reasoning
   model, `Delta.Content` stays empty while the model thinks, so `ttftRecorded`
   never flips, the watchdog fires, and `cancel()` kills a healthy run.

**Raising the timeout cannot fix this.** Confirmed empirically: at 60 s the run
died at ~60 s; raised to 5 minutes, it died at 302 s — still mid-stream. A
longer timeout only postpones the kill, because the condition being measured
(“no `delta.content` yet”) may remain true for the entire reasoning phase.

### Secondary: tool-call deltas also don't record TTFT

`choice.Delta.ToolCalls` does not call `recordTTFT()` either. A model that
responds with a tool call and no prose — the **normal** case for this driver,
whose entire purpose is tool-calling agent loops — is equally exposed to being
killed by the watchdog.

## Impact

High. This makes reasoning models unusable on the `openai-compatible` driver,
including `gemma-4-26B` — the model documented as the verified llama.cpp target
in [[open-provider-support-2]]. It presents as a load/VRAM failure, so the
operator is sent to check hardware and increase timeouts, neither of which can
help.

## Fix guidance

1. **Add `reasoning_content` to the streaming delta struct** so those chunks are
   decoded rather than silently dropped.
2. **Record TTFT on the first sign of *any* generation** — `content`,
   `reasoning_content`, or a tool-call delta. First-token-received is the
   condition the watchdog actually cares about.
3. Consider surfacing reasoning tokens as a distinct progress stage (the
   `warming_up` / `generating` status events already exist; a `reasoning` stage
   would make a long think visible rather than looking hung).
4. Decide whether `reasoning_content` should be teed into the run log body — it
   is useful for diagnosis but can be large.
5. Add a regression test streaming only `reasoning_content` chunks and asserting
   the run is **not** killed by the load-timeout watchdog.

## Notes

Found immediately after
[[agent-run-wall-clock-exceeds-reported-duration]]-adjacent investigation into a
misleading "Model failed to load into memory" message. That message was itself a
misclassification: `ErrModelLoadTimeout` and genuine OOM both map to
`FailureReasonModelUnloaded`, so a timeout is reported with VRAM remediation
("confirm the host has enough free RAM/VRAM"). Worth separating those two
conditions as part of this fix — the model in question loads fine and, once
resident, answers in 0.6 s.
