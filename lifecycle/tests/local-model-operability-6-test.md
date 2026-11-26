---
title: "Local-Model Operability and UI Error Surfacing — Test Suite"
type: test
status: draft
lineage: local-model-operability
parent: lifecycle/test-plans/local-model-operability-5-test.md
created: "2026-08-25T12:13:40+10:00"
---

# Local-Model Operability and UI Error Surfacing — Test Suite

## Overview

This artifact documents the integration tests added by the test-developer role for [[local-model-operability-5-test]]. Milestones 1, 3, and part of 4 of that plan describe **unit** tests co-located with the implementation (`internal/agent/*_test.go`, `internal/config/config_test.go`, `internal/initcmd/*_test.go`) — those are outside `test-developer`'s write scope (`tests/**`, `lifecycle/tests/`, `lifecycle/architecture/decisions/`) and were already implemented by the backend-developer alongside FR-1/FR-2/FR-3/FR-5 (see `internal/agent/openai_preflight_availability_test.go`, `internal/agent/errors_test.go`, `internal/agent/openai_warmup_test.go`, `internal/agent/prompt_defaults_test.go`, `internal/config/config_test.go`). This suite adds the **integration**-level coverage that plan calls out and that unit tests alone can't observe: full HTTP-API + git-backed-project behaviour, and Vitest coverage for the Vue components/stores.

## Test Files

| File | Milestone / Area |
|------|-------------------|
| `tests/integration/local_model_fastfail_test.go` | M2 — Manager Fast-Fail & Lineage Lock Release (full HTTP API) |
| `tests/integration/local_model_prompt_fallback_test.go` | M4 — Local-model prompt fallback reaches the provider over HTTP |
| `tests/web/RunFailureBanner.localModels.test.ts` | M5 — `RunFailureBanner.vue`, all 9 structured failure codes |
| `tests/web/AgentRunningBanner.warmup.test.ts` | M5 — `AgentRunningBanner.vue` warmup badge |
| `tests/web/agentsStore.localModelFailure.test.ts` | M5 — `agents` Pinia store: failure + warmup WS events |

## Milestone 2 — Manager Fast-Fail & Lineage Lock Release (`local_model_fastfail_test.go`)

`internal/agent/openai_preflight_availability_test.go` (`TestStartRun_Preflight`) already exercises fast-fail directly against `Manager`. These tests drive the same two scenarios through the full HTTP API + git-backed project harness (`openAIAgentTestEnv` from `openai_agent_run_test.go`), adding assertions the plan calls out that aren't observable from inside the `agent` package: request wall-clock time, and zero new git commits.

- **TestOpenAIAgentRun_ModelNotFound_FastFail** — provider's `/v1/models` omits the configured model. Asserts: response in < 3s, `409` with `model_not_found` in the error message, lineage lock free, no new git commit, target artifact status unchanged, and a `failed` run record in the index with `failure_reason=model_not_found`.
- **TestOpenAIAgentRun_EndpointUnreachable_FastFail** — provider connection refused (mock server closed before the request). Same assertion set with `failure_reason=endpoint_unreachable`.

Both share the `assertFastFail` helper in that file.

## Milestone 4 — Local-Model Prompt Fallback Reaches the Provider (`local_model_prompt_fallback_test.go`)

`internal/agent/prompt_defaults_test.go` already covers `LocalModelPromptDefaults` selection and its <1200-token budget at the `Manager` level. This test proves the fallback is wired end-to-end: an `openai-compatible` agent named `backend-developer` (added to the shared `openAIAgentCfgTemplate` in `openai_agent_run_test.go`, deliberately without a `prompt_templates` entry for its role) is run through the full HTTP API, and the recorded `/v1/chat/completions` request bodies are inspected for the first line of `agent.PromptDefaultBackendDeveloper` — confirming the fallback prompt is what actually reaches the wire, not just what `Manager.StartRun` selects internally.

- **TestOpenAIAgentRun_LocalModelPromptFallback_ReachesProvider**

## Milestone 5 — Frontend Store & Component Tests

- **`RunFailureBanner.localModels.test.ts`** — table-driven over all 9 structured `failure_reason` codes from `web/src/lib/failureReasons.ts` (`tools_unsupported`, `model_not_found`, `model_unloaded`, `endpoint_unreachable`, `context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`, `auth_error`, `timeout`). Per code: heading + explanatory body render; backend-supplied remediation renders as numbered `<code>`-annotated steps, falling back to the static list when absent; Provider Settings (`/p/:project/settings/providers`) and Agent Config (`/p/:project/agents`) links render (suppressed when `denialCount` is set); `role="alert"` is present. Also asserts the banner never renders a raw secret pattern, only the `"***"` marker the backend already sent, and that `context_window_exceeded`/`turn_token_ceiling` intentionally share copy. Existing `tests/web/RunFailureBanner.test.ts` (legacy `permission_mode_default`/`precheck_timeout` codes) is untouched and this file is additive.
- **`AgentRunningBanner.warmup.test.ts`** — mounts with real Pinia `queue` + `agents` stores (matching the pattern in `ProjectQueuePanel.realtime.test.ts`): the `.warmup-badge` renders with the live `warmup_message` when the matching run row has `warmup_state: 'model_loading'` or `'warming_up'`, falls back to the default label text when no message is set, disappears once the run transitions to `'generating'`, and stays hidden for a non-warming run.
- **`agentsStore.localModelFailure.test.ts`** — extends the pattern in `agentsStore.precheckFailure.test.ts` to the new fields: an `agent.failed` event with `failure_reason`/`remediation`/`error_details` updates the matching `AgentRunRow` (including passthrough of an already-masked `"***"` value, and `null` defaults when absent); `agent.status`(`model_loading`)/`agent.progress`(`warming_up`→`generating`) events update `warmup_state`/`warmup_message` on the matching running run, including the pending-warmup-before-`agent.started` race; `warmup_state`/`warmup_message` are cleared when the run finishes; an `agent.status` event for an unrelated `target_path` does not mutate other runs.

## Milestone 6 — Verification

Ran locally against this branch:

- `go vet -tags=integration ./tests/...` — clean.
- `go test -tags=integration ./tests/integration/ -run "TestOpenAIAgentRun_ModelNotFound_FastFail|TestOpenAIAgentRun_EndpointUnreachable_FastFail|TestOpenAIAgentRun_LocalModelPromptFallback_ReachesProvider"` — pass.
- `go test -tags=integration ./tests/integration/ -run "TestOpenAI"` — all pass except the pre-existing `TestOpenAIRegression_ExistingDriversWork/ollama` failure (see Open Issues below; not caused by this change).
- `make lint` — pre-existing `gosec` false-positive on `internal/agent/errors.go:23` (`FailureReasonTurnTokenCeiling` constant name flagged as a possible hardcoded credential); not touched by this change.
- `make test-unit` — all packages pass.
- `cd tests/web && pnpm vitest run` — 106/108 files pass; 2 pre-existing failures unrelated to this change (see Open Issues below). The 3 new files added here (69 tests) all pass.
- `cd web && pnpm build` — succeeds.

### Open issues found during verification (not part of this plan's scope; flagged for triage)

1. **`web/src/components/agent/RunDetailModal.vue` uses `computed` without importing it from `vue`** (only `ref, watch, onMounted, onBeforeUnmount` are imported). This throws `ReferenceError: computed is not defined` on every mount and breaks all 22 tests in `tests/web/RunDetailModal.test.ts`. Neither `make lint` nor `make build-web`/`pnpm build` catch it (no type-check step in the build script) — only `pnpm test` does. This is a real runtime bug in the shipped app (introduced in the "Milestone 4 — Live Warmup & Weight Loading Visual Indicators" frontend work), outside `test-developer`'s write scope (`web/src`) to fix directly.
2. **`tests/web/AppSidebar.test.ts`** expects a nav item labelled "Ollama" that no longer exists in `AppSidebar.vue` (superseded by "Providers", consistent with the native Ollama driver's removal in commit `4bfca5b7`, "Outright Removal of Native Ollama Driver & Surface"). Stale test, unrelated to this plan.
3. **`tests/integration/openai_regression_test.go`'s `TestOpenAIRegression_ExistingDriversWork/ollama` subtest** still asserts that the legacy `ollama` driver value loads without error, contradicting the deliberate hard-rejection added in commit `4bfca5b7` (`internal/config/config.go`: `agent %q uses deprecated driver "ollama"`). The companion artifact `lifecycle/tests/open-provider-support-6-test.md` still documents this as NFR-2 ("legacy `ollama` configurations continue to load and validate without regression"), which is now stale. Neither this test's assertion nor its artifact were updated when the driver was removed.

None of the three are caused by or fixed in this change; they predate it and are outside this plan's write scope. Recommend routing to the backend/frontend-developer roles (or qa, to raise formal defects) separately.
