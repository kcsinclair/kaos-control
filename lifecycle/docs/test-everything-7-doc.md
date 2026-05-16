---
title: Test Everything — Run All Tests and Auto-file Defects
type: doc
status: done
lineage: test-everything
created: "2026-05-16T13:32:01+10:00"
parent: lifecycle/ideas/test-everything.md
---

Documentation for the `test-runner` agent and the test-everything feature.

## Output

Full documentation written to `docs/test-everything.md`.

## Coverage

The documentation covers:

- **What it is and why** — the manual triage problem it replaces.
- **How it works** — end-to-end flow diagram: trigger → executor → parsers → mapper → deduplicator → defect filer → run summary.
- **Trigger modes** — manual (agent launcher), DevOps pipeline (`test-all.yaml`), and scheduled.
- **Agent configuration** — `lifecycle/config.yaml` entry with key field explanations.
- **Test output parsers** — Go `go test -json`, Vitest `--reporter=json`, and Playwright `--reporter=json`; the `TestFailure` struct; graceful handling of non-JSON output (NF4).
- **Artifact mapping** — three-tier lookup (filename → label → lineage); orphaned failures; coverage gap detection.
- **Deduplication** — by test identifier, assertion location, and normalised error message; `NormaliseError` details; within-run assertion grouping; `AppendWitness`.
- **Defect creation** — file path convention, frontmatter fields, body structure, role routing rules.
- **Run summary** — `RunSummary` struct; agent run history panel rendering.
- **Frontend UI** — target-less agent launcher; `TestRunSummaryCard`; auto-filed badge.
- **Internal package layout** — `internal/testrunner/` file-by-file table.
- **Non-goals and limitations** — NF1–NF5 constraints; what the feature deliberately does not do.
- **Related documentation** links.
