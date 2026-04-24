# Project Plan — kaos-control / Innovation Maker

Living document summarising project state. Updated on every commit per the Commit Conventions in [CLAUDE.md](../CLAUDE.md).

**Current stage:** M2 complete — artifact indexing, SQLite index, read-only HTTP API. Next: M3 (write path + git integration).

---

## Recent Changes

Rolling log — add a dated bullet per commit.

- **2026-04-24** — Initial commit: original idea captured.
- **2026-04-24** — Requirements flow completed: Q&A rounds, detailed spec distilled, lifecycle directory structure established, `CLAUDE.md` created with commit conventions, first implementation-plan artifact (`plans/create-claude-md.md`) saved.
- **2026-04-24** — Three development plans generated from the detailed requirements: backend (`…-2-be.md`), frontend (`…-3-fe.md`), test (`…-4-test.md`). Each phased into 6 milestones with cross-plan coordination noted.
- **2026-04-24** — M1 + M2 implemented: repo scaffold (`cmd/`, `internal/`, `web/`, `Makefile`), `internal/artifact` parser, `internal/index` SQLite layer, `internal/project` container, full read-only HTTP API (`/artifacts`, `/graph`, `/labels`, `/lineages`, `/parse-errors`). Fixed chi wildcard routing (greedy wildcard requires inline dispatch for suffix-matched sub-routes). Acceptance verified: 7 lifecycle artifacts indexed, graph returns 7 nodes + 19 edges.

---

## Completed

- Original idea captured: [lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md](../lifecycle/ideas/Innovation Maker - Making Releases from Ideas.md)
- Clarifying Q&A (two rounds): [lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md](../lifecycle/ideas/Innovation Maker - Making Releases from Ideas-questions.md)
- Detailed requirements spec: [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](../lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md)
- Repo guidance for Claude Code: [CLAUDE.md](../CLAUDE.md) (includes commit conventions)
- Plan: Create CLAUDE.md — [plans/create-claude-md.md](create-claude-md.md)
- Backend development plan — [lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md](../lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md)
- Frontend development plan — [lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md](../lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md)
- Test plan — [lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md](../lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md)
- **M1 (skeleton)**: `cmd/kaos-control/main.go`, `Makefile`, `web/embed.go`, config loading, project registry, signal handling
- **M2 (artifact indexing)**: `internal/artifact` (parser, links, types), `internal/index` (SQLite schema, scan, all queries), `internal/project` (runtime container), HTTP API (`/artifacts`, `/graph`, `/labels`, `/lineages`, `/parse-errors`)

---

## Planned

### Next: M3 — Write path + Git (≈ 3 days)
- `POST /artifacts`, `PUT /artifacts/*`, `DELETE /artifacts/*`
- Slug rename with link rewrite (atomic git commit)
- go-git wrapper: commit per change with templated message, branch-per-lineage creation
- fsnotify watcher → incremental re-index + WebSocket `file.changed` event
- **Acceptance** (from BE plan §18): create artifact via API → file created, branch created, commit emitted, event broadcast; rename slug → inbound links rewritten in one commit; external file drop → re-indexed in < 1 s

---

## Open Questions (parked)

Carry-overs from §17 of the spec — decide during implementation.
- Finalise product name: **kaos-control** vs **Innovation Maker**.
- Styling system: Tailwind vs small custom CSS layer.
- Agent prompt template storage location.
- SQLite schema migration strategy for index rebuilds across app versions.
- Auto-collection cadence for labels/types (realtime fsnotify vs manual re-index).

---

## References

- Authoritative spec: [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](../lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md)
- Workflow log & prompt library: [project-notes.md](../project-notes.md)
- Agent/Claude guidance: [CLAUDE.md](../CLAUDE.md)
