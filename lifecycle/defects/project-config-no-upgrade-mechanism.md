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

There is a first-class way to **upgrade an existing project's `config.yaml` to
the current schema** — persisting the built-in defaults for any
missing/empty sections and templates while preserving the project's existing
customisations. After running it, the warnings stop and the on-disk file
matches what the server expects.

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

Add a **`kaos-control migrate-config`** (or `upgrade-config`) CLI command, and
ideally a matching UI action, that:

- loads the project config, runs the same `ValidateAndRepair` pass, and **writes
  the repaired config back to disk**, persisting the built-in defaults for any
  missing sections/templates;
- **preserves existing customisations** — only fills what's missing/empty, never
  clobbers hand-tuned values;
- follows the `migrate-directives` UX: show a **diff / dry-run** and require
  confirmation (or `--force`) before writing, so the user sees exactly what
  changes;
- reports the applied `RepairNote`s (the `ConfigHealthResponse` shape already
  models these).

## Notes

Surfaced while running devops against `kaos-control.io`. Non-blocking (the
project works via in-memory self-repair), so deferred to KC-Release6. Interim
workaround: manually add the missing `architecture_wizard:` section and the
`idea-capture` `defect-generate` template (copy from a current project's
`config.yaml` or a fresh `kaos-control init`).
