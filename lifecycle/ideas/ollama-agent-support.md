---
title: Ollama Agent Support
type: idea
status: approved
lineage: ollama-agent-support
created: "2026-05-06T12:00:56+10:00"
priority: high
labels:
    - feature
    - agent
    - agent-runner
    - backend
    - frontend
---

# Ollama Agent Support

Add support for Ollama as an agent driver alongside the existing Claude Code driver. Users should be able to register local or remote Ollama instances (e.g. `http://localhost:11434` or a remote URL) through a simple UI, browse the list of models available on that instance, and create named agents backed by a chosen model.

The backend needs a new driver implementation that speaks the Ollama REST API (`/api/tags` for model listing, `/api/chat` or `/api/generate` for inference), wired into the existing agent runner and supervisor infrastructure. Instance connection details and credentials (if any) should be stored in project or app config in a consistent way with existing agent configuration.

The frontend requires a settings or configuration panel where users can add/edit/remove Ollama instances, trigger a model refresh, and select a model when creating a new agent. The agent creation flow should surface Ollama as an available driver type next to Claude Code, keeping the experience consistent with the existing agent management UI.
