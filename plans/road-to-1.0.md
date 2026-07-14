# Versioning — 0.x, and the Road to 1.0

Rationale for how kaos-control is versioned, and the concrete criteria that would
earn a **1.0**. Written at the **0.2.0** release (2026-07-13).

## Why 0.2.0 (not 0.1.4, not 1.0)

- **Not 0.1.4.** The release from v0.1.3 was ~405 commits with several new
  features and some behaviour changes. In 0.x, features land in a **minor** bump,
  so `0.2.0` is the honest number; `0.1.4` would read as a patch and muddy the
  changelog.
- **Not 1.0.** 1.0 is a promise — of a stable contract, backward compatibility,
  and production readiness. kaos-control isn't ready to make that promise yet
  (see the criteria below). Version numbers are cheap; a broken 1.0 promise
  (breaking changes in 1.1) is not.

**Stance for now:** stay in **0.x**, where the artifact format and config schema
can still evolve without a major bump, and reliability is still being hardened.
Move to 1.0 deliberately, when it's *earned*.

## What 1.0 actually promises

For a tool like this — a single binary whose real "public API" is the **files on
disk** and the **config that drives the agents** — 1.0 should mean:

1. **A frozen contract.** The artifact file format (frontmatter schema, lineage /
   monotonic-index rules, status & type vocabularies), the `config.yaml` schema
   (agents, drivers, workflow transitions), and the REST + WebSocket API are
   stable — with a backward-compatibility commitment (breaking changes ⇒ major
   bump) and a migration story for existing `lifecycle/` trees.
2. **Proven core reliability.** The foundational engine (index, watcher, agent
   runner, workflow state machine) is battle-tested and no longer surfacing
   *structural* bugs.
3. **Safe-by-default agent workflow.** Automated agents cannot break the shared
   branch; risky automation is gated by verification.
4. **Used beyond the author.** At least one real user other than the maintainer
   has run the full ideas→release lifecycle end-to-end.
5. **Settled identity.** The product name and positioning are decided
   (CLAUDE.md still says the name is "to be finalised").

## Where kaos-control stands today (0.2.0)

| Criterion | Status | Notes |
|---|---|---|
| Frozen format/API | ❌ still moving | This cycle added `rel_path`, the `raw` status, and `architecture`/`tech-stack` types. Healthy 0.x evolution — the opposite of a frozen contract. |
| Proven reliability | ⚠️ hardening | 0.2.0 fixes *foundational* bugs: index status desync ("approve didn't stick"), a stdout-drain race (false truncated-stream), dot-dir indexing, auto-block edge cases, the test-runner parking, broken agent commits reaching `main`. That these keep surfacing means the engine isn't "1.0 solid" yet. |
| Safe agents | ❌ not yet | Agents still commit non-compiling code to the shared branch. The fix — `agent-run-branch-isolation` (work in an isolated branch/worktree, merge only when a verify gate is green) — is **designed but unbuilt**. For an *agent-orchestration* product, this is a 1.0 blocker. |
| External users | ❌ single user | Not yet run in anger by anyone but the maintainer. |
| Settled identity | ❌ | Product name still "to be finalised". |

## Road to 1.0 — checklist

Pre-1.0 (earn the promise):

- [ ] **Freeze the artifact format** — document the frontmatter schema, lineage
      rules, and status/type vocabularies as stable; add a version marker and a
      migration path for older trees.
- [ ] **Freeze the config schema** — same for `config.yaml` (agents, drivers,
      transitions), with validation and clear errors.
- [ ] **Config hot-reload** — editing agents/prompts should take effect without a
      server restart (today it silently doesn't; see the papercut below).
- [ ] **Safe-by-default agents** — build `agent-run-branch-isolation` +
      pre-commit build/vet/test gate so a bad agent run can't break the shared
      branch. Harden the runner (e.g. reject `ScheduleWakeup` structurally —
      `agent-runner-strip-schedulewakeup`).
- [ ] **Reliability soak** — a stretch where the foundational bug rate goes to
      ~zero (index, watcher, runner, workflow), backed by the test-runner running
      the full suite cleanly on demand.
- [ ] **Second user** — someone other than the maintainer runs the full
      lifecycle end-to-end and it holds up.
- [ ] **Name & positioning** — decide the product name; align README/CLAUDE.md.
- [ ] **Public API stability** — REST/WebSocket contracts documented and
      committed (the SPA and the devops CLI already depend on them).

When those are checked, cut **1.0** deliberately.

## Loose papercut worth fixing on the way

- **Config isn't hot-reloaded.** `handleUpdateConfig` writes `config.yaml` but
  doesn't refresh the cached `p.Cfg`, and the file isn't watched — so agent/prompt
  edits silently do nothing until a server restart. Small, but exactly the kind
  of surprise a 1.0 shouldn't have.
