---
title: "Single-Submit Idea & Defect Capture – Backend Plan"
type: plan-backend
status: done
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
---

# Single-Submit Idea & Defect Capture – Backend Plan

This plan implements the server-side components for the single-submit "brain dump" capture flow described in [[prompt-to-idea]]. It adds a stateless `POST /ideas/generate` endpoint that sends user text to the LLM in a single request and returns a fully-formed artifact proposal (slug, title, labels, body, frontmatter) without writing anything to disk. Defect brain-dump capture is included per Open Question 2 in the requirement. The existing conversational `/ideas/converse` endpoint is preserved unchanged (NFR-4).

The frontend plan ([[prompt-to-idea]]-fe) consumes this endpoint. The test plan ([[prompt-to-idea]]-test) validates its behaviour.

---

## Milestone 1 – `Generate` Function (Single-Shot LLM Call)

### Description

Add a `Generate` function to `internal/ideachat` that accepts raw user text and returns a complete artifact proposal in a single LLM round-trip. The function builds a system prompt instructing the LLM to return structured JSON (`action: "propose"`, `slug`, `title`, `labels`, `body`) in one response, calls the existing `CallLLM` from `llm.go`, parses the result using the existing `extractJSON` / `parseAction` helpers in `converse.go`, validates the slug via `resolveSlug`, and filters labels via `filterLabels`.

### Files to change

- **`internal/ideachat/generate.go`** (new) — Contains:
  - `GenerateOptions` struct: `Input string`, `ArtifactType string` (`"idea"` or `"defect"`, default `"idea"`), `ExistingLabels []string`, `ExistingSlugs []string`, `ModelCfg ModelConfig`.
  - `GenerateResult` struct: `Slug`, `Title`, `Labels []string`, `Body string`, `Frontmatter map[string]any`, `TargetDir string`.
  - `ErrInputTooShort` sentinel error.
  - `Generate(ctx, opts) (*GenerateResult, error)`:
    1. Validates input (≥5 words with discernible content); returns `ErrInputTooShort` otherwise.
    2. Selects system prompt from `ModelCfg` by template key: `"idea-generate"` for ideas, `"defect-generate"` for defects. Returns error for unknown type.
    3. Builds `[]LLMMessage` with one user message containing the raw input plus a label vocabulary list.
    4. Calls `CallLLM` (from `llm.go:35`).
    5. Parses JSON via `extractJSON` / `parseAction` (reused from `converse.go`).
    6. Runs `sanitiseSlug` then `resolveSlug` against `ExistingSlugs`.
    7. Filters labels via `filterLabels` against `ExistingLabels`. For defects, ensures `"defect"` label is always present.
    8. Constructs frontmatter map: `type` (`idea` or `defect`), `status: draft`, `lineage: <slug>`, `labels`, `priority` (default `normal`).
    9. Sets `TargetDir` to `"lifecycle/ideas"` for ideas, `"lifecycle/defects"` for defects.
    10. Returns `*GenerateResult`.

- **`internal/ideachat/converse.go`** — No changes needed; `extractJSON`, `parseAction`, `resolveSlug`, `filterLabels`, `sanitiseSlug` are already package-level functions. Verify they remain accessible (not shadowed by test files).

### Acceptance criteria

- [ ] `Generate` returns a complete `GenerateResult` with valid slug, title, ≥1 label, and non-empty body for reasonable input.
- [ ] `Generate` returns `ErrInputTooShort` for input with fewer than 5 words.
- [ ] For `ArtifactType: "defect"`, frontmatter has `type: defect` and labels include `"defect"`.
- [ ] Unknown `ArtifactType` values return an error.
- [ ] Labels are restricted to the `ExistingLabels` vocabulary.
- [ ] Slug matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$` and collisions are resolved with numeric suffix.
- [ ] A single LLM call is made — no multi-turn conversation.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 – Disk-Based Slug Collision Detection

### Description

FR-2.2 requires slug collision detection against existing files on disk, not just the SQLite index. Add a helper that scans the target directory for files whose names start with the proposed slug, so that even un-indexed files are caught. The `Generate` function merges disk slugs with the provided `ExistingSlugs` before calling `resolveSlug`.

### Files to change

- **`internal/ideachat/generate.go`** — Add `CollectDiskSlugs(projectPath, targetDir string) ([]string, error)` that globs `<projectPath>/<targetDir>/*.md`, extracts the slug portion of each filename (stripping lineage index and stage suffixes), and returns the deduplicated set. Call this within `Generate` and merge results with `opts.ExistingSlugs`.

### Acceptance criteria

- [ ] `CollectDiskSlugs` returns slugs from all `.md` files in the target directory.
- [ ] Filenames with lineage suffixes (e.g., `my-idea-2.md`) are correctly parsed to extract the base slug.
- [ ] On collision, `resolveSlug` appends a numeric suffix (e.g., `my-idea-2`).
- [ ] Unit test with a temp directory containing fixture `.md` files verifies collision detection.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3 – HTTP Handler for `/ideas/generate`

### Description

Wire up the `Generate` function as a new HTTP endpoint. The handler reads the project context, resolves the `idea-capture` agent config, gathers the label and slug vocabularies, calls `Generate`, and returns the proposal as JSON. No artifact is written to disk — the response is preview-only.

### Files to change

- **`internal/http/idea_generate.go`** (new) — `handleIdeaGenerate(w, r)`:
  - Request body: `{ "input": string, "type"?: "idea" | "defect" }` (type defaults to `"idea"`).
  - Validates `input` is non-empty (400 if missing).
  - Resolves project from URL param, user from session (401 if unauthenticated).
  - Looks up the `idea-capture` agent config from `p.Cfg.Agents` and selects the prompt template by key (`"idea-generate"` or `"defect-generate"`).
  - Gathers `existingLabels` from `p.Idx.Labels()` (index.go:486) and `existingSlugs` by calling `CollectDiskSlugs`.
  - Calls `ideachat.Generate(...)`.
  - On `ErrInputTooShort`, returns 400 with `{ "error": "<user-facing message>" }`.
  - On success, returns 200 with:
    ```json
    {
      "slug": "...",
      "title": "...",
      "labels": [...],
      "body": "...",
      "frontmatter": { ... },
      "target_dir": "lifecycle/ideas"
    }
    ```

- **`internal/http/server.go`** — Register `r.Post("/ideas/generate", s.handleIdeaGenerate)` in the per-project route group (near the existing `/ideas/converse` route at line ~145).

### Acceptance criteria

- [ ] `POST /api/p/:project/ideas/generate` with valid input returns 200 with slug, title, labels, body, frontmatter, and target_dir.
- [ ] `POST /api/p/:project/ideas/generate` with `{ "input": "hi" }` returns 400 with user-friendly error.
- [ ] `POST /api/p/:project/ideas/generate` with `{ "input": "...", "type": "defect" }` returns a defect-shaped proposal with `target_dir: "lifecycle/defects"`.
- [ ] Missing or empty `input` returns 400.
- [ ] Unauthenticated requests return 401.
- [ ] No file is written to disk by this endpoint.
- [ ] The existing `/ideas/converse` endpoint continues to function unchanged (NFR-4).
- [ ] The existing `POST /api/p/:project/artifacts` endpoint continues to function unchanged (NFR-3).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 4 – Refactor `resolveIdeaCaptureConfig` for Multiple Templates

### Description

The existing `resolveIdeaCaptureConfig` in `idea_chat.go:190–209` looks up the `idea-capture` agent and returns its single prompt template. Refactor it to accept a `templateKey string` parameter so it can resolve `"idea-capture"`, `"idea-generate"`, or `"defect-generate"` templates from the same agent entry. The existing `handleIdeaConverse` passes `"idea-capture"` to maintain backwards compatibility.

### Files to change

- **`internal/http/idea_chat.go`** — Change `resolveIdeaCaptureConfig` signature to accept `templateKey string`. Update its template lookup from `a.PromptTemplates["idea-capture"]` to `a.PromptTemplates[templateKey]`. Update the single call site in `handleIdeaConverse` to pass `"idea-capture"`.

- **`internal/http/idea_generate.go`** — Call `resolveIdeaCaptureConfig` with `"idea-generate"` or `"defect-generate"` based on the request's `type` field.

### Acceptance criteria

- [ ] `resolveIdeaCaptureConfig("idea-capture", ...)` returns the existing conversational prompt — no behaviour change for `/ideas/converse`.
- [ ] `resolveIdeaCaptureConfig("idea-generate", ...)` returns the single-shot idea prompt.
- [ ] `resolveIdeaCaptureConfig("defect-generate", ...)` returns the defect prompt.
- [ ] Missing template key returns a clear error.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 5 – Agent Configuration Update

### Description

Update the `idea-capture` agent entry in `lifecycle/config.yaml` with two new prompt templates: `idea-generate` (FR-7.1, FR-7.2) for single-shot idea capture and `defect-generate` for single-shot defect capture. Both prompts instruct the LLM to return structured JSON in a single response with no clarifying questions.

### Files to change

- **`lifecycle/config.yaml`** — Under the `idea-capture` agent's `prompt_templates`, add:
  - `idea-generate`: System prompt instructing the LLM to produce a single JSON response with `action: "propose"`, `slug`, `title`, `labels` (from provided vocabulary only), and `body` (self-contained markdown with level-1 heading, 1–3 paragraphs). Explicitly forbids `action: "clarify"`. Instructs the LLM to select `priority: "high"` only if the input signals urgency, otherwise `"normal"`.
  - `defect-generate`: Similar prompt but instructs the LLM to produce a defect-shaped body with `## Reproduction Steps`, `## Expected Behaviour`, `## Actual Behaviour` sections, and to always include `"defect"` in labels.

### Acceptance criteria

- [ ] `lifecycle/config.yaml` contains `prompt_templates.idea-generate` under the `idea-capture` agent.
- [ ] `lifecycle/config.yaml` contains `prompt_templates.defect-generate` under the `idea-capture` agent.
- [ ] The `idea-generate` prompt explicitly forbids `action: "clarify"` — single-shot only.
- [ ] The `defect-generate` prompt instructs structured defect sections in the body.
- [ ] The existing `idea-capture` template is unchanged.
- [ ] `go build ./...` and `go vet ./...` pass.
