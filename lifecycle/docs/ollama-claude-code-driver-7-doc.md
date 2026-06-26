---
title: Require documentation on how to configure this and any caveats and limitations
type: doc
status: done
lineage: ollama-claude-code-driver
created: "2026-06-26T18:27:18+10:00"
parent: lifecycle/requirements/ollama-claude-code-driver-2.md
output: docs/ollama-claude-code-driver.md
---

Require documentation on how to configure this and any caveats and limitations.

## Produced

Documentation written to `docs/ollama-claude-code-driver.md`. Covers:

- Why `claude-env` vs the `ollama` driver (comparison table)
- How the driver works internally (env injection, arg vector reuse, process lifecycle)
- Configuration fields (`base_url`, `auth_token`, `model`) with two complete examples
- Ollama setup walkthrough (install, pull model, verify shim, configure)
- Config validation errors and their messages
- Secret hygiene behaviour (masking, log exclusion)
- Runtime behaviour (concurrency, cancellation, endpoint unreachable, env precedence)
- Caveats and limitations (model tool-use fidelity, Ollama-only testing, v1 config-file-only, no Ollama model management, bypass permissions only, context window)
- Troubleshooting section (URL validation, connection refused, model not found, broken tool calls)
