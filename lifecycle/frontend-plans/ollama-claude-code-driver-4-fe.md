---
title: "Env-Override Claude Code Driver — Frontend Plan"
type: plan-frontend
status: in-development
lineage: ollama-claude-code-driver
parent: lifecycle/requirements/ollama-claude-code-driver-2.md
---

# Env-Override Claude Code Driver — Frontend Plan

## Overview

**No new frontend features are built for v1.** Resolved question 4 of
[[ollama-claude-code-driver]] is explicit: *"For this version no frontend
required."* Non-goal §"A frontend instance-picker UI" confirms it — v1 is
config-file driven.

This plan therefore exists to (a) record that decision, and (b) define
*verification-only* milestones ensuring the existing SPA degrades gracefully
when a `claude-env` agent is present and that no secret (`auth_token`) ever
reaches the browser. There are no new Vue components, stores, or routes.

The backend ([[ollama-claude-code-driver]] backend plan) keeps `auth_token` out
of every API payload and may expose the non-secret `base_url` on the agent
summary; the frontend must neither require nor display the token.

Cross-references: [[ollama-claude-code-driver]] backend plan (BE-6 secret
hygiene defines the API contract this plan verifies), [[ollama-claude-code-driver]]
test plan.

---

## Milestone FE-1 — Existing agent list renders a `claude-env` agent gracefully

**Description.** The agents view consumes `GET /api/p/:project/agents`. With a
`claude-env` agent configured, the response now includes a `driver:
"claude-env"` entry (and optionally `base_url`). Confirm the existing list/badge
UI renders it without errors — driver shown as a plain label, ready-count badge
behaving as for any other driver — without any code change. Only if the current
UI hard-codes a known-driver allowlist (and would hide/break on an unrecognised
driver) is a minimal additive change needed: treat unknown drivers as a generic
label rather than throwing.

**Files to inspect (change only if a hard-coded allowlist is found).**
- `web/src/components/agent/` (agent list / card / `AgentLaunchModal.vue`).
- Any agent Pinia store + the `Agent` TypeScript type that mirrors
  `agentSummary` (add optional `base_url?: string` only if the type is strict
  and would otherwise drop/forbid the field).

**Acceptance criteria.**
- With a `claude-env` agent in config, the agents view loads, lists the agent,
  and shows its driver label without console errors.
- Launching a run against the `claude-env` agent from the existing UI works via
  the unchanged `POST …/agents/:name/run` path.
- If no code change was required, that is recorded explicitly as the outcome.

---

## Milestone FE-2 — No secret reaches the browser

**Description.** Verify, from the client side, that the `auth_token` never
appears in any payload the SPA receives — agents list, run detail, run log
viewer, or WebSocket `agent.progress` events. This is the frontend-side mirror
of backend NFR-1 and is checked by inspecting network/WS payloads (no code).

**Files to inspect.**
- Agent list/store fetch, run-detail/run-log views, and the WS event handler in
  `web/src/`.

**Acceptance criteria.**
- No view binds to or renders an `auth_token` field (none is sent).
- The run-log viewer for a `claude-env` run shows the standard streamed output
  with no token string present.
- If `base_url` is surfaced by the backend, it may optionally be displayed
  read-only on the agent card; the token is never displayed.

---

## Out of scope (v1)

- No driver-type picker / create-edit agent form for `claude-env`
  (mirroring the Ollama picker) — explicitly deferred.
- No base-url / token input fields in the UI.
- Any of the above is a candidate for a future lineage if a UI is later
  requested.
