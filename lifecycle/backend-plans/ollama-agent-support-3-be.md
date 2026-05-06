---
title: "Ollama Agent Support — Backend Plan"
type: plan-backend
status: in-development
lineage: ollama-agent-support
parent: lifecycle/requirements/ollama-agent-support-2.md
---

# Ollama Agent Support — Backend Plan

## Overview

Implement backend support for Ollama as a second agent driver. This covers instance registration in app-level config, a new `OllamaDriver` conforming to the existing `Driver` interface, Ollama-specific REST endpoints (health, model discovery), and agent config extensions so `driver: ollama` routes runs through the new driver.

Cross-references: [[ollama-agent-support]] frontend plan for UI integration, [[ollama-agent-support]] test plan for integration tests.

---

## Milestone 1 — App-Level Instance Config

### Description

Extend the app-level config (`~/.kaos-control/config.yaml`) with an `ollama_instances` list. Each entry stores `name`, `base_url`, and optional `api_key`. Provide load/save helpers and validation.

### Files to change

- `internal/config/config.go` — Add `OllamaInstance` struct and `OllamaInstances []OllamaInstance` field to `App`.
- `internal/config/config.go` — Add validation in `LoadApp`: unique names, non-empty `base_url`, valid URL format.

### Acceptance criteria

- [ ] `OllamaInstance` struct has fields: `Name string`, `BaseURL string`, `APIKey string` (yaml tags: `name`, `base_url`, `api_key`).
- [ ] `App.OllamaInstances` is populated from YAML and round-trips correctly (load → save → load).
- [ ] Duplicate instance names are rejected at load time with a clear error.
- [ ] Empty or malformed `base_url` is rejected.

---

## Milestone 2 — Instance CRUD API

### Description

Expose REST endpoints for managing Ollama instances. These operate on app-level config and are not project-scoped (instances are shared across projects, per resolved question #2 in the requirement).

### Files to change

- `internal/http/ollama.go` — New file. Handlers: `handleListOllamaInstances`, `handleCreateOllamaInstance`, `handleUpdateOllamaInstance`, `handleDeleteOllamaInstance`.
- `internal/http/server.go` — Register routes under `/api/ollama/instances`.
- `internal/config/config.go` — Add `SaveApp(path string, cfg App) error` for persisting config changes.

### Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/api/ollama/instances` | `handleListOllamaInstances` | Masks `api_key` → `"***"` when set (NFR-2) |
| POST | `/api/ollama/instances` | `handleCreateOllamaInstance` | Body: `{name, base_url, api_key?}` |
| PUT | `/api/ollama/instances/{name}` | `handleUpdateOllamaInstance` | Body: `{base_url, api_key?}` |
| DELETE | `/api/ollama/instances/{name}` | `handleDeleteOllamaInstance` | Rejects if any project agent references this instance |

### Acceptance criteria

- [ ] `GET` returns all instances with `api_key` masked.
- [ ] `POST` creates a new instance; returns 409 on duplicate name.
- [ ] `PUT` updates an existing instance; returns 404 if not found.
- [ ] `DELETE` removes an instance; returns 409 if referenced by an agent config.
- [ ] All mutations persist to `~/.kaos-control/config.yaml` atomically (write-tmp + rename).
- [ ] `api_key` is never logged or returned in plain text.

---

## Milestone 3 — Health & Model Discovery Endpoints

### Description

Add endpoints that proxy Ollama's own REST API to check instance connectivity and list available models. These are project-scoped routes because agent runs are project-scoped, but they resolve the instance from app config.

### Files to change

- `internal/http/ollama.go` — Add `handleOllamaHealth` and `handleOllamaModels` handlers.
- `internal/http/server.go` — Register under `/api/ollama/instances/{name}/health` and `/api/ollama/instances/{name}/models`.

### Routes

| Method | Path | Handler | Proxies to |
|--------|------|---------|------------|
| GET | `/api/ollama/instances/{name}/health` | `handleOllamaHealth` | `GET {base_url}/api/tags` with 10s timeout |
| GET | `/api/ollama/instances/{name}/models` | `handleOllamaModels` | `GET {base_url}/api/tags` with 10s timeout |

### Acceptance criteria

- [ ] Health endpoint returns `{"ok": true, "latency_ms": N}` when the instance responds, or `{"ok": false, "error": "..."}` on failure.
- [ ] Models endpoint returns `{"models": [{"name": "llama3:8b", "size": 4000000000}, ...]}` extracted from Ollama's `/api/tags` response.
- [ ] Both endpoints respect a 10-second timeout (NFR-1).
- [ ] `api_key`, when configured, is sent as `Authorization: Bearer <key>` on proxied requests.
- [ ] Unknown instance name returns 404.

---

## Milestone 4 — OllamaDriver (Driver Interface Implementation)

### Description

Implement `OllamaDriver` conforming to the `Driver` interface. The driver makes HTTP requests to the Ollama `/api/chat` or `/api/generate` endpoint (based on config), streams the response, and emits `ProgressEvent` messages. It supports separate system and user prompts (resolved question #1).

### Files to change

- `internal/agent/ollama.go` — New file. `OllamaDriver` struct, `Start()` method, `ollamaProcess` struct implementing `Process`.
- `internal/config/config.go` — Add `OllamaInstance` and `OllamaModel` fields to `AgentConfig`: `OllamaInstanceName string` (`yaml:"ollama_instance"`), `OllamaEndpoint string` (`yaml:"ollama_endpoint"`, values: `chat` or `generate`, default `chat`).

### Design

```
OllamaDriver struct {
    Instances []config.OllamaInstance   // resolved from app config at construction
    HTTPClient *http.Client             // with configurable timeout
}

ollamaProcess struct {
    cancel   context.CancelFunc
    progress chan ProgressEvent
    stderr   *ringBuf
    done     chan error
}
```

- `Start()`:
  1. Resolve instance by name from `Instances` list.
  2. Build request body: `{"model": run.Model, "system": <system prompt>, "prompt"/"messages": <user prompt>, "stream": true}`.
  3. POST to `{base_url}/api/chat` or `{base_url}/api/generate` based on `OllamaEndpoint`.
  4. Emit `ProgressEvent{Raw: "started"}`.
  5. Spawn goroutine to read NDJSON response stream, emitting `ProgressEvent` per chunk.
  6. On completion, emit final `ProgressEvent` with full response and close channel.

- `Process.Wait()` blocks on `done` channel.
- `Process.Kill()` calls `cancel()` which aborts the HTTP request.
- `Process.StderrTail()` returns any HTTP error details from `ringBuf`.
- `Process.Progress()` returns the progress channel.

### System vs user prompt separation

The driver inspects `Run.PromptText` for a `---SYSTEM---` / `---USER---` delimiter convention. If present, the text before the delimiter is sent as the system prompt and the remainder as the user message. If absent, the entire text is sent as the user message. This is transparent to callers.

### Acceptance criteria

- [ ] `OllamaDriver` implements `Driver` interface (compiles).
- [ ] `Start()` returns an `ollamaProcess` that emits `started`, streamed `output`, and `completed`/`error` events.
- [ ] `/api/chat` is used by default; `/api/generate` when `ollama_endpoint: generate` is set.
- [ ] System and user prompts are separated correctly.
- [ ] HTTP timeout defaults to 5 minutes (NFR-1); configurable via `TimeoutMinutes` on `AgentConfig`.
- [ ] Context cancellation aborts the in-flight HTTP request.
- [ ] `StderrTail()` captures HTTP-level errors (non-200 status, connection errors).
- [ ] Progress channel capacity is 64 (consistent with `ClaudeCodeDriver`).

---

## Milestone 5 — Agent Runner Driver Selection

### Description

Update the agent `Manager` to select the correct driver based on `AgentConfig.Driver`. Currently `Manager` holds a single `driver Driver` field initialised to `&ClaudeCodeDriver{}`. Refactor to a driver registry / factory so `ollama` routes to `OllamaDriver`.

### Files to change

- `internal/agent/agent.go` — Replace single `driver` field with a `drivers map[string]Driver`. Update `New()` to register both drivers. Update `StartRun()` to look up `agents[i].Driver` in the map.
- `internal/project/project.go` — Pass app-level `OllamaInstances` into `agent.New()` so the `OllamaDriver` can resolve instances.

### Acceptance criteria

- [ ] `Manager.drivers` maps `"claude-code-cli"` → `&ClaudeCodeDriver{}` and `"ollama"` → `&OllamaDriver{...}`.
- [ ] `StartRun()` uses `m.drivers[agentCfg.Driver]` to select the driver; returns a clear error for unknown driver values.
- [ ] Existing `claude-code-cli` agents continue to work without any change.
- [ ] Ollama agent runs participate in the same `sem` semaphore (NFR-3).
- [ ] All supervisor logic (progress forwarding, git commit, status lifecycle, lock management) applies identically to Ollama runs.

---

## Milestone 6 — Agent Config Extensions & Validation

### Description

Extend `AgentConfig` parsing and validation to support `driver: ollama` with its required fields (`ollama_instance`, `model`, `ollama_endpoint`). Validate at config load time that referenced instances exist.

### Files to change

- `internal/config/project.go` — Add validation in `LoadProject` or the project config loader: when `Driver == "ollama"`, require `OllamaInstanceName` and `Model` to be non-empty.
- `internal/config/config.go` — Extend `AgentConfig` struct (already done in Milestone 4).

### Acceptance criteria

- [ ] An agent with `driver: ollama` but missing `ollama_instance` or `model` causes a config validation error at load time.
- [ ] An agent referencing a non-existent `ollama_instance` name produces a clear error.
- [ ] `ollama_endpoint` defaults to `"chat"` when omitted.
- [ ] Agents with `driver: claude-code-cli` ignore `ollama_instance` / `ollama_endpoint` fields.
- [ ] The `handleListAgents` API response includes `driver`, `ollama_instance`, and `model` fields for Ollama agents.
