---
title: Ollama (local LLMs)
type: feature
status: approved
lineage: feature-ollama-local-llms
created: "2026-08-21T15:11:00+10:00"
summary: Configure multiple Ollama endpoints and run agents against local models alongside Claude.
function: Agents
labels:
    - feature
    - agent
    - ollama
related_to:
    - lifecycle/ideas/ollama-agent-support.md
    - lifecycle/requirements/ollama-agent-support-2.md
    - lifecycle/ideas/ollama-claude-code-driver.md
    - lifecycle/requirements/ollama-claude-code-driver-2.md
---

# Ollama (local LLMs)

Run agents against locally-hosted models instead of a hosted Claude
subscription.

## What it does

- **Multi-instance management.** Configure multiple Ollama endpoints
  app-wide (CRUD via `/api/ollama/instances`); per-instance health + model
  listing.
- **Agent driver.** Use any Ollama instance + model as an agent driver
  alongside Claude.

Reachable at **Settings → Ollama instances**; see also
[[agent-orchestration]] for how drivers are selected per agent.
