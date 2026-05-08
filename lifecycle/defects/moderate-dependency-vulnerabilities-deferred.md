---
title: Moderate Dependency Vulnerabilities (Deferred — See Vitest Upgrade Plan)
type: defect
status: draft
lineage: moderate-dependency-vulnerabilities-deferred
created: "2026-05-09T09:53:43+10:00"
priority: normal
labels:
    - defect
    - security
    - testing
release: June2026
---

# Moderate Dependency Vulnerabilities (Deferred — See Vitest Upgrade Plan)

## Reproduction Steps

1. Run `pnpm audit` (or `npm audit`) in the repository root or `tests/web/`.
2. Observe moderate severity vulnerability warnings in the dependency tree.

## Expected Behaviour

All dependencies should be free of known moderate or higher severity vulnerabilities, or have accepted mitigations in place.

## Actual Behaviour

Moderate severity vulnerabilities are present in the current dependency tree. These cannot be remediated immediately due to a dependency on a vitest upgrade that is in progress. See `plans/vitest-upgrade.md` for the planned resolution path and current blockers.
