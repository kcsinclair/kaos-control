---
title: 'Generate endpoint returns 500 instead of 400 for short input when agent is not configured'
type: defect
status: done
lineage: prompt-to-idea
parent: lifecycle/tests/prompt-to-idea-14.md
labels:
    - defect
    - backend
assignees:
    - role: backend-developer
      who: agent
release: KC-OG-Sprint
---

# Generate endpoint returns 500 instead of 400 for short input when agent is not configured

## Reproduction Steps

1. Start a test environment with no `idea-capture`/`idea-generate` agent configured in `lifecycle/config.yaml` (e.g. the bare `newTestEnv(t, nil)` helper used in integration tests).
2. Log in as `admin@test.local`.
3. `POST /api/p/testproject/ideas/generate` with body `{"input": "hi"}` (one word).
4. Observe the HTTP response status and body.
5. Repeat with body `{"input": "fix bug"}` (two words).

## Expected Behaviour

Both requests should return **HTTP 400** with a JSON body containing an `"error"` field and a user-facing message such as `"Please provide at least 5 words describing your idea."`. Input length validation must fire **before** any agent configuration is resolved.

## Actual Behaviour

Both requests return **HTTP 500** with:

```json
{"error":{"code":"config_error","message":"idea-capture agent not configured"}}
```

The `resolveIdeaCaptureConfig()` call in `internal/http/idea_generate.go` (line 55) runs before `ideachat.Generate()` is called (line 73). Because `ErrInputTooShort` is only raised inside `ideachat.Generate()`, a missing agent config causes a 500 to be returned before the word-count check is ever reached.

## Logs / Output

```
=== RUN   TestIdeaGenerate_TooShort
    idea_generate_test.go:130: expected status 400, got 500: {"error":{"code":"config_error","message":"idea-capture agent not configured"}}
--- FAIL: TestIdeaGenerate_TooShort (0.11s)

=== RUN   TestIdeaGenerate_FewWords
    idea_generate_test.go:172: expected status 400, got 500: {"error":{"code":"config_error","message":"idea-capture agent not configured"}}
--- FAIL: TestIdeaGenerate_FewWords (0.11s)

FAIL    github.com/kaos-control/kaos-control/tests/integration  0.558s
```

## Fix guidance

In `internal/http/idea_generate.go`, add word-count validation immediately after the empty-input check (around line 43), before `resolveIdeaCaptureConfig` is called. The word-count helper `ideachat.CountWords` (or an equivalent inline check using `strings.Fields`) should be used to enforce the 5-word minimum and return 400 with `apiError("input_too_short", "Please provide at least 5 words describing your idea.")`.
