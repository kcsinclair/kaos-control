---
title: DevOps CLI with Linux-User Identity Mapping — Frontend Plan
type: plan-frontend
status: done
lineage: kaos-control-devops-cli
parent: lifecycle/requirements/kaos-control-devops-cli-2.md
release: KC-Release4
assignees:
    - role: frontend-developer
      who: agent
---

# DevOps CLI with Linux-User Identity Mapping — Frontend Plan

The feature in [[kaos-control-devops-cli-2]] is delivered primarily by the backend
([[kaos-control-devops-cli-3-be]]): the new surface is a terminal subcommand group, not a
web view. There is **no new page or user-management UI** — managing users and tokens is an
explicit non-goal of the requirement (owned by [[cli-auth-user-management]]).

The frontend footprint is therefore deliberately small and confined to two seams the
backend opens:

1. The project config gains a `linux_user` field on each user binding (backend Milestone 1).
   Any UI that displays project users should surface it so operators can see and trust the
   mapping the CLI relies on.
2. `devops run` triggers pipeline runs attributed to the resolved kaos-control user
   (backend F11). The existing devops run history UI must render CLI-originated runs without
   regressions.

If, on inspection, the project users panel is read-only or absent in the current SPA, the
frontend-developer should reduce Milestone 1 to read-only display (or skip with a note) and
**not** introduce a new editing surface — that would cross the requirement's non-goal.

## Milestone 1 — Surface the `linux_user` mapping in the project users view

### Description

Where the SPA already displays a project's configured users (project settings/detail), add a
column or field showing the bound `linux_user`, so the operator can confirm which Linux
account maps to which kaos-control identity for CLI use.

### Files to change

- **`web/src/components/project/EditProjectModal.vue`** (and/or the project detail view
  under `web/src/views/project/`)
  - If the users list is rendered here, add a read-only `Linux user` column/field bound to
    the `linux_user` value returned by the project config API.
- **`web/src/api/` types** — extend the project/user-binding TypeScript type with an
  optional `linux_user?: string` field so the value is typed end-to-end.

### Acceptance criteria

- [ ] A project whose config has `linux_user: alice` on a binding shows `alice` against that
      user in the UI.
- [ ] A binding with no `linux_user` renders cleanly (empty/placeholder), no console errors.
- [ ] No new editing affordance for user management is introduced (respects the non-goal).
- [ ] `pnpm build` succeeds with no TypeScript errors.

## Milestone 2 — CLI-originated devops runs render correctly in run history

### Description

`devops run --follow` (backend Milestone 5) produces ordinary pipeline runs, attributed to
the resolved user (F11), reusing the streaming surface of [[devops-pipeline-log-streaming]].
Verify the existing run-history and log components handle runs whose trigger origin is the
CLI — primarily a correct-attribution and no-regression check, not new UI.

### Files to change

- **`web/src/components/devops/RunHistory.vue`** — confirm the run list renders the
  triggering user for CLI-originated runs; if the user/attribution field is already shown,
  ensure it reflects the resolved kaos-control user rather than a blank/system value.
- **`web/src/components/devops/PipelineLogPane.vue`** / **`LogViewer.vue`** — confirm a run
  started from the CLI streams and terminates in the live log view identically to a
  UI-started run (the WS/NDJSON event shape is unchanged).

### Acceptance criteria

- [ ] A pipeline run started via `kaos-control devops run` appears in the web run history
      attributed to the resolved kaos-control user, matching F11.
- [ ] Its live log streams and reaches a terminal state in the UI exactly as a UI-triggered
      run does.
- [ ] No regressions to existing UI-triggered run display; `pnpm build` succeeds.

## Notes for the developer

- The bulk of acceptance for this requirement is verified at the CLI/HTTP layer in
  [[kaos-control-devops-cli-5-test]]; this plan exists to keep the web UI consistent with
  the new backend fields and run origins, not to add a CLI console to the browser.
