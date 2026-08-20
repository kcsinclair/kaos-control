---
created: "2026-07-14T19:34:44+10:00"
title: Isolate agent runs in sub-branches so unverified work can't break the shared branch
type: idea
status: draft
lineage: agent-run-branch-isolation
priority: medium
labels:
    - agent
    - git
    - reliability
    - ci
assignees:
    - role: product-owner
      who: agent
---

# Isolate agent runs in sub-branches so unverified work can't break the shared branch

## Context

Agent runs currently commit their produced files **directly to the active
working branch** (e.g. `kc-dev`), and they commit **regardless of run
status**. The commit block in
[internal/agent/agent.go](../../internal/agent/agent.go) (around
[L920-988](../../internal/agent/agent.go#L920)) gates only on denied tool
calls, not on whether the run succeeded:

```go
if m.git != nil && !hasDenials {   // no status check — commits even on failed / killed-timeout
    ...
    commitMsg := fmt.Sprintf("agent(%s): run %s [%s]", run.AgentName, run.RunID, status)
```

On 2026-06-29 this let **four broken test files** from four separate
`test-developer` runs land on `kc-dev`, leaving the whole
`tests/integration` package non-compiling for every other in-flight agent:

- `cbcdfc8888ce3f83` (`killed-timeout`) — called two hallucinated helpers.
- `843e959dd61ba5e8` (`killed-timeout`) — a test that asserted values it never set up.
- `7499574d6e5b00f5` (**`[done]`**) — an unused import (compile error).
- `9ce2fe6afbef80f5` (**`[done]`**) — asserted 202 where the guard returns 409.

Two of these were even recorded as `done`. The failures were only found and
repaired by hand. The runner *already* does post-run status overrides
(auth-error [L871](../../internal/agent/agent.go#L871), truncated-stream
[L885](../../internal/agent/agent.go#L885)) that downgrade `done → failed`,
but nothing verifies that produced **code** actually builds/passes before it
is committed, and nothing stops a failed run from committing anyway.

## The problem to solve

A single bad agent run should not be able to break the shared branch for
every other agent and for the human. Broken or unverified output should be
quarantined until it is known good.

## The tension (why the obvious fix is not obviously right)

The naive fix — "run the tests before committing, and only commit if they
pass" — is expensive. **`make test-integration` takes 20+ minutes.** If every
agent run had to wait out a full integration pass before it could commit,
agents would take *dramatically* longer to get anything done, and parallel
agents would serialise on the test box. A cheap gate (build + `go vet` only,
~seconds) would have caught all four of the above failures, but it would
**not** catch logic failures (wrong assertions) — only a real test run does
that. So there is a genuine trade-off between *safety* and *throughput*, and
where to draw the line is the thing to decide.

## Options considered

### Option 1 — Verification gate on the current branch (cheap, partial)
Add a post-run check (sibling to the truncated-stream override): if a run
produced code files, run a **fast** check (build + `go vet`, maybe
`test-unit -short`) before committing. On failure, downgrade `done → failed`
(`failureReason: "verification_failed"`) and **skip the commit** so broken
work never lands. Reuse the existing devops pipelines
(`build.yaml`, `test-lint.yaml`) via a new per-agent `verify_pipeline:` field
in `config.yaml` (next to `timeout_minutes`).

- **Pros:** small change, fast, mirrors existing override pattern, would have
  blocked all four incidents (they were compile errors).
- **Cons:** a fast gate can't run the 20-min integration suite, so it won't
  catch *logic* failures; and while it stops the bad commit, it still assumes
  one agent at a time is safe to touch the shared branch.

### Option 2 — Per-run sub-branch isolation (preferred, larger)
Each agent run works in its **own branch / git worktree** off the current
branch. It commits freely there (no risk to anyone), and the run's output is
only **merged into `kc-dev` once it passes its verification** — which can be
the *full* pipeline (including the 20-min integration suite) because it runs
**asynchronously and in parallel**, off the critical path of other agents. A
run that fails verification never merges; its branch stays for inspection and
re-run. This also removes the serialisation problem: N agents can run and
verify concurrently without stepping on each other's working tree.

- **Pros:** the shared branch is *always* green; supports the full (slow)
  test suite without blocking anyone; parallel agents don't collide; failed
  work is preserved in its branch, not lost.
- **Cons:** bigger change — needs worktree/branch lifecycle management, a
  merge/verify queue, conflict handling on merge, and UI to surface per-run
  branches and their verification state.

**Current lean (Keith):** Option 2. The 20-min integration cost makes a
blocking on-branch gate impractical, and sub-branches let the expensive
verification happen off the critical path.

## Open questions (to think about)

- **Verify depth per role.** Fast gate (build/vet) for every run, full suite
  only before merge? Or full suite always, since it's off-branch in Option 2?
- **Merge policy.** Auto-merge on green, or hold for human/reviewer approval?
  Fast-forward vs. merge commit? How to handle merge conflicts between two
  runs that both went green against an older base?
- **Concurrency + cost.** How many verification pipelines run at once before
  the test box is saturated (the 20-min suite ×N)? Does this need its own
  queue / concurrency cap, separate from the agent-run cap?
- **Branch lifecycle & cleanup.** Naming (`agent/<run-id>`?), retention of
  failed branches, and when they're pruned.
- **Reuse of existing infra.** Devops pipelines already encode the verify
  commands; the lock manager already handles lineage locks — how much of the
  merge/verify queue can be built on those rather than new machinery?
- **Interim safety.** Should Option 1's "skip commit on failed status" ship
  now as a cheap stop-gap while Option 2 is designed, so the shared branch
  stops breaking in the meantime?

## Related

- [agent-auth-error-fail-fast-requeue.md](../defects/agent-auth-error-fail-fast-requeue.md)
  — sibling runner-reliability work (fail fast + re-queue on transient auth).
- [agent-sequencing-and-concurrency.md](agent-sequencing-and-concurrency.md)
  — existing thinking on running agents in parallel.
