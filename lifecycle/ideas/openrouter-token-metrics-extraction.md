---
title: Extract Full Token Metrics from OpenRouter Driver
type: idea
status: draft
lineage: openrouter-token-metrics-extraction
created: "2026-09-04T10:30:50+10:00"
priority: normal
labels:
    - driver
    - provider
    - observability
    - cost
---

# Extract Full Token Metrics from OpenRouter Driver

Currently, runs using the OpenRouter driver show the message "Token metrics not available for this driver," even though OpenRouter's JSON responses typically include detailed usage data such as prompt tokens, completion tokens, total tokens, and potentially cost-related fields. This suggests the driver implementation is not parsing or surfacing these available metrics.

We should investigate the actual JSON structure returned by OpenRouter API calls to identify all available token and usage metrics, then update the driver to properly extract and report them. This would improve cost tracking, observability, and consistency with other provider drivers that already report token usage.

This fix would benefit dashboards, cost reporting, and run-level observability by ensuring OpenRouter-based runs have the same level of detail as other supported providers.
