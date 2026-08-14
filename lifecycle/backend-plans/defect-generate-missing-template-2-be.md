---
title: 'Backend plan — defect-generate template fallback, config self-repair, and graceful degradation'
type: plan-backend
status: done
lineage: defect-generate-missing-template
parent: lifecycle/defects/defect-generate-missing-template.md
release: KC-Release5
---

# Backend plan — "New Defect → Generate" missing-template fix

Fixes the hard failure where the defect-generation flow resolves the
`idea-capture` agent and asks it for a `defect-generate` prompt template that
the agent does not define, surfacing the raw error
`idea-capture agent has no template "defect-generate"`.

Root cause (verified):
- `internal/http/idea_generate.go:68` maps `type=defect` → `templateKey =
  "defect-generate"`, then calls `resolveIdeaCaptureConfig(p, templateKey)`.
- `internal/http/idea_chat.go:219` (`resolveIdeaCaptureConfig`) looks the key up
  on the `idea-capture` agent and returns a hard error at line 230 when the key
  is absent. The built-in fallback at line 243 covers **only** the
  `idea-capture` conversational key — `idea-generate`, `defect-generate`, and
  `doc-generate` have no default.
- `internal/initcmd/templates/config.yaml.tmpl:318-355` ships an `idea-capture`
  agent with only `idea-capture` and `idea-generate` templates, so **every
  fresh project** created from the init template reproduces the bug.

Strategy (per the defect's "Possible Solution"): (a) never hard-error — provide
built-in default prompt templates for every generation key; (b) validate the
project config on load and self-repair missing generation templates/agents from
the embedded init template, non-destructively; (c) fix the init template so new
projects are correct out of the box; (d) return an actionable error code when a
genuine misconfiguration remains.

Frontend consumes the new error code and config-health signal — see
[[defect-generate-missing-template-3-fe]]. Test coverage is specified in
[[defect-generate-missing-template-4-test]].

---

## Milestone 1 — Built-in default templates for every generation key

**Description.** Extend `resolveIdeaCaptureConfig` so that when the resolved
agent exists but lacks the requested template key, and when no agent is
configured at all, a built-in default prompt is returned for **every** single-
shot generation key (`idea-generate`, `defect-generate`, `doc-generate`) — not
just the conversational `idea-capture` key. This alone makes the endpoint stop
throwing `... agent has no template "defect-generate"`.

**Files to change.**
- `internal/http/idea_chat.go` — add package-level constants
  `defaultIdeaGeneratePrompt`, `defaultDefectGeneratePrompt`,
  `defaultDocGeneratePrompt` (mirroring the templates in
  `internal/initcmd/templates/config.yaml.tmpl` and `lifecycle/config.yaml` so
  the JSON contract — `action`/`reply`/`slug`/`title`/`labels`/`body`, and the
  three `## Reproduction Steps` / `## Expected Behaviour` / `## Actual
  Behaviour` sections for defects — is identical). Rework
  `resolveIdeaCaptureConfig` so both the "agent found but key missing" branch
  (line 228-231) and the "no agent configured" branch (line 242-249) fall back
  to a `defaultTemplateFor(templateKey)` helper, returning
  `(ModelConfig{}, error)` only for genuinely unknown keys.
- Keep the existing `Model` default (`claude-sonnet-4-6`) for fallback configs.

**Acceptance criteria.**
- `resolveIdeaCaptureConfig(p, "defect-generate")` returns a non-empty
  `SystemPrompt` and no error when the `idea-capture` agent has no
  `defect-generate` key, or when no `idea-capture` agent is configured.
- The same holds for `idea-generate` and `doc-generate`.
- An unknown key (e.g. `"bogus"`) still returns a descriptive error.
- The returned default `defect-generate` prompt instructs the model to emit the
  three required `##` sections and always include the `defect` label, matching
  the existing config template contract.

---

## Milestone 2 — Startup config validation + non-destructive self-repair

**Description.** When a project config loads, detect missing required agents and
generation templates and repair them from the embedded init template defaults,
logging a warning per repair. Repairs are **in-memory** (the runtime
`config.Project` used to serve requests); the user's on-disk `config.yaml` is
never rewritten silently. Existing user-defined templates are never overwritten.
The set of required generation capabilities is: `idea-capture` agent with
`idea-capture`, `idea-generate`, `defect-generate`; `docs-capture` agent with
`doc-generate`.

**Files to change.**
- `internal/config/config.go` — add `func (c *Project) ValidateAndRepair()
  []RepairNote` (or extend `validateProject`) that: (1) ensures an
  `idea-capture` agent exists with the three keys, adding any missing key from
  the embedded defaults; (2) ensures a `docs-capture` agent exists with
  `doc-generate`; (3) returns a slice of `RepairNote{Agent, TemplateKey,
  Reason}` describing what was filled. Define `RepairNote` here.
- `internal/initcmd/embed.go` / a small accessor — expose the embedded default
  templates (or a parsed map keyed by agent+template) so `config` can source
  repair values without importing HTTP-layer constants. Prefer a single shared
  source of truth (see Milestone 5) to avoid divergence.
- Call site: `internal/config/config.go:679` (`LoadProject`) invokes
  `ValidateAndRepair` after `validateProject`, and the project runtime container
  (`internal/project/`) records the returned notes for later exposure.

**Acceptance criteria.**
- Loading a config whose `idea-capture` agent lacks `defect-generate` yields a
  runtime config where that agent now has a non-empty `defect-generate`
  template, and one `RepairNote` naming the agent + key.
- Loading a config that already defines a **custom** `defect-generate` leaves it
  byte-for-byte unchanged and produces **no** `RepairNote` for that key.
- Loading a config with no `idea-capture` agent adds one (driver `inline`, model
  `claude-sonnet-4-6`, `allowed_write_paths: [lifecycle/ideas]`) with all three
  keys, and emits repair notes.
- Repairs are logged at WARN with the project name, agent, and template key.
- `LoadProject` still returns a hard error for structurally invalid configs
  (empty stages, agent missing driver, etc.) — repair does not mask those.

---

## Milestone 3 — Fix the init template so fresh projects are correct

**Description.** Add the `defect-generate` template to the `idea-capture` agent
in the init template so newly initialised projects generate defects out of the
box without relying on runtime repair.

**Files to change.**
- `internal/initcmd/templates/config.yaml.tmpl` — insert a `defect-generate: |`
  block under `idea-capture.prompt_templates` (after `idea-generate`, around
  L377), identical in contract to `lifecycle/config.yaml:590-616`.

**Acceptance criteria.**
- A project scaffolded via `initcmd` has an `idea-capture` agent whose
  `prompt_templates` contains `idea-capture`, `idea-generate`, and
  `defect-generate`.
- Loading that scaffolded config produces **zero** `RepairNote`s (the template
  already satisfies the required-capabilities set from Milestone 2).
- `make lint` and `go test ./internal/initcmd/...` pass.

---

## Milestone 4 — Actionable error + config-health surface for the API

**Description.** Ensure the generation endpoint never leaks the raw
`... has no template ...` string. With Milestone 1 the endpoint should always
resolve a template, so a resolution error now only occurs for an unknown key —
return it as a stable, actionable error code. Additionally expose the repair
notes so the frontend can show a config-health hint (consumed by
[[defect-generate-missing-template-3-fe]]).

**Files to change.**
- `internal/http/idea_generate.go` — when `resolveIdeaCaptureConfig` errors,
  respond `422` with `apiError("template_unavailable", <actionable message>)`
  instead of the current `500 config_error` carrying the raw string.
- `internal/http/idea_chat.go` — same treatment for the conversational path if
  its resolve ever errors.
- A read endpoint for config health: extend the existing project/config info
  handler (or add `GET /api/p/:project/config/health`) to return
  `{ repairs: [{agent, template_key, reason}] }` from the notes recorded in
  Milestone 2. No secrets (e.g. `auth_token`) are included.

**Acceptance criteria.**
- `POST /ideas/generate` with `type=defect` on a config missing the template
  returns `200` with a proposal (via the Milestone 1 default), **not** `500`.
- If resolution genuinely fails (unknown key), the response is `422` with code
  `template_unavailable` and a human-actionable message; the raw
  `has no template` string never reaches the client.
- The config-health endpoint returns the repair notes for a repaired project and
  an empty list for a clean project, and never includes secret fields.

---

## Milestone 5 — Single source of truth for default templates (consolidation)

**Description.** Remove the risk of three copies of each prompt drifting
(HTTP constants, init template, this project's `lifecycle/config.yaml`).
Establish one embedded source consumed by both the HTTP fallback (Milestone 1)
and config repair (Milestone 2).

**Files to change.**
- `internal/initcmd/` (or a new `internal/config/defaults` package) — hold the
  canonical default template strings and expose a `DefaultGenerationTemplates()
  map[string]string` accessor.
- `internal/http/idea_chat.go` — source `defaultTemplateFor` from that accessor
  rather than local literals.
- `internal/config/config.go` — source repair values from the same accessor.

**Acceptance criteria.**
- There is exactly one canonical definition of each default generation prompt in
  Go; the HTTP fallback and config repair both read from it.
- A test asserts the init template's `defect-generate` and the Go default carry
  the same required `##` sections and label contract (guards against drift) —
  see [[defect-generate-missing-template-4-test]].
- `make lint` and `make test-unit` pass.
