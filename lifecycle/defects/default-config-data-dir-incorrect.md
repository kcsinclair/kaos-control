---
title: Default config.yaml data directory not set to /home/keith/.kaos-control/data
type: defect
status: done
lineage: default-config-data-dir-incorrect
created: "2026-05-12T09:47:50+10:00"
priority: normal
labels:
    - defect
    - backend
    - go
    - onboarding
release: KC-Release1
---

# Default config.yaml data directory not set to /home/keith/.kaos-control/data

## Reproduction Steps

1. Run kaos-control for the first time on a system where no config.yaml exists.
2. Allow the application to generate the default config.yaml file.
3. Inspect the generated config.yaml (typically at ~/.kaos-control/config.yaml).

## Expected Behaviour

The generated default config.yaml should set the data directory to `/home/keith/.kaos-control/data`.

## Actual Behaviour

The generated default config.yaml does not set the data directory to `/home/keith/.kaos-control/data`. The data directory field is either absent, set to a different default path, or incorrectly configured.
