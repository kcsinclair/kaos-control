---
title: "Single-Submit Idea Capture – Backend Plan"
type: plan-backend
status: rejected
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
labels:
    - agent
    - workflow
    - usability
    - enhancement
---

# Single-Submit Idea Capture – Backend Plan

This plan implements the server-side components for the single-submit "brain dump" idea capture flow described in [[prompt-to-idea]]. The backend adds a new stateless `POST /ideas/generate` endpoint that sends user text to the LLM in a single request, returns a fully-formed artifact proposal (slug, title, labels, body, frontmatter), and handles slug collision detection. It also extends the `idea-capture` agent configuration with a new `idea-generate` prompt template. The existing conversational `/ideas/converse` endpoint is preserved unchanged per NFR-4.

The frontend plan ([[prompt-to-idea]]-fe) consumes this endpoint. The test plan ([[prompt-to-idea]]-test) validates its behaviour.

---

## Milestone 1 – Single-Shot Generate Function

### Description

Add a new `Generate` function to the `internal/ideachat` package that accepts raw user text and returns a complete artifact proposal in a single LLM round-trip — no session state, no multi-turn conversation. This function builds a system prompt instructing the LLM to return structured JSON (`slug`, `title`, `labels`, `body`) in one response, calls `CallLLM`, parses the result, validates the slug, filters labels against the existing vocabulary, and returns the proposal.

### Files to change

- **`internal/ideachat/generate.go`** (new) – `Generate(ctx, input, existingLabels, existingSlugs, modelCfg) (*GenerateResult, error)`. `GenerateResult` contains `Slug`, `Title`, `Labels []string`, `Body string`, `Frontmatter artifact.Frontmatter`. The function:
  1. Validates input length (≥5 words with discernible content); returns a typed `ErrInputTooShort` otherwise.
  2. Builds an `[]LLMMessage` with a single user message containing the raw input, plus context about available labels.
  3. Calls `CallLLM` with the `idea-generate` system prompt from `ModelConfig`.
  4. Parses the JSON response using the existing `parseAction` / `extractJSON` helpers (reused from `converse.go`; ensure they are package-internal, not file-private).
  5. Validates and resolves the slug via the existing `resolveSlug` function.
  6. Filters labels via the existing `filterLabels` function.
  7. Constructs the `Frontmatter` with `type: idea`, `status: draft`, `lineage: <slug>`, and a `priority` field (default `normal`, or `high` if the LLM signals urgency).
  8. Returns `*GenerateResult`.

- **`internal/ideachat/converse.go`** – Ensure `parseAction`, `extractJSON`, `resolveSlug`, `filterLabels`, and `sanitiseSlug` are exported or at minimum accessible within the package (they already are, since they are package-level functions — verify no conflicts).

### Acceptance criteria

- [ ] `Generate` returns a complete `GenerateResult` with valid slug, title, at least one label, and a non-empty body for reasonable input.
- [ ] `Generate` returns `ErrInputTooShort` for input with fewer than 5 words or no discernible intent.
- [ ] Labels in the result are restricted to the `existingLabels` vocabulary (no invented labels).
- [ ] Slug matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$` and collisions are resolved with numeric suffix.
- [ ] Frontmatter has `type: idea`, `status: draft`, `lineage` matching slug, `priority: normal` by default.
- [ ] The function uses a single LLM call — no multi-turn conversation.
- [ ] Unit test covers: happy path, short input rejection, slug collision resolution, label filtering.

---

## Milestone 2 – Defect Generate Support

### Description

Extend the `Generate` function to support defect brain-dump capture as well as ideas (FR from Open Question 2 in the requirement). The function accepts an optional `artifactType` parameter (`"idea"` or `"defect"`) which selects the appropriate system prompt and frontmatter defaults. For defects, the LLM prompt instructs it to produce a body with `## Reproduction Steps`, `## Expected Behaviour`, `## Actual Behaviour` sections, and frontmatter uses `type: defect`, `status: draft`, and adds a `defect` label.

### Files to change

- **`internal/ideachat/generate.go`** – Add an `ArtifactType string` field to the input options (defaulting to `"idea"` when empty). Branch on this to select the correct system prompt key (`idea-generate` vs `defect-generate`) and set the frontmatter `Type` accordingly. For defects, the target directory is `lifecycle/defects/` and the slug collision check must look at defect slugs.
- **`internal/ideachat/generate.go`** – Add `GenerateOptions` struct: `{ Input string, ArtifactType string, ExistingLabels []string, ExistingSlugs []string, ModelCfg ModelConfig }` to keep the function signature clean.

### Acceptance criteria

- [ ] `Generate` with `ArtifactType: "idea"` behaves identically to Milestone 1.
- [ ] `Generate` with `ArtifactType: "defect"` returns frontmatter with `type: defect`, `status: draft`, and includes `defect` in labels.
- [ ] Defect body contains structured sections (Reproduction Steps, Expected Behaviour, Actual Behaviour).
- [ ] Unknown `ArtifactType` values return an error.
- [ ] Unit test covers defect generation happy path and type validation.

---

## Milestone 3 – HTTP Handler for `/ideas/generate`

### Description

Wire up the `Generate` function as a new HTTP endpoint. The handler reads the project context, resolves the `idea-capture` agent config (with the new `idea-generate` prompt template), gathers the label and slug vocabularies, calls `Generate`, and returns the proposal as JSON. Slug collision detection runs against files on disk in `lifecycle/ideas/` (for ideas) or `lifecycle/defects/` (for defects).

### Files to change

- **`internal/http/idea_generate.go`** (new) – `handleIdeaGenerate(w, r)`:
  - Request body: `{ "input": string, "type"?: "idea" | "defect" }` (type defaults to `"idea"`).
  - Validates `input` is non-empty.
  - Resolves project, user (authenticated), and agent config.
  - Looks up the prompt template: `idea-generate` for ideas, `defect-generate` for defects, from the `idea-capture` agent entry.
  - Gathers `existingLabels` from `p.Idx.Labels()` and `existingSlugs` from `collectSlugs(p)`.
  - Calls `ideachat.Generate(...)`.
  - On `ErrInputTooShort`, returns `400` with a user-facing message.
  - On success, returns `200` with:
    ```json
    {
      "slug": "...",
      "title": "...",
      "labels": [...],
      "body": "...",
      "frontmatter": { ... },
      "target_dir": "lifecycle/ideas" | "lifecycle/defects"
    }
    ```
  - The response includes a `target_dir` field so the frontend knows the file destination.

- **`internal/http/server.go`** – Register `r.Post("/ideas/generate", s.handleIdeaGenerate)` in the per-project route group, alongside the existing `/ideas/converse` route.

### Acceptance criteria

- [ ] `POST /api/p/:project/ideas/generate` with `{ "input": "A long enough description of a feature" }` returns 200 with slug, title, labels, body, and frontmatter.
- [ ] `POST /api/p/:project/ideas/generate` with `{ "input": "hi" }` returns 400 with a user-friendly error message.
- [ ] `POST /api/p/:project/ideas/generate` with `{ "input": "...", "type": "defect" }` returns a defect-shaped proposal.
- [ ] Missing or empty `input` returns 400.
- [ ] Unauthenticated requests return 401.
- [ ] The existing `/ideas/converse` endpoint continues to function unchanged (NFR-4).
- [ ] The existing `POST /api/p/:project/artifacts` endpoint continues to function unchanged (NFR-3).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 4 – Slug Collision Detection Against Disk

### Description

The requirement (FR-2.2) specifies that slug collision detection must run against existing files in `lifecycle/ideas/` (not just the index). The existing `resolveSlug` in `converse.go` checks against lineage slugs from the index. For the generate endpoint, add a helper that also scans the target directory on disk for files starting with the proposed slug, so that even un-indexed files are caught.

### Files to change

- **`internal/ideachat/generate.go`** – Add a `CollectDiskSlugs(projectPath, targetDir string) ([]string, error)` function that globs `lifecycle/<targetDir>/*.md`, parses each filename to extract the slug, and returns the set. The `Generate` function merges these with the `existingSlugs` from the index before calling `resolveSlug`.

### Acceptance criteria

- [ ] `CollectDiskSlugs` returns all slugs present as files in the target directory.
- [ ] Slug collision detection catches files not yet in the SQLite index.
- [ ] On collision, the returned slug has a numeric suffix (e.g., `my-idea-2`).
- [ ] Unit test with a temp directory containing fixture `.md` files verifies collision detection.

---

## Milestone 5 – Agent Configuration Update

### Description

Update the `idea-capture` agent entry in `lifecycle/config.yaml` to include the new `idea-generate` prompt template (FR-7.1, FR-7.2) and a `defect-generate` template. Update `resolveIdeaCaptureConfig` to support looking up templates by key.

### Files to change

- **`lifecycle/config.yaml`** – Add `prompt_templates.idea-generate` and `prompt_templates.defect-generate` entries under the `idea-capture` agent. The `idea-generate` prompt instructs the LLM to return structured JSON (matching the `llmAction` schema: `action: "propose"`, `slug`, `title`, `labels`, `body`) in a single response with no clarifying questions. The `defect-generate` prompt instructs the same but for defect-shaped output.

- **`internal/http/idea_generate.go`** – The handler looks up the correct prompt template key from the agent config based on the requested artifact type.

- **`internal/http/idea_chat.go`** – Refactor `resolveIdeaCaptureConfig` to accept a `templateKey string` parameter so it can resolve either `idea-capture`, `idea-generate`, or `defect-generate` templates from the same agent entry. The existing `handleIdeaConverse` passes `"idea-capture"` to maintain backwards compatibility.

### Acceptance criteria

- [ ] `lifecycle/config.yaml` contains `prompt_templates.idea-generate` under the `idea-capture` agent.
- [ ] `lifecycle/config.yaml` contains `prompt_templates.defect-generate` under the `idea-capture` agent.
- [ ] `resolveIdeaCaptureConfig` accepts a template key and returns the correct prompt for each key.
- [ ] The existing conversational flow (`/ideas/converse`) still resolves the `idea-capture` template correctly.
- [ ] The `idea-generate` prompt explicitly instructs the LLM to produce `action: "propose"` with no `"clarify"` action.
- [ ] `go build ./...` and `go vet ./...` pass.
