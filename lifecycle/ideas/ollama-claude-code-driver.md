---
title: Ollama Claude code driver
type: idea
status: draft
lineage: ollama-claude-code-driver
priority: high
labels:
    - agent
    - agent-runner
    - driver
    - ollama
    - integration
    - portability
release: KC-Release4
---

## Raw Idea

## Raw Idea

Create a new driver so we can use Claude code with environment variables.  This will provide the ability to use Claude with local AI 
 ```
export ANTHROPIC_AUTH_TOKEN="ollama"
export ANTHROPIC_BASE_URL=http://leia.packsin.com:11434
claude --model gemma4:26b-a4b-it-qat

ollama pull gemma4:26b-mlx
export ANTHROPIC_AUTH_TOKEN="ollama"
export ANTHROPIC_BASE_URL=http://localhost:11434
claude --model gemma4:26b-mlx
```

## Idea

Add a new Claude Code driver that routes API calls through environment variable overrides (`ANTHROPIC_AUTH_TOKEN` and `ANTHROPIC_BASE_URL`), enabling locally-hosted Ollama models (or any OpenAI-compatible endpoint) to be used as a drop-in replacement for the Anthropic API. This allows the agent runner to work entirely offline or against self-hosted inference without any code changes — just environment configuration.

The driver would accept a `--model` flag (or equivalent config field) pointing to an Ollama model tag (e.g. `gemma4:26b-mlx` or `gemma4:26b-a4b-it-qat`), and set the auth token to the literal string `"ollama"` as required by the Ollama Claude-compatible shim. The base URL would point to the local or network Ollama instance (e.g. `http://localhost:11434` or a LAN host).

This unlocks cost-free local development runs, air-gapped deployments, and experimentation with community-fine-tuned models — without forking the existing ClaudeCodeDriver or touching the supervisor. A config-level `driver: ollama` or `driver: claude-env` field in `lifecycle/config.yaml` (or per-agent override) would select the new driver at runtime.
