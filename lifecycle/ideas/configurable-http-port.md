---
title: Configurable HTTP Port via YAML Config
type: idea
status: approved
lineage: configurable-http-port
priority: normal
labels:
    - backend
    - feature
    - go
---

# Configurable HTTP Port via YAML Config

Allow the HTTP server's listening port to be configured via the application YAML config file (`~/.kaos-control/config.yaml`), giving operators the flexibility to run the server on any available port without recompiling or passing environment variables.

The default port should be `8042`. If the config field is absent or zero, the server must fall back to this default, ensuring backwards compatibility with existing deployments that have no explicit port setting.

The config field should be clearly documented and validated at startup — an invalid port value (e.g. out of range 1–65535) should produce a descriptive fatal error rather than a silent fallback.
