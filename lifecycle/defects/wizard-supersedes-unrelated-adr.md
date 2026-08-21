---
title: Architecture Wizard supersedes an unrelated pre-existing ADR on retrofit
type: defect
status: draft
lineage: wizard-supersedes-unrelated-adr
created: "2026-08-21T11:20:00+10:00"
priority: high
labels:
    - defect
    - architecture
    - wizard
    - adr
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
---

# Architecture Wizard supersedes an unrelated pre-existing ADR on retrofit

## Reproduction Steps

1. Have a project that already contains a hand-authored ADR numbered
   `adr-0001-*` on some topic unrelated to architecture selection (e.g.
   `adr-0001-no-header-based-client-ip-trust.md`, a security decision).
2. Run the Architecture Wizard and confirm a selection (promote an
   architecture + tech-stack for the first time).
3. Inspect `lifecycle/architecture/decisions/`.

## Expected Behaviour

A new architecture-selection ADR should only supersede a **prior
architecture-selection ADR**. When no prior *selection* ADR exists (only
unrelated ADRs), the run is effectively a first selection: it should author its
own selection ADR and supersede nothing.

## Actual Behaviour

The wizard marked the unrelated `adr-0001-no-header-based-client-ip-trust`
(`status: draft`, a security decision) as `status: superseded`, appended
`**Superseded by:** adr-0002-…`, and wrote `Supersedes: adr-0001-…` into the new
architecture ADR. Two unrelated decisions were incorrectly chained.

## Root Cause

`detectPriorSelection` in [internal/http/architecture_wizard.go:80](internal/http/architecture_wizard.go#L80)
identifies "the prior architecture-selection ADR" purely by the `adr-0001-`
**filename prefix**, assuming ADR-0001 is always the selection ADR. That holds
for a greenfield flow (where `WriteADR0001` writes the selection as ADR-0001)
but is false on a **retrofit**: any pre-existing `adr-0001-*` on another topic
is misidentified as the prior selection, so `isFirstRun` is false, the run takes
the `case changed:` branch, and `architecture.Supersede` is called against the
wrong ADR ([architecture_wizard.go:379-391](internal/http/architecture_wizard.go#L379-L391)).

## Fix Guidance

Identify the architecture-selection ADR by a **durable marker**, not by the
`0001` number:

- Stamp selection ADRs written by the wizard (`WriteADR0001` and the `readopt-…`
  `CreateADR`) with a distinguishing frontmatter marker — e.g. a
  `labels: [architecture-selection]` entry or a dedicated `selection: true`
  field.
- Change `detectPriorSelection` to find the most-recent ADR carrying that
  marker, ignoring unrelated ADRs. When none is found, treat the run as a first
  selection (`isFirstRun = true`) and supersede nothing.
- Consider a lightweight migration/back-compat path for greenfield projects
  whose existing `adr-0001` selection ADR predates the marker (detect the
  wizard's own `readopt-`/first-run slug shape as a fallback).

Also worth guarding: `WriteADR0001` should not collide with a pre-existing
`adr-0001-*` of a different lineage — allocate the next free ADR number for the
selection ADR rather than assuming 0001 is free.

## Notes

Surfaced while retrofitting the architecture process onto kaos-control itself.
The corrupted state in this repo was corrected by hand (adr-0001 reverted; the
bogus `Supersedes` line removed from adr-0002); this defect tracks the code fix
so it cannot recur.
