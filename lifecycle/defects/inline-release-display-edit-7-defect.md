---
title: patch_release_test.go redeclares strPtr — integration test build broken
type: defect
status: in-development
lineage: inline-release-display-edit
parent: lifecycle/tests/inline-release-display-edit-6-test.md
labels:
    - defect
    - release-blocker
release: KC-Release1
assignees:
    - role: test-developer
      who: agent
---

# patch_release_test.go redeclares strPtr — integration test build broken

`tests/integration/patch_release_test.go` defines a `strPtr` helper at line 79.
The same function is already declared in `tests/integration/dashboard_stats_test.go`
at line 16. Since all `_test.go` files in a package compile into the same binary,
this causes a `strPtr redeclared in this block` compilation error that prevents
the entire `tests/integration` package from building.

## Reproduction Steps

1. `go vet -tags integration ./tests/integration/`
2. Observe build failure.

## Expected Behaviour

The integration test package compiles cleanly; `strPtr` is defined exactly once.

## Actual Behaviour

```
# github.com/kaos-control/kaos-control/tests/integration
vet: tests/integration/patch_release_test.go:79:6: strPtr redeclared in this block
     tests/integration/dashboard_stats_test.go:16:6: other declaration of strPtr

go test -count=1 -tags integration -run "TestQueue" ./tests/integration/ -timeout 120s
FAIL    github.com/kaos-control/kaos-control/tests/integration [build failed]
```

## Fix Required

Remove the local `strPtr` definition from `tests/integration/patch_release_test.go`
(lines 77–79) and rely on the existing declaration in `dashboard_stats_test.go`.
All call-sites in `patch_release_test.go` will continue to work as-is since both
files are in the same package.

## Logs / Output

```
# github.com/kaos-control/kaos-control/tests/integration [build failed]
tests/integration/patch_release_test.go:79:6: strPtr redeclared in this block
        tests/integration/dashboard_stats_test.go:16:6: other declaration of strPtr
FAIL    github.com/kaos-control/kaos-control/tests/integration [build failed]
FAIL
```
