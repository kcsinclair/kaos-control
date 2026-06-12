---
title: AgentConfigForm.vue is missing claude-mediated, gemini, and gemini-cli driver options
type: defect
status: done
lineage: agent-config-form-missing-drivers
created: "2026-05-22T18:30:00+10:00"
labels:
    - defect
    - frontend
    - agent
release: KC-Release3
assignees:
    - role: frontend-developer
      who: agent
---

# AgentConfigForm.vue is missing claude-mediated, gemini, and gemini-cli driver options

## Reproduction Steps

1. Open kaos-control in the browser and navigate to **Agents → New Agent**
   (or edit any existing agent).
2. Inspect the **Driver** radio group.

## Expected Behaviour

The radio group should list every driver registered in the backend
([internal/agent/agent.go:405-412](internal/agent/agent.go#L405-L412)) that
is appropriate for end-user configuration:

- `claude-code-cli`
- `claude-mediated`
- `codex-cli` (landing via PR #9)
- `ollama`
- `gemini`
- `gemini-cli`

(`shell-stub` is test-only and should stay hidden.)

## Actual Behaviour

The form's `driver` union type and the radio template only include
`claude-code-cli`, `codex-cli`, and `ollama`. Users editing an agent
whose YAML already specifies `claude-mediated`, `gemini`, or
`gemini-cli` cannot select that driver from the UI; choosing any
radio overwrites the YAML value on save.

Affected lines in [web/src/components/agent/AgentConfigForm.vue](web/src/components/agent/AgentConfigForm.vue):

- Line 12 — `driver` prop type union
- Lines 41-42 — `driver` ref type union
- Lines 101-106 — validation branches (need cases for new drivers)
- Lines 207-220 — radio group template
- Lines 224, 232 — conditional model-field rendering (claude-mediated and gemini-cli need a model input; gemini needs model + a hint about `GEMINI_API_KEY`)

## Logs / Output

N/A — UI-only defect.

## Notes

The `gemini` (REST API) driver additionally needs the `GEMINI_API_KEY`
environment variable; consider surfacing a hint in the form when that
driver is selected.

Deferred until after PR #9 (`codex-agent-support`) merges to main and
main is merged into kc-dev, to avoid an unnecessary merge conflict in
this file.
