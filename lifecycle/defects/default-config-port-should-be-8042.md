---
title: Default config.yaml uses wrong port (8080 instead of 8042)
type: defect
status: in-development
lineage: default-config-port-should-be-8042
created: "2026-05-12T09:45:46+10:00"
priority: normal
labels:
    - defect
    - backend
    - go
    - onboarding
---

# Default config.yaml uses wrong port (8080 instead of 8042)

## Reproduction Steps

1. Run kaos-control for the first time on a machine with no existing `~/.kaos-control/config.yaml`.
2. Allow the application to auto-generate the default configuration file.
3. Inspect the generated `~/.kaos-control/config.yaml`.

## Expected Behaviour

The generated `config.yaml` should contain `port: 8042` as the default server port.

## Actual Behaviour

The generated `config.yaml` contains `port: 8080` as the default server port, which is incorrect and may conflict with other common services running on port 8080.
