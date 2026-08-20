---
created: "2026-07-14T19:34:44+10:00"
title: 'go vet lint failure: unused artifactRow/parseArtifactList/extractJSONField in devopscmd'
type: defect
status: done
lineage: kaos-control-devops-cli
parent: cmd/kaos-control/devopscmd/client.go
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

## Resolution (2026-07-09)

Removed the dead `artifactRow` type and the unused `parseArtifactList` /
`extractJSONField` functions from `cmd/kaos-control/devopscmd/client.go`
(the `encoding/json` import is still used elsewhere in the file). staticcheck
clean; `make lint` green.

# go vet lint failure: unused artifactRow/parseArtifactList/extractJSONField in devopscmd

## Reproduction Steps

1. Run `make lint`
2. Observe `go vet ./...` (staticcheck U1000) fails on `cmd/kaos-control/devopscmd/client.go`

## Expected Behaviour

`make lint` should pass with no unused-code warnings.

## Actual Behaviour

`staticcheck` reports three unused declarations in `client.go`, left over from
an earlier iteration of the `devops list`/`devops status` CLI commands. None
of the three are referenced anywhere else in the package:

```
cmd/kaos-control/devopscmd/client.go:154:6: type artifactRow is unused (U1000)
cmd/kaos-control/devopscmd/client.go:162:6: func parseArtifactList is unused (U1000)
cmd/kaos-control/devopscmd/client.go:174:6: func extractJSONField is unused (U1000)
```

Delete the unused type/functions (or wire them into the current `list`/`status`
command implementation if they were meant to replace what those commands use
today).

## Logs / Output

```
go vet ./...
cmd/kaos-control/devopscmd/client.go:154:6: type artifactRow is unused (U1000)
cmd/kaos-control/devopscmd/client.go:162:6: func parseArtifactList is unused (U1000)
cmd/kaos-control/devopscmd/client.go:174:6: func extractJSONField is unused (U1000)
make[1]: *** [lint] Error 1
```
