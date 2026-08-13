---
title: Embedded security-scan devops pipeline with pluggable scanners and auto defect tracking
type: idea
status: draft
lineage: security-scan-pipeline
priority: medium
labels:
    - security
    - devops
    - pipeline
    - agents
    - defects
assignees:
    - role: product-owner
      who: agent
---

# Embedded security-scan devops pipeline with pluggable scanners and auto defect tracking

## Context

Dependency-vulnerability scanning is currently **half-covered and manual**:

- **Go is covered locally.** `make lint` runs `govulncheck` + `gosec` +
  `gitleaks` ([Makefile L67](../../Makefile#L67)), and the `test-lint` /
  `all-tests` devops pipelines run it — so Go dependency vulns, SAST, and secret
  leaks are caught pre-commit.
- **The JavaScript/npm side has no local scanner.** There is no `pnpm audit`,
  `osv-scanner`, or equivalent for `web/`'s dependencies — the only thing
  watching them is GitHub Dependabot.
- **Dependabot alone is not enough.** It only runs against the GitHub default
  branch, and it does **not resolve pnpm `overrides`** — so on 2026-08-11/13 we
  had to triage a batch by hand: real fixes (`js-yaml` 4.3.1, `brace-expansion`
  2.1.4) plus **false positives** (`postcss`, `ws`) that were already safe in the
  lockfile but that Dependabot wouldn't auto-close. That manual loop is exactly
  what this idea automates.

kaos-control already has the machinery to fix this without new engine code:
devops pipelines are simple YAML `steps` ([all-tests.yaml](../devops/all-tests.yaml)),
the scheduler can run a pipeline nightly, and the `test-runner` agent already
demonstrates the pattern of **parsing tool output and filing defect artifacts**
([test-runner-parks-on-schedulewakeup](../defects/test-runner-parks-on-schedulewakeup.md)).

## The problem to solve

Ship a first-class **security-scan pipeline** that (a) closes the JS gap, (b) is
**pluggable** across scanners, defaulting to a free/local tool that fits the
"single binary, no SaaS, no lock-in" ethos, and (c) **feeds the agent
defect-filing loop** so a found vulnerability becomes a tracked, routed defect
automatically — replacing the manual Dependabot triage.

## Sketch

1. **Embedded pipeline.** A `lifecycle/devops/security-scan.yaml` shipped with
   kaos-control and seeded into new projects at init (like the architecture
   catalog). Steps produce machine-readable (JSON) output for the agent to parse.
2. **Default scanner: osv-scanner.** Free, local, no account, multi-ecosystem
   (Go modules **and** `pnpm-lock.yaml` and more), and it **resolves lockfiles
   correctly** — so it catches the real npm vulns *and* correctly treats
   override-fixed ones as safe, avoiding the Dependabot false-positive problem
   entirely.
3. **Pluggable scanners.** The scan command is a config knob (a scanner driver),
   so `osv-scanner` is the default and others — **Snyk** (`snyk test`,
   `SNYK_TOKEN`), `pnpm audit`, `grype`, etc. — can be swapped in. Documentation
   lists the supported scanners and how to configure each (Snyk needs a token +
   account; the free/local ones don't).
4. **Auto defect tracking.** A `security-runner` agent (sibling to `test-runner`)
   parses the scanner JSON, **dedups against existing `lifecycle/defects/`**, and
   files a defect per new finding — severity → priority, and ecosystem → role
   (Go → backend-developer, npm → frontend-developer). Reuse the test-runner's
   parse + dedup + file pattern.
5. **Scheduled.** Register the pipeline with the scheduler for a nightly run, so
   new advisories surface as defects without anyone watching a security tab.

## Open questions (to think about)

- **Scanner availability.** osv-scanner is a separate binary — bundle it, require
  it on `PATH`, or `go install`? Mirror `make lint`'s "run if present, else print
  install hint" pattern ([Makefile L97](../../Makefile#L97)) so a missing scanner
  is a soft skip, not a hard failure.
- **Overlap with `make lint`.** Go vulns are already caught by `govulncheck` in
  `make lint`. Keep that as the fast pre-commit gate and let the security-scan
  pipeline be the broader, scheduled, multi-ecosystem + defect-filing pass? Or
  consolidate? (Lean: keep both — different cadence and purpose.)
- **Defect lifecycle & dedup.** How to avoid re-filing the same vuln each nightly
  run, and how to **auto-close** a defect when the advisory is patched (parallel
  to the open-questions auto-block/unblock). Key to not drowning in duplicates.
- **Allowlist / accepted risk.** A suppression file the scanner/agent respects,
  so intentionally-dismissed findings (the pnpm-override false positives, or
  accepted-risk items like the deferred moderate CVEs) aren't re-filed every run.
  This is the durable home for the "dismiss with a reason" decisions we made by
  hand on GitHub.
- **Snyk driver specifics.** Token handling (a secret, like the `claude-env`
  `auth_token` — never logged), free-tier limits, and cloud (`snyk monitor`) vs
  purely local (`snyk test`) modes.
- **Severity → priority mapping and routing.** Confirm the mapping (critical/high
  → high, moderate → medium, low → low) and the ecosystem → developer-role
  assignment.

## Related

- [test-runner-parks-on-schedulewakeup](../defects/test-runner-parks-on-schedulewakeup.md)
  — the existing agent that parses tool output and files defects; the pattern to
  reuse for the `security-runner`.
- The 2026-08 Dependabot triage (js-yaml/brace-expansion fixed; postcss/ws
  dismissed as false positives) — the manual loop this pipeline automates.
- [moderate-dependency-vulnerabilities-deferred](../defects/moderate-dependency-vulnerabilities-deferred.md)
  — an accepted-risk item that an allowlist would carry cleanly across runs.
