---
title: "Conversational Idea Capture – Backend Plan"
type: plan-backend
status: done
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-2.md
labels:
    - artefacts
    - workflow
    - agent
---

# Conversational Idea Capture – Backend Plan

This plan implements the server-side components for the conversational idea capture feature described in [[prompt-to-idea]]. The backend provides a stateful conversation endpoint, an in-memory session store with TTL-based expiry, slug generation with collision detection, artifact writing, and agent configuration.

---

## Milestone 1 – Conversation Session Store

### Description

Introduce an in-memory session store that holds conversational state for idea-capture sessions. Each session tracks a unique ID, the project it belongs to, the authenticated user, the message history, the current conversation status (`conversing`, `proposed`, `created`), a clarification counter, and a proposed artifact draft. Sessions expire after 30 minutes of inactivity and are cleaned up by a background reaper.

### Files to change

- **`internal/ideachat/session.go`** (new) – Define `Session` struct, `Store` with `sync.RWMutex`-guarded map, `NewStore()`, `Create(projectSlug, userEmail) *Session`, `Get(sessionID) (*Session, bool)`, `Touch(sessionID)`, `Delete(sessionID)`, and `StartReaper(ctx, interval)`.

### Acceptance criteria

- [ ] `Session` struct holds: `ID string`, `ProjectSlug string`, `UserEmail string`, `Messages []Message`, `Status string` (`conversing`|`proposed`|`created`), `ClarifyCount int`, `ProposedFM artifact.Frontmatter`, `ProposedBody string`, `ProposedSlug string`, `CreatedAt time.Time`, `LastActivity time.Time`.
- [ ] `Message` struct holds: `Role string` (`user`|`assistant`), `Content string`.
- [ ] `Create` generates a crypto/rand hex session ID (16 bytes) and sets `LastActivity` to `time.Now()`.
- [ ] `Get` returns `nil, false` for unknown or expired sessions.
- [ ] `Touch` updates `LastActivity`.
- [ ] `StartReaper` runs in a goroutine, sweeps every 5 minutes, deletes sessions whose `LastActivity` is older than 30 minutes.
- [ ] Unit test covers create, get, touch, expiry, and delete.

---

## Milestone 2 – LLM Conversation Logic

### Description

Implement the core conversation logic that takes a user message, the session's history, and the project's label vocabulary, then calls the configured LLM to produce either a clarifying question, a proposed idea artifact, or a final confirmation response. This is a synchronous request/response call (no streaming for v1). The LLM prompt instructs the model to:
1. Ask up to 3 short clarifying questions if the input is vague.
2. Once sufficient context exists, generate a slug, frontmatter (title, type, status, lineage, labels from the project's existing label set), and a markdown body.
3. Return structured JSON so the backend can parse the response deterministically.

### Files to change

- **`internal/ideachat/converse.go`** (new) – `Converse(ctx, session *Session, userMsg string, existingLabels []string, existingSlugs []string, modelConfig ModelConfig) (*Response, error)`. Builds the system prompt from the agent's `prompt_templates` entry, appends conversation history, calls the LLM API, parses the structured response.
- **`internal/ideachat/llm.go`** (new) – `ModelConfig` struct and `CallLLM(ctx, model string, messages []LLMMessage) (string, error)` – thin wrapper around the Anthropic API (using the model specified in `lifecycle/config.yaml` for the `idea-capture` agent, defaulting to `sonnet`). Uses `ANTHROPIC_API_KEY` from the environment.

### Acceptance criteria

- [ ] `Converse` returns a `Response` with fields: `Reply string`, `Status string` (`conversing`|`proposed`), `ProposedSlug string`, `ProposedFM *artifact.Frontmatter`, `ProposedBody string`.
- [ ] When `session.ClarifyCount >= 3`, the function forces the LLM to produce a proposal rather than asking further questions.
- [ ] The system prompt instructs the LLM to pick labels only from the provided `existingLabels` list.
- [ ] The system prompt instructs the LLM to return a JSON block with fields: `action` (`clarify`|`propose`), `reply`, `slug`, `title`, `labels`, `body`.
- [ ] `Converse` validates the generated slug against `slugRe` (`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`) and checks it against `existingSlugs`; if collision detected, asks the LLM to adjust (one retry) or appends a disambiguating suffix.
- [ ] On `action: propose`, session status transitions to `proposed` and the proposed artifact is stored on the session.
- [ ] `CallLLM` respects a 30-second context timeout.
- [ ] Errors from the LLM are wrapped with context and surfaced to the caller (not swallowed).

---

## Milestone 3 – HTTP Endpoint (`POST /api/p/:project/ideas/converse`)

### Description

Wire up the conversation endpoint in the HTTP router. The endpoint accepts a JSON body, delegates to the session store and conversation logic, and returns the response. It also handles the `accept` and `reject` actions for the proposal confirmation flow.

### Files to change

- **`internal/http/idea_chat.go`** (new) – `handleIdeaConverse(w, r)` handler.
- **`internal/http/server.go`** – Add route `r.Post("/ideas/converse", s.handleIdeaConverse)` inside the `/api/p/{project}` group. Add `ideachat.Store` to `Server` struct (initialised in constructor).

### Acceptance criteria

- [ ] Request schema: `{ "session_id": string|null, "message": string }`. The `message` field is required and non-empty. Special message values `"__accept__"` and `"__reject__"` trigger the confirmation flow.
- [ ] When `session_id` is null/empty, a new session is created via the store and the returned `session_id` is included in the response.
- [ ] When `session_id` is provided but not found (expired or invalid), returns HTTP 404 with `{ "code": "session_not_found", "message": "..." }`.
- [ ] Response schema: `{ "session_id": string, "reply": string, "status": "conversing"|"proposed"|"created", "preview": { "frontmatter": object, "body": string }|null, "artifact_path": string|null }`.
- [ ] `preview` is non-null only when `status` is `proposed`.
- [ ] `artifact_path` is non-null only when `status` is `created`.
- [ ] On `__accept__` when session status is `proposed`: writes the artifact to disk (see Milestone 4), returns `status: "created"`.
- [ ] On `__reject__`: deletes the session, returns `{ "session_id": null, "reply": "Idea discarded.", "status": "conversing" }`.
- [ ] On `__accept__` when session status is not `proposed`: returns HTTP 409 with `{ "code": "no_proposal", "message": "..." }`.
- [ ] Endpoint requires authentication (existing session middleware).
- [ ] `go vet` and `go build` pass.

---

## Milestone 4 – Artifact Writing and Indexing

### Description

When the user accepts a proposed idea, the backend writes the artifact file to `lifecycle/ideas/<slug>.md`, triggers indexing, and broadcasts the appropriate WebSocket event. This reuses the existing `buildMarkdown` helper and index/hub infrastructure.

### Files to change

- **`internal/http/idea_chat.go`** – Add `writeIdeaArtifact(p *project.Project, session *Session) (string, error)` helper within the handler file.
- **`internal/http/write.go`** – Export `BuildMarkdown` (or extract the helper so it can be called from `idea_chat.go`; alternatively, since both are in the same package, the unexported `buildMarkdown` is already accessible).

### Acceptance criteria

- [ ] The artifact file is written to `lifecycle/ideas/<slug>.md` using `os.WriteFile` with mode `0644`.
- [ ] The file path is validated via `sandbox.Resolve(projectRoot, relPath)` before writing.
- [ ] Frontmatter fields: `title` (from LLM), `type: idea`, `status: draft`, `lineage` (same as slug), `labels` (from LLM, constrained to existing labels).
- [ ] The body contains a level-1 heading matching the title and 1–3 paragraphs.
- [ ] After writing, `p.Idx.IndexFile(absPath)` is called to update the SQLite cache.
- [ ] After indexing, `p.Hub.Broadcast(hub.Event{Type: "artifact.indexed", Payload: ...})` is called.
- [ ] If the file already exists on disk (race condition with another session), returns an error rather than overwriting.
- [ ] The session status transitions to `created` and the session is deleted from the store after successful write.
- [ ] The relative artifact path is returned to the caller for inclusion in the HTTP response.

---

## Milestone 5 – Agent Configuration in `lifecycle/config.yaml`

### Description

Add the `idea-capture` agent entry to the project configuration so the prompt template is tuneable without code changes. The agent does not use the standard `agent.Manager` run flow (it is an inline, synchronous conversation rather than a long-running CLI process), but its config entry provides the model selection and prompt template.

### Files to change

- **`lifecycle/config.yaml`** – Add `idea-capture` agent block with `role: [product-owner]`, `driver: inline`, `model: sonnet`, `allowed_write_paths: [lifecycle/ideas]`, and `prompt_templates.idea-capture: |` containing the system prompt.
- **`internal/config/config.go`** – No structural changes needed; `AgentConfig` already supports arbitrary `prompt_templates` keys and a `driver` field. The `inline` driver value is simply a convention recognised by the idea-chat handler (not by `agent.Manager`).

### Acceptance criteria

- [ ] `lifecycle/config.yaml` contains an `idea-capture` agent with the fields listed above.
- [ ] The system prompt template in `prompt_templates.idea-capture` instructs the LLM to: produce a JSON response, respect the label vocabulary, generate a valid slug, keep clarifying questions to single sentences, and produce self-contained idea bodies.
- [ ] `config.LoadProject` parses the new entry without error.
- [ ] The `handleIdeaConverse` handler reads the prompt template from the project config at runtime (not hardcoded).
- [ ] The model field from the config is passed to `CallLLM`.

---

## Milestone 6 – Session Cleanup and Lifecycle Integration

### Description

Ensure the session store reaper is started alongside the project runtime, and that the conversation endpoint integrates cleanly with the existing project lifecycle (watcher, index, hub).

### Files to change

- **`internal/project/project.go`** – Initialise `ideachat.Store` in `Open()`, start `Store.StartReaper(ctx)` in `StartWatcher()` (or a new `StartSessionReaper` method). Store the `ideachat.Store` as a field on `Project`.
- **`internal/http/server.go`** – Access the store via `projectFromCtx(ctx).IdeaChatStore` in the handler, removing the need for a server-level store.

### Acceptance criteria

- [ ] Each project has its own `ideachat.Store` instance (not a global singleton).
- [ ] The reaper goroutine is started when the project starts and cancelled when the project closes.
- [ ] `Project.Close()` stops the reaper cleanly (context cancellation).
- [ ] No goroutine leaks: the reaper exits when its context is cancelled.
- [ ] Existing endpoints (`POST /api/p/:project/artifacts`, `PUT /api/p/:project/artifacts/*`) continue to function unchanged (NFR-3).
- [ ] `go build ./...` and `go vet ./...` pass with all changes.

---

## Cross-references

- The [[prompt-to-idea]] frontend plan defines the chat UI that consumes this endpoint.
- The [[prompt-to-idea]] test plan covers integration tests for the conversation flow, slug collision handling, session expiry, and artifact creation.
