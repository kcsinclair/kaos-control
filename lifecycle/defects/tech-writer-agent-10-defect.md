---
title: Artifact detail view renders 'project not found' when docs-stage fixtures are active
type: defect
status: blocked
lineage: tech-writer-agent
parent: lifecycle/tests/tech-writer-agent-6-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Artifact detail view renders "project not found" when docs-stage fixtures are active

## Reproduction Steps

1. Run the E2E harness with the fixture `config.yaml` that includes the `docs` stage and `tech-writer` agent (`tests/e2e/fixtures/lifecycle/config.yaml`).
2. Log in as `admin@kaos-e2e.local`.
3. Navigate to any artifact detail URL, e.g.:
   - `/p/testproject/artifacts/lifecycle/requirements/smoke-req-done.md`
   - `/p/testproject/artifacts/lifecycle/requirements/smoke-req-01.md`
   - `/p/testproject/artifacts/lifecycle/docs/smoke-doc-approved.md`
4. Wait up to 10 s for the page to load.

## Expected Behaviour

The artifact detail component renders the artifact's content and metadata, including a `.status-badge` or `[data-status]` element.

## Actual Behaviour

The main content area displays `"project not found: testproject"` instead of the artifact. The sidebar shows the project navigation correctly (confirming the project IS registered), but the artifact view component fails to load its data.

The ARIA tree for the main section is:
```
- main:
  - button "← artifacts"
  - text: "project not found: testproject"
```

## Logs / Output

```
Error: expect(locator).toBeVisible() failed
Locator: locator('.status-badge, [data-status]').first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found
```

This blocks all of Flow 06 (TC1, TC2) and Flow 08 (TC1, TC2), preventing verification of the "Request docs" button and the Queue Work button.

**Failing tests:** `Flow 06 TC1`, `Flow 06 TC2` (`tests/e2e/flows/06-doc-request.spec.ts`); `Flow 08 TC1`, `Flow 08 TC2` (`tests/e2e/flows/08-doc-queue.spec.ts`).

## Frontend Investigation Findings

After thorough investigation of `web/src/**`, the frontend developer has determined:

1. **Root cause is backend, not frontend.** The error `"project not found: testproject"` originates from `internal/http/project_ctx.go`. It is returned when `s.getProject("testproject")` finds no entry in `s.projects`. This happens because `project.Open()` fails at server startup: the fixture `lifecycle/config.yaml` configures the `tech-writer` agent with `roles:` (plural key), which the old config parser did not accept, causing the project to be silently skipped on startup.

2. **Backend fix was already applied.** Commit `ffaba80d fix(config): accept both "role:" and "roles:" YAML keys for AgentConfig` resolves the project startup failure. After this fix the project opens correctly, all artifacts are indexed, and artifact detail API calls succeed.

3. **Frontend code is correct for all test scenarios.** The `ArtifactEditorView`, `QueueWorkButton`, `StatusDropdown`, `FrontmatterPanel`, and `agentForArtifact` composable already handle `doc` type artifacts, `approved` status Queue Work visibility, `done` status Request docs visibility, and `tech-writer` agent routing exactly as the failing tests require. `vue-tsc --noEmit` and `pnpm build` both pass with zero errors.

4. **No frontend code change can fix this.** The error occurs because the project does not exist in the backend's live map. Retry logic or better error messages would not make the E2E tests pass; they require the artifact to actually load.

## Open Questions

1. **Should this defect be closed or re-assigned?** The root cause was a backend bug in `AgentConfig` YAML parsing, resolved by commit `ffaba80d`. Now that the backend fix is in place, the failing flows (06 TC1/TC2, 08 TC1/TC2) should pass without any frontend changes. Please confirm whether this defect can be marked `done` (resolved by the backend fix) or whether it needs re-assignment to the backend developer who applied the fix.

2. **Is a rebuilt binary required for E2E verification?** The E2E harness (`tests/e2e/harness/kaos-control.ts`) only rebuilds `dist/kaos-control` if the binary is absent. If the binary pre-dates `ffaba80d`, the tests will still fail. Should the test developer ensure a fresh build before verifying this defect closes?
