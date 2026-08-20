---
created: "2026-08-15T10:09:10+10:00"
title: Route config-owning defects to a config owner at triage, not to developer agents
type: idea
status: draft
lineage: config-change-defect-routing
priority: medium
labels:
    - agents
    - governance
    - workflow
    - qa
assignees:
    - role: product-owner
      who: agent
---

# Route config-owning defects to a config owner at triage, not to developer agents

## Context

Two defects on 2026-08-15
([adr-write-path](../defects/architectural-artefacts-agent-directives-adr-write-path-7-defect.md),
[propose-adr](../defects/architectural-artefacts-agent-directives-propose-adr-7-defect.md))
were auto-assigned to **backend-developer**, but their entire fix was a data
edit to `lifecycle/config.yaml`. backend-developer's write scope is
`internal/**` / `cmd/**`, so it **structurally could not** make the change. It
did the right thing — blocked and asked the product-owner via `## Open
Questions` — but the result was an agent bouncing a config change up the chain
("agents asking questions of other agents"), one round trip per defect.

This is not a one-off. Any defect whose fix lives in `lifecycle/config.yaml`
(agent roster, `allowed_write_paths`, prompt templates, gates, kanban) hits the
same wall, because **no developer agent has — or should have — write access to
`config.yaml`**: it defines every agent's own permissions and prompts, so
granting an agent write access to it is a self-modification / privilege-
escalation hazard. The block is correct; the **routing** is what's wrong.

## The problem to solve

Recognise "this defect's fix is in `lifecycle/config.yaml` (or another
config-owning artifact)" at **triage time** and route it to a config owner
(product-owner / a dedicated config-owning role), so it never lands on a
developer agent that can only block on it.

## Sketch

- **Triage-time detection.** The `qa` agent already routes defects to a
  developer role by which layer failed
  ([lifecycle/config.yaml](../config.yaml) qa prompt). Add a rule: if the fix
  is a change to `lifecycle/config.yaml` (or files outside any developer's
  write scope), assign `role: product-owner` (or a config-owning role) rather
  than a developer. Signals: the failing test reads/asserts `config.yaml`
  (e.g. the `TestAgentDirectives_*` family), or the defect's target path is
  `lifecycle/config.yaml`.
- **Config-owning role (optional).** Introduce a narrowly-scoped role whose
  `allowed_write_paths` is just `lifecycle/config.yaml` (+ the scaffold
  template), so config changes can be actioned by an agent without giving any
  developer that power. Keeps the privilege boundary intact.
- **Fail-fast hint.** When an agent is dispatched on a defect whose fix is
  provably outside its write scope, surface that at assignment (or in the
  precheck) as "misrouted — reassign", instead of spending a run to discover it
  and block.
- **Ownership of self-tests.** Tests that assert `config.yaml` matches the spec
  (`TestAgentDirectives_*`, agent-roster checks) are product-owner/governance
  concerns, not developer tasks — reflect that in how their defects are routed.

## Open questions (to think about)

- **Dedicated config role vs. human-only?** A `config-owner` agent role keeps
  it automatable but means *an* agent can rewrite `config.yaml` — acceptable if
  that role is tightly scoped and itself governed, or should config edits stay
  human-only?
- **Detection heuristic.** Is "target path / failing test touches
  `lifecycle/config.yaml`" enough, or does qa need an explicit map of
  file → owning role to route reliably?
- **Scope of the boundary.** Does the same routing apply to other
  agent-governing files (devops pipelines, gates, the scaffold template), or
  just `config.yaml`?

## Related

- [architectural-artefacts-agent-directives-adr-write-path-7-defect](../defects/architectural-artefacts-agent-directives-adr-write-path-7-defect.md)
  and
  […-propose-adr-7-defect](../defects/architectural-artefacts-agent-directives-propose-adr-7-defect.md)
  — the two config-change defects that motivated this.
- [agent-config-requires-restart](../defects/agent-config-requires-restart.md)
  — related config-lifecycle gap (config isn't hot-reloaded).
