---
title: Please review this idea, create the necessary documentation and feature documentation
type: doc
status: done
lineage: local-model-operability
created: "2026-08-27T12:31:02+10:00"
priority: normal
parent: lifecycle/ideas/local-model-operability.md
release: KC-Release6
output: docs/local-model-operability.md
---

Please review this idea, create the necessary documentation and feature documentation

## Produced

Documentation written to `docs/local-model-operability.md`, covering
[[local-model-operability-2]] (Workstream 3 of [[open-provider-support]]),
verified against the implementation in `internal/agent/prompt_defaults.go`,
`internal/agent/openai_preflight.go`, `internal/agent/openai_compatible.go`,
`internal/agent/errors.go`, and the frontend components/store in
`web/src/lib/failureReasons.ts`, `RunFailureBanner.vue`,
`AgentRunningBanner.vue`, `RunDetailModal.vue`, and `AgentsRunsView.vue`:

- Local-model-tuned prompt fallbacks (`LocalModelPromptDefaults`) and how
  they interact with per-agent `prompt_templates` overrides
- The `/v1/models` availability preflight (3s timeout, fast-fail before
  lock acquisition)
- Warmup/lazy-load detection (5s warmup signal, configurable
  `model_loading_timeout_seconds` load timeout) and its UI surfacing,
  including the fixed reasoning-model TTFT defect
  ([[openai-driver-ttft-ignores-reasoning-content]])
- The 10-code structured error taxonomy, its remediation steps, and the
  `RunFailureBanner.vue` / `RunDetailModal.vue` / `AgentsRunsView.vue`
  surfacing
- An example local-provider configuration (mirroring this project's own
  `qa` agent) and the benchmark-verified model/quantization
  recommendations from [[open-provider-support-2]]
- A troubleshooting section covering each failure mode

Feature record added: `lifecycle/features/local-llm-operability.md` (`function:
Agents`), replacing the Ollama-specific scope of
`lifecycle/features/ollama-local-llms.md` per FR-6 — that file is left in
place as a historical/superseded marker (already pointing to
[[provider-management]]) with an added forward link to the new feature.

Also updated `docs/open-provider-support.md`: promoted local-model
operability from "in progress" to a third shipped workstream, with the
"Related and in-progress work" entry marked **Done** and linked to the new
doc/feature.
