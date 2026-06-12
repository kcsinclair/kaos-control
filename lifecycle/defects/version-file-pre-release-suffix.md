---
title: "VERSION file contains pre-release suffix (0.1.3a) violating bare semver requirement"
type: defect
status: approved
lineage: version-file-pre-release-suffix
created: "2026-06-12T00:00:00+10:00"
labels:
  - defect
assignees:
  - role: backend-developer
    who: agent
---

# VERSION file contains pre-release suffix (0.1.3a) violating bare semver requirement

## Reproduction Steps

1. Read the repository-root `VERSION` file:
   ```
   cat VERSION
   ```
2. Observe the content.
3. Run the version integration test:
   ```
   go test -tags integration ./tests/integration/... -run TestVersionFile_ExistsAndIsValidSemver
   ```

## Expected Behaviour

`VERSION` contains a bare semver string matching `^[0-9]+\.[0-9]+\.[0-9]+$` (e.g. `0.1.3`).

## Actual Behaviour

`VERSION` contains `0.1.3a`, which fails the bare semver pattern:

```
version_test.go:136: VERSION file contents "0.1.3a" do not match bare semver pattern (e.g. 0.1.0)
```

## Failing Test

- `TestVersionFile_ExistsAndIsValidSemver` (`version_test.go:136`)

## Fix

Remove the `a` suffix from the `VERSION` file so it reads `0.1.3`. If a pre-release label is needed for internal tracking, use the git tag (e.g. `v0.1.3-alpha`) rather than mutating the `VERSION` file, which the test asserts must be a bare 3-part semver.
