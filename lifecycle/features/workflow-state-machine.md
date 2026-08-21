---
title: Workflow & state machine
type: feature
status: approved
lineage: feature-workflow-state-machine
created: "2026-08-21T15:02:00+10:00"
summary: Role-gated, plan-gated status transitions enforced by a per-project state machine, with a lineage-wide status checker.
function: Workflow
labels:
    - feature
    - workflow
related_to:
    - lifecycle/ideas/open-questions-gui.md
    - lifecycle/requirements/open-questions-gui-2.md
    - lifecycle/ideas/status-checker-button.md
    - lifecycle/requirements/status-checker-button-2.md
---

# Workflow & state machine

Artifacts move through a defined lifecycle, and moving them is governed by
configuration, not convention.

## What it does

- **Status vocabulary.** `draft → clarifying → planning → in-development →
  in-qa → approved → done` plus `rejected`, `abandoned`, `blocked`.
- **Role-gated transitions.** Per-edge rules in `lifecycle/config.yaml`
  decide which roles may move which artifact type from which status to
  which. Type-aware (e.g. `test` artifacts have their own
  `approved → in-qa` cycle).
- **Plan-completion gate.** A requirement can only leave `planning` once
  every `required_plan` type for it has at least one approved artifact.
- **Product-owner bypass.** The `product-owner` role can take any transition
  between known states, for recovery / smoothing edge cases.
- **Self-transition guard.** Same-state transitions are rejected up front
  (no `draft → draft` no-ops).
- **Concurrency-safe transitions.** Per-path mutex + on-disk from-status
  re-check, so two parallel calls to the same artifact produce one advance,
  not two.
- **Auto-block on Open Questions.** Saving an artifact whose body has a
  populated `## Open Questions` H2 forces `status: blocked` and routes it to
  a `product-owner` agent assignee.
- **Lineage status checker.** A view that shows every lineage at the same
  time and one-click advances stale ones to their next valid state.
- **Allowed-targets API.** `GET /artifacts/.../allowed-targets` for
  per-user, per-status, per-type valid next-state lookups.

Reachable via the status pill on any artifact, and the **Lineage Status**
view; API under `/artifacts/.../transition` and `/artifacts/.../allowed-targets`.
