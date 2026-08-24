---
created: "2026-08-25T08:50:39+10:00"
title: "OpenAI-Compatible Agent Driver & Provider Support — Test Suite"
type: test
status: in-qa
lineage: open-provider-support
parent: lifecycle/test-plans/open-provider-support-5-test.md
---

# OpenAI-Compatible Agent Driver & Provider Support — Test Suite

## Overview

Comprehensive integration test suite covering the OpenAI-compatible agent driver, provider configuration, REST APIs, preflight capability verification, multi-turn tool calling, sandboxing and security enforcement, live target verification, and regression coverage for existing drivers.

## Test Files

| File | Package | Milestone / Area |
|------|---------|------------------|
| `tests/integration/testutil/openai_mock.go` | `testutil` | M1 — Mock OpenAI Server Infrastructure |
| `tests/integration/provider_config_test.go` | `integration` | M2 — Provider & Agent Configuration |
| `tests/integration/provider_api_test.go` | `integration` | M3 — Provider REST API Endpoints |
| `tests/integration/openai_driver_test.go` | `integration` | M4 — OpenAI Driver & Sandboxing |
| `tests/integration/openai_agent_run_test.go` | `integration` | M5 — Agent Runner E2E Lifecycle |
| `tests/integration/openai_regression_test.go` | `integration` | M6 — Driver Regression & Secret Masking |
| `tests/integration/openai_driver_live_test.go` | `integration` | M4 — Live Target Verification |

---

## Milestone 1 — Mock OpenAI Server (`testutil/openai_mock.go`)

`MockOpenAIServer` is a configurable `httptest.Server` simulating OpenAI-compatible inference backends (llama.cpp, vLLM, OpenRouter, Ollama). It provides:

- **Model Catalog Proxying** (`GET /v1/models`): returns standard model objects with `supported_parameters` (e.g. `tools`).
- **Configurable Preflight Capability Modes**:
  - `mode-c` (Token Delta): returns token differentials (25 tokens with tools vs 5 without tools) to verify tool support.
  - `mode-a` (Silent Drop): returns identical token counts to simulate backend tool ignorance.
  - `mode-b` (Explicit Rejection): returns HTTP 400 with `<model> does not support tools`.
- **Multi-Turn Tool Simulation**: supports `ScriptedTurns` specifying tool calls (`read_file`, `write_file`, `list_dir`) and final completions via SSE streaming (`text/event-stream`).
- **Native Syntax Emulation**: emits text with XML tags (`<function=...>`) and JSON tags (`<tool_call>...`) for recovery parser validation.
- **Latency & Error Injection**: allows custom stream delays, context cancellation verification, and error code injection.
- **Header & Auth Verification**: validates `Authorization: Bearer <key>` and custom `extra_headers`.
- **Request Inspection**: records incoming requests (`Requests()`, `LastRequest()`, `RequestsForPath()`).

---

## Milestone 2 — Provider & Agent Config (`provider_config_test.go`)

Covers app-level `providers` YAML and project-level agent configuration:

- **TestProviderConfig_LoadWithProviders** — app config YAML with multiple providers parses all fields (`name`, `base_url`, `driver`, `api_key`, `extra_headers`).
- **TestProviderConfig_RoundTrip** — Load → SaveApp → Load round-trip preserves all provider records without data loss.
- **TestProviderConfig_DuplicateNameRejected** — duplicate provider names trigger validation error.
- **TestProviderConfig_EmptyBaseURLRejected** — missing `base_url` is rejected with validation error.
- **TestProviderConfig_InvalidURLRejected** — unsupported URL schemes are rejected.
- **TestProviderConfig_EmptyDriverRejected** — missing `driver` field is rejected.
- **TestProviderConfig_NoProvidersKey** — app config without `providers` defaults to empty slice without errors.
- **TestProviderConfig_LegacyOllamaInstancesMigration** — legacy `ollama_instances` YAML keys migrate automatically into `Provider` records.
- **TestProviderConfig_AgentWithProvider** — `driver: openai-compatible` agent with `provider`, `model`, and `max_tool_iterations` loads and validates cleanly.
- **TestProviderConfig_AgentMissingProviderRejected** — `driver: openai-compatible` agent without `provider` is rejected.
- **TestProviderConfig_AgentMissingModelRejected** — `driver: openai-compatible` agent without `model` is rejected.

---

## Milestone 3 — Provider REST API (`provider_api_test.go`)

Covers the `/api/providers` HTTP API with authentication and project integration:

- **TestProviderAPI_List** — `GET /api/providers` returns providers with `api_key: "***"` and `has_api_key: true` when keyed, omitting raw secrets.
- **TestProviderAPI_Create** — `POST /api/providers` creates a new provider (201 Created) and saves to disk.
- **TestProviderAPI_CreateDuplicate** — `POST` with an existing name returns 409 Conflict.
- **TestProviderAPI_Update** — `PUT /api/providers/{name}` updates provider configuration while preserving existing API keys.
- **TestProviderAPI_Delete** — `DELETE /api/providers/{name}` deletes unreferenced providers.
- **TestProviderAPI_DeleteReferenced** — `DELETE` returns 409 Conflict when a provider is in use by an agent in any project.
- **TestProviderAPI_TestProbe** — `POST /api/providers/test` probes provider connectivity and tool calling capability.
- **TestProviderAPI_ModelsList** — `GET /api/providers/{name}/models` proxies model discovery to the underlying backend.

---

## Milestone 4 — OpenAI Driver & Security Sandboxing (`openai_driver_test.go` & `openai_driver_live_test.go`)

Direct driver-level testing of `OpenAICompatibleDriver` executing the multi-turn tool loop and enforcing sandbox boundaries:

- **TestOpenAIDriver_SingleRoundTrip** — executes `read_file` tool call and completes with final synthesized response.
- **TestOpenAIDriver_MultiStepExecution** — executes multi-turn tool chain (`read_file` followed by `write_file`) and verifies disk modification.
- **TestOpenAIDriver_RefusedWriteRecovery** — writes outside `allowed_write_paths` are rejected by `ToolExecutor` and fed back to the model as errors; verifies no unauthorized files are created.
- **TestOpenAIDriver_PathTraversalRefusal** — attempts with `../` relative path traversal are blocked by `sandbox.Resolver`.
- **TestOpenAIDriver_MaxIterationsCap** — loops exceeding `max_tool_iterations` terminate gracefully with an iteration cap error.
- **TestOpenAIDriver_StreamingAndTTFT** — SSE chunks stream progress events, and `OnTTFT` callback measures Time-to-First-Token accurately.
- **TestOpenAIDriver_KillCancellation** — `proc.Kill()` terminates in-flight streaming requests promptly (< 2 s).
- **TestOpenAIDriver_RunLogFormatAndMasking** — run log file records headers, prompts, turn tool calls, results, and masks API keys with `***`.
- **TestOpenAIDriver_NativeXMLRecovery** — recovers XML tool calls emitted in plaintext by models without native tool support.
- **TestOpenAIDriver_NativeJSONRecovery** — recovers JSON `<tool_call>` tag structures.
- **TestOpenAIDriver_LiveTarget_LlamaServer** — live target verification against llama.cpp (`leia.packsin.com:7442`) with `gemma-4-26B-A4B-it-UD-Q8_K_XL` and `gpt-oss-20b-Q8_0` (gracefully skipped if host is unreachable).
- **TestOpenAIDriver_LiveTarget_Ollama** — live target verification against Ollama (`leia.packsin.com:11434`) with `qwen3-coder:30b` and `gemma4:26b` (gracefully skipped if host is unreachable).

---

## Milestone 5 — Agent Runner E2E Lifecycle (`openai_agent_run_test.go`)

End-to-end server integration tests through HTTP endpoints and background supervisors:

- **TestOpenAIAgentRun_DriverSelection** — verifies OpenAI-compatible agents route to `OpenAICompatibleDriver`.
- **TestOpenAIAgentRun_Completes** — full execution lifecycle from 202 Accepted through background execution to `status=done`.
- **TestOpenAIAgentRun_PreflightFailureHardFails** — Mode A silent drop fails preflight, aborting the run immediately without modifying disk artifacts.
- **TestOpenAIAgentRun_ExplicitRejectionFails** — Mode B explicit HTTP 400 fails preflight and reports backend error message verbatim.
- **TestOpenAIAgentRun_ConcurrencySemaphore** — project-level agent concurrency limit (`max_concurrent_agents`) blocks excess runs with HTTP 503.
- **TestOpenAIAgentRun_Kill** — `POST /api/p/{project}/agents/runs/{id}/kill` terminates active run and marks record `status=killed`.
- **TestOpenAIAgentRun_StatusLifecycle** — agents configured with `active_status: in-development` and `done_on_success: true` transition target artifacts through the workflow state machine and SQLite index.
- **TestOpenAIAgentRun_HubEvents** — verifies real-time WebSocket broadcast of `agent.started`, `agent.progress`, and `agent.finished` events.

---

## Milestone 6 — Regression & Security Standards (`openai_regression_test.go`)

Guarantees non-interference with existing drivers and adherence to architecture standards:

- **TestOpenAIRegression_ExistingDriversWork** — ensures `claude-code-cli`, `claude-mediated`, `codex-cli`, `gemini`, `gemini-cli`, `shell-stub`, and legacy `ollama` configurations continue to load and validate without regression (NFR-2).
- **TestOpenAIRegression_MixedAgentsConfig** — validates projects mixing Claude, OpenAI-compatible, and shell-stub agents concurrently.
- **TestOpenAIRegression_SecretMasking** — verifies raw API keys never leak through agent or provider API responses (NFR-1).
