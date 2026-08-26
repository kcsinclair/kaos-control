---
title: Ollama (local LLMs)
type: feature
status: done
lineage: feature-ollama-local-llms
created: "2026-08-21T15:11:00+10:00"
summary: 'Superseded: single-shot Ollama driver and instance manager, replaced by the Provider entity and the openai-compatible agent driver.'
function: Agents
labels:
    - feature
    - agent
    - ollama
    - superseded
related_to:
    - lifecycle/ideas/ollama-agent-support.md
    - lifecycle/requirements/ollama-agent-support-2.md
    - lifecycle/ideas/ollama-claude-code-driver.md
    - lifecycle/requirements/ollama-claude-code-driver-2.md
    - lifecycle/ideas/open-provider-support.md
parent: lifecycle/docs/open-provider-support-7-doc.md
---

# Ollama (local LLMs)

**Superseded by [[provider-management]].** This page is kept for history;
the capability described below no longer exists as such.

Originally: run agents against locally-hosted Ollama models via a
dedicated `ollama_instances` config block, `/api/ollama/instances` CRUD
API, and an `OllamaSettingsView` settings page, using a native `ollama`
driver that issued single-shot `/api/chat` requests — no tool use, no file
edits, no multi-turn loop.

## What changed

The [[open-provider-support]] epic removed the native `ollama` driver, the
`/api/ollama/instances` API, and `OllamaSettingsView` outright — Ollama
already exposes an OpenAI-compatible `/v1/` endpoint, so it needs no
special-casing. On startup, any existing `ollama_instances` entries are
migrated automatically into `providers` records
(`driver: openai-compatible`), so registered instances (e.g. `Loki` →
`http://leia.packsin.com:11434`) carry over with no operator action.
Agents pointed at Ollama now get the **full agentic loop** (tool use, file
edits, streamed progress, run logging) that the old driver never had.

See [[provider-management]] and
[open-provider-support](../../docs/open-provider-support.md) for the
current feature.
