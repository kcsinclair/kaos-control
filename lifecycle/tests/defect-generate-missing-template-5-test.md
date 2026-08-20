---
created: "2026-08-15T10:23:32+10:00"
title: "Integration Tests — Defect Generate Template Fallback"
type: test
status: approved
lineage: defect-generate-missing-template
parent: lifecycle/test-plans/defect-generate-missing-template-4-test.md
---

# Integration Tests — Defect Generate Template Fallback

Covers Milestone 4 (backend integration) and Milestone 6 (Playwright e2e) of
the test plan. Verifies "New Defect → Generate" (GitHub issue #16) never
hard-errors with `idea-capture agent has no template "defect-generate"`,
end to end, against a real HTTP server and (for the e2e case) the real
browser UI.

Milestones 1–3 (Go unit tests in `internal/http`, `internal/config`,
`internal/initcmd`) and Milestone 5 (web vitest in `web/src`) are out of
scope for this artifact — they fall under `internal/**` and `web/src/**`,
outside the test-developer role's write scope (`tests/**`,
`lifecycle/tests/**`). Consult the backend/frontend developer run history for
that coverage.

All Go tests carry the `//go:build integration` tag and are gated on
`ANTHROPIC_API_KEY` via `skipIfNoAPIKey` (they drive the real `ideachat.Generate`
LLM call). Run with:

```sh
ANTHROPIC_API_KEY=... go test -tags integration ./tests/integration/... -v -run TestDefectGenerate
```

The e2e spec runs against a built binary with the real `claude` CLI (`make test-e2e`).

---

## Test files

| File | Covers |
|---|---|
| `tests/integration/defect_generate_test.go` | Milestone 4 |
| `tests/e2e/flows/13-defect-generate.spec.ts` | Milestone 6 |

---

## Scenarios covered

### Milestone 4 — Backend integration (`defect_generate_test.go`)

- **TestDefectGenerate_FreshProjectFromInitTemplate** — scaffolds a project via
  `initcmd.Run` (the exact `kaos-control init` template), sanity-checks it loads
  cleanly through `config.LoadProject`, then boots a live HTTP server against
  that scaffolded `lifecycle/config.yaml` and calls
  `POST /api/p/:project/ideas/generate {type: "defect"}`. Asserts `200`, a body
  containing the three required `##` sections (Reproduction Steps, Expected
  Behaviour, Actual Behaviour), the mandatory `defect` label, and
  `target_dir: lifecycle/defects`.
- **TestDefectGenerate_MissingDefectGenerateKeyFallsBack** — a config with the
  `idea-capture` agent present but only an `idea-capture` prompt template (the
  `defect-generate` key stripped). Asserts the same request still returns `200`
  via the built-in default-template fallback, not `500`, and that the raw
  response body never contains the substring `has no template`.
- **TestDefectGenerate_NoAgentConfiguredFallsBack** — the `newTestEnv` default
  config, which has no `agents:` section at all (no idea-capture agent
  configured). Asserts `200` via the "no agent configured" fallback branch of
  `resolveIdeaCaptureConfig`, and that the raw response never leaks
  `has no template`.

**Not covered here:** the third Milestone 4 acceptance criterion — a genuinely
unresolvable template key returning `422 template_unavailable` — is not
reachable through the public HTTP API. All four known template keys
(`idea-capture`, `idea-generate`, `defect-generate`, `doc-generate`) have a
built-in default in `internal/config/defaults`, so `resolveIdeaCaptureConfig`
never errors for a request the router can actually construct. The test plan
anticipates this ("asserted ... via a handler-level test if no request can
trigger an unknown key") — that case belongs with the Milestone 1 unit tests
in `internal/http`, outside this artifact's scope.

### Milestone 6 — Playwright e2e (`13-defect-generate.spec.ts`)

- **TC1: generating a defect shows a structured preview with no template
  error** — opens the New Defect modal from the artifact list, enters a
  qualifying (≥5-word) bug description, clicks **Generate**, and asserts:
  - the `POST /ideas/generate` call returns `200`;
  - no `.bdm-error` banner is shown, and the modal never renders the raw
    `idea-capture agent has no template "defect-generate"` string;
  - the preview shows the defect title and the three structured sections
    (Reproduction Steps / Expected Behaviour / Actual Behaviour).

  Then clicks **Accept** and asserts the resulting `POST /artifacts` call
  returns `201` with a path matching
  `lifecycle/defects/<slug>.md`, and that the app navigates to that artifact.

  The e2e fixture project (`tests/e2e/fixtures/lifecycle/config.yaml`) has no
  `idea-capture`/`docs-capture` agent configured at all, so this flow already
  exercises the "no agent configured" fallback path in the real browser —
  the same scenario as `TestDefectGenerate_NoAgentConfiguredFallsBack` above,
  driven end to end through the UI instead of directly via HTTP.

---

## Notes

- Verified locally: `TestDefectGenerate_*` compile and skip cleanly with
  `go vet -tags integration ./tests/integration/...` when `ANTHROPIC_API_KEY`
  is unset (matching the existing `skipIfNoAPIKey` convention used throughout
  `tests/integration`). Manually forcing a (non-functional) API key value
  confirmed the resolver/self-repair path is reached correctly (`WARN project
  config: self-repaired missing generation template` logged, request proceeds
  to the LLM call) before failing only on this sandbox's `claude` CLI OAuth
  vs. API-key auth precedence — unrelated to the fix under test.
- `13-defect-generate.spec.ts` was run end to end against a freshly built
  binary (`make build`) in this environment and passed, including the real
  `ideachat.Generate` LLM call via the `claude` CLI.
