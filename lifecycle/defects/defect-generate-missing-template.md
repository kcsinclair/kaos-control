---
title: "New Defect → Generate" fails with 'idea-capture agent has no template "defect-generate"'
type: defect
status: draft
lineage: defect-generate-missing-template
created: "2026-08-11T00:00:00+10:00"
priority: medium
release: KC-Release5
labels:
    - defect
    - agent
    - defects
    - config
assignees:
    - role: backend-developer
      who: agent
---

# "New Defect → Generate" fails with 'idea-capture agent has no template "defect-generate"'

## Source

GitHub issue [#16](https://github.com/kcsinclair/kaos-control/issues/16)
("Missing default") — reported by **aburow**, 2026-06-21.

## Summary

In the **New Defect** modal, entering a description and clicking **Generate**
returns the error:

> idea-capture agent has no template "defect-generate"

so a defect cannot be generated.

## Likely root cause

The defect-generation flow resolves the **`idea-capture`** agent and asks it for
a **`defect-generate`** prompt template, but that agent does not define one. In
this project's own config, `idea-capture`
([config.yaml L491](../../lifecycle/config.yaml#L491)) defines only
`idea-capture` / `idea-generate` templates, while the `defect-generate` template
([config.yaml L551](../../lifecycle/config.yaml#L551)) lives on a **different**
(defect-capture) agent. So the flow is either resolving the wrong agent for
defect generation, or requires a `defect-generate` template on `idea-capture`
that isn't guaranteed to exist — and it surfaces a raw error with no default or
fallback (hence the issue title "Missing default"). A user whose config lacks a
`defect-generate` template on the resolved agent hits a dead end.

Resolution path is around
[internal/http/idea_chat.go L110-111](../../internal/http/idea_chat.go#L110)
(`resolveIdeaCaptureConfig(p, "idea-capture")`) and the defect-generate
counterpart.

## Expected

The defect-generation flow resolves an agent that actually defines
`defect-generate` (or falls back to a built-in default template), and/or
degrades gracefully with actionable guidance instead of a hard error. Fresh
projects should be able to generate a defect out of the box.
