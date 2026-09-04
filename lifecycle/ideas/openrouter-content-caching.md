---
title: Enable OpenRouter Content Caching for Google and Anthropic
type: idea
status: draft
lineage: openrouter-content-caching
created: "2026-09-04T19:01:25+10:00"
priority: normal
labels:
    - cache
    - open-provider-support
---

# Enable OpenRouter Content Caching for Google and Anthropic

Add support for content caching when using OpenRouter with Google or Anthropic models. The implementation should allow callers to enable caching via a request header `X-OpenRouter-Cache: true` or by setting `cache_enabled: true` in the client configuration.

Enable more than 5 minutes of caching, 2 or 4 hours is probably sufficient. Manage how long responses stay cached using X-OpenRouter-Cache-TTL (from 1 second up to 24 hours; default is 5 minutes) or clear it using X-OpenRouter-Cache-clear: true

When caching is active, OpenRouter will store and reuse model responses for identical prompts, reducing latency and cost. This feature should be configurable per request and documented clearly for developers integrating OpenRouter into their applications.
