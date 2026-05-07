---
title: Improved Agent Handling
type: idea
status: draft
lineage: improved-agent-handling
priority: medium
labels:
    - agent
    - workflow
    - process
    - enhancement
release: May2026-2
---

NOTE: This is an example of an Epic idea, many requests related to agents in a single artefact.

Using multiple models, free and paid to make it work.

Track usage of models, which model for which task.

Track token usage and cost of each piece of work.

Logs coming from agent runs include details about the tokens used, parse all this into a database for each task completed.

Add driver for Ollama, you can name a local initially, remote later.  You select which model you want to use for the agent.

What is needed to smooth out handling Claude as an agent as well as using Gemini or Ollama agents.

Improved display for RUN LOG, summarise key JSON logs, the important elements, include click for raw log which pops up modal of larger display to see it better.

Before an agent start can we check if usage is almost expired, so we don't start it now?

Can we auto-restart an agent when credit becomes available?

When an agent stops, include a button to restart/retry this activity.

When an agent stops check the last message for errors and display the reason why if failed, e.g. 

```
{"type":"assistant","message":{"id":"23779de5-146b-4f9e-ab04-2c0e20130547","container":null,"model":"<synthetic>","role":"assistant","stop_reason":"stop_sequence","stop_sequence":"","type":"message","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":null,"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":0},"inference_geo":null,"iterations":null,"speed":null},"content":[{"type":"text","text":"You're out of extra usage · resets 8pm (Australia/Brisbane)"}],"context_management":null},"parent_tool_use_id":null,"session_id":"7715e54b-9183-434e-adf5-28482255695f","uuid":"a415b8cd-684f-41bb-addf-0bb64f56acf8","error":"rate_limit"}
{"type":"result","subtype":"success","is_error":true,"duration_ms":210476,"duration_api_ms":197546,"num_turns":38,"result":"You're out of extra usage · resets 8pm (Australia/Brisbane)","stop_reason":"stop_sequence","session_id":"7715e54b-9183-434e-adf5-28482255695f","total_cost_usd":0.8173092000000001,"usage":{"input_tokens":35,"cache_creation_input_tokens":56458,"cache_read_input_tokens":1382939,"output_tokens":12707,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":56458},"inference_geo":"","iterations":[{"input_tokens":1,"output_tokens":304,"cache_read_input_tokens":56810,"cache_creation_input_tokens":263,"cache_creation":{"ephemeral_5m_input_tokens":263,"ephemeral_1h_input_tokens":0},"type":"message"}],"speed":"standard"},"modelUsage":{"claude-sonnet-4-6":{"inputTokens":35,"outputTokens":12707,"cacheReadInputTokens":1382939,"cacheCreationInputTokens":56458,"webSearchRequests":0,"costUSD":0.8173092000000001,"contextWindow":200000,"maxOutputTokens":32000}},"permission_denials":[],"terminal_reason":"completed","fast_mode_state":"off","uuid":"9ae98e37-d115-4dae-9ddf-1c989d67e7a3"}
```

Brainstorm on more things we need for improved agent handling.
