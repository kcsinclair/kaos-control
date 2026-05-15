---
title: Claude Driver with Permission Hooks (alternate, hardened)
type: idea
status: done
lineage: claude-hooks-driver
created: "2026-05-15T08:00:00+10:00"
priority: medium
labels:
    - agent
    - security
assignees:
    - role: product-owner
      who: agent
---

# Claude Driver with Permission Hooks (alternate, hardened)

## Context

The existing `ClaudeCodeDriver` in
[internal/agent/agent.go](../../internal/agent/agent.go) invokes
`claude` with `--permission-mode bypassPermissions` and
`--dangerously-skip-permissions`. In this mode the model does
whatever it wants on disk and via shell: writes anywhere, runs
arbitrary commands, makes network calls. Kaos Control's
`AllowedPaths` allowlist is only enforced *after* the run when
scoping the git commit — during the run itself, the agent is
unconstrained.

That's fine (and fast) for trusted private environments, but it's
not a security model. Multi-user installs, hosted scenarios, or
agents using less-trusted prompts would benefit from a real sandbox.

## Proposal

Add a **second, alternate Claude driver** alongside the existing
one. The current `claude-code-cli` driver stays as-is — it remains
the default for users who want maximum speed in a trusted
environment. The new driver (working name: `claude-hooks-cli` or
`claude-mediated`) invokes Claude *without* bypass mode and uses
Claude Code's documented `hooks` API to mediate every tool call
through Kaos Control.

Architecture:

```
Kaos Control (Go binary)
     │
     │ spawn claude --permission-mode default \
     │       --settings <per-run hook config>
     ▼
claude  ──(emits PreToolUse event)──►  hook helper script
                                         │
                                         │ POST localhost:<port>/api/agent/<run_id>/permission
                                         │ body: {tool, path, command, …}
                                         ▼
                                       Kaos Control
                                         │
                                         │ check AllowedPaths, bash allowlist,
                                         │ lineage scope, deny-list, …
                                         ▼
                                       {"decision":"allow"|"deny","reason":"…"}
                                         │
hook helper exits with that decision   ◄┘
     │
     ▼
claude either executes or skips the tool call
```

### Go side

- New driver `ClaudeHooksDriver` registered as e.g. `claude-mediated`
  in `m.drivers`. Same `Driver` interface as the existing one;
  reuses 90 % of `ClaudeCodeDriver`.
- New permission endpoint in `internal/http/`:
  `POST /api/agent/<run_id>/permission`. Authenticated via a
  per-run secret passed to the subprocess as an env var; payload
  describes the tool invocation; response is allow/deny/ask.
- Permission policy in `internal/agent/` (probably a new
  `permission.go`) that evaluates each request against:
  - `AllowedPaths` allowlist for Write/Edit (currently only used
    for git scoping — promoted to a hard run-time check).
  - Bash command allowlist/denylist (new, configurable per agent
    in `lifecycle/config.yaml`).
  - Lineage scope (refuse writes outside the current lineage's
    paths even if `AllowedPaths` would otherwise permit them).
- Per-run hook config: generate a temporary `settings.json` that
  wires the `PreToolUse` hook to our helper script, then pass it
  to `claude --settings <path>`.
- Precheck refactor: today `runPrecheck` requires
  `permissionMode == bypassPermissions`. For the new driver,
  require the *absence* of bypass and the presence of the hook
  config in the init event. Existing precheck stays unchanged for
  the original driver.

### Hook helper

Could be either a tiny embedded shell script written per-run, or
a new `kaos-control hook-helper` subcommand of the main binary
(preferred — keeps everything in one artefact, no shell-script
escaping headaches). The helper reads the tool-call JSON on stdin,
POSTs to the permission endpoint, prints the JSON response, exits.

### Config wiring

`lifecycle/config.yaml` agents pick driver `claude-mediated`
instead of `claude-code-cli`. New optional fields:

```yaml
- name: backend-developer-strict
  driver: claude-mediated
  bash_allowlist: ["go test", "go build", "go vet", "git status"]
  bash_denylist: ["rm -rf", "curl * | sh", "sudo *"]
  …
```

Sensible defaults for the deny-list so users don't have to
configure everything from scratch (`rm -rf /`, `curl ... | sh`,
`sudo`, anything writing to `$HOME` outside the project root,
etc.).

## Why have both drivers

- **Speed.** Hook round-trips add latency (probably milliseconds
  on loopback, but tool-heavy runs make hundreds of calls).
  Private/trusted environments may not want to pay that cost.
- **Migration path.** Existing installs keep working unchanged.
  Users opt in per agent.
- **Risk tolerance varies by role.** `backend-developer` working
  in `internal/` could stay on the fast driver; a hypothetical
  `external-prompt-agent` taking user-supplied prompts could be
  forced onto the mediated driver in config.
- **Failure modes differ.** If Kaos Control's HTTP layer wobbles,
  the mediated driver starts failing tool calls. The fast driver
  keeps working because it never talks back. Having both lets
  operators pick which failure mode they prefer.

## Why this is genuinely better (when used)

- **Sandbox is enforced, not aspirational.** `AllowedPaths`
  becomes a hard barrier rather than a post-hoc commit filter.
- **Audit trail.** Every tool call hits Kaos Control before it
  executes — every denied call is logged with reason. UI can
  surface "agent X tried to Write to Y, denied".
- **Granular Bash control.** Per-agent allow/deny lists for shell
  commands.
- **No bypass-permissions UX papercut.** The "you must run
  `claude` once and approve bypass mode" first-run step (the
  thing Ben hit on 0.1.0) is no longer needed for this driver.

## Effort estimate

| Piece | Effort |
|---|---|
| `ClaudeHooksDriver` (mostly copy of `ClaudeCodeDriver` with arg changes) | ~half day |
| Permission endpoint + per-run auth handshake | ~half day |
| `kaos-control hook-helper` subcommand | ~half day |
| Permission policy module (`AllowedPaths`, bash lists, lineage scope) | ~day |
| Per-run `settings.json` generation | ~hour |
| Precheck refactor (verify hook configured for new driver) | ~half day |
| Config plumbing (`bash_allowlist`, `bash_denylist`) + sensible defaults | ~half day |
| Tests + UI for "denied tool calls" | ~day |
| **Total** | **~4 days of focused work** |

## Caveats

- **Round-trip latency per tool call.** Probably negligible on
  loopback (~ms), but every Claude run does hundreds of calls.
- **Server liveness coupling.** If kaos-control restarts mid-run,
  hook calls fail and the agent will start refusing tool calls.
  Mitigation: hook helper falls back to a local-allowlist check if
  the server is unreachable, then logs a warning.
- **Claude Code version pinning.** Hooks API needs to be stable in
  the supported Claude versions. Document the minimum version.
- **Stream-json contract.** The new driver needs the init event to
  carry hook-config confirmation; need to check what Claude
  actually emits in non-bypass mode.

## Resolved Questions

- What's the default deny-list shipped with the binary? Should it
  include things like `rm -rf`, `curl … | sh`, `sudo`, network
  egress, anything writing outside the project root?

> That list is a good start.

- Should there be a "dry run" / observe-only mode where the
  permission endpoint logs decisions but doesn't enforce them, so
  operators can see what an agent *would have done* before turning
  on strict mode?

> Yes

- Per-tool deny vs. abort-run: if a Write is denied, does the
  agent get the deny response and keep going (current Claude
  hooks contract), or does Kaos Control kill the run on the first
  denial? Probably configurable, but what's the default?

> Make it configurable, default behaviour, log what happen to the job log, agent gets the deny response and keeps going.

- How does this interact with `auto-commit on success`? If most
  tool calls succeeded but a few were denied, do we still commit
  the partial work, or roll back?

> Do not commit work with any denials.  Tell the human and pause any further work, e.g. pause the queue, until the human starts it.

- Should the permission decisions feed into the UI run timeline
  the way `agent.progress` events already do? (Probably yes —
  "denied: Write to /etc/passwd" is exactly the kind of thing
  operators want to see.)

> Yes

- Does the existing `RequireBypassPermissions` config flag become
  per-driver, or do we deprecate it in favour of "your driver
  choice implies the bypass posture"?

> Leave in place for existing claude driver.
