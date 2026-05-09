# Security Policy

## Supported Versions

kaos-control is **pre-1.0** and under active development. There are no
released versions yet; security fixes always land on `main` and are
included in the next built artefact.

| Version | Status                                       |
| ------- | -------------------------------------------- |
| `main`  | ✅ Supported — fixes go here                  |
| Anything else | ❌ No back-porting until a stable release exists |

Once 1.x ships, this table will change to track the most recent minor
release.

## Reporting a Vulnerability

**Please do not file a public GitHub issue for a security vulnerability.**

Two private channels, in order of preference:

1. **GitHub Private Vulnerability Reporting** (preferred) — open the
   [repository's Security tab](https://github.com/kcsinclair/kaos-control/security)
   and click "Report a vulnerability". This routes the report directly
   to the maintainer and lets us collaborate on a fix and coordinated
   disclosure within GitHub.
2. **Email** — `keith@sinclair.org.au` with the subject prefix
   `[kaos-control security]`. PGP is not currently set up; email
   contents are protected only by transport encryption.

Please include:

- Affected component (which file/handler/agent role).
- Steps to reproduce (a curl command or test case is ideal).
- Impact assessment in your own words — what an attacker can do.
- Whether you've published or shared the finding anywhere already.

## What to expect

- **Acknowledgement** within 7 days. Solo-maintained project; if you've
  not heard back, please re-send.
- **Initial assessment** within 14 days — we'll either accept the
  finding, push back with reasoning, or ask for more detail.
- **Fix timeline** depends on severity. Critical and high-severity
  issues are prioritised over feature work; medium and low are queued
  alongside other backlog. We don't commit to fixed-time SLAs at this
  stage of the project.
- **Coordinated disclosure** — we'd like to publish the fix before the
  finding is public. Default embargo is 90 days from the
  acknowledgement, shorter if the fix lands faster, longer if mutually
  agreed. Once disclosed, we'll credit you in the advisory unless you
  prefer to remain anonymous.

## Scope

The maintained codebase covers:

- The Go server (`cmd/`, `internal/`).
- The Vue 3 frontend (`web/`).
- The build/release pipeline definitions in `lifecycle/devops/`.
- Default configuration shipped at `~/.kaos-control/config.yaml`.

### High-attention surfaces

These are the components where a vulnerability would have the largest
blast radius — researcher attention is welcome here:

- **Authentication & sessions** ([internal/auth/](internal/auth/), session
  cookies, CSRF token handling, role checks).
- **Path resolution & sandboxing** ([internal/sandbox/](internal/sandbox/))
  — anything that lets an API write files outside the project root.
- **Agent runner** ([internal/agent/](internal/agent/)) — it spawns
  subprocesses (Claude CLI, Ollama, etc.) with project-relative working
  directories and a configured `allowed_write_paths` allowlist; bypassing
  the allowlist is in scope.
- **DevOps pipelines** ([internal/devops/](internal/devops/)) — pipeline
  YAML steps execute as shell commands; injection or privilege escalation
  via crafted YAML is in scope.
- **Frontmatter & markdown parsing** ([internal/artifact/](internal/artifact/))
  — exploits that crash the parser, smuggle YAML into HTML rendering, or
  evade indexing are in scope.
- **WebSocket hub** ([internal/hub/](internal/hub/)) — message
  authorisation across project boundaries.

### Out of scope

- **Third-party dependency vulnerabilities** — these are tracked via
  GitHub Dependabot and don't need a separate report. If you can
  demonstrate a working exploit path through one of our handlers, that
  *is* in scope; the dependency itself is not.
- **Self-hosted deployments where the operator has elected to run
  in single-user / unauthenticated mode** (e.g. binding to `0.0.0.0`
  without a reverse proxy). Hardening guidance is a documentation
  concern, not a vulnerability.
- **Theoretical issues without a reproducer** — we'll read them, but
  fixes are not prioritised without evidence.
- **Issues requiring physical access to the host** running the server,
  or write access to its filesystem.
- **Issues in modified forks** that aren't reproducible against an
  unmodified `main`.

## Hardening defaults

Operator-side notes for anyone running kaos-control on a multi-user
host:

- Run behind a reverse proxy that terminates TLS. The bundled HTTP server
  is HTTP/1.1 plaintext; production exposure should always be behind
  nginx, Caddy, or similar.
- Bind the listener to `127.0.0.1` unless you have a reason to expose it
  to other hosts. The default `:8042` listens on all interfaces.
- The `agent` and `devops` roles can execute shell commands via
  configured pipelines. Don't grant these roles to users you don't
  trust to run code on the server.
- Pipeline YAML files live in the project's `lifecycle/devops/`
  directory and are committed to git. Restrict who can push to that
  directory — anyone who can land a YAML file there can run shell
  commands as the server process.

## Acknowledgements

When fixes land, the advisory and release notes will credit the
reporter unless they request otherwise. There is no monetary bounty
program at this stage of the project.
