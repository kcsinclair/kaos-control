---
title: "catalogfs embedded architecture catalog has drifted from lifecycle/architecture/ source files"
type: defect
status: approved
lineage: architectural-artefacts
parent: lifecycle/backend-plans/architectural-artefacts-3-be.md
created: "2026-08-20T12:21:00+10:00"
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# catalogfs embedded architecture catalog has drifted from lifecycle/architecture/ source files

## Reproduction Steps

1. `make test-unit` (or `make test-integration`)
2. Observe `TestFS_MatchesRepoCatalog` fail in `internal/architecture/catalogfs`.

## Expected Behaviour

`internal/architecture/catalogfs/{architectures,tech-stacks}/*.md` — the copies embedded into the Go binary via `embed.FS` — stay byte-identical to their source-of-truth counterparts under `lifecycle/architecture/{architectures,tech-stacks}/*.md`, enforced by `TestFS_MatchesRepoCatalog`.

## Actual Behaviour

All 10 `architectures/*.md` and 14 `tech-stacks/*.md` embedded copies have drifted from the `lifecycle/architecture/` originals. Commit `1f66e6b8` ("feat(architecture): stamp created: on ADRs, summary, and promoted copies") added a `created:` frontmatter field to every file under `lifecycle/architecture/architectures/` and `lifecycle/architecture/tech-stacks/`, but did not regenerate/copy the corresponding files in `internal/architecture/catalogfs/architectures/` and `internal/architecture/catalogfs/tech-stacks/`, so the embedded copies now lack the `created:` field the disk originals have.

## Logs / Output

```
--- FAIL: TestFS_MatchesRepoCatalog (0.00s)
    embed_test.go:54: architectures/cloud-native-microservices.md: embedded copy has drifted from lifecycle/architecture/architectures/cloud-native-microservices.md
    embed_test.go:54: architectures/edge-hybrid.md: embedded copy has drifted from lifecycle/architecture/architectures/edge-hybrid.md
    ... (24 files total: 10 architectures/*.md + 14 tech-stacks/*.md)
FAIL
FAIL	github.com/kaos-control/kaos-control/internal/architecture/catalogfs	2.610s
```

## Fix guidance

Copy the current `lifecycle/architecture/architectures/*.md` and `lifecycle/architecture/tech-stacks/*.md` files over their `internal/architecture/catalogfs/architectures/` and `internal/architecture/catalogfs/tech-stacks/` counterparts (see `internal/architecture/catalogfs/README.md` for the intended sync process), then re-run `make test-unit`. Consider whether the `created:` stamping commit's workflow should also update the embedded copies going forward to prevent recurrence.
