---
title: New agents added to config.yaml don't appear until the app is restarted
type: defect
status: done
lineage: agent-config-requires-restart
created: "2026-08-11T00:00:00+10:00"
priority: low
release: KC-Release5
labels:
    - defect
    - config
    - agents
    - ux
assignees:
    - role: backend-developer
      who: agent
parent: lifecycle/tests/defect-generate-missing-template-5-test.md
---

# New agents added to config.yaml don't appear until the app is restarted

## Source

GitHub issue [#13](https://github.com/kcsinclair/kaos-control/issues/13) —
reported by **aburow**, 2026-06-23.

## Summary

Adding new agent profiles (e.g. two GPT/codex agents) to the Agents section of
`lifecycle/config.yaml` does not make them available in the running app. The new
config is visible under **System → Config** (the raw text reloads), but the
agents do not appear where they can be selected/run until the server is
restarted. After a restart they show up.

## Steps to reproduce

1. Add one or more agents to the `agents:` list in `lifecycle/config.yaml` (via
   the config editor or on disk).
2. Observe the agent picker / run surfaces — the new agents are absent.
3. Restart the app → the new agents appear.

## Likely root cause

The project's config is snapshotted at project-open and cached on `p.Cfg`
([internal/project/project.go L81](../../internal/project/project.go#L81)
`config.LoadProject`). The runtime agent roster (dispatch + the agent picker)
reads that cached snapshot. Two gaps:

- The fsnotify watcher only watches `lifecycle/**` markdown; it does **not**
  watch `lifecycle/config.yaml`, so an edit triggers nothing live.
- Only a few read-only *view* endpoints reload config per request
  (`handleGetKanbanConfig` / `…RoadmapConfig` /
  [config.go L46](../../internal/http/config.go#L46)); the **live agent set is
  never refreshed** after project open.

This is the same "config is not hot-reloaded" gap noted in
[test-runner-parks-on-schedulewakeup](test-runner-parks-on-schedulewakeup.md).

## Expected

Adding an agent to config takes effect without a restart — either by watching
`config.yaml` and refreshing `p.Cfg`, or via an explicit "reload config" action
that re-reads the agent roster. At minimum, the UI should tell the user a
restart is required.
