---
title: Show Usage Guide by Default; Require -d/--daemon for Daemon Mode
type: idea
status: clarifying
lineage: daemon-flag-usage-guide
created: "2026-06-27T14:04:17+10:00"
priority: normal
labels:
    - backend
    - go
    - usability
    - operability
    - onboarding
---

# Show Usage Guide by Default; Require -d/--daemon for Daemon Mode

When the `kaos-control` binary is invoked with no arguments it should print a usage guide — command synopsis, available flags, and a brief description of what the tool does — then exit with code 0. This gives new users immediate orientation without having to hunt for documentation.

Running as a background daemon should require an explicit `-d` or `--daemon` flag. This makes the intent unambiguous, prevents accidental server starts, and aligns with conventional CLI design where destructive or long-running modes opt in via a flag.

The usage output should cover at minimum: the binary name, the daemon flag, any other top-level flags (config path, port, log level, etc.), and a one-line description of the product. It can be generated via the standard `flag` package's `Usage` hook or a custom help printer.
