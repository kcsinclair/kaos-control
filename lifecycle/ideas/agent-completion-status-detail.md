---
title: Richer Agent Completion Status
type: idea
status: in-qa
lineage: agent-completion-status-detail
created: "2026-04-27T13:56:41+10:00"
priority: normal
labels:
    - agent
    - enhancement
    - workflow
    - artefacts
release: KC-Release1
---

relates-to: [[ideas/improved-agent-handling]]

When an agent run finishes, the current status update is too sparse — it simply marks the run as done. The completion message should instead reflect what the agent actually accomplished: whether it produced new artifacts, updated existing ones, or (for the QA agent) ran tests and what the outcomes were.

Artifact changes should be surfaced with clear distinction between produced vs updated, listing the relevant file paths. For example: `ARTIFACTS PRODUCED: lifecycle/frontend-plans/prompt-to-idea-12-fe.md` or `ARTIFACTS UPDATED: lifecycle/requirements/login-2.md`. This gives developers immediate visibility into what changed without having to inspect git diff or re-read the run log.

For the QA agent specifically, if all tests pass the completion status should include a summary line (e.g. `ALL TESTS PASSED`) along with the relevant output line from the test runner. If tests fail, the failing test names and error snippets should be surfaced in the status update so the developer can act without opening the full run log.

```
{"type":"result","subtype":"success","is_error":false,"duration_ms":35804,"duration_api_ms":31133,"num_turns":7,"result":"All tests pass cleanly on a fresh run. Here is the QA summary:\n\n---\n\n## QA Report — `ignore-readme-files-in-lifecycle-dir`\n\n**Artifact:** `lifecycle/tests/ignore-readme-files-in-lifecycle-dir-5-test.md`\n\n### Results\n\n| Test suite | Tests run | Passed | Failed |\n|---|---|---|---|\n| `internal/config/` — `TestShouldIgnore` | 8 subtests | 8 | 0 |\n| `internal/config/` — `TestLoadProjectIgnoreField` | 3 subtests | 3 | 0 |\n| `tests/integration/` — `TestIgnorePatterns_StartupScan` | 1 | 1 | 0 |\n| `tests/integration/` — `TestIgnorePatterns_WatcherSkipsIgnored` | 1 | 1 | 0 |\n| `tests/integration/` — `TestIgnorePatterns_IndexFileRejectsIgnored` | 1 | 1 | 0 |\n| `tests/integration/` — `TestIgnorePatterns_APIExcludesIgnored` | 1 | 1 | 0 |\n\n**All 15 tests pass.** No defect artifacts were raised.\n\nThe feature is ready to progress to `approved`.","stop_reason":"end_turn","session_id":"b6498194-b214-49cd-8bc8-884a8f922b3f","total_cost_usd":0.14507025,"usage":{"input_tokens":7,"cache_creation_input_tokens":24961,"cache_read_input_tokens":119685,"output_tokens":1036,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":24961,"ephemeral_5m_input_tokens":0},"inference_geo":"","iterations":[{"input_tokens":1,"output_tokens":328,"cache_read_input_tokens":25355,"cache_creation_input_tokens":2041,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":2041},"type":"message"}],"speed":"standard"},"modelUsage":{"claude-sonnet-4-6":{"inputTokens":7,"outputTokens":1036,"cacheReadInputTokens":119685,"cacheCreationInputTokens":24961,"webSearchRequests":0,"costUSD":0.14507025,"contextWindow":200000,"maxOutputTokens":32000}},"permission_denials":[],"terminal_reason":"completed","fast_mode_state":"off","uuid":"7ae2be96-2ae1-4ab0-975f-b9cebd47f60b"}
```
