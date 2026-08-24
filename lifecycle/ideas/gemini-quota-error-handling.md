---
title: Handle Gemini Quota Errors Gracefully
type: idea
status: draft
lineage: gemini-quota-error-handling
created: "2026-08-25T09:53:37+10:00"
priority: normal
labels:
    - agent-runner
    - provider
    - queue
    - reliability
    - driver
---

# Handle Gemini Quota Errors Gracefully

Google Gemini returns a distinct error message when an individual quota is reached: "Error: Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 2h22m22s." The agent runner currently does not recognise this format, so the error is likely treated as a generic failure rather than a recoverable quota exhaustion event.

The driver should detect this specific error pattern (and similar Gemini quota messages) and translate it into a structured quota-exceeded signal. When this signal is raised, the run queue should pause automatically for the provider, surfacing the reset time where available, rather than retrying immediately or failing the run outright.

This mirrors any existing quota-pause logic for other providers (e.g. Ollama rate limits) and ensures Gemini quota exhaustion is handled as a first-class, recoverable condition rather than an unexpected crash.
