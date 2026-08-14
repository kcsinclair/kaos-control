---
title: "Integration Tests — Architectural Artefacts On-Disk Model"
type: test
status: draft
lineage: architectural-artefacts
parent: lifecycle/test-plans/architectural-artefacts-5-test.md
---

# Integration Tests — Architectural Artefacts On-Disk Model

Implements the six milestones of [[architectural-artefacts-5-test]] against the backend
([[architectural-artefacts-3-be]]) and frontend ([[architectural-artefacts-4-fe]])
implementations. All new tests live in `tests/integration/` (package `integration`, build tag
`integration`), following the repository's existing integration-test convention rather than the
plan's literal `tests/` path.

## Test files

| File | Milestone | Covers |
|---|---|---|
| `tests/integration/architecture_types_test.go` | 1 (FR-18) | `architecture`/`tech-stack`/`adr` index cleanly end to end; an unknown type still surfaces a parse error. |
| `tests/integration/architecture_zones_test.go` | 2 (FR-1–FR-3) | Catalog and project-own zones index simultaneously; catalog `README.md` is excluded; catalog source bytes are unchanged after a promotion. |
| `tests/integration/architecture_promote_test.go` | 3 (FR-4–FR-7, OQ-3, NFR-3) | `POST .../architecture/promote` — first promotion, idempotent re-promotion, changed-choice archive-on-replace, two archived generations with the `-1` disambiguator, path-traversal 400, role-gate 403. |
| `tests/integration/architecture_adr_test.go` | 4 (FR-11, FR-12, FR-14, OQ-4) | `GET/POST .../architecture/adrs[/next]` — monotonic zero-padded numbering (including superseded numbers), default `status: draft`, filename format guard, and `WriteADR0001` idempotency (title, Q&A trail, ranked rejected alternatives, no `adr-0002` spawned). |
| `tests/integration/architecture_lineage_test.go` | 5 (FR-19, FR-20, NFR-2) | Clean-slug files index without lineage errors; a promoted copy's catalog `parent:` is accepted and resolves to a graph edge; the relaxation is path-scoped (regression guard for `lifecycle/requirements/`); live-watch indexing of a new architecture file. |
| `tests/integration/architecture_directives_test.go` | 6 (FR-21, FR-22) | Loads the real `lifecycle/config.yaml` via `config.LoadProject` and asserts, for each of the six design/build agents, the read-architecture and propose-ADR directive slots, plus the ADR-authoring write path for the analyst/developer agents FR-13 names. |

## Scenarios covered

- Type registration: `architecture`, `tech-stack`, `adr` retrievable via `GET /artifacts` with no
  parse error; `type: bogus` still records `unknown type "bogus"` (vocabulary guard).
- Zone coexistence: catalog (`architectures/`, `tech-stacks/`) and project-own
  (`architecture-summary.md`, `decisions/`, `standards/`) artefacts all index; catalog
  `README.md` excluded; catalog bytes untouched by a promotion run.
- Promotion: first-promotion parent stamping, idempotent re-promotion (no duplicate, no
  `archive/`), changed-selection archiving, `archive/foo.md` + `archive/foo-1.md` coexisting
  without overwrite, `../../etc/passwd` → 400, a `reviewer`-only user → 403.
- ADRs: empty `decisions/` → `next == 1`; sequential creation → `adr-0001`, `adr-0002`, `next ==
  3`; a `status: superseded` `adr-0003` fixture still counts (`next == 4`); filename regex guard;
  `WriteADR0001` called twice with identical inputs produces exactly one `adr-0001-*.md`.
- Lineage relaxation: a clean-slug architecture file parses with `Index == 0` and no parse error;
  a promoted copy's `parent:` pointing at a catalog entry is accepted and produces a resolvable
  `parent`-kind graph edge; `lifecycle/requirements/` files without `lineage:` still fail
  (regression guard); a file written directly to `lifecycle/architecture/` after boot is picked
  up by the live fsnotify watcher.
- Agent directives: read-architecture and propose-ADR directive presence per agent; ADR-authoring
  write-path presence for the FR-13-named analyst/developer agents.

## Result: 8 of 25 new tests currently fail against the real implementation

`go test -tags integration ./tests/integration/ -run 'TestArchitecture|TestPromote|TestADR|TestAgentDirectives'`
passes 17/25. The 8 failures are not test bugs — they reproduce reliably and trace to two
implementation gaps outside this test-developer's write scope (`internal/`, `lifecycle/config.yaml`):

1. **Startup scan is stage-scoped; `architecture` is not a configured stage (5 failures).**
   `index.Index.Scan(stages)` (`internal/index/index.go:335`) only walks
   `filepath.Join(lifecycle, stage.Dir)` for each entry in the project's configured `stages:`
   list. Neither the built-in `defaultStages` (`internal/config/config.go:610`) nor this
   project's own `lifecycle/config.yaml` lists `architecture` as a stage, so files under
   `lifecycle/architecture/` present *before boot* are never picked up by the initial scan —
   only by an explicit `IndexFile` call (e.g. the promote/ADR endpoints re-indexing the paths
   they just wrote) or by the live fsnotify watcher (which does watch the whole `lifecycle/`
   tree, confirmed passing). This contradicts the "all three index paths" / NFR-2 acceptance
   criterion and CLAUDE.md's "Startup — full scan of `lifecycle/**/*.md`" description.
   Affects: `TestArchitectureTypes_IndexCleanly`, `TestArchitectureTypes_UnknownTypeStillRejected`,
   `TestArchitectureZones_CoexistAndIndex`, `TestArchitectureLineage_CleanSlugIndexesWithoutError`,
   `TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge`.
2. **Agent directive prose/write-paths are incomplete in `lifecycle/config.yaml` (3 failures).**
   Only `planning-analyst` currently carries an explicit "propose an ADR in
   `lifecycle/architecture/decisions/`" directive and has that path in `allowed_write_paths`.
   `requirements-analyst`, `backend-developer`, `frontend-developer`, and `test-developer` direct
   the agent to treat a forced deviation as "being stuck" (Open Questions / blocked) rather than
   proposing an ADR, and lack the `lifecycle/architecture/decisions` write path; `qa` has neither
   the read- nor the propose-directive at all. CLAUDE.md and FR-22 note that concrete directive
   prose is owned by the separate, still-in-progress [[agent-directives-generation]] lineage —
   this is a known, not-yet-closed gap rather than a regression.
   Affects: `TestAgentDirectives_ReadArchitectureFirst`, `TestAgentDirectives_ProposeADROnDeviation`,
   `TestAgentDirectives_ADRAuthoringWritePath`.

The remaining 17 tests pass, and the full `tests/integration/` suite otherwise shows no
regressions (`go test -tags integration ./tests/integration/...` — only the 8 above fail).
