---
title: Add handling for OpenRouter token exceeded error
type: defect
status: draft
lineage: openrouter-token-exceeded-error-handling
created: "2026-09-04T12:51:51+10:00"
priority: normal
labels:
    - defect
    - open-provider-support
---

# Add handling for OpenRouter token exceeded error

## Reproduction Steps

1. Execute a run that uses the **openrouter** provider.
2. Initiate enough concurrent/in‑flight requests to exceed the available credits.
3. Observe the run completing with a STDERR message:
   `provider "openrouter" returned HTTP 402: This request would exceed your available credits …`.

## Expected Behaviour

- The system should detect the HTTP 402 “credits exceeded” response.
- It should handle the situation gracefully, e.g., by:
  * Queuing or throttling further requests until in‑flight requests settle.
  * Providing a clear, user‑friendly error message suggesting to add credits or wait.
  * Optionally marking the run as failed with a specific error code rather than just dumping to STDERR.

## Actual Behaviour

- The run finishes, but an unhandled error message is printed to STDERR.
- No retry, throttling, or user guidance is performed; the error is merely logged.
