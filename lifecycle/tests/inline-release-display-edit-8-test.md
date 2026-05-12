---
title: "Test Fix: Remove duplicate strPtr declaration in patch_release_test.go"
type: test
status: draft
lineage: inline-release-display-edit
parent: lifecycle/defects/inline-release-display-edit-7-defect.md
---

## Overview

This artifact documents the fix applied to resolve defect
`inline-release-display-edit-7-defect.md`, which prevented the entire
`tests/integration` package from compiling due to a `strPtr` function being
declared in two files.

## Fix Applied

**File modified:** `tests/integration/patch_release_test.go`

Removed the local `strPtr` helper (lines 77–79 of the original file):

```go
// strPtr is a convenience helper to obtain a *string from a string literal.
func strPtr(s string) *string { return &s }
```

The identical function already exists in `tests/integration/dashboard_stats_test.go`
(line 16). Because all `_test.go` files in a package share a single compilation
unit, both declarations were visible in the same scope, causing a
`strPtr redeclared in this block` error.

All call-sites in `patch_release_test.go` continue to resolve to the declaration
in `dashboard_stats_test.go` — no callers required changes.

## Verification

```sh
go vet -tags integration ./tests/integration/
```

Exits 0 with no output after the fix.

## Scenarios unaffected

All ten `TestReleasePatch_*` scenarios documented in
`lifecycle/tests/inline-release-display-edit-6-test.md` remain intact and
continue to compile and run correctly.
