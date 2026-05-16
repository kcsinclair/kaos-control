---
title: "Test fix: stub-agent prompt_templates and TC2 regex (defects 7 & 8)"
type: test
status: draft
lineage: artefacts-agent-run-count-column
parent: lifecycle/defects/artefacts-agent-run-count-column-7-defect.md
---

# Test fix: stub-agent prompt_templates and TC2 regex (defects 7 & 8)

Addresses two defects that prevented the Flow 10 E2E suite from running:

- **Defect 7** (`artefacts-agent-run-count-column-7-defect.md`): stub-agent in the E2E fixture config lacked a `prompt_templates` block, causing every `triggerRun()` call to return HTTP 409. TC1, TC3, and TC4 all failed at their first run trigger.
- **Defect 8** (`artefacts-agent-run-count-column-8-defect.md`): TC2's Runs header locator used a case-sensitive regex `/^Runs$/` against `element.innerText`, which Chrome renders as `"RUNS"` after applying `text-transform: uppercase` from `ArtifactListView.vue`. The locator matched nothing.

## Changes Made

### `tests/e2e/fixtures/lifecycle/config.yaml`

Added `prompt_templates` to the `stub-agent` entry:

```yaml
prompt_templates:
  product-owner: "Process {target_path}"
```

This satisfies the `ag.PromptTemplates[role]` lookup in `internal/agent/agent.go:StartRun()` and allows TC1, TC3, and TC4 to POST runs successfully.

### `tests/e2e/flows/10-artefact-run-count-column.spec.ts`

Changed both occurrences of the Runs header locator regex from case-sensitive `/^Runs$/` to case-insensitive `/^runs$/i` (lines ~115 and ~152). `innerText` reflects CSS-transformed uppercase; the `/i` flag makes the filter match regardless of case.

## Scenarios Now Passing

All four TC cases in `tests/e2e/flows/10-artefact-run-count-column.spec.ts` should now pass:

- **TC1** — Runs column present, positioned after Type and before Created; rc-idea-a shows 2, rc-idea-b shows 1, rc-idea-c shows 0.
- **TC2** — Runs header is clickable for ascending then descending sort.
- **TC3** — `agent-status-pill[data-status="running"]` appears while stub run is active and disappears on completion.
- **TC4** — Run count increments without page reload on `agent.finished` WebSocket event.

## Test Files

| File | Type | Command |
|------|------|---------|
| `tests/e2e/flows/10-artefact-run-count-column.spec.ts` | Playwright E2E | `cd tests/e2e && pnpm exec playwright test flows/10-artefact-run-count-column.spec.ts --reporter=list` |
| `tests/e2e/fixtures/lifecycle/config.yaml` | E2E fixture config | (loaded automatically by the E2E test environment) |
