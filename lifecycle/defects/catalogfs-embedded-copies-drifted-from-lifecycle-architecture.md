---
title: catalogfs embedded architecture catalog has drifted from lifecycle/architecture/ source files
type: defect
status: draft
lineage: architectural-artefacts
created: "2026-08-20T12:21:00+10:00"
parent: lifecycle/backend-plans/architectural-artefacts-3-be.md
labels:
    - defect
release: KC-Release6
assignees:
    - role: product-owner
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

## Resolution (done)

Real but transient. The `created:` **backfill** (not commit `1f66e6b8`, which only added the
stamping *code*) had walked all of `lifecycle/` and stamped `created:` into the catalog markdown,
drifting the embedded copies — the failure this defect captured. It was fixed in commit
`973460ac`: the catalog `created:` additions were reverted (the shipped/embedded catalog must stay
clean), and `backfill-created` now **skips** `lifecycle/architecture/{architectures,tech-stacks}/`
so it can't recur. `TestFS_MatchesRepoCatalog` is green and no catalog file carries `created:`.
The investigating agent below correctly found it non-reproducing — it had just been resolved.

## Resolved Questions

Investigated on 2026-08-20 (current HEAD `cb984312`) before writing any code:

- `diff -rq lifecycle/architecture/architectures internal/architecture/catalogfs/architectures` and the `tech-stacks` equivalent report **zero** differences — all 24 files plus `README.md` are byte-identical between `lifecycle/architecture/` and `internal/architecture/catalogfs/`.
- Neither tree contains a `created:` field on any `architectures/*.md` or `tech-stacks/*.md` file — commit `1f66e6b8` only added the *stamping code* (`internal/architecture/promote.go`, `adr.go`, `summary.go`) for future writes; it never touched the existing catalog markdown, so no drift was actually introduced by it.
- `go test ./internal/architecture/catalogfs/... -run TestFS_MatchesRepoCatalog -v` passes, and a full `make test-unit` run (all packages) is green — no other failures related to this catalog.
- `git log --oneline 1f66e6b8..HEAD -- internal/architecture/catalogfs/ lifecycle/architecture/architectures/ lifecycle/architecture/tech-stacks/` is empty — nothing has touched either tree since the commit blamed for the drift, yet they are already in sync.

I cannot reproduce the failure this defect describes on current HEAD, and the repro steps (`make test-unit` / `make test-integration`) do not fail here. Rather than guess at a fix for drift that isn't present (or fabricate a no-op commit), flagging back:

1. Was this defect filed against a different commit/branch state that has since been rebased or otherwise resolved without closing this artifact?
2. Should this defect be transitioned to `done`/closed as non-reproducing, or is there a different observed failure (e.g. from CI, a different checkout, or `make test-integration` specifically) that should be captured here instead?
