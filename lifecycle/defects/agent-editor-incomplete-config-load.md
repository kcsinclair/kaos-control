---
title: Agent Editor Does Not Load All Data from config.yaml
type: defect
status: draft
lineage: agent-editor-incomplete-config-load
created: "2026-05-22T10:14:42+10:00"
priority: normal
labels:
    - defect
    - agent
    - editor
    - frontend
---

# Agent Editor Does Not Load All Data from config.yaml

## Reproduction Steps

1. Open the application and navigate to the agent configuration section.
2. Select an existing agent to edit.
3. Observe the fields populated in the editor form.

## Expected Behaviour

All fields defined for the agent in `config.yaml` (e.g. prompt template, allowed_write_paths, model, and any other agent-specific configuration) should be loaded and displayed in the editor when opening an agent for editing.

## Actual Behaviour

The agent editor only loads a partial subset of the agent's configuration data from `config.yaml`. Some fields are missing or blank despite being present in the config file, meaning edits made via the UI may overwrite or lose data that was not surfaced in the form.
