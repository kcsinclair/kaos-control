---
title: No way to upgrade a project's config.yaml to the current schema (self-repair never persists)
type: defect
status: draft
lineage: project-config-no-upgrade-mechanism
created: "2026-08-24T16:00:00+10:00"
priority: normal
labels:
    - defect
    - config
    - cli
    - onboarding
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

# No way to upgrade a project's config.yaml to the current schema

## Reproduction Steps

1. Have a project whose `lifecycle/config.yaml` predates a newer kaos-control
   version (e.g. `kaos-control.io`, missing the `architecture_wizard:` section
   and the `idea-capture` agent's `defect-generate` template).
2. Run any command against it (e.g. `kaos-control devops run …`, or start the
   server).
3. Observe repeated WARN lines on every load:
   ```
   WARN project config: self-repaired missing generation template
     agent=idea-capture template_key=defect-generate reason="template missing"
   WARN project config: self-repaired missing generation template
     agent=architecture_wizard template_key=questions
     reason="architecture_wizard section missing or empty, filled from built-in defaults"
   ```

## Expected Behaviour

A first-class **config upgrade** flow that:

1. **Tells the user in the GUI** that the project's config is behind (which
   sections/templates would be filled) and offers an explicit "Upgrade config"
   action to proceed. The existing `ConfigHealthBanner` — backed by
   `GET …/config/health`, which already returns the self-repair `RepairNote`s —
   is the natural home.
2. On proceed, **backs up the existing `config.yaml`** (timestamped, e.g.
   `config.yaml.bak-<RFC3339>`) and **writes the upgraded config** — persisting
   built-in defaults for missing/empty sections while preserving customisations.
3. **Applies without a manual restart.** Writing `config.yaml` already triggers
   the watcher-driven `project.ReloadConfig()` and a `config.reloaded` WS
   broadcast, so the running server picks it up live and the GUI can confirm the
   reload. (A "Reload config now" button — an explicit `POST …/config/reload` —
   is a nice fallback, but the auto-reload already covers the common case.)

After upgrading, the self-repair WARNs stop and the on-disk file matches the
expected schema.

## Actual Behaviour

Config self-repair (`ValidateAndRepair`, `internal/config/config.go`) fills the
missing sections **in memory only** — the on-disk file is never rewritten
(config.go: *"lifecycle/config.yaml is never rewritten by this method — repairs
apply only to this in-memory Project"*; `RepairNotes` is `yaml:"-"`). So:

- the WARN self-repair lines recur on every load, forever;
- the on-disk config drifts further from the expected schema with each release;
- the only workaround is to hand-edit `config.yaml`, copying the missing
  sections from the built-in defaults or another project's config.

There is a `migrate-directives` command for the AGENTS.md upgrade, but **no
equivalent for `config.yaml`**.

## Fix guidance

Most of the plumbing already exists — this is largely wiring it together:

- **Persist path.** Add `POST …/config/upgrade` (and a `kaos-control
  migrate-config` CLI) that runs the same `ValidateAndRepair` pass and writes
  the result back to disk, **after copying the current file to a timestamped
  backup** (`config.yaml.bak-<RFC3339>`). Only fill missing/empty sections;
  never clobber hand-tuned values. Show a **diff / dry-run** and require
  confirmation (or `--force`), mirroring `migrate-directives`.
- **GUI.** Extend `ConfigHealthBanner.vue` (already renders the pending
  `RepairNote`s from `GET …/config/health`) with an **"Upgrade config"** button
  that calls the endpoint, then reflects the resulting `config.reloaded` event.
- **Reload (already exists).** Reuse the watcher → `project.ReloadConfig()` →
  `config.reloaded` WS path (`internal/project/project.go`). **Verify a reload
  fully re-derives agents / stages / kanban from the new config** — some of
  those may be captured at `project.Open` and not refreshed by `ReloadConfig`;
  if so, either re-wire them on reload or prompt for a restart only for those
  specific changes. Optionally add an explicit `POST …/config/reload` +
  "Reload config now" button for manual control.

## Notes

Surfaced while running devops against `kaos-control.io`. Non-blocking (the
project works via in-memory self-repair), so deferred to KC-Release6. Interim
workaround: manually add the missing `architecture_wizard:` section and the
`idea-capture` `defect-generate` template (copy from a current project's
`config.yaml` or a fresh `kaos-control init`).
