---
title: "Test Plan: OpenAI-Compatible Agent Driver (Tool-Calling)"
type: plan-test
status: approved
lineage: open-provider-support
parent: lifecycle/requirements/open-provider-support-2.md
created: "2026-08-25T07:10:33+10:00"
---

# Test Plan: OpenAI-Compatible Agent Driver (Tool-Calling)

Parent: [[open-provider-support-2]].

This plan defines unit tests, integration tests, and verification targets for
all acceptance criteria and non-functional requirements of the
[[llama-cpp-driver]] requirement. Backend implementation is in
[[open-provider-support-3-be]] and frontend implementation is in
[[open-provider-support-4-fe]].

---

## Milestone 1 — Tool Executor, Native Recovery & Preflight Unit Tests

### Description

Implement unit tests for the sandboxed tool executor, native-format tool-call
recovery parser, and preflight capability verification. These tests are pure Go
logic without external network dependencies.

### Files to change

- **`internal/agent/openai_tools_test.go`** (new file):
  - `TestToolExecutor_ReadFile`: sandboxed file reading within project root.
  - `TestToolExecutor_WriteFile_Allowed`: writing to files matching `AllowedPaths`.
  - `TestToolExecutor_WriteFile_Disallowed`: attempting to write outside `AllowedPaths` returns a tool error string without altering disk.
  - `TestToolExecutor_PathTraversal`: attempting relative paths (`../`) or absolute escapes returns path traversal errors.
  - `TestToolExecutor_ListDir`: directory listing within root.
  - `TestToolExecutor_Grep`: searching regex/plain patterns in project files.

- **`internal/agent/openai_recovery_test.go`** (new file):
  - `TestParseNativeCalls_XML`: parses `<function=write_file><parameter=path>foo.md</parameter><parameter=content>bar</parameter></function>`.
  - `TestParseNativeCalls_JSONTag`: parses `<tool_call>{"name":"read_file","arguments":{"path":"foo.md"}}</tool_call>`.
  - `TestParseNativeCalls_PlainText`: plain assistant responses with no tool calls return zero recovered calls and unaltered content.
  - `TestParseNativeCalls_Malformed`: malformed tags degrade gracefully to plain text without crashing.

- **`internal/agent/openai_preflight_test.go`** (new file):
  - `TestPreflight_SilentDrop_ModeA`: mock server returning identical `prompt_tokens` with/without tools triggers `ErrToolsSilentlyDropped` and aborts.
  - `TestPreflight_ExplicitReject_ModeB`: mock server returning HTTP 400 `"does not support tools"` surfaces error verbatim.
  - `TestPreflight_CleanDelta_ModeC`: mock server returning positive token delta passes preflight.
  - `TestPreflight_OpenRouterDiscovery`: verifies `supported_parameters` containing `tools` allows execution.

### Acceptance criteria

- [ ] All tool operations are thoroughly tested with positive and boundary cases.
- [ ] Path traversal and write scoping enforcement is 100% covered.
- [ ] Native tool-call formats parse cleanly into standard tool calls.
- [ ] Preflight failure modes (Mode A silent drop, Mode B 400 reject) are verified.
- [ ] `go test ./internal/agent/ -run "TestToolExecutor|TestParseNativeCalls|TestPreflight"` passes.

---

## Milestone 2 — Mock HTTP Server Integration Tests for Driver Loop

### Description

Implement end-to-end driver integration tests using `httptest.Server` to
simulate an OpenAI-compatible endpoint with SSE streaming deltas and multi-turn
tool-calling loops.

### Files to change

- **`internal/agent/openai_compatible_test.go`** (new file):
  - `TestOpenAIDriver_SingleRoundTrip`:
    - Mock server returns a `tool_calls` delta for `read_file`.
    - Driver executes `read_file` and sends back the result in the next turn.
    - Mock server responds with `finish_reason: stop` and final text.
    - Asserts run completes with `status: done`.
  - `TestOpenAIDriver_MultiStepExecution`:
    - Mock server drives sequential `read_file` followed by `write_file`.
    - Asserts both tools execute, file is created, and run completes.
  - `TestOpenAIDriver_RefusedWriteRecovery`:
    - Mock server requests write to a path outside `allowed_write_paths`.
    - Asserts driver returns error message to model, model concludes with explanation, and no unauthorised file is created.
  - `TestOpenAIDriver_MaxIterationsCap`:
    - Mock server continuously issues tool calls without stopping.
    - Asserts driver terminates at `max_tool_iterations` (default 25) with a clear error.
  - `TestOpenAIDriver_StreamingAndTTFT`:
    - Asserts `ProgressEvent`s stream during execution and `OnTTFT` callback is invoked on first content token.
  - `TestOpenAIDriver_KillAndTimeout`:
    - Asserts calling `proc.Kill()` or expiring timeout terminates HTTP connection and marks run failed.
  - `TestOpenAIDriver_RunLogFormat`:
    - Asserts generated log file contains header, system/user prompts, turn-by-turn tool calls/results, and summary footer.

### Acceptance criteria

- [ ] Single-turn and multi-turn tool loops pass end-to-end against mock servers.
- [ ] Refused write attempts are handled gracefully without run crashes.
- [ ] Iteration caps protect against infinite loops.
- [ ] TTFT and streaming progress events are recorded accurately.
- [ ] `Kill()` and timeout cancellations terminate cleanly.
- [ ] `go test ./internal/agent/ -run TestOpenAIDriver` passes.

---

## Milestone 3 — Configuration Validation & Provider API Tests

### Description

Verify configuration parsing, validation rules, secret masking, and REST API
endpoints for provider management.

### Files to change

- **`internal/config/config_test.go`**:
  - `TestConfig_ProviderValidation`: tests unique names, valid URLs, missing drivers.
  - `TestConfig_AgentProviderBinding`: tests agent validation referencing valid and invalid provider names.
  - `TestConfig_LegacyOllamaMigration`: verifies `ollama_instances` parses into `Provider` records.

- **`internal/http/providers_test.go`** (new file):
  - `TestAPI_ProvidersCRUD`: creates, lists, updates, and deletes providers via REST endpoints.
  - `TestAPI_ProviderSecretMasking`: verifies `api_key` is masked in `GET /api/providers` and never exposed.
  - `TestAPI_ProviderProbe`: tests `/api/providers/{name}/test` endpoint against a mock server.

### Acceptance criteria

- [ ] Config loader accepts valid provider configs and rejects invalid entries with clear error messages.
- [ ] Provider API endpoints enforce authentication and input validation.
- [ ] API keys are masked in all JSON responses.
- [ ] `go test ./internal/config/... ./internal/http/...` passes.

---

## Milestone 4 — Live Target Verification & Benchmark Documentation

### Description

Verify driver performance and tool compatibility against live targets (llama.cpp
and Ollama) and document recommended flags and model baselines in
`~/Code/agent-benchmark` (FR-5b, Acceptance Criteria).

### Files to change / test targets

- **`tests/integration/openai_driver_live_test.go`** (tagged integration test):
  - Live target tests (executed with `-tags=integration` against live servers):
    - llama.cpp (`leia.packsin.com:7442`):
      - Model: `gemma-4-26B-A4B-it-UD-Q8_K_XL` (verified baseline).
      - Model: `gpt-oss-20b-Q8_0` (verified alternative).
    - Ollama (`leia.packsin.com:11434`):
      - Model: `qwen3-coder:30b`.
      - Model: `gemma4:26b`.
  - Asserts full two-turn round trips execute in < 15 seconds with valid artifact creation.

- **`~/Code/agent-benchmark` documentation**:
  - Document required `llama-server` flags (`--jinja` for tool templates, `--api-key` if configured, `--ctx-size 8192+`).
  - Document model recommendations and quantization impact (e.g. Q8 vs Q4 tool fidelity).

### Acceptance criteria

- [ ] Live integration test confirms tool round-trip against llama.cpp baseline (`gemma-4-26B-A4B-it-UD-Q8_K_XL`).
- [ ] Live integration test confirms tool round-trip against Ollama baseline (`qwen3-coder:30b`).
- [ ] Setup and flag requirements are documented in the benchmark repo.

---

## Milestone 5 — Frontend Store & Component Tests

### Description

Implement frontend tests for provider store actions, the Provider Settings view,
and agent configuration forms.

### Files to change

- **`web/src/stores/__tests__/providers.test.ts`** (new file):
  - Tests `fetchProviders`, `saveProvider`, `deleteProvider`, and `probeProvider`.

- **`web/src/views/project/__tests__/ProviderSettingsView.spec.ts`** (new file):
  - Tests rendering provider list, opening add/edit modals, secret field masking, and triggering connection probes.

- **`web/src/components/agent/__tests__/AgentConfigForm.spec.ts`**:
  - Tests driver dropdown selection (`openai-compatible`), dynamic provider selection, and model autocomplete.

### Acceptance criteria

- [ ] Pinia provider store tests pass.
- [ ] ProviderSettingsView renders providers and responds to user actions.
- [ ] `cd web && pnpm test` passes.

---

## Milestone 6 — Regression & Backwards Compatibility Verification

### Description

Ensure that introducing `openai-compatible` and provider records causes zero
regressions for existing drivers and respects global concurrency limits (NFR-2, NFR-3).

### Files to change

- **`internal/agent/agent_test.go`**:
  - Run the full driver test suite covering `claude-code-cli`, `claude-mediated`, `claude-env`, `gemini`, `codex-cli`, and `shell-stub`.
  - Verify global concurrency semaphore (`max_concurrent_agents`) functions across mixed concurrent driver runs.
  - Verify error handling for non-2xx responses and malformed payloads (NFR-3).

### Acceptance criteria

- [ ] All existing driver tests pass without alteration (NFR-2).
- [ ] Global semaphore correctly throttles concurrent runs.
- [ ] No secret material appears in test logs or stderr tails.
- [ ] `make test-unit` passes cleanly.
