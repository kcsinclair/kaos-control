---
title: Catalog FS drift detected
type: defect
status: in-development
lineage: internal/architecture/catalogfs
created: "2026-08-27T12:45:00Z"
parent: lifecycle/tests/2d-graph-layout-selector-6-test.md
labels:
    - defect
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

## Reproduction Steps

1. Run unit tests:
   `make test-unit`

## Expected Behaviour

The embedded architecture catalog in `internal/architecture/catalogfs` should be byte-identical (ignoring `status:` and `created:` fields) to the source of truth in `lifecycle/architecture/`.

## Actual Behaviour

The test `TestFS_MatchesRepoCatalog` fails because the embedded version of `tech-stacks/go-vue.md` has drifted from the file in `lifecycle/architecture/tech-stacks/go-vue.md`.

## Logs / Output

```
--- FAIL: TestFS_MatchesRepoCatalog (0.00s)
    embed_test.go:93: tech-stacks/go-vue.md: embedded copy has drifted from lifecycle/architecture/tech-stacks/go-vue.md (content differs beyond project-local status/created)
FAIL
FAIL	github.com/kaos-control/kaos-control/internal/architecture/catalogfs	2.542s
```
