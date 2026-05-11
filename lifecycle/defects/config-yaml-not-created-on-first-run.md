---
title: Config file not auto-created on first run
type: defect
status: in-development
lineage: config-yaml-not-created-on-first-run
created: "2026-05-12T09:32:22+10:00"
priority: normal
labels:
    - defect
    - backend
    - onboarding
    - go
    - operability
---

# Config file not auto-created on first run

## Reproduction Steps

1. Install kaos-control on a machine with no existing `~/.kaos-control/` directory.
2. Run the `kaos-control` binary.
3. Observe the contents of `~/.kaos-control/`.

## Expected Behaviour

`~/.kaos-control/config.yaml` is created automatically with sensible defaults if it does not already exist, allowing the application to start without manual configuration.

## Actual Behaviour

`~/.kaos-control/config.yaml` is not created on first run. The directory and/or file must be created manually before the application can be configured and used, resulting in a poor out-of-the-box experience on new machines.
