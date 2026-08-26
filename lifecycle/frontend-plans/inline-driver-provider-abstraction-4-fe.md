---
title: Inline Conversational Driver Provider Abstraction — Frontend Plan
type: plan-frontend
status: approved
lineage: inline-driver-provider-abstraction
parent: lifecycle/requirements/inline-driver-provider-abstraction-2.md
created: "2026-08-25T14:30:00+10:00"
release: KC-Release6
labels:
    - driver
    - provider
    - inline
    - ideachat
    - frontend
---

# Frontend Plan: Inline Conversational Driver Provider Abstraction

Parent requirement: [[inline-driver-provider-abstraction]]
(`lifecycle/requirements/inline-driver-provider-abstraction-2.md`). Companion
plans: [[inline-driver-provider-abstraction]] backend (`-3-be`) and test
(`-5-test`).

## Scope: no net-new UI in v1

This is a **backend-only** change. The requirement is explicit
(§Goals/Non-goals):

> - **A UI for choosing the inline provider; selection is config-only in v1.**
>   — Non-goal.
> - Changing how Provider records are stored, validated, discovered, or surfaced
>   in the UI — that is [[provider-model-for-agents]], already done. — Non-goal.

Provider selection for inline agents is made entirely in
`lifecycle/config.yaml` (`agents[].provider`) and app config
(`~/.kaos-control/config.yaml` `providers:`). The existing Idea Capture chat and
the idea/defect/doc generate flows call the same endpoints
(`POST /ideas/converse`, `POST /ideas/generate`) with the same request/response
shapes; whether a completer is backed by the Claude CLI or an OpenAI-compatible
provider is invisible to the client by design.

Therefore this plan ships **no new components, stores, routes, or props**. Its
milestones are confined to *confirming non-regression* and *guarding the secret
boundary* on the frontend, so the change is safe to release without UI work.

## Architecture conformance

Conforms to [[go-vue]] (Vue 3 SPA, embedded) and [[secrets-handling]]. The
critical frontend-facing rule: the SPA must **never** receive provider
`api_key`/credential material. Provider records are already masked server-side
(`internal/http/providers.go:maskedProviders` sends `has_api_key` + `"***"`),
and the inline endpoints return only `reply`/`preview`/`artifact_path` — no
provider block. This plan adds no code path that would surface a secret; it only
verifies that invariant still holds.

---

## Milestone 1 — Confirm inline chat/generate UI is provider-agnostic (no change)

**Description.** Verify the Idea Capture conversational UI and the idea/defect/
doc generation UI behave identically regardless of which completer backs the
inline agent — because the request/response contracts are unchanged.

**Files to change.** None expected. Review only:
- `web/src/` idea-capture chat view/component and the generate/preview
  view(s) that call `POST /ideas/converse` and `POST /ideas/generate`.

**Acceptance criteria.**
- [ ] The converse and generate request/response shapes consumed by the SPA are
      unchanged (`session_id`, `reply`, `status`, `preview`, `artifact_path`
      for converse; `slug`/`title`/`labels`/`body`/`frontmatter`/`target_dir`
      for generate).
- [ ] With an inline agent pointed at an OpenAI-compatible provider (backend
      `-3-be`), the chat and generate flows produce the same UI states and the
      same preview/accept behaviour as with the Claude CLI default.
- [ ] No frontend source change is required to support provider-backed inline
      completers (documented outcome; if any change *is* found necessary, raise
      it as a defect against this lineage rather than silently expanding scope).

## Milestone 2 — Secret boundary + error surfacing check (no change expected)

**Description.** Confirm the SPA never receives provider credentials and that an
inline provider failure (backend FR-7) renders as an actionable error, reusing
the existing error-display path.

**Files to change.** None expected. Review only:
- The error-handling branch of the converse/generate views (they already render
  API `error` bodies, e.g. `template_unavailable`, `llm_error`,
  `generate_error`).

**Acceptance criteria.**
- [ ] No API response consumed by the inline UI contains `api_key` or any
      credential material (network-tab / response-shape verification); provider
      records, where shown elsewhere, remain masked (`has_api_key` / `"***"`).
- [ ] A non-2xx / unreachable inline provider surfaces the backend's returned
      error through the existing error UI (no blank state, no leaked internals),
      consistent with today's `llm_error` / `generate_error` handling.
- [ ] No new secret-bearing field is added to any inline endpoint response.

---

## Out of scope

Any inline-provider picker, provider status badge on inline agents, or config
editing UI — deferred beyond v1 per the requirement. Provider CRUD/masking UI is
owned by [[provider-model-for-agents]] and unchanged here.
