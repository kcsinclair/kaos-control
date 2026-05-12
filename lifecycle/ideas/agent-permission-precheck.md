---
title: Detect & Fail-Fast When Claude Code Ignores Bypass-Permissions Flag
type: idea
status: in-development
lineage: agent-permission-precheck
created: "2026-05-12T15:30:00+10:00"
priority: high
labels:
    - agent
    - reliability
    - backend
    - operability
release: KC-Release1
---

# Detect & Fail-Fast When Claude Code Ignores Bypass-Permissions Flag

## Problem

When a user installs kaos-control 0.1.0 on a fresh machine and kicks off
an agent run, the agent run can silently fail to do any actual work. The
visible symptom is an opaque agent message such as:

> "I need write permission for `lifecycle/requirements/agent-frontend-2.md`
> to create the requirement artifact. Please approve the write so I can
> save the file."

— with no clear cause and no obvious fix.

Captured from a real user's log
(`support/73acc29471c85c3a.log`,
`support/d4e2556e-accd-45a0-a5fb-8a801b76de66.jsonl`):

- The agent runner passed `--dangerously-skip-permissions` to the
  `claude` subprocess (visible in the log header).
- The session's `init` event reported `"permissionMode":"default"`,
  not `"bypassPermissions"` — i.e. the flag was silently ignored.
- The resulting `permission_denials` array contained **every** `Bash`
  and `Write` attempt. The agent eventually surrendered and produced
  the human-readable message above.

## Why Claude Code ignores the flag

`--dangerously-skip-permissions` is silently disabled by recent Claude
Code releases in any of these conditions (none of which are surfaced
to the calling process):

1. **The user has never accepted the dangerous-mode warning
   interactively.** Until you run `claude` once with a tty and accept
   the bypass prompt, the flag is a no-op.
2. **Running as root.**
3. **A `~/.claude/settings.json`** (or project-local equivalent) sets
   `permissions.defaultMode` to a restrictive value, which overrides
   the CLI flag.
4. **An older binary** that does not recognise the flag and silently
   drops it.

The newer canonical form is `--permission-mode bypassPermissions`.
The older flag is kept as an alias on recent binaries, ignored on
older ones, and gated by all of (1)–(3) regardless of which name is
used.

## Cost today

- **Confusing first run.** Anyone installing kaos-control onto a clean
  machine without first running `claude` interactively gets a
  multi-minute agent run that produces no artefact and no clear error.
- **Long log triage.** Diagnosing the failure required reading 73 KB
  of stream-json and noticing two unrelated facts: the `init` event's
  `permissionMode` value, and the `permission_denials` array at the
  end. Neither is surfaced anywhere in the kaos-control UI.
- **Adoption blocker.** This was hit by the first non-author user
  trying kaos-control 0.1.0 on his own machine.

## Vision

1. **Pass both flag forms** so kaos-control works on the widest range
   of Claude Code versions: `--permission-mode bypassPermissions`
   alongside the legacy `--dangerously-skip-permissions`.

2. **Pre-check the runtime.** When the `claude` subprocess emits its
   first stream-json event (the `init` event), the agent supervisor
   inspects `permissionMode`. If it is anything other than
   `bypassPermissions`, the supervisor aborts the run immediately,
   before any tool calls or token spend.

3. **Surface a useful error.** The agent run terminates with a
   structured failure event whose message points at the three
   resolutions the user can take, in the order most likely to work:

   - Run `claude` interactively once and accept the bypass prompt.
   - Upgrade to the latest Claude Code.
   - Remove or correct `permissions.defaultMode` in
     `~/.claude/settings.json` and the project's `.claude/settings.json`.

4. **Fail visibly.** The UI run-detail panel renders the message
   prominently; the run's terminal state is `failed` with a stable
   reason code `permission_mode_default` so the frontend can render
   help inline rather than as raw text.

## Out of scope

- **Automatic remediation.** kaos-control does not modify the user's
  `~/.claude/settings.json` or attempt to accept the dangerous-mode
  prompt on their behalf.
- **Per-tool allow-listing.** A scoped `--allowedTools` pass is a
  reasonable future option for tighter security but is not required
  to close this defect.
- **Detection of mid-run permission revocation.** The check happens
  at the `init` event only; if a user somehow flips their settings
  mid-run, that is not detected.

## Related

- Captured user logs in `support/73acc29471c85c3a.log` and
  `support/d4e2556e-accd-45a0-a5fb-8a801b76de66.jsonl`.
- Driver invocation at [internal/agent/agent.go:134-145](internal/agent/agent.go#L134-L145).
- Stream-json reader at [internal/agent/agent.go:189](internal/agent/agent.go#L189).
