# CodeQL configuration

Advanced setup, so the analysis can load `codeql-config.yml` and with it the
local model pack in `models/`.

## Why the model pack exists

`go/path-injection` reported 116 alerts under default setup. Many traced
through this repository's own path sanitisers, which CodeQL does not model as
barriers, so taint was tracked past the validation into the file operation.

`models/sandbox-barriers.model.yml` declares those sanitisers as `barrierModel`
entries for the `path-injection` kind.

## Reproducing locally

```sh
brew install codeql
codeql pack download codeql/go-queries

# The Go build embeds web/dist, so build the frontend first.
cd web && pnpm install && pnpm build && cd ..
codeql database create /tmp/kc-db --language=go --command='go build ./...' --overwrite

Q=~/.codeql/packages/codeql/go-queries/*/Security/CWE-022/TaintedPath.ql

# Baseline — reproduces code scanning exactly (116 alerts).
codeql database analyze /tmp/kc-db $Q --format=csv --output=/tmp/baseline.csv --rerun

# With the barrier models (79 alerts).
codeql database analyze /tmp/kc-db $Q \
  --additional-packs=.github/codeql/models \
  --model-packs=kaos-control/go-models \
  --format=csv --output=/tmp/withpack.csv --rerun
```

## Verified results (CodeQL 2.26.4, 2026-08-28)

| | alerts |
|---|---|
| baseline | 116 |
| with barrier models | 79 |
| removed | 37 |

Cleared where the sandbox is genuinely applied — `internal/http/write.go`
23 → 2, `internal/release/disksync.go` 5 → 0, `internal/docs/docs.go` 4 → 0.

## The remaining 79 are not all false positives

Grouped by where the tainted value enters:

| Source | Alerts |
|---|---|
| `handleCreateProject` (`internal/http/projects.go:566`) | 48 |
| `internal/http/agents.go:171` (`target_path`) | 9 |
| `handleCheckDirectory` (`internal/http/projects.go:346`) | 5 |
| `internal/http/architecture.go:38` | 4 |
| others | 13 |

The 48 from project registration are a design property, not a modelling gap: an
authenticated user supplies an arbitrary absolute path and the server then reads
and writes beneath it. `config.ValidatePathFormat` only requires the path to be
absolute and outside `~/.kaos-control`, and `handleCreateProject` has no role
check beyond authentication. **Do not add a barrier for these** — that would
hide the question rather than answer it.

The `architecture.go` group does check `sandbox.ErrPathTraversal`, so the
sanitiser is applied deeper in the call chain than these models reach; those are
likely modellable once the actual sanitising function is identified.

`agents.go:171` passes `target_path` to `Manager.StartRun` with no visible
sanitisation at that layer. Unreviewed — it needs the flow followed to its sink
before any conclusion.
