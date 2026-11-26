---
title: Reporting for Open Provider Support and Gemini JSON Stream
type: idea
status: approved
lineage: reporting-open-provider-gemini-stream
created: "2026-08-25T18:23:28+10:00"
priority: normal
labels:
    - reports
    - open-provider-support
    - gemini
    - provider
    - agent-runner
    - observability
    - backend
release: KC-Release7
parent: lifecycle/ideas/release-goal-and-description.md
---

# Reporting for Open Provider Support and Gemini JSON Stream

Reporting infrastructure must be extended to capture and surface run data across all providers enabled by the open-provider-support initiative, including Gemini when operating in JSON stream mode. Currently reporting may be tightly coupled to a single provider's response shape, meaning usage metrics, token counts, cost data, and run summaries are incomplete or missing for non-default providers.

The work involves normalising provider-specific response envelopes — including Gemini's streaming JSON chunks — into a common reporting schema so that agent run reports reflect accurate, comparable data regardless of which provider executed the run. This includes capturing model identity, input/output token usage, finish reason, and any cost metadata the provider exposes.

The goal is that the analytics aggregation layer and any downstream report artifacts receive consistent, provider-agnostic data, giving operators full visibility into agent activity and expenditure across the whole provider fleet.
