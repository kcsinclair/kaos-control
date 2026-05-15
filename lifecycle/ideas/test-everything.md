---
title: Test Everything
type: idea
status: clarifying
lineage: test-everything
created: "2026-05-08T17:47:27+10:00"
priority: high
labels:
    - test
    - testing
    - qa
    - workflow
    - artefacts
release: KC-Release2
---

Every X period of time (daily) or before a release this can be run to prepare and ensure all tests have passed and if not get things fixed.

A small extension to the existing `qa` agent (or a sibling `qa-release` agent if you want to keep them separate):

1. **Trigger**: invoked without a `target_path`, or with a special suite token. Reads no specific test artifact.
2. **Action**: runs `make test` + the frontend Vitest suite (or just runs the `test` DevOps pipeline directly), capturing structured output.
3. **Parsing**: walks the test output — Go's JSON output (`go test -json`) and Vitest's `--reporter=json` give you per-test pass/fail with file/line.
4. **Defect creation**: for each genuine failure, look up the corresponding `lifecycle/tests/*.md` artifact (test code → artifact via filename or label), then write a defect under that test's lineage. Failures with no matching artifact get filed under a `tests-orphaned` lineage — itself a useful signal.
5. **Deduplication**: simple — group failures by file/line of the assertion, then by error message. Five failures in the same release-store call → one defect with five witnesses listed.
6. **Routing**: defect role is set from the test artifact's labels (`backend`, `frontend`, etc.) or the file path of the test code (`tests/web/...` → frontend-developer). Already the convention in your existing defects.

This composes cleanly with what already exists. The DevOps pipeline runs the tests; the QA agent turns failures into actionable defect artifacts. Pipeline = "did it pass?". Agent = "if not, who fixes what?".
