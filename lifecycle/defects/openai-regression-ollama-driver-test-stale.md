---
title: "TestOpenAIRegression_ExistingDriversWork/ollama asserts a driver the backend plan deliberately removed"
type: defect
status: draft
lineage: open-provider-support
parent: lifecycle/tests/open-provider-support-6-test.md
labels:
    - defect
created: "2026-08-25T09:58:00+10:00"
assignees:
    - role: test-developer
      who: agent
---

# TestOpenAIRegression_ExistingDriversWork/ollama asserts a driver the backend plan deliberately removed

## Reproduction Steps

1. From the repository root, run:
   ```
   go test -tags=integration ./tests/integration/... -run TestOpenAIRegression_ExistingDriversWork -v
   ```
2. Observe the `ollama` subtest fails at `config.LoadProject`.

## Expected Behaviour

Per `tests/integration/openai_regression_test.go:18-29`, the test asserts that
a project config with `driver: ollama` continues to **load and validate
without regression (NFR-2)**, alongside `claude-code-cli`, `claude-mediated`,
`codex-cli`, `gemini`, `gemini-cli`, and `shell-stub`.

This no longer matches the approved implementation. `lifecycle/backend-plans/open-provider-support-3-be.md`
Milestone 6 explicitly scopes: *"sunset the native single-shot `ollama` driver
by routing Ollama instances through `openai-compatible`"* and *"Deprecate/remove
the legacy single-shot `OllamaDriver` registration, replacing it with the
`openai-compatible` driver."* `internal/config/config.go:1119-1121`
(commit `4bfca5b7`, "Milestone 3 — Outright Removal of Native Ollama Driver &
Surface") implements this by hard-rejecting `driver: ollama` with a
descriptive migration error. This is the intended, approved behaviour, not an
architecture or standards deviation — no ADR is warranted.

The test itself is stale: it was written against the original NFR-2 wording
before the plan converged on removing the native driver outright, and was
never updated to match.

## Actual Behaviour

```
openai_regression_test.go:67: LoadProject for driver "ollama" failed: project config: agent "test-agent-ollama" uses deprecated driver "ollama"; please configure a provider with driver "openai-compatible" and reference it via provider: <name>
--- FAIL: TestOpenAIRegression_ExistingDriversWork (0.02s)
    --- FAIL: TestOpenAIRegression_ExistingDriversWork/ollama (0.00s)
```

## Suggested Fix

Update `tests/integration/openai_regression_test.go`:
- Remove `"ollama"` from the `drivers` regression list (it is covered instead
  by `TestOpenAIRegression_MixedAgentsConfig`/`TestProviderConfig_LegacyOllamaInstancesMigration`,
  which correctly test migration of legacy `ollama_instances` into `Provider`
  records).
- Add a dedicated assertion (or extend `TestProviderConfig_*`) that
  `driver: ollama` in an agent entry is rejected with the deprecation error
  from `internal/config/config.go:1120`, so the removal itself stays under
  regression coverage.

## Logs / Output

```
=== RUN   TestOpenAIRegression_ExistingDriversWork/ollama
    openai_regression_test.go:67: LoadProject for driver "ollama" failed: project config: agent "test-agent-ollama" uses deprecated driver "ollama"; please configure a provider with driver "openai-compatible" and reference it via provider: <name>
--- FAIL: TestOpenAIRegression_ExistingDriversWork (0.02s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/claude-code-cli (0.00s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/claude-mediated (0.00s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/codex-cli (0.00s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/gemini (0.00s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/gemini-cli (0.00s)
    --- PASS: TestOpenAIRegression_ExistingDriversWork/shell-stub (0.00s)
    --- FAIL: TestOpenAIRegression_ExistingDriversWork/ollama (0.00s)
```
