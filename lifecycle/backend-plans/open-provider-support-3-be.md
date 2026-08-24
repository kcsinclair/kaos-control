---
title: "Backend Plan: OpenAI-Compatible Agent Driver (Tool-Calling)"
type: plan-backend
status: approved
lineage: open-provider-support
parent: lifecycle/requirements/open-provider-support-2.md
created: "2026-08-25T07:10:33+10:00"
---

# Backend Plan: OpenAI-Compatible Agent Driver (Tool-Calling)

Parent: [[open-provider-support-2]].

This plan covers the Go backend implementation for FR-1 through FR-8 and NFR-1
through NFR-4 of the [[llama-cpp-driver]] requirement (Workstream 1 of the
[[open-provider-support]] epic). Frontend changes are defined in
[[open-provider-support-4-fe]] and test coverage in [[open-provider-support-5-test]].

---

## Milestone 1 — Provider & AgentConfig Data Model & Validation

### Description

Extend the app-level and project-level configuration to support first-class
`Provider` records and update `AgentConfig` to reference a provider by name.
Implement validation and secret masking to satisfy FR-2, FR-3, and NFR-1.

### Files to change

- **`internal/config/config.go`**:
  - Add `Provider` struct:
    ```go
    type Provider struct {
        Name         string            `yaml:"name"`
        BaseURL      string            `yaml:"base_url"`
        Driver       string            `yaml:"driver"` // "openai-compatible"
        APIKey       string            `yaml:"api_key,omitempty"`
        ExtraHeaders map[string]string `yaml:"extra_headers,omitempty"`
    }
    ```
  - Update `App` struct to include `Providers []Provider` (replacing/generalising `OllamaInstances`).
  - Update `AgentConfig` struct:
    - Add `Provider string` (`yaml:"provider,omitempty"`)
    - Add `Model string` (`yaml:"model,omitempty"`)
    - Add `MaxToolIterations int` (`yaml:"max_tool_iterations,omitempty"`)
  - Update `validateApp(cfg *App)`:
    - Enforce unique, non-empty `Provider.Name`.
    - Enforce valid HTTP/HTTPS `BaseURL`.
    - Enforce non-empty `Driver`.
  - Update `validateProject(cfg *Project)`:
    - For agents using `driver: openai-compatible` (or when `provider` is set), enforce that `model` is specified.
  - Maintain backward compatibility for existing `ollama_instances` by migrating them in-memory to equivalent Provider records during unmarshal.

- **`internal/config/config_test.go`**:
  - Add test cases verifying validation of valid and malformed `Provider` definitions.
  - Test validation failures for missing `provider` or `model` on `openai-compatible` agents.

### Acceptance criteria

- [ ] `Provider` struct and `AgentConfig` extensions are implemented with accurate YAML tags.
- [ ] `validateApp` rejects empty provider names, duplicate provider names, and malformed `base_url` strings.
- [ ] `validateProject` validates `openai-compatible` agent configurations.
- [ ] Legacy `ollama_instances` entries are seamlessly mapped to `Provider` records.
- [ ] `go test ./internal/config/...` passes.

---

## Milestone 2 — Sandboxed Local Tool Execution Engine

### Description

Implement the driver's local tool executor. The driver exposes OpenAI function-calling
tool definitions for creating and editing lifecycle artifacts (`read_file`, `write_file`,
`list_dir`, `grep`), executes them locally, and scopes all operations via `internal/sandbox/`
against `allowed_write_paths` (FR-5, Resolved Questions 1 & 2, NFR-4).

### Files to change

- **`internal/agent/openai_tools.go`** (new file):
  - Define standard OpenAI tool schema specifications (`read_file`, `write_file`, `list_dir`, `grep`).
  - Implement `ToolExecutor`:
    ```go
    type ToolExecutor struct {
        ProjectRoot  string
        AllowedPaths []string
    }

    func (e *ToolExecutor) Execute(ctx context.Context, name string, argsJSON string) (string, error)
    ```
  - Implement tool handlers:
    - `read_file(path)`: resolves path via `sandbox.Resolve(e.ProjectRoot, path)` and returns file contents.
    - `write_file(path, content)`: validates path against `AllowedPaths` using `sandbox.Resolve`; if out of scope, returns a clear error message string (e.g. `"permission denied: path is outside allowed_write_paths"`) to be fed back to the model without crashing the run.
    - `list_dir(path)`: lists directories/files within project root.
    - `grep(pattern, path)`: searches text within sandboxed directory/file.
  - Prohibit shell execution (`bash` tool is omitted in v1 per Resolved Question 2).

- **`internal/agent/openai_tools_test.go`** (new file):
  - Unit tests for all tool operations.
  - Verify path traversal attempts (`../`) are rejected with `sandbox.ErrPathTraversal`.
  - Verify writes outside `AllowedPaths` return tool errors without altering files.

### Acceptance criteria

- [ ] OpenAI tool schema definitions are valid function-calling specifications.
- [ ] `read_file`, `write_file`, `list_dir`, and `grep` operate strictly within `ProjectRoot`.
- [ ] Out-of-scope writes are refused with descriptive tool error feedback and do not mutate disk.
- [ ] `go test ./internal/agent/ -run TestOpenAITools` passes.

---

## Milestone 3 — Native-Format Tool-Call Recovery Parser

### Description

Implement the fallback parser for models that emit tool calls in native text formats
rather than structured JSON `tool_calls` (FR-5a). Recovered calls are normalised to
the OpenAI `tool_calls` structure, executed identically, and counted/logged distinctly.

### Files to change

- **`internal/agent/openai_recovery.go`** (new file):
  - Implement regex-based and token-based fallback parsers:
    - XML/Tag format: `<function=NAME><parameter=KEY>VALUE</parameter></function>`
    - JSON Tag format: `<tool_call>{"name": "...", "arguments": {...}}</tool_call>`
  - Implement `ParseNativeCalls(content string) (recovered []ToolCall, remainingContent string)`
  - Provide helper to convert parsed calls into OpenAI `tool_calls` JSON payloads.

- **`internal/agent/openai_recovery_test.go`** (new file):
  - Unit tests covering Qwen-style XML function calls, raw `<tool_call>` blocks, mixed text, and plain text without tool calls.
  - Verify recovered calls parse tool name and arguments accurately.

### Acceptance criteria

- [ ] XML-style `<function=...>` and JSON-style `<tool_call>` blocks are accurately parsed into `tool_calls`.
- [ ] Plain assistant messages without tool call syntax are untouched.
- [ ] Recovered calls are flagged for distinct logging and metrics.
- [ ] `go test ./internal/agent/ -run TestOpenAIRecovery` passes.

---

## Milestone 4 — Preflight Capability Verification & Token Delta Probing

### Description

Implement preflight capability verification to detect models that silently ignore
`tools` parameters or explicitly reject them (FR-5b). Perform token-delta comparison
and check provider metadata (e.g. OpenRouter `supported_parameters`).

### Files to change

- **`internal/agent/openai_preflight.go`** (new file):
  - Implement `VerifyToolCapability(ctx context.Context, client *http.Client, provider config.Provider, model string, tools []OpenAITool) error`
  - Mode A check: issue a lightweight test completion with and without `tools`. Compare `usage.prompt_tokens`. If `prompt_tokens` is unchanged, return `ErrToolsSilentlyDropped` to trigger an immediate hard-fail.
  - Mode B check: handle explicit HTTP 400 responses (e.g. Ollama `"does not support tools"`) and return clear non-retryable error.
  - Gateway discovery: if provider is OpenRouter, query `/v1/models` and verify `tools` in `supported_parameters`.

- **`internal/agent/openai_preflight_test.go`** (new file):
  - Unit tests using mock HTTP handlers simulating Mode A (silent drop), Mode B (HTTP 400), Mode C (valid token delta), and OpenRouter models metadata.

### Acceptance criteria

- [ ] Silent tool drops (Mode A) are detected via prompt token deltas and result in immediate hard failure.
- [ ] Explicit HTTP 400 errors (Mode B) surface the server error message directly.
- [ ] Gateway capability metadata is checked when available.
- [ ] No artifacts or files are modified when preflight fails.
- [ ] `go test ./internal/agent/ -run TestOpenAIPreflight` passes.

---

## Milestone 5 — OpenAI-Compatible Driver Core & Multi-turn Loop

### Description

Implement `OpenAICompatibleDriver` satisfying `agent.Driver` and `agent.Process`.
The driver manages SSE streaming deltas, TTFT measurement, multi-turn tool-calling
loops, timeout handling, iteration caps (default 25), and run logging (FR-1, FR-4, FR-5, FR-6, FR-7, FR-8).

### Files to change

- **`internal/agent/openai_compatible.go`** (new file):
  - Define `OpenAICompatibleDriver` implementing `Driver`:
    ```go
    type OpenAICompatibleDriver struct {
        Providers  []config.Provider
        HTTPClient *http.Client
    }
    ```
  - Define `openAIProcess` implementing `Process` (`Wait`, `Kill`, `Progress`, `StderrTail`).
  - Implement `Start(ctx context.Context, run Run) (Process, error)`:
    1. Resolve `Provider` by name and validate `Model`.
    2. Run preflight capability check (`openai_preflight.go`).
    3. Split system and user prompts (`splitPrompt`).
    4. Build initial messages array and tools schema array.
    5. Launch worker goroutine executing the bounded multi-turn loop:
       - POST to `<base_url>/v1/chat/completions` with `stream: true`, `model`, `messages`, `tools`.
       - Stream SSE chunks (`data: {...}`).
       - Accumulate content tokens and parse `tool_calls` chunks.
       - Fire `Run.OnTTFT` on the first streamed content token.
       - Check for native tool-call recovery (FR-5a) if `tool_calls` is empty.
       - If tool calls exist: execute each via `ToolExecutor`, append assistant message and `tool` results to `messages`, and repeat.
       - If `finish_reason: stop` (and no tool calls): emit completed event and exit loop.
       - If turn count exceeds `max_tool_iterations` (default 25): abort with terminal error.
    6. Write per-run log file (`run.LogPath`) matching `ollama` driver format with header, turns, tool calls, results, recovered call counts, and summary footer.
    7. Mask `api_key` in all logs, events, and error outputs (NFR-1).

- **`internal/agent/openai_compatible_test.go`** (new file):
  - Integration tests with `httptest.Server` testing single-turn tool calls, multi-turn tool calls, out-of-scope write recoveries, max iteration caps, and streaming progress events.

### Acceptance criteria

- [ ] `OpenAICompatibleDriver` fulfills the `Driver` interface and streams `ProgressEvent`s.
- [ ] TTFT is recorded on the first assistant token.
- [ ] Multi-turn tool execution loop runs until `finish_reason: stop` or iteration cap is reached.
- [ ] Bounded iteration cap terminates runaway loops cleanly.
- [ ] Run log file is structured with complete headers, turns, tool outputs, and footers.
- [ ] `Process.Kill()` and context cancellations terminate in-flight requests promptly.
- [ ] `api_key` is masked in logs and progress events.

---

## Milestone 6 — Manager Integration & Driver Registration

### Description

Register `"openai-compatible"` in `agent.Manager`, wire provider resolution from
`App` config, and sunset the native single-shot `ollama` driver by routing
Ollama instances through `openai-compatible` (FR-1, Scope).

### Files to change

- **`internal/agent/agent.go`**:
  - Update `New(...)` signature or options to accept `[]config.Provider`.
  - Register `"openai-compatible"` in `m.drivers`.
  - In `StartRun`:
    - Resolve agent's provider from `AgentConfig.Provider`.
    - Populate `Run` with provider details, `Model`, `MaxToolIterations`, and `AllowedPaths`.
    - Wire `OnTTFT` callback.
  - Deprecate/remove the legacy single-shot `OllamaDriver` registration, replacing it with the `openai-compatible` driver.

- **`internal/agent/agent_test.go`**:
  - Verify `openai-compatible` driver is registered and instantiated properly.
  - Verify `StartRun` correctly resolves provider config and starts runs.

### Acceptance criteria

- [ ] `"openai-compatible"` driver is registered and selectable via `AgentConfig.Driver`.
- [ ] `StartRun` executes `openai-compatible` agent runs with proper lineage locking and semaphore checks.
- [ ] Concurrency semaphore (`max_concurrent_agents`) is respected.
- [ ] `go test ./internal/agent/...` passes with all existing and new tests green.

---

## Milestone 7 — Provider Management REST API Endpoints

### Description

Expose REST endpoints for managing app-level `Provider` configurations and testing
provider connectivity and models (replacing `/api/ollama/instances`).

### Files to change

- **`internal/http/providers.go`** (new file):
  - `GET /api/providers`: list configured providers (with `api_key` masked as `"***"`).
  - `POST /api/providers`: create a new provider.
  - `PUT /api/providers/{name}`: update provider.
  - `DELETE /api/providers/{name}`: delete provider.
  - `POST /api/providers/{name}/test`: probe provider connectivity, test `/v1/models`, and return available models and tool support capabilities.

- **`internal/http/server.go`**:
  - Register provider routes under `/api/providers`.
  - Remove deprecated `/api/ollama/instances` routes.

- **`internal/http/providers_test.go`** (new file):
  - Test provider CRUD operations, input validations, secret masking, and probe endpoints.

### Acceptance criteria

- [ ] `/api/providers` endpoints perform CRUD on configured providers.
- [ ] `api_key` is never returned in plaintext in API responses.
- [ ] Provider test endpoint probes `/v1/models` and returns capability info.
- [ ] `go test ./internal/http/...` passes.
