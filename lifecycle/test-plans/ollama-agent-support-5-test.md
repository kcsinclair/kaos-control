---
title: "Ollama Agent Support — Test Plan"
type: plan-test
status: done
lineage: ollama-agent-support
parent: lifecycle/requirements/ollama-agent-support-2.md
---

# Ollama Agent Support — Test Plan

## Overview

Integration and unit tests for Ollama agent support covering config, API endpoints, driver behaviour, and end-to-end agent runs. Tests use a mock Ollama HTTP server (httptest) to avoid requiring a running Ollama instance in CI.

Cross-references: [[ollama-agent-support]] backend plan for implementation details, [[ollama-agent-support]] frontend plan for UI integration.

---

## Milestone 1 — Mock Ollama Server

### Description

Build a reusable `httptest.Server` that mimics the Ollama REST API. This server is used across all subsequent test milestones and provides deterministic responses for `/api/tags`, `/api/chat`, `/api/generate`, and the root health endpoint.

### Files to change

- `tests/integration/testutil/ollama_mock.go` — New file. Helper to start a mock Ollama server with configurable responses.

### Mock behaviour

| Endpoint | Response |
|----------|----------|
| `GET /api/tags` | JSON with configurable model list (default: `[{"name":"testmodel:latest","size":1000000}]`) |
| `POST /api/chat` | NDJSON stream: `{"message":{"content":"chunk1"}}`, `{"message":{"content":"chunk2"}}`, `{"done":true}` |
| `POST /api/generate` | NDJSON stream: `{"response":"chunk1"}`, `{"response":"chunk2"}`, `{"done":true}` |
| `GET /` | `200 OK` (health) |

Supports:
- Configurable latency per endpoint (for timeout tests).
- Configurable error responses (HTTP 500, connection refused simulation).
- Request recording for assertion (verify sent prompts, model, system prompt).
- Optional `Authorization` header validation.

### Acceptance criteria

- [ ] Mock server starts on a random port and returns the base URL.
- [ ] Responses are configurable (model list, chat output, latency, errors).
- [ ] Recorded requests are accessible for assertion (method, path, body, headers).
- [ ] Server properly handles streaming (NDJSON lines with flush).
- [ ] Cleanup (`Close()`) is safe to call multiple times.

---

## Milestone 2 — App Config Tests (Instance CRUD)

### Description

Test that Ollama instance configuration loads, saves, validates, and round-trips correctly through the app config system.

### Files to change

- `tests/integration/ollama_config_test.go` — New file. Tests for instance config operations.

### Test cases

1. **Load config with ollama_instances** — YAML with valid instances parses correctly.
2. **Round-trip** — Load → Save → Load produces identical config.
3. **Duplicate instance name rejected** — Config with two instances sharing a name returns a validation error.
4. **Empty base_url rejected** — Instance with missing/empty `base_url` returns a validation error.
5. **API key masking** — API response masks `api_key` as `"***"` when set; omits field when not set.
6. **Agent config with driver=ollama** — `AgentConfig` with `ollama_instance` and `model` parses correctly.
7. **Agent config validation** — `driver: ollama` without `ollama_instance` fails validation.

### Acceptance criteria

- [ ] All 7 test cases pass.
- [ ] Tests use temporary config files (no side effects on real config).
- [ ] Validation error messages are specific (mention the field and constraint).

---

## Milestone 3 — Ollama API Endpoint Tests

### Description

Test the HTTP endpoints for instance management, health checks, and model discovery against the mock Ollama server.

### Files to change

- `tests/integration/ollama_api_test.go` — New file. HTTP-level tests for Ollama endpoints.

### Test cases

1. **List instances** — `GET /api/ollama/instances` returns configured instances with masked keys.
2. **Create instance** — `POST /api/ollama/instances` adds a new instance; re-fetch confirms persistence.
3. **Create duplicate** — `POST` with an existing name returns 409.
4. **Update instance** — `PUT /api/ollama/instances/{name}` updates base_url.
5. **Delete instance** — `DELETE` removes the instance; re-fetch confirms removal.
6. **Delete referenced instance** — `DELETE` when an agent references the instance returns 409.
7. **Health check — healthy** — `GET /api/ollama/instances/{name}/health` returns `ok: true` with latency.
8. **Health check — unreachable** — Mock server down → returns `ok: false` with error message.
9. **Health check — timeout** — Mock server delays beyond 10s → returns `ok: false`.
10. **List models** — `GET /api/ollama/instances/{name}/models` returns model names and sizes.
11. **List models — instance not found** — Unknown instance name → 404.
12. **Auth header forwarded** — When `api_key` is set, proxied requests include `Authorization: Bearer <key>`.

### Acceptance criteria

- [ ] All 12 test cases pass.
- [ ] Tests start a real HTTP server (using `httptest` or the app's own test harness) and the mock Ollama server.
- [ ] No test depends on external network access.
- [ ] Timeout test uses mock latency, not real sleep (keeps test fast).

---

## Milestone 4 — OllamaDriver Unit Tests

### Description

Test the `OllamaDriver` in isolation, verifying it correctly implements the `Driver` interface, emits expected events, and handles error conditions.

### Files to change

- `tests/integration/ollama_driver_test.go` — New file. Driver-level tests.

### Test cases

1. **Successful chat run** — Driver emits `started`, output chunks, `completed` events in order. Full response text is concatenated correctly.
2. **Successful generate run** — Same as above but via `/api/generate` endpoint.
3. **System prompt separation** — Prompt with `---SYSTEM---`/`---USER---` delimiter sends system and user messages separately. Verify via mock request recording.
4. **No system prompt** — Prompt without delimiter sends entire text as user message.
5. **Process.Wait() blocks** — `Wait()` blocks until the response stream completes.
6. **Process.Kill() cancels** — Calling `Kill()` mid-stream cancels the HTTP request; `Wait()` returns a context error.
7. **HTTP error** — Ollama returns HTTP 500 → driver emits an `error` event with details in `StderrTail()`.
8. **Connection refused** — Instance unreachable → driver fails fast with a clear error.
9. **Timeout** — Mock delays beyond configured timeout → driver emits error event.
10. **Model field forwarded** — The `model` field in the request body matches `Run.Model`.

### Acceptance criteria

- [ ] All 10 test cases pass.
- [ ] Events are verified in order (channel reads with test timeout).
- [ ] Mock request recording verifies the exact JSON body sent to Ollama.
- [ ] Kill/cancel tests complete in under 2 seconds (no real timeouts).
- [ ] `StderrTail()` contains actionable error text for failure cases.

---

## Milestone 5 — Agent Runner Integration Tests

### Description

Test end-to-end agent run flow through the `Manager` with an Ollama-backed agent, verifying driver selection, supervisor behaviour, concurrency, and status lifecycle.

### Files to change

- `tests/integration/ollama_agent_run_test.go` — New file. Manager-level integration tests.

### Test cases

1. **Driver selection** — Agent with `driver: ollama` uses `OllamaDriver`; agent with `driver: claude-code-cli` uses `ClaudeCodeDriver` (verify both in same test).
2. **Unknown driver** — Agent with `driver: unknown` returns an error from `StartRun`.
3. **Ollama run completes** — Full run lifecycle: `StartRun` → progress events broadcast → run record shows `status=completed`.
4. **Ollama run fails** — Mock returns error → run record shows `status=failed` with `stderr_tail`.
5. **Concurrency semaphore** — Start `max_concurrent_agents` Ollama runs + 1 Claude run → last run returns `ErrBusy`.
6. **Kill Ollama run** — Start a long-running Ollama mock → `Kill()` → run status is `killed`.
7. **Status lifecycle** — Agent with `active_status: in-development` and `done_on_success: true` → target artifact transitions through statuses correctly.
8. **Hub events** — Verify `agent.progress`, `agent.finished` events are broadcast via the hub.

### Acceptance criteria

- [ ] All 8 test cases pass.
- [ ] Tests use the mock Ollama server from Milestone 1.
- [ ] Tests create a temporary project directory with a minimal `lifecycle/config.yaml`.
- [ ] No test requires a real Claude Code CLI or real Ollama instance.
- [ ] Concurrency test verifies the semaphore without race conditions (use `sync.WaitGroup` or channel synchronisation).

---

## Milestone 6 — Regression Tests

### Description

Verify that existing Claude Code agent functionality is unaffected by the Ollama changes. These are targeted checks against the refactored `Manager` to catch any regressions introduced by the driver registry refactor.

### Files to change

- `tests/integration/ollama_regression_test.go` — New file. Regression-focused tests.

### Test cases

1. **Claude Code agent still works** — Existing `claude-code-cli` agent config loads and `StartRun` selects the correct driver.
2. **Default driver** — If `driver` field is omitted in agent config, defaults to `claude-code-cli` (backward compatibility).
3. **Config without ollama_instances** — App config with no `ollama_instances` key loads successfully (empty list).
4. **Mixed agents** — Config with both `claude-code-cli` and `ollama` agents loads and validates correctly.
5. **Agent list API** — `GET /api/p/{project}/agents` returns both Claude and Ollama agents with correct driver fields.

### Acceptance criteria

- [ ] All 5 test cases pass.
- [ ] Tests confirm backward compatibility: no existing config or behaviour is broken.
- [ ] Default driver test uses a config YAML that omits the `driver` field entirely.
